package cpu

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProgram_Debug(t *testing.T) {
	assert := assert.New(t)

	prog := &Program{
		Opcodes: []Opcode{
			{LineNo: 1, Ip: 0, Words: []string{"write", "r0", "0x10"},
				Codes: []Code{MakeCodeAlu(COND_ALWAYS, ALU_OP_SET, IR_REG_R0, IR_IMMEDIATE_16, 0x10)}},
			{LineNo: 2, Ip: 1, Words: []string{"write", "r1", "0x20"},
				Codes: []Code{MakeCodeAlu(COND_ALWAYS, ALU_OP_SET, IR_REG_R1, IR_IMMEDIATE_16, 0x20)}},
			{LineNo: 3, Ip: 2, Words: []string{"alu", "add", "r0", "r1"},
				Codes: []Code{MakeCodeAlu(COND_ALWAYS, ALU_OP_ADD, IR_REG_R0, IR_REG_R1)}},
		},
	}

	dbg := prog.Debug(0)
	assert.NotNil(dbg.Opcode)
	assert.Equal(1, dbg.Opcode.LineNo)
	assert.Equal(0, dbg.Index)

	dbg = prog.Debug(1)
	assert.NotNil(dbg.Opcode)
	assert.Equal(2, dbg.Opcode.LineNo)
	assert.Equal(0, dbg.Index)

	dbg = prog.Debug(2)
	assert.NotNil(dbg.Opcode)
	assert.Equal(3, dbg.Opcode.LineNo)
	assert.Equal(0, dbg.Index)
}

func TestProgram_Debug_NotFound(t *testing.T) {
	assert := assert.New(t)

	prog := &Program{
		Opcodes: []Opcode{
			{LineNo: 1, Ip: 0, Words: []string{"write", "r0", "0x10"},
				Codes: []Code{MakeCodeAlu(COND_ALWAYS, ALU_OP_SET, IR_REG_R0, IR_IMMEDIATE_16, 0x10)}},
		},
	}

	dbg := prog.Debug(10)
	assert.Nil(dbg.Opcode)
	assert.Equal(0, dbg.Index)
}

func TestProgram_Debug_MultipleCodesPerOpcode(t *testing.T) {
	assert := assert.New(t)

	prog := &Program{
		Opcodes: []Opcode{
			{LineNo: 1, Ip: 0, Words: []string{"call", "FUNC"},
				Codes: []Code{
					MakeCodeAlu(COND_ALWAYS, ALU_OP_SET, IR_STACK, IR_IMMEDIATE_16, 1),
					MakeCodeAlu(COND_ALWAYS, ALU_OP_ADD, IR_STACK, IR_IP),
					MakeCodeAlu(COND_ALWAYS, ALU_OP_SET, IR_IP, IR_IMMEDIATE_16, 10),
				},
				LinkLabel: "FUNC"},
		},
	}

	dbg := prog.Debug(0)
	assert.Equal(0, dbg.Index)

	dbg = prog.Debug(1)
	assert.Equal(1, dbg.Index)

	dbg = prog.Debug(2)
	assert.Equal(2, dbg.Index)

	dbg = prog.Debug(3)
	assert.Nil(dbg.Opcode)
}

func TestProgram_Binary(t *testing.T) {
	assert := assert.New(t)

	prog := &Program{
		Opcodes: []Opcode{
			{LineNo: 1, Ip: 0, Words: []string{"write", "r0", "0x10"},
				Codes: []Code{MakeCodeAlu(COND_ALWAYS, ALU_OP_SET, IR_REG_R0, IR_IMMEDIATE_16, 0x10)}},
			{LineNo: 2, Ip: 1, Words: []string{"write", "r1", "0x20"},
				Codes: []Code{MakeCodeAlu(COND_ALWAYS, ALU_OP_SET, IR_REG_R1, IR_IMMEDIATE_16, 0x20)}},
		},
	}

	bins := prog.Binary()
	assert.NotEmpty(bins)

	assert.Equal(uint32(ARENA_CODE), bins[0]&ARENA_MASK)
	assert.Equal(uint32(0x10), bins[0]&0xFFFF)

	assert.Equal(uint32(ARENA_CODE), bins[1]&ARENA_MASK)

	assert.Equal(uint32(ARENA_CODE), bins[2]&ARENA_MASK)
	assert.Equal(uint32(0x20), bins[2]&0xFFFF)

	assert.Equal(uint32(ARENA_CODE), bins[3]&ARENA_MASK)
}

func TestProgram_Binary_WithMultipleImmediates(t *testing.T) {
	assert := assert.New(t)

	code := MakeCodeAlu(COND_ALWAYS, ALU_OP_SET, IR_REG_R0, IR_IMMEDIATE_32, 0x1234, 0x5678)
	prog := &Program{
		Opcodes: []Opcode{
			{LineNo: 1, Ip: 0, Words: []string{"write", "r0", "0x12345678"},
				Codes: []Code{code}},
		},
	}

	bins := prog.Binary()
	assert.Equal(3, len(bins))

	assert.Equal(uint32(ARENA_CODE), bins[0]&ARENA_MASK)
	assert.Equal(uint32(0x1234), bins[0]&0xFFFF)

	assert.Equal(uint32(ARENA_CODE), bins[1]&ARENA_MASK)
	assert.Equal(uint32(0x5678), bins[1]&0xFFFF)

	assert.Equal(uint32(ARENA_CODE), bins[2]&ARENA_MASK)
	assert.Equal(uint16(code.Word), uint16(bins[2]&0xFFFF))
}

func TestProgram_Codes(t *testing.T) {
	assert := assert.New(t)

	prog := &Program{
		Opcodes: []Opcode{
			{LineNo: 1, Ip: 0, Words: []string{"write", "r0", "0x10"},
				Codes: []Code{MakeCodeAlu(COND_ALWAYS, ALU_OP_SET, IR_REG_R0, IR_IMMEDIATE_16, 0x10)}},
			{LineNo: 2, Ip: 1, Words: []string{"write", "r1", "0x20"},
				Codes: []Code{MakeCodeAlu(COND_ALWAYS, ALU_OP_SET, IR_REG_R1, IR_IMMEDIATE_16, 0x20)}},
			{LineNo: 3, Ip: 2, Words: []string{"alu", "add", "r0", "r1"},
				Codes: []Code{MakeCodeAlu(COND_ALWAYS, ALU_OP_ADD, IR_REG_R0, IR_REG_R1)}},
		},
	}

	ips := []uint16{}
	codes := []Code{}
	for ip, code := range prog.Codes() {
		ips = append(ips, ip)
		codes = append(codes, code)
	}

	assert.Equal(3, len(ips))
	assert.Equal(3, len(codes))
	assert.Equal(uint16(0), ips[0])
	assert.Equal(uint16(1), ips[1])
	assert.Equal(uint16(2), ips[2])
}

func TestProgram_Codes_EarlyReturn(t *testing.T) {
	assert := assert.New(t)

	prog := &Program{
		Opcodes: []Opcode{
			{LineNo: 1, Ip: 0, Words: []string{"write", "r0", "0x10"},
				Codes: []Code{MakeCodeAlu(COND_ALWAYS, ALU_OP_SET, IR_REG_R0, IR_IMMEDIATE_16, 0x10)}},
			{LineNo: 2, Ip: 1, Words: []string{"write", "r1", "0x20"},
				Codes: []Code{MakeCodeAlu(COND_ALWAYS, ALU_OP_SET, IR_REG_R1, IR_IMMEDIATE_16, 0x20)}},
		},
	}

	count := 0
	for range prog.Codes() {
		count++
		if count == 1 {
			break
		}
	}

	assert.Equal(1, count)
}

func TestProgram_Codes_Empty(t *testing.T) {
	assert := assert.New(t)

	prog := &Program{
		Opcodes: []Opcode{},
	}

	count := 0
	for range prog.Codes() {
		count++
	}

	assert.Equal(0, count)
}

func TestProgram_Codes_MultipleCodesPerOpcode(t *testing.T) {
	assert := assert.New(t)

	prog := &Program{
		Opcodes: []Opcode{
			{LineNo: 1, Ip: 0, Words: []string{"call", "FUNC"},
				Codes: []Code{
					MakeCodeAlu(COND_ALWAYS, ALU_OP_SET, IR_STACK, IR_IMMEDIATE_16, 1),
					MakeCodeAlu(COND_ALWAYS, ALU_OP_ADD, IR_STACK, IR_IP),
					MakeCodeAlu(COND_ALWAYS, ALU_OP_SET, IR_IP, IR_IMMEDIATE_16, 10),
				},
				LinkLabel: "FUNC"},
		},
	}

	count := 0
	for ip, _ := range prog.Codes() {
		assert.Equal(uint16(count), ip)
		count++
	}

	assert.Equal(3, count)
}

func TestProgram_Integration_ParseAndBinary(t *testing.T) {
	assert := assert.New(t)

	asm := &Assembler{}
	program := strings.Join([]string{
		"write r0 0x100",
		"write r1 0x200",
		"alu add r0 r1",
	}, "\n")

	prog, err := asm.Parse(strings.NewReader(program))
	assert.NoError(err)

	bins := prog.Binary()
	assert.NotEmpty(bins)

	for _, bin := range bins {
		assert.Equal(uint32(ARENA_CODE), bin&ARENA_MASK)
	}
}

func TestProgram_Integration_ParseAndDebug(t *testing.T) {
	assert := assert.New(t)

	asm := &Assembler{}
	program := strings.Join([]string{
		"write r0 0x100",
		"write r1 0x200",
		"alu add r0 r1",
	}, "\n")

	prog, err := asm.Parse(strings.NewReader(program))
	assert.NoError(err)

	dbg := prog.Debug(0)
	assert.NotNil(dbg.Opcode)
	assert.Equal(1, dbg.Opcode.LineNo)

	dbg = prog.Debug(1)
	assert.NotNil(dbg.Opcode)
	assert.Equal(2, dbg.Opcode.LineNo)

	dbg = prog.Debug(2)
	assert.NotNil(dbg.Opcode)
	assert.Equal(3, dbg.Opcode.LineNo)
}
