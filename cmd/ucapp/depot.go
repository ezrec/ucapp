package main

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	capp_io "github.com/ezrec/ucapp/io"
)

// selectDrum selects a drum of the current depot.
func selectDrum(depot *capp_io.Depot, drum uint32) (err error) {
	if drum > 0xff_ffff {
		err = fmt.Errorf("invalid drum ID 0x%06x (must be <= 0xffffff)", drum)
		return
	}

	resp := make(chan uint32, 1)
	defer close(resp)

	depot.Alert(capp_io.DEPOT_OP_SELECT|drum, resp)
	value := <-resp
	if value == ^uint32(0) {
		err = fmt.Errorf("drum 0x%06x does not exist", drum)
		return
	}

	return
}

// selectRing selects a ring of the current depot's drum, and rewinds it read index.
func selectRing(depot *capp_io.Depot, ring uint8) (err error) {
	resp := make(chan uint32, 1)
	defer close(resp)

	depot.Alert(capp_io.DEPOT_OP_DRUM|capp_io.DRUM_OP_SELECT|uint32(ring), resp)
	value := <-resp
	if value == ^uint32(0) {
		err = fmt.Errorf("ring 0x%02x does not exist", ring)
		return
	}

	depot.Alert(capp_io.DEPOT_OP_DRUM|capp_io.DRUM_OP_RING|capp_io.RING_OP_REWIND_WRITE, resp)
	value = <-resp
	if value == ^uint32(0) {
		err = fmt.Errorf("ring 0x%02x cannot be written to", ring)
		return
	}

	return
}

// CliDepot handles CLI 'depot' commands.
type CliDepot struct {
	List   CliDepotList   `cmd:"" help:"List entries in the depot"`
	Save   CliDepotSave   `cmd:"" help:"Save a file to a drum"`
	Delete CliDepotDelete `cmd:"" help:"Delete a file from a drum"`
}

// CliDepot handles CLI 'depot list' commands.
type CliDepotList struct {
	Deleted bool   `help:"Show deleted dirents as well"`
	Drum    uint32 `help:"Drum to list" default:"0xffffffff"`
}

// Run executes the 'depot list' command.
func (cmd *CliDepotList) Run(opt *Options) (err error) {
	for id, drum := range opt.Emulator.Depot.Drums {
		if cmd.Drum == ^uint32(0) || cmd.Drum == id {
			for dirent := range drum.Dirents() {
				if cmd.Deleted || !dirent.Deleted() {
					fmt.Printf("0x%06x.%02x: %v\n", id, dirent.Ring, dirent.Name)
				}
			}
		}
	}

	return
}

// CliDepotSave handles CLI 'depot save' commands.
type CliDepotSave struct {
	Drum   uint32   `help:"Drum to load into" default:"0x000000"`
	Name   string   `arg:"" help:"Name for the ring, or 0x00-0xff for a specific ring number"`
	Source *os.File `arg:"" help:"File to load (64K max)"`
}

// Run executes the 'depot save' command.
func (cmd *CliDepotSave) Run(opt *Options) (err error) {
	defer cmd.Source.Close()

	// Select drum
	err = selectDrum(&opt.Emulator.Depot, cmd.Drum)
	if err != nil {
		return
	}

	if strings.HasPrefix(cmd.Name, "0x") {
		var ring uint64
		ring, err = strconv.ParseUint(cmd.Name, 0, 8)
		if err != nil {
			return
		}

		if ring > 0xff {
			err = fmt.Errorf("ring id 0x%x invalid, must be <= 0xff", ring)
			return
		}

		err = selectRing(&opt.Emulator.Depot, uint8(ring))
		if err != nil {
			return
		}

		var content []byte
		content, err = io.ReadAll(cmd.Source)
		for _, value := range content {
			capp_io.SendAsUint8(&opt.Emulator.Depot, value)
		}
	} else {
		err = opt.Emulator.Depot.Save(cmd.Name, cmd.Source)
		if err != nil {
			return
		}
	}

	return
}

// CliDepotDelete handles 'depot delete' command.
type CliDepotDelete struct {
	Drum uint32 `help:"Drum to delete from" default:"0x000000"`
	Name string `arg:"" help:"Name of the entry to delete"`
}

// Run executes the 'depot delete' command.
func (cmd *CliDepotDelete) Run(opt *Options) (err error) {
	// Select drum
	err = selectDrum(&opt.Emulator.Depot, cmd.Drum)
	if err != nil {
		return
	}

	err = opt.Emulator.Depot.Delete(cmd.Name)
	if err != nil {
		return
	}

	return
}
