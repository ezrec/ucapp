// Copyright 2025, Jason S. McMullan <jason.mcmullan@gmail.com>

package main

import (
	"flag"
	"io"
	"io/fs"
	"log"
	"os"

	"github.com/ezrec/ucapp/cpu"
	"github.com/ezrec/ucapp/emulator"
	capp_io "github.com/ezrec/ucapp/io"
)

type createFS struct {
	Root *os.Root
}

func (cf *createFS) Mkdir(name string, mode fs.FileMode) (err error) {
	return cf.Root.Mkdir(name, mode)
}

func (cf *createFS) Create(name string) (file io.WriteCloser, err error) {
	file, err = cf.Root.Create(name)
	return
}

func (cf *createFS) Sub(name string) (sub capp_io.CreateFS, err error) {
	subroot, err := cf.Root.OpenRoot(name)
	if err != nil {
		return
	}
	sub = &createFS{Root: subroot}
	return
}

func main() {
	var compile string
	var depot_path string
	var drum int
	var ring int
	var save bool
	var exec bool
	var input string
	var output string
	var verbose bool

	flag.StringVar(&compile, "c", "", ".uc file to compile")
	flag.StringVar(&depot_path, "D", "", "Depot path (default is none)")
	flag.IntVar(&drum, "d", 0, "Drum to use (default is 0)")
	flag.IntVar(&ring, "r", 0, "Ring to use (default is 0)")
	flag.BoolVar(&save, "s", false, "Save program to ring, do not execute")
	flag.BoolVar(&exec, "x", false, "Save program to ring, then execute")
	flag.StringVar(&input, "i", "-", "Tape input")
	flag.StringVar(&output, "o", "-", "Tape output")
	flag.BoolVar(&verbose, "v", false, "Verbose mode")

	flag.Parse()

	if flag.NArg() != 0 {
		log.Fatalf("%v: Unknown arguments: %v", os.Args[0], flag.Args())
	}

	prog := &cpu.Program{}

	emu := emulator.NewEmulator()
	defer emu.Close()

	emu.Verbose = verbose

	resp := make(chan uint32, 1)
	defer close(resp)

	boot := cpu.CHANNEL_ID_TEMP

	var root *os.Root
	var err error

	if len(depot_path) != 0 {
		// Unmarshal the depot.
		root, err = os.OpenRoot(depot_path)
		if err != nil {
			log.Fatalf("depot '%v', %v", depot_path, err)
		}
		emu.Depot.Unmarshal(root.FS())

		if _, ok := emu.Depot.Drums[uint32(drum)]; !ok {
			emu.Depot.Drums[uint32(drum)] = &capp_io.Drum{}
		}

		if verbose {
			for n, drum := range emu.Depot.Drums {
				log.Printf("drum %06x:", n)
				for i, ring := range drum.Rings {
					log.Printf("  ring %02x: %v bytes", i, len(ring.Data))
				}
			}
		}

		// Select active ring
		emu.Depot.Alert(uint32(capp_io.DEPOT_OP_SELECT|drum), resp)
		var value uint32
		value = <-resp
		if value == ^uint32(0) {
			log.Fatalf("drum %v/%06x.drum missing: 0x%08x", depot_path, drum, value)
		}
		emu.Depot.Alert(uint32(capp_io.DEPOT_OP_DRUM|capp_io.DRUM_OP_SELECT|ring), resp)
		value = <-resp
		if value == ^uint32(0) {
			log.Fatalf("ring %v/%06x.drum/%02x.ring missing: 0x%08x", depot_path, drum, ring, value)
		}

		if verbose {
			log.Printf("depot: drum %06x, ring %02x", drum, ring)
		}

		boot = cpu.CHANNEL_ID_DEPOT
	}

	depot_changed := false

	// Compile a new instruction stream.
	if len(compile) != 0 {
		inf, err := os.Open(compile)
		if err != nil {
			log.Fatalf("%v: %v", compile, err)
		}
		defer inf.Close()

		asm := &cpu.Assembler{}
		for define, value := range emu.Defines() {
			asm.Predefine(define, value)
		}
		prog, err = asm.Parse(inf)
		if err != nil {
			log.Fatalf("%v: %v", compile, err)
		}

		emu.Program = prog

		// Only save.
		if save || exec {
			emu.Depot.Alert(uint32(capp_io.DEPOT_OP_DRUM|capp_io.DRUM_OP_RING|capp_io.RING_OP_REWIND_WRITE), resp)
			value := <-resp
			if value == ^uint32(0) {
				log.Fatalf("ring %v/%x.drum/%x.ring corrupted", depot_path, drum, ring)
			}
			for _, item := range prog.Binary() {
				capp_io.SendAsUint32(&emu.Depot, item)
			}

			depot_changed = true
		} else {
			boot = cpu.CHANNEL_ID_MONITOR
		}
	}

	if !save {
		if input == "-" {
			emu.Tape.Input = os.Stdin
		} else {
			inf, err := os.Open(input)
			if err != nil {
				log.Fatalf("%v: %v", input, err)
			}
			defer inf.Close()
			emu.Tape.Input = inf
		}

		if output == "-" {
			emu.Tape.Output = os.Stdout
		} else {
			ouf, err := os.Create(output)
			if err != nil {
				log.Fatalf("%v: %v", output, err)
			}
			defer ouf.Close()
			emu.Tape.Output = ouf
		}

		if verbose {
			log.Printf("emu: reset, boot from %v", boot)
		}

		err = emu.Reset(boot)
		if err != nil {
			log.Fatal(err)
		}

		depot_changed = true

		for done, err := emu.Tick(); !done; done, err = emu.Tick() {
			if err != nil {
				log.Fatal(err)
			}
		}
	}

	if len(depot_path) > 0 && depot_changed {
		if verbose {
			log.Printf("saving depot state")
		}
		cfs := &createFS{Root: root}
		err = emu.Depot.Marshal(cfs)
		if err != nil {
			log.Fatalf("depot '%v', %v", depot_path, err)
		}
	}
}
