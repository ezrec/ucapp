package cpu

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAssembler(t *testing.T) {
	assert := assert.New(t)

	asm := &Assembler{}

	prog, err := asm.Parse(strings.NewReader(""))
	assert.NoError(err)
	assert.Equal(0, len(prog.Opcodes))

	assert.Equal("0", asm.Equate["LINENO"])
	assert.Equal(fmt.Sprintf("%#v", ARENA_MASK), asm.Equate["ARENA_MASK"])
	assert.Equal(fmt.Sprintf("%#v", ARENA_CODE), asm.Equate["ARENA_CODE"])
	assert.Equal(fmt.Sprintf("%#v", ARENA_IO), asm.Equate["ARENA_IO"])
	assert.Equal(fmt.Sprintf("%#v", ARENA_FREE), asm.Equate["ARENA_FREE"])
	assert.Equal(fmt.Sprintf("%#v", CAPP_SIZE), asm.Equate["CAPP_SIZE"])
}

func opEqual(t *testing.T, expected, opcodes []Opcode) {
	assert := assert.New(t)

	assert.Equal(len(expected), len(opcodes))
	if len(expected) == len(opcodes) {
		for n := range len(expected) {
			assert.Equal(expected[n], opcodes[n])
		}
	}
}

func TestAssemblerIo(t *testing.T) {
	assert := assert.New(t)

	asm := &Assembler{}

	program := []string{
		"? trap",
	}

	prog, err := asm.Parse(strings.NewReader(strings.Join(program, "\n")))
	assert.NoError(err)
	if err != nil {
		t.Fatal(err)
		return
	}

	expected := []Opcode{
		{1, 0, []string{"?", "trap"}, []Code{0x4_dfce}, ""},
	}

	opEqual(t, expected, prog.Opcodes)
}

func TestAssemblerRegisters(t *testing.T) {
	assert := assert.New(t)

	asm := &Assembler{}

	program := []string{
		"list of 0x123 0x7ff", // match & mask
		"write r0 0x10",       // r0
		"write r1 0x20",       // r1
		"write r2 0x30",       // r2
		"write r3 0x40",       // r3
		"list all",            // first = 0x123, count = 2
	} // ip = 6

	prog, err := asm.Parse(strings.NewReader(strings.Join(program, "\n")))
	assert.NoError(err)
	if err != nil {
		t.Fatal(err)
		return
	}

	expected := []Opcode{
		{1, 0, []string{"list", "of", "0x123", "0x7ff"}, []Code{
			0x1_07ff, 0x1_0123, 0x0_a8ff}, ""},
		{2, 3, []string{"write", "r0", "0x10"}, []Code{
			0x1_0010, 0x0_00fe}, ""},
		{3, 5, []string{"write", "r1", "0x20"}, []Code{
			0x1_0020, 0x0_01fe}, ""},
		{4, 7, []string{"write", "r2", "0x30"}, []Code{
			0x1_0030, 0x0_02fe}, ""},
		{5, 9, []string{"write", "r3", "0x40"}, []Code{
			0x1_0040, 0x0_03fe}, ""},
		{6, 11, []string{"list", "all"}, []Code{0x0_88cc}, ""},
	}

	opEqual(t, expected, prog.Opcodes)
}

func TestAssemblerAlu(t *testing.T) {
	assert := assert.New(t)

	asm := &Assembler{}
	program := []string{
		"write r0 0x10", // r0
		"alu add r0 1",
		"alu sub r0 1",
		"write r1 0x200", // r1
		"alu xor r1 r0",
		"alu and r1 0xf",
		"alu shl r1 2",
		"alu and r1 0x20",
		"write r2 0x100", // r2
		"alu or r2 0x200",
		"alu shr r2 4",
		"write r3 0x40", // r3
	} // ip = 6

	prog, err := asm.Parse(strings.NewReader(strings.Join(program, "\n")))
	assert.NoError(err)
	if err != nil {
		t.Fatal(err)
		return
	}

	expected := []Opcode{
		{1, 0, []string{"write", "r0", "0x10"}, []Code{0x1_0010, 0x0_00fe}, ""},
		{2, 2, []string{"alu", "add", "r0", "1"}, []Code{0x0_30de}, ""},
		{3, 3, []string{"alu", "sub", "r0", "1"}, []Code{0x0_38de}, ""},
		{4, 4, []string{"write", "r1", "0x200"}, []Code{0x1_0200, 0x0_01fe}, ""},
		{5, 6, []string{"alu", "xor", "r1", "r0"}, []Code{0x0_090e}, ""},
		{6, 7, []string{"alu", "and", "r1", "0xf"}, []Code{0x1_000f, 0x0_11fe}, ""},
		{7, 9, []string{"alu", "shl", "r1", "2"}, []Code{0x1_0002, 0x0_21fe}, ""},
		{8, 11, []string{"alu", "and", "r1", "0x20"}, []Code{0x1_0020, 0x0_11fe}, ""},
		{9, 13, []string{"write", "r2", "0x100"}, []Code{0x1_0100, 0x0_02fe}, ""},
		{10, 15, []string{"alu", "or", "r2", "0x200"}, []Code{0x1_0200, 0x0_1afe}, ""},
		{11, 17, []string{"alu", "shr", "r2", "4"}, []Code{0x1_0004, 0x0_2afe}, ""},
		{12, 19, []string{"write", "r3", "0x40"}, []Code{0x1_0040, 0x0_03fe}, ""},
	}

	opEqual(t, expected, prog.Opcodes)
}

func TestAssemblerEqu(t *testing.T) {
	assert := assert.New(t)

	asm := &Assembler{}
	program := []string{
		".equ CONST_10 0x10",
		"write r0 CONST_10",               // r0
		"write r1 $(CONST_10 + CONST_10)", // r1
		".equ CONST_30 $(2 * CONST_10 + CONST_10)",
		"write r2 CONST_30",
		"write r3 $(LINENO * 8 + 0x10)", // r3
	} // ip = 4

	prog, err := asm.Parse(strings.NewReader(strings.Join(program, "\n")))
	assert.NoError(err)
	if err != nil {
		t.Fatal(errors.Unwrap(err))
	}

	assert.Equal(4, len(prog.Opcodes))
}

func TestAssemblerMacro(t *testing.T) {
	assert := assert.New(t)

	asm := &Assembler{}
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
		".macro NESTED VALUE",
		"SETADD r0 VALUE $(~VALUE)",
		"SETADD r1 $(~VALUE) VALUE",
		".endm",
		"NESTED 0",
	} // ip = 6

	prog, err := asm.Parse(strings.NewReader(strings.Join(program, "\n")))
	assert.NoError(err)
	if err != nil {
		log.Fatal(err)
	}

	expected := []Opcode{
		{2, 0, []string{"write", "r0", "8"}, []Code{0x1_0008, 0x0_00fe}, ""},
		{3, 2, []string{"alu", "add", "r0", "8"}, []Code{0x1_0008, 0x0_30fe}, ""},
		{2, 4, []string{"write", "r1", "0x10"}, []Code{0x1_0010, 0x0_01fe}, ""},
		{3, 6, []string{"alu", "add", "r1", "0x10"}, []Code{0x1_0010, 0x0_31fe}, ""},
		{2, 8, []string{"write", "r2", "0x20"}, []Code{0x1_0020, 0x0_02fe}, ""},
		{3, 10, []string{"alu", "add", "r2", "r0"}, []Code{0x0_320e}, ""},
		{2, 11, []string{"write", "r3", "r2"}, []Code{0x0_032e}, ""},
		{3, 12, []string{"alu", "add", "r3", "r0"}, []Code{0x0_330e}, ""},
		{2, 13, []string{"write", "r0", "0"}, []Code{0x0_00ce}, ""},
		{3, 14, []string{"alu", "add", "r0", "0xffffffff"}, []Code{0x0_30ee}, ""},
		{2, 15, []string{"write", "r1", "0xffffffff"}, []Code{0x0_01ee}, ""},
		{3, 16, []string{"alu", "add", "r1", "0"}, []Code{0x0_31ce}, ""},
	}

	opEqual(t, expected, prog.Opcodes)
}

func TestAssemblerLabel(t *testing.T) {
	assert := assert.New(t)

	asm := &Assembler{}
	program := []string{
		"jump R0",
		"R1: write r1 0x20",
		"jump R2",
		"R0: AND_ALSO:",
		"write r0 0x10",
		"jump R1",
		"R2:",
		"",
		"write r2 0x30",
		"write r3 0x40",
	}

	prog, err := asm.Parse(strings.NewReader(strings.Join(program, "\n")))
	assert.NoError(err)

	assert.Equal(7, len(prog.Opcodes))
}

func TestAssemblerCall(t *testing.T) {
	assert := assert.New(t)

	asm := &Assembler{}
	program := []string{
		"call FUNC",
		"jump EXIT",
		"FUNC:",
		"vcall 0x1234",
		"return",
		"EXIT:",
		"exit",
	}

	prog, err := asm.Parse(strings.NewReader(strings.Join(program, "\n")))
	assert.NoError(err)

	expected := []Opcode{
		{1, 0, []string{"call", "FUNC"}, []Code{0x1_0006, 0x07de, 0x376e, 0x06fe}, "FUNC"},
		{2, 4, []string{"jump", "EXIT"}, []Code{0x1_000b, 0x06fe}, "EXIT"},
		{4, 6, []string{"vcall", "0x1234"}, []Code{0x1_1234, 0x0_07de, 0x0_376e, 0x0_06fe}, ""},
		{5, 10, []string{"return"}, []Code{0x067e}, ""},
		{7, 11, []string{"exit"}, []Code{0x6ee}, ""},
	}

	opEqual(t, expected, prog.Opcodes)

}

func TestAssemblerSystemMacro(t *testing.T) {
	assert := assert.New(t)

	asm := &Assembler{}
	program := []string{
		"fetch 0",
		"write r0 0x10",
		"write r1 0x20",
		"write r2 0x30",
		"write r3 0x40",
		"; this does not count as an opcode.",
		"list of 0xf000 0xf000",
		"write list ARENA_IO ARENA_MASK",
		"list of ARENA_IO ARENA_MASK",
		"list all",
		"store 0",
	}

	prog, err := asm.Parse(strings.NewReader(strings.Join(program, "\n")))
	assert.NoError(err)
	if err != nil {
		log.Fatal(err)
	}

	if !assert.Equal(10, len(prog.Opcodes)) {
		assert.Equal(&Program{}, prog)
	}
}

func TestAssemblerErrSyntax(t *testing.T) {
	assert := assert.New(t)

	asm := &Assembler{}

	// Various syntax errors
	table := [](struct {
		prog string
		line int
	}){
		{"DUP:\nDUP:\n", 2},
		{"write r0 nothing", 1},
		{"write r0 $(\"aaa\")", 1},
		{"write r0 $(more(\"aaa\"))", 1},
		{"write r0 $(0x10000000000000000)", 1},
		{"list", 1},
		{"list invalid", 1},
		{"if none? list", 1},
		{".equ", 1},
		{".equ A", 1},
		{".equ A 1\n.equ A 2\n", 2},
		{".macro A B C\n.endm\nA 1\n", 3},
		{".macro A B C\nB C\n.endm\nA list all\nA invalid word\n", 5},
		{".macro A B\n.macro C\n.endm\n.endm", 2},
		{".macro A B\n.endm\n.macro A\n.endm\n", 3},
		{".macro A B\n.endm\n.endm\n", 3},
		{".macro A\nwrite r0 1\n", 2},
		{"alu add match 0\n", 1},
		{"alu zed r0 0\n", 1},
		{"alu\n", 1},
		{"if false?\n", 1},
		{"nop bad\n", 1},
		{"set\n", 1},
		{"set of\n", 1},
		{"list of 1 2 3\n", 1},
		{"list of r9 2\n", 1},
		{"list of 2 r9\n", 1},
		{"tag\n", 1},
		{"list all all\n", 1},
		{"tag bad\n", 1},
		{"list only 1 2 3", 1},
		{"list only", 1},
		{"list only r9", 1},
		{"list only 1 r9", 1},
		{"list next 1", 1},
		{"jump", 1},
		{"jump all over", 1},
		{"jump nowhere", 1},
		{"write", 1},
		{"write r0", 1},
		{"write r0 1 2 3", 1},
		{"write r0 1 r9", 1},
		{"write bad 1 2", 1},
		{"alu", 1},
		{"alu add", 1},
		{"alu add r0", 1},
		{"alu add r0 1 2", 1},
		{"alu add r0 r9", 1},
	}

	for _, entry := range table {
		_, err := asm.Parse(strings.NewReader(entry.prog))
		var se *ErrSyntax
		assert.NotNil(err, entry.prog)
		if err != nil {
			assert.True(errors.As(err, &se), entry.prog)
			assert.Equal(entry.line, se.LineNo, entry.prog)
		}
	}

}
