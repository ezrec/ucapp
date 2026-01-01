package cpu

import (
	"errors"
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
)

func TestAssemblerInclude(t *testing.T) {
	assert := assert.New(t)

	asm := &Assembler{}
	asm.Clear()
	asm.FS = fstest.MapFS{
		"common.asm": &fstest.MapFile{Data: []byte("write r1 0x20\nwrite r2 0x30\n")},
	}

	program := strings.Join([]string{
		"write r0 0x10",
		".include common.asm",
		"write r3 0x40",
	}, "\n")

	err := asm.Parse(strings.NewReader(program))
	assert.NoError(err)

	prog, err := asm.Link()
	assert.NoError(err)
	assert.NotNil(prog)
	if !assert.Equal(4, len(prog.Opcodes)) {
		return
	}

	assert.Equal(0, prog.Opcodes[0].Ip)
	assert.Equal([]string{"write", "r0", "0x10"}, prog.Opcodes[0].Words)
	assert.Equal(MakeCodeAlu(COND_ALWAYS, ALU_OP_SET, IR_REG_R0, IR_IMMEDIATE_16, 0x10), prog.Opcodes[0].Codes[0])

	assert.Equal(1, prog.Opcodes[1].Ip)
	assert.Equal([]string{"write", "r1", "0x20"}, prog.Opcodes[1].Words)
	assert.Equal(MakeCodeAlu(COND_ALWAYS, ALU_OP_SET, IR_REG_R1, IR_IMMEDIATE_16, 0x20), prog.Opcodes[1].Codes[0])

	assert.Equal(2, prog.Opcodes[2].Ip)
	assert.Equal([]string{"write", "r2", "0x30"}, prog.Opcodes[2].Words)
	assert.Equal(MakeCodeAlu(COND_ALWAYS, ALU_OP_SET, IR_REG_R2, IR_IMMEDIATE_16, 0x30), prog.Opcodes[2].Codes[0])

	assert.Equal(3, prog.Opcodes[3].Ip)
	assert.Equal([]string{"write", "r3", "0x40"}, prog.Opcodes[3].Words)
	assert.Equal(MakeCodeAlu(COND_ALWAYS, ALU_OP_SET, IR_REG_R3, IR_IMMEDIATE_16, 0x40), prog.Opcodes[3].Codes[0])
}

func TestAssemblerIncludePathError(t *testing.T) {
	assert := assert.New(t)

	asm := &Assembler{}
	asm.Clear()

	err := asm.Parse(strings.NewReader(".include"))
	assert.Error(err)
	assert.ErrorIs(err, ErrIncludePath)

	var se *ErrSyntax
	assert.True(errors.As(err, &se))
	assert.Equal("stdin", se.Filename)
	assert.Equal(1, se.LineNo)
}

func TestAssemblerIncludeMissingFile(t *testing.T) {
	assert := assert.New(t)

	asm := &Assembler{}
	asm.Clear()
	asm.FS = fstest.MapFS{}

	err := asm.Parse(strings.NewReader(".include missing.asm"))
	assert.Error(err)

	var se *ErrSyntax
	assert.True(errors.As(err, &se))
	assert.Equal("stdin", se.Filename)
	assert.Equal(1, se.LineNo)

	var pe *fs.PathError
	assert.True(errors.As(err, &pe))
}

func TestAssemblerIncludeEquateScope(t *testing.T) {
	assert := assert.New(t)

	asm := &Assembler{}
	asm.Clear()
	asm.FS = fstest.MapFS{
		"defs.asm": &fstest.MapFile{Data: []byte(".equ CONST_10 0x10\n")},
	}

	err := asm.Parse(strings.NewReader(".include defs.asm\nwrite r0 CONST_10"))
	assert.NoError(err)

	prog, err := asm.Link()
	assert.NoError(err)
	assert.NotNil(prog)
	if !assert.Equal(1, len(prog.Opcodes)) {
		return
	}

	assert.Equal([]string{"write", "r0", "0x10"}, prog.Opcodes[0].Words)
	assert.Equal(MakeCodeAlu(COND_ALWAYS, ALU_OP_SET, IR_REG_R0, IR_IMMEDIATE_16, 0x10), prog.Opcodes[0].Codes[0])
}
