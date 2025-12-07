// Copyright 2025, Jason S. McMullan <jason.mcmullan@gmail.com>

package main

import (
	"log"
	"os"

	"github.com/ezrec/ucapp/cpu"
	capp_io "github.com/ezrec/ucapp/io"
)

type CliRun struct {
	Drum   uint32 `help:"Drum in the depot to run (default is 0x000000)"`
	Ring   uint8  `help:"Ring in the drum to run (default is 0x00)"`
	Input  string `help:"Tape input" default:"-"`
	Output string `help:"Tape input" default:"-"`
}

func (cr *CliRun) Run(opt *Options) (err error) {
	emu := opt.Emulator

	resp := make(chan uint32, 1)
	defer close(resp)

	// Select active ring
	emu.Depot.Alert(uint32(capp_io.DEPOT_OP_SELECT|cr.Drum), resp)
	var value uint32
	value = <-resp
	if value == ^uint32(0) {
		log.Fatalf("drum %06x.drum missing: 0x%08x", cr.Drum, value)
	}
	emu.Depot.Alert(capp_io.DEPOT_OP_DRUM|capp_io.DRUM_OP_SELECT|uint32(cr.Ring), resp)
	value = <-resp
	if value == ^uint32(0) {
		log.Fatalf("ring %06x.drum/%02x.ring missing: 0x%08x", cr.Drum, cr.Ring, value)
	}

	boot := cpu.CHANNEL_ID_DEPOT

	if cr.Input == "-" {
		emu.Tape.Input = os.Stdin
	} else {
		inf, err := os.Open(cr.Input)
		if err != nil {
			log.Fatalf("%v: %v", cr.Input, err)
		}
		defer inf.Close()
		emu.Tape.Input = inf
	}

	if cr.Output == "-" {
		emu.Tape.Output = os.Stdout
	} else {
		ouf, err := os.Create(cr.Output)
		if err != nil {
			log.Fatalf("%v: %v", cr.Output, err)
		}
		defer ouf.Close()
		emu.Tape.Output = ouf
	}

	err = emu.Reset(boot)
	if err != nil {
		log.Fatal(err)
	}

	for done, err := emu.Tick(); !done; done, err = emu.Tick() {
		if err != nil {
			log.Fatal(err)
		}
	}

	if opt.Verbose {
		for n := range 6 {
			log.Printf("r%v: 0x%08x", n, emu.Cpu.Register[n])
		}
		for !emu.Cpu.Stack.Empty() {
			val, _ := emu.Cpu.Stack.Pop()
			log.Printf("stack: 0x%08x", val)
		}

		for val := range capp_io.ReceiveAsUint8(&emu.Temporary) {
			log.Printf("temp: 0x%02x", val)
		}
	}

	return
}
