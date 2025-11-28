// Copyright 2025, Jason S. McMullan <jason.mcmullan@gmail.com>

package main

import (
	"flag"
	"log"
	"os"

	"github.com/ezrec/ucapp/cpu"
	"github.com/ezrec/ucapp/emulator"
)

func main() {
	var compile string
	var ring string
	var drum string
	var save bool
	var input string
	var output string
	var verbose bool

	flag.StringVar(&compile, "c", "", ".uc file to compile")
	flag.StringVar(&ring, "r", "", ".ring file to use")
	flag.StringVar(&drum, "d", "", ".drum file to use")
	flag.BoolVar(&save, "s", false, "Save CAPP to ring, do not execute")
	flag.StringVar(&input, "i", "-", "Tape input")
	flag.StringVar(&output, "o", "-", "Tape output")
	flag.BoolVar(&verbose, "v", false, "Verbose mode")

	flag.Parse()

	if flag.NArg() != 0 {
		log.Fatalf("%v: Unknown arguments: %v", os.Args[0], flag.Args())
	}

	prog := &cpu.Program{}

	// Compile a new instruction stream.
	if len(compile) != 0 {
		inf, err := os.Open(compile)
		if err != nil {
			log.Fatalf("%v: %v", compile, err)
		}
		defer inf.Close()

		asm := &cpu.Assembler{}
		prog, err = asm.Parse(inf)
		if err != nil {
			log.Fatalf("%v: %v", compile, err)
		}
	}

	if !save {
		emu := emulator.NewEmulator()
		emu.Program = prog
		emu.Verbose = verbose

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

		emu.Reset()
		for done, err := emu.Tick(); !done; done, err = emu.Tick() {
			if err != nil {
				log.Fatal(err)
			}
		}
	}
}
