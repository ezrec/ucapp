// Copyright 2025, Jason S. McMullan <jason.mcmullan@gmail.com>

package main

import (
	"log"
	"os"
	"strings"

	"github.com/ezrec/ucapp/cpu"
	"github.com/ezrec/ucapp/sio"
)

type CliBuild struct {
	Output string   `help:"Output file name. Default is <source>.ur"`
	Source *os.File `arg:"" help:"Source file (*.uc) to compile"`
}

func (cb *CliBuild) Run(opt *Options) (err error) {
	// Create an emulator (to get defines from)
	emu := opt.Emulator

	// Compile a new instruction stream.
	defer cb.Source.Close()

	asm := &cpu.Assembler{}
	for define, value := range emu.Defines() {
		asm.Predefine(define, value)
	}

	asm.Clear()
	err = asm.Parse(cb.Source)
	if err != nil {
		log.Fatalf("%v: %v", cb.Source.Name(), err)
	}
	prog, err := asm.Link()
	if err != nil {
		log.Fatalf("%v: %v", cb.Source.Name(), err)
	}

	if len(cb.Output) == 0 {
		cb.Output = strings.TrimSuffix(cb.Source.Name(), ".uc") + ".ur"
	}

	output, err := os.Create(cb.Output)
	if err != nil {
		return
	}
	defer output.Close()

	ring := &sio.Ring{}
	ring.Rewind()
	for _, item := range prog.Binary() {
		sio.SendAsUint32(ring, item)
	}

	err = ring.Marshal(output)
	if err != nil {
		return
	}

	return
}
