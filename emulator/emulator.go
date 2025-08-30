// Copyright 2024, Jason S. McMullan <jason.mcmullan@gmail.com>

package emulator

import (
	"errors"
	"log"

	"github.com/ezrec/ucapp/channel"
	"github.com/ezrec/ucapp/cpu"
)

const (
	CAPP_TICK_COST = 1
	ALU_TICK_COST  = 4
	STACK_LIMIT    = 16
	CAPP_SIZE      = 8192 // 4K for program text, 1K for compiled, 3K for work
)

type Emulator struct {
	Verbose bool
	*cpu.Cpu
	Program *cpu.Program

	Temporary channel.Temporary
	Tape      channel.Tape
	Depot     channel.Depot
	Rom       channel.Rom
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

	return
}

// Reset the assembler state
func (emu *Emulator) Reset() (err error) {
	cp := emu.Cpu.Capp

	emu.Cpu.Verbose = false

	if emu.Verbose {
		log.Printf("reset")
	}
	emu.Rom.Data = emu.Program.Binary()

	err = emu.Cpu.Reset()
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
