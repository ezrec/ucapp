package emulator

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ezrec/ucapp/cpu"
)

func TestEmulator(t *testing.T) {
	assert := assert.New(t)

	emu := NewEmulator()

	assert.False(emu.Verbose)
	assert.NotNil(emu.Cpu.Capp)
}

func doRunSingle(emu *Emulator, program []string, input []byte, t *testing.T) (output []byte) {
	assert := assert.New(t)

	asm := &cpu.Assembler{}
	prog, err := asm.Parse(strings.NewReader(strings.Join(program, "\n")))
	assert.NoError(err)
	emu.Program = prog

	err = emu.Reset()
	assert.NoError(err)

	emu.Tape.Input = bytes.NewReader(input)
	tape_output := &bytes.Buffer{}
	emu.Tape.Output = tape_output

	for _, op := range prog.Opcodes {
		assert.Equal(emu.LineNo(), op.LineNo)
		here := program[emu.LineNo()-1]
		for c := range len(op.Codes) {
			assert.Equal(emu.Cpu.Ip, uint32(op.Ip+c), here)
			debug := emu.Program.Debug(uint16(emu.Cpu.Ip))
			done, err := emu.Tick()
			assert.NoError(err)
			if err != nil {
				t.Log(emu.Cpu.String())
				t.Fatalf("%v", err)
			}
			assert.Equal(debug.Codes[debug.Index], op.Codes[c])
			assert.NoError(err, here)
			assert.False(done, here)
		}
	}
	done, err := emu.Tick()
	assert.NoError(err)
	assert.True(done)

	output = tape_output.Bytes()
	return
}

func doRunBranch(emu *Emulator, program []string, input []byte, t *testing.T) (output []byte) {
	assert := assert.New(t)

	asm := &cpu.Assembler{}
	prog, err := asm.Parse(strings.NewReader(strings.Join(program, "\n")))
	assert.NoError(err)
	emu.Program = prog

	err = emu.Reset()
	assert.NoError(err)

	emu.Tape.Input = bytes.NewReader(input)
	tape_output := &bytes.Buffer{}
	emu.Tape.Output = tape_output

	var done bool
	for !done {
		line := emu.LineNo()
		if line == 0 {
			line = 1
		}
		done, err = emu.Tick()
		here := program[line-1]
		assert.NoError(err, here)
		if err != nil {
			t.Fatal(err)
		}
	}

	output = tape_output.Bytes()
	return
}

func TestEmulatorRegisters(t *testing.T) {
	assert := assert.New(t)

	emu := NewEmulator()

	program := []string{
		"list of ARENA_FREE ARENA_MASK",
		"list all",
		"fetch tape 0xffff",
		"list not",
		"write list ARENA_IO 0xffff0000",
		"list of $(ARENA_IO | 0x123) $(ARENA_MASK | 0x7ff)", // match & mask
		"write r0 0x10", // r0
		"write r1 0x20", // r1
		"write r2 0x30", // r2
		"write r3 0x40", // r3
		"list all",      // first = 0x123, count = 2
		"store tape 0xffff",
		"list not",
	} // ip = 6

	input := []uint8{0x23, 0x00, 0x23, 0x01, 0x23, 0x09}
	output := doRunSingle(emu, program, input, t)

	assert.Equal(uint32(cpu.ARENA_IO|0x123), emu.Cpu.Match)
	assert.Equal(uint32(cpu.ARENA_MASK|0x7ff), emu.Cpu.Mask)
	assert.Equal(uint32(0x123|cpu.ARENA_IO), emu.Cpu.Capp.First())
	assert.Equal(uint(2), emu.Cpu.Capp.Count())
	assert.Equal(uint32(0x10), emu.Cpu.Register[0])
	assert.Equal(uint32(0x20), emu.Cpu.Register[1])
	assert.Equal(uint32(0x30), emu.Cpu.Register[2])
	assert.Equal(uint32(0x40), emu.Cpu.Register[3])
	assert.Equal([]uint8{0x23, 0x01, 0x23, 0x09}, output)
}

func TestEmulatorAlu(t *testing.T) {
	assert := assert.New(t)

	emu := NewEmulator()
	program := []string{
		"write r1 0x10", // r0
		"alu add r1 1",
		"alu xor r0 r0",
		"alu sub r0 r1",
		"write r1 0x200", // r1
		"alu xor r1 r0",
		"alu and r1 0xf",
		"alu shl r1 2",
		"alu and r1 ~0x3",
		"alu and r1 0x20",
		"write r2 0x100", // r2
		"alu or r2 0x200",
		"alu shr r2 4",
		"alu and r2 ~0xf000_0000",
		"write r3 0x40", // r3
	} // ip = 6

	doRunSingle(emu, program, []byte{}, t)

	neg := func(v int32) uint32 { return uint32(-v) }

	assert.Equal(neg(0x11), emu.Cpu.Register[0])
	assert.Equal(uint32(0x20), emu.Cpu.Register[1])
	assert.Equal(uint32(0x30), emu.Cpu.Register[2])
	assert.Equal(uint32(0x40), emu.Cpu.Register[3])
}

func TestEmulatorEqu(t *testing.T) {
	assert := assert.New(t)

	emu := NewEmulator()
	program := []string{
		".equ CONST_10 0x10",
		"write r0 CONST_10",               // r0
		"write r1 $(CONST_10 + CONST_10)", // r1
		".equ CONST_30 $(2 * CONST_10 + CONST_10)",
		"write r2 CONST_30",
		"write r3 $(LINENO * 8 + 0x10)", // r3
	} // ip = 4

	doRunSingle(emu, program, []byte{}, t)

	assert.Equal(uint32(0x10), emu.Cpu.Register[0])
	assert.Equal(uint32(0x20), emu.Cpu.Register[1])
	assert.Equal(uint32(0x30), emu.Cpu.Register[2])
	assert.Equal(uint32(0x40), emu.Cpu.Register[3])
}

func TestEmulatorMacro(t *testing.T) {
	assert := assert.New(t)

	emu := NewEmulator()
	program := []string{
		".macro SETADD rn a b",
		"write rn a",
		"alu add rn b",
		".endm",
		"SETADD r0 8 8",
		".equ CONST_10 0x10",
		"SETADD r1 CONST_10 CONST_10",
		"SETADD r2 $(CONST_10 + CONST_10) r0",
		"SETADD r3 r2 r0",
	} // ip = 6

	doRunSingle(emu, program, []byte{}, t)

	assert.Equal(uint32(0x10), emu.Cpu.Register[0])
	assert.Equal(uint32(0x20), emu.Cpu.Register[1])
	assert.Equal(uint32(0x30), emu.Cpu.Register[2])
	assert.Equal(uint32(0x40), emu.Cpu.Register[3])
}

func TestEmulatorLabel(t *testing.T) {
	assert := assert.New(t)

	emu := NewEmulator()
	program := []string{
		"jump R0",
		"AddOneToR0:",
		"alu add r0 1",
		"return",
		"R1: write r1 0x20",
		"jump R2",
		"R0: AND_ALSO:",
		"write r0 0x10",
		"jump R1",
		"R2:",
		"call AddOneToR0",
		"call AddOneToR0",
		"",
		"write r2 0x30",
		"write r3 0x40",
	}

	doRunBranch(emu, program, []byte{}, t)

	assert.Equal(uint32(0x12), emu.Cpu.Register[0])
	assert.Equal(uint32(0x20), emu.Cpu.Register[1])
	assert.Equal(uint32(0x30), emu.Cpu.Register[2])
	assert.Equal(uint32(0x40), emu.Cpu.Register[3])
}

func TestEmulatorSystemMacro(t *testing.T) {
	assert := assert.New(t)

	emu := NewEmulator()
	program := []string{
		"list of ARENA_FREE ARENA_MASK",
		"list all",
		"fetch tape 0xffff",
		"list not",
		"write list ARENA_IO 0xffff0000",
		"write r0 0x10",
		"write r1 0x20",
		"write r2 0x30",
		"write r3 0x40",
		"; this does not count as an opcode.",
		"list of $(ARENA_IO | 0xf000) $(ARENA_MASK | 0xf000)",
		"write list ARENA_IO ARENA_MASK",
		"list of ARENA_IO ARENA_MASK",
		"list all",
		"store tape 0xffff",
		"list not",
	}

	input := []uint8{0xa0, 0x01, 0xb0, 0x02, 0xc0, 0x03, 0xd0, 0x04}

	output := doRunSingle(emu, program, input, t)

	assert.Equal(uint32(0x10), emu.Cpu.Register[0])
	assert.Equal(uint32(0x20), emu.Cpu.Register[1])
	assert.Equal(uint32(0x30), emu.Cpu.Register[2])
	assert.Equal(uint32(0x40), emu.Cpu.Register[3])

	assert.Equal([]uint8{0xa0, 0x01, 0xb0, 0x02, 0xc0, 0x03, 0xd0, 0x04}, output)
}

func TestEmulatorTemp(t *testing.T) {
	assert := assert.New(t)

	emu := NewEmulator()
	program := []string{
		"list of ARENA_FREE ARENA_MASK",
		"list all",
		"fetch tape 0xffff",
		"list not",
		"write list ARENA_IO 0xffff0000",
		"list of ARENA_IO ARENA_MASK",
		"store temp 0xffff", // store to temp
		"list not",
		"write list ARENA_FREE ARENA_MASK",
		"fetch temp 0xffff",
		"list not",
		"write list 0x9000 0xf000",
		"store tape 0xffff",
		"list not",
	}

	input := []uint8{0x34, 0x12, 0x78, 0x56, 0xcd, 0xab}

	output := doRunSingle(emu, program, input, t)

	assert.Equal([]uint8{0x34, 0x92, 0x78, 0x96, 0xcd, 0x9b}, output)
}
