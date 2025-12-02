// Copyright 2024, Jason S. McMullan <jason.mcmullan@gmail.com>

package emulator

import (
	"errors"
	"fmt"
	"iter"
	"maps"

	"github.com/ezrec/ucapp/cpu"
	"github.com/ezrec/ucapp/internal"
	"github.com/ezrec/ucapp/io"
)

const (
	CAPP_TICK_COST = 1    // Cost of a single CAPP tick.
	ALU_TICK_COST  = 4    // Cost of an ALU tick.
	CAPP_SIZE      = 8192 // 4K for program text, 1K for compiled, 3K for work
)

var _emulator_defines = map[string]string{
	"CAPP_SIZE": fmt.Sprintf("%v", CAPP_SIZE),
}

// Emulator state. CPU + CAPP + IO channels.
type Emulator struct {
	Verbose  bool         // If set, enables verbose logging.
	*cpu.Cpu              // Reference to the CPU simulation.
	Program  *cpu.Program // Reference to the currently running program listing.

	Temporary io.Temporary // Temporary buffer IO channel.
	Tape      io.Tape      // Tape IO channel.
	Depot     io.Depot     // Depot (Drum and Ring) IO channel.
	Rom       io.Rom       // ROM IO channel.

	TrapRequest chan uint32
}

// NewEmulator creates a new emulator.
func NewEmulator() (emu *Emulator) {
	emu = &Emulator{
		Cpu:     cpu.NewCpu(CAPP_SIZE),
		Program: &cpu.Program{},
	}

	emu.Temporary.Capacity = 8192

	emu.Cpu.SetChannel(cpu.CHANNEL_ID_TEMP, &emu.Temporary)
	emu.Cpu.SetChannel(cpu.CHANNEL_ID_MONITOR, &emu.Rom)
	emu.Cpu.SetChannel(cpu.CHANNEL_ID_TAPE, &emu.Tape)
	emu.Cpu.SetChannel(cpu.CHANNEL_ID_DEPOT, &emu.Depot)

	// Map the trap channel
	_, emu.TrapRequest, _ = emu.Cpu.GetChannel(cpu.CHANNEL_ID_MONITOR)
	emu.Rom.Alert(io.ROM_OP_TRAP, emu.TrapRequest)

	return
}

// Defines returns an iterator over all of the defines
func (emu *Emulator) Defines() iter.Seq2[string, string] {
	return internal.IterSeq2Concat(maps.All(_emulator_defines),
		emu.Cpu.Defines(),
		emu.Temporary.Defines(),
		emu.Rom.Defines(),
		emu.Tape.Defines(),
		emu.Depot.Defines(),
	)
}

// Close the emulator
func (emu *Emulator) Close() (err error) {
	emu.Cpu.Close()

	return
}

// Reset the assembler state
func (emu *Emulator) Reset(boot cpu.CodeChannel) (err error) {
	cp := emu.Cpu.Capp

	emu.Cpu.Verbose = false

	emu.Rom.Data = emu.Program.Binary()

	err = emu.Cpu.Reset(boot)
	if err != nil {
		return
	}

	// Tick till out of the boot rom
	for (emu.Cpu.Ip & cpu.IP_MODE_MASK) != cpu.IP_MODE_CAPP {
		err = emu.Cpu.Tick()
		if err != nil {
			return
		}
	}

	// Reset power stats.
	cp.BitsFlipped = 0

	emu.Cpu.Verbose = emu.Verbose

	return
}

// Ticks returns the total ticks since a reset.
func (emu *Emulator) Ticks() int {
	return emu.Cpu.Ticks
}

// Power returns the total power consumed.
func (emu *Emulator) Power() int {
	return emu.Cpu.Power
}

// Ip returns current instruction pointer.
func (emu *Emulator) Ip() int {
	return int(emu.Cpu.Ip)
}

// Code returns the current instruction code.
func (emu *Emulator) Code() cpu.Code {
	for ip, code := range emu.Program.Codes() {
		if uint16(emu.Cpu.Ip & ^cpu.IP_MODE_MASK) == ip {
			return code
		}
	}

	return cpu.Code{}
}

// LineNo returns the current line number for the executing opcode.
func (emu *Emulator) LineNo() int {
	for _, op := range emu.Program.Opcodes {
		if int(emu.Cpu.Ip) >= op.Ip && int(emu.Cpu.Ip) < (op.Ip+len(op.Codes)) {
			return op.LineNo
		}
	}

	return 0
}

// Tick performs a single tick of the emulator.
func (emu *Emulator) Tick() (done bool, err error) {
	// Set CPU verbosity
	emu.Cpu.Verbose = emu.Verbose

	lineno := emu.LineNo()
	defer func() {
		if err != nil {
			err = &ErrRuntime{LineNo: lineno, Err: err}
		}
	}()

	// Tick past boot code.
	for {
		err = emu.Cpu.Tick()
		if errors.Is(err, cpu.ErrIpEmpty) {
			err = nil
			done = true
			return
		}
		if err != nil {
			return
		}
		if (emu.Cpu.Ip & cpu.IP_MODE_MASK) != cpu.IP_MODE_REG {
			break
		}
	}

	return
}
