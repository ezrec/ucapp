package io

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

const (
	// DEPOT_OP_MASK masks the depot operation type from an alert request.
	DEPOT_OP_MASK = (1 << 23)
	// DEPOT_OP_SELECT indicates a drum selection operation.
	DEPOT_OP_SELECT = (0 << 23)
	// DEPOT_OP_DRUM indicates a drum-level operation.
	DEPOT_OP_DRUM = (1 << 23)
	// DEPOT_OP_SELECT_MASK masks the drum ID from a select operation.
	DEPOT_OP_SELECT_MASK = ((1 << 23) - 1)
)

// Depot represents a collection of drums providing persistent storage.
// It implements the Channel interface and manages multiple Drum instances,
// allowing selection between them via Alert operations.
type Depot struct {
	*Drum
	Drums map[uint32](*Drum)
}

var _ Channel = &Depot{}

// Unmarshal loads depot data from a file system by scanning for drum directories
// matching the pattern XXXXXX.drum (6 hex digits).
func (depot *Depot) Unmarshal(filesys fs.FS) (err error) {
	return fs.WalkDir(filesys, ".", func(path string, d fs.DirEntry, err_in error) (err error) {
		if !d.IsDir() {
			return
		}
		name := d.Name()
		ok, err := regexp.MatchString("(?i)[0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f].drum", name)
		if err != nil {
			return
		}
		if !ok {
			return
		}
		drum_index, err := strconv.ParseUint(strings.TrimSuffix(name, filepath.Ext(name)), 16, 24)
		if err != nil {
			return
		}

		// Ensure the drum exists, and unmarshal it.
		if depot.Drums == nil {
			depot.Drums = make(map[uint32](*Drum))
		}

		drum := &Drum{}
		depot.Drums[uint32(drum_index)] = drum
		subsys, err := fs.Sub(filesys, name)
		if err != nil {
			return
		}
		drum.Unmarshal(subsys)

		return
	})
}

// Marshal writes the depot's drums to a file system, creating directories
// named XXXXXX.drum for each drum.
func (depot *Depot) Marshal(filesys CreateFS) (err error) {
	for index, drum := range depot.Drums {
		var subsys CreateFS
		drum_name := fmt.Sprintf("%06x.drum", index)
		subsys, err = filesys.Sub(drum_name)
		if err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				return
			}
			// Create the directory
			err = filesys.Mkdir(drum_name, 0755)
			if err != nil {
				return
			}
			subsys, err = filesys.Sub(drum_name)
			if err != nil {
				return
			}
		}

		err = drum.Marshal(subsys)
		if err != nil {
			return
		}
	}

	return
}

// Rewind resets all drums in the depot to their initial positions.
func (depot *Depot) Rewind() {
	for _, drum := range depot.Drums {
		drum.Rewind()
	}
}

// Alert handles depot control operations including drum selection and
// forwarding drum-specific operations to the currently selected drum.
func (depot *Depot) Alert(request uint32, response chan uint32) {
	if depot == nil {
		response <- ^uint32(0)
		return
	}

	switch request & DEPOT_OP_MASK {
	case DEPOT_OP_SELECT:
		drum_id := request & DEPOT_OP_SELECT_MASK
		drum, ok := depot.Drums[drum_id]
		if !ok {
			depot.Drum = nil
			response <- ^uint32(0)
		} else {
			depot.Drum = drum
			response <- uint32(0)
		}
	case DEPOT_OP_DRUM:
		depot.Drum.Alert(request, response)
	}
}
