// Copyright 2025, Jason S. McMullan <jason.mcmullan@gmail.com>

package main

import (
	"io"
	"io/fs"
	"log"
	"os"

	"github.com/ezrec/ucapp/emulator"
	capp_io "github.com/ezrec/ucapp/io"

	"github.com/alecthomas/kong"
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

type Cli struct {
	Verbose   bool   `help:"Enter vebose mode"`
	DepotPath string `help:"Path to the depot to use." name:"depot" default:"depot/"`

	Build CliBuild `cmd:"" help:"Build a ucapp program"`
	Depot CliDepot `cmd:"" help:"Manage the drum depot"`
	Run   CliRun   `cmd:"" help:"Run a ucapp program in the emulator"`
}

type Options struct {
	Verbose bool

	Emulator *emulator.Emulator
}

func main() {
	var err error

	var cli Cli
	ctx := kong.Parse(&cli)

	emu := emulator.NewEmulator()
	defer emu.Close()

	emu.Verbose = cli.Verbose

	var root *os.Root
	if len(cli.DepotPath) != 0 {
		// Unmarshal the depot.
		root, err = os.OpenRoot(cli.DepotPath)
		if err != nil {
			log.Fatalf("depot '%v', %v", cli.DepotPath, err)
		}
		emu.Depot.Unmarshal(root.FS())
	}

	opt := Options{
		Verbose:  cli.Verbose,
		Emulator: emu,
	}

	err = ctx.Run(&opt)
	if err != nil {
		log.Fatal(err)
	}

	if len(cli.DepotPath) != 0 && emu.Depot.Dirty() {
		cfs := &createFS{Root: root}
		err = emu.Depot.Marshal(cfs)
		if err != nil {
			log.Fatalf("depot '%v', %v", cli.Depot, err)
		}
	}
}
