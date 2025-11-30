package cpu

import (
	"bytes"
	"errors"
	"iter"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ezrec/ucapp/capp"
	"github.com/ezrec/ucapp/io"
)

type dummyIo struct {
	response chan uint32
}

func (di *dummyIo) Rewind()                 {}
func (di *dummyIo) Send(bool) error         { return nil }
func (di *dummyIo) Receive() iter.Seq[bool] { return func(func(bool) bool) {} }
func (di *dummyIo) Alert(value uint32, response chan uint32) {
	di.response = response
	di.response <- value
}

var _ io.Channel = &dummyIo{}

func TestCpu_String(t *testing.T) {
	assert := assert.New(t)

	cpu := NewCpu(64)
	cpu.Ip = 0x1234_0ABC
	cpu.Cond = true
	cpu.Register[0] = 0x12345678
	cpu.Register[1] = 0xABCDEF01
	cpu.Match = 0x87654321
	cpu.Mask = 0xFEDCBA98
	cpu.Stack.Push(0xDEADBEEF)

	str := cpu.String()
	assert.Contains(str, "ip")
	assert.Contains(str, "cond")
	assert.Contains(str, "r0")
	assert.Contains(str, "stack")
	assert.Contains(str, "match")
	assert.Contains(str, "mask")
	assert.Contains(str, "first")
	assert.Contains(str, "count")
	assert.Contains(str, "true")
}

func TestCpu_Reset(t *testing.T) {
	assert := assert.New(t)

	cpu := NewCpu(64)
	cpu.Ip = 0x12345678
	cpu.Cond = true
	cpu.Register[0] = 0x11111111
	cpu.Register[1] = 0x22222222
	cpu.Stack.Push(0x33333333)
	cpu.Ticks = 100
	cpu.Power = 500

	tape := &io.Tape{}
	cpu.SetChannel(CHANNEL_ID_TAPE, tape)

	err := cpu.Reset(CHANNEL_ID_MONITOR)
	assert.NoError(err)

	assert.Equal(uint32(IP_MODE_REG), cpu.Ip)
	assert.Equal(0, cpu.Ticks)
	assert.Equal(0, cpu.Power)
	assert.True(cpu.Stack.Empty())

	for _, reg := range cpu.Register {
		assert.NotEqual(uint32(0), reg)
	}
}

func TestCpu_GetChannel(t *testing.T) {
	assert := assert.New(t)

	cpu := NewCpu(64)
	defer cpu.Close()

	_, _, err := cpu.GetChannel(CHANNEL_ID_TAPE)
	assert.Error(err)
	assert.ErrorIs(err, ErrChannelInvalid)

	tape := &io.Tape{}
	cpu.SetChannel(CHANNEL_ID_TAPE, tape)

	ch, resp, err := cpu.GetChannel(CHANNEL_ID_TAPE)
	assert.NoError(err)
	assert.Equal(tape, ch)
	assert.NotNil(resp)
}

func TestCpu_FetchCode_CAPP(t *testing.T) {
	assert := assert.New(t)

	cpu := NewCpu(64)
	defer cpu.Close()

	cpu.Capp.Action(capp.SET_OF, ARENA_FREE, ARENA_MASK)
	cpu.Capp.Action(capp.LIST_ALL, 0, 0)
	cpu.Capp.Action(capp.WRITE_FIRST, ARENA_CODE|0x1234, 0xffffffff)
	cpu.Capp.Action(capp.LIST_NOT, 0, 0)
	cpu.Capp.Action(capp.SET_OF, ARENA_CODE, ARENA_MASK)

	cpu.Ip = 0x0000
	code, err := cpu.FetchCode()
	assert.NoError(err)
	assert.Equal(uint16(0x1234), code.Word)
	assert.Empty(code.Immediates)
}

func TestCpu_FetchCode_CAPP_WithImmediates(t *testing.T) {
	assert := assert.New(t)

	cpu := NewCpu(64)
	defer cpu.Close()

	cpu.Capp.Action(capp.SET_OF, ARENA_FREE, ARENA_MASK)
	cpu.Capp.Action(capp.LIST_ALL, 0, 0)
	cpu.Capp.Action(capp.WRITE_FIRST, ARENA_CODE|0x1234, 0xffffffff)
	cpu.Capp.Action(capp.LIST_NEXT, 0, 0)
	cpu.Capp.Action(capp.WRITE_FIRST, ARENA_CODE|0x5678, 0xffffffff)
	cpu.Capp.Action(capp.LIST_NEXT, 0, 0)
	cpu.Capp.Action(capp.WRITE_FIRST, ARENA_CODE|0xABCD, 0xffffffff)
	cpu.Capp.Action(capp.LIST_NOT, 0, 0)
	cpu.Capp.Action(capp.SET_OF, ARENA_CODE, ARENA_MASK)

	cpu.Ip = 0x0000
	code, err := cpu.FetchCode()
	assert.NoError(err)
	assert.Equal(uint16(0xABCD), code.Word)
	assert.Equal([]uint16{0x1234, 0x5678}, code.Immediates)
}

func TestCpu_FetchCode_Stack(t *testing.T) {
	assert := assert.New(t)

	cpu := NewCpu(64)
	defer cpu.Close()
	cpu.Ip = IP_MODE_STACK
	cpu.Stack.Push(0x5678)

	code, err := cpu.FetchCode()
	assert.NoError(err)
	assert.Equal(uint16(0x5678), code.Word)
}

func TestCpu_FetchCode_Stack_Empty(t *testing.T) {
	assert := assert.New(t)

	cpu := NewCpu(64)
	defer cpu.Close()
	cpu.Ip = IP_MODE_STACK

	_, err := cpu.FetchCode()
	assert.Error(err)
	assert.ErrorIs(err, ErrIpEmpty)
}

func TestCpu_FetchCode_Reg(t *testing.T) {
	assert := assert.New(t)

	cpu := NewCpu(64)
	defer cpu.Close()
	cpu.Register[3] = 0x87654321
	cpu.Ip = IP_MODE_REG | 3

	code, err := cpu.FetchCode()
	assert.NoError(err)
	assert.Equal(uint16(0x4321), code.Word)
}

func TestCpu_FetchCode_Reg_OutOfBounds(t *testing.T) {
	assert := assert.New(t)

	cpu := NewCpu(64)
	defer cpu.Close()
	cpu.Ip = IP_MODE_REG | 10

	_, err := cpu.FetchCode()
	assert.Error(err)
	assert.ErrorIs(err, ErrIpEmpty)
}

func TestCpu_FetchCode_InvalidIP(t *testing.T) {
	assert := assert.New(t)

	cpu := NewCpu(64)
	defer cpu.Close()
	cpu.Ip = 0xffffffff

	_, err := cpu.FetchCode()
	assert.Error(err)
	assert.ErrorIs(err, ErrIpEmpty)
}

func TestCpu_FetchCode_CAPP_Empty(t *testing.T) {
	assert := assert.New(t)

	cpu := NewCpu(64)
	defer cpu.Close()
	cpu.Ip = 0x0000

	_, err := cpu.FetchCode()
	assert.Error(err)
	assert.ErrorIs(err, ErrIpEmpty)
}

func TestCpu_Tick_WithTrap(t *testing.T) {
	assert := assert.New(t)

	cpu := NewCpu(64)
	defer cpu.Close()
	cpu.Register[0] = 0x12345678
	cpu.Ip = IP_MODE_REG

	monitor := &io.Tape{}
	cpu.SetChannel(CHANNEL_ID_MONITOR, monitor)
	cpu.channel[CHANNEL_ID_MONITOR].Response <- 1

	code := MakeCodeAlu(COND_ALWAYS, ALU_OP_SET, IR_REG_R1, IR_REG_R0)
	cpu.Register[0] = uint32(code.Word)

	err := cpu.Tick()
	assert.Error(err)
	assert.ErrorIs(err, ErrIpTrap)
}

func TestCpu_Tick_NoMonitorChannel(t *testing.T) {
	assert := assert.New(t)

	cpu := NewCpu(64)
	defer cpu.Close()
	cpu.Ip = IP_MODE_REG
	code := MakeCodeAlu(COND_ALWAYS, ALU_OP_SET, IR_REG_R1, IR_CONST_0)
	cpu.Register[0] = uint32(code.Word)

	err := cpu.Tick()
	assert.NoError(err)
	assert.Equal(uint32(0), cpu.Register[1])
}

func TestCpu_Execute_CondNever(t *testing.T) {
	assert := assert.New(t)

	cpu := NewCpu(64)
	defer cpu.Close()
	code := MakeCodeAlu(COND_NEVER, ALU_OP_SET, IR_REG_R0, IR_CONST_0)

	err := cpu.Execute(code)
	assert.Error(err)
}

func TestCpu_Execute_CondTrue_WithCond(t *testing.T) {
	assert := assert.New(t)

	cpu := NewCpu(64)
	defer cpu.Close()
	cpu.Cond = true
	cpu.Ip = 0
	code := MakeCodeAlu(COND_TRUE, ALU_OP_SET, IR_REG_R0, IR_IMMEDIATE_16, 0x1234)

	err := cpu.Execute(code)
	assert.NoError(err)
	assert.Equal(uint32(0x1234), cpu.Register[0])
}

func TestCpu_Execute_CondTrue_NoCond(t *testing.T) {
	assert := assert.New(t)

	cpu := NewCpu(64)
	defer cpu.Close()
	cpu.Cond = false
	cpu.Ip = 0
	cpu.Register[0] = 0xFFFFFFFF
	code := MakeCodeAlu(COND_TRUE, ALU_OP_SET, IR_REG_R0, IR_IMMEDIATE_16, 0x1234)

	err := cpu.Execute(code)
	assert.NoError(err)
	assert.Equal(uint32(0xFFFFFFFF), cpu.Register[0])
}

func TestCpu_Execute_CondFalse_WithCond(t *testing.T) {
	assert := assert.New(t)

	cpu := NewCpu(64)
	defer cpu.Close()
	cpu.Cond = true
	cpu.Ip = 0
	cpu.Register[0] = 0xFFFFFFFF
	code := MakeCodeAlu(COND_FALSE, ALU_OP_SET, IR_REG_R0, IR_IMMEDIATE_16, 0x1234)

	err := cpu.Execute(code)
	assert.NoError(err)
	assert.Equal(uint32(0xFFFFFFFF), cpu.Register[0])
}

func TestCpu_Execute_CondFalse_NoCond(t *testing.T) {
	assert := assert.New(t)

	cpu := NewCpu(64)
	defer cpu.Close()
	cpu.Cond = false
	cpu.Ip = 0
	code := MakeCodeAlu(COND_FALSE, ALU_OP_SET, IR_REG_R0, IR_IMMEDIATE_16, 0x1234)

	err := cpu.Execute(code)
	assert.NoError(err)
	assert.Equal(uint32(0x1234), cpu.Register[0])
}

func TestCpu_Execute_ALU_AllOps(t *testing.T) {
	assert := assert.New(t)

	tests := []struct {
		op       CodeAluOp
		input    uint32
		arg      uint32
		expected uint32
	}{
		{ALU_OP_SET, 0x12345678, 0xABCDEF01, 0xABCDEF01},
		{ALU_OP_XOR, 0x12345678, 0xFFFFFFFF, 0xEDCBA987},
		{ALU_OP_AND, 0x12345678, 0xFF00FF00, 0x12005600},
		{ALU_OP_OR, 0x12345678, 0x0F0F0F0F, 0x1F3F5F7F},
		{ALU_OP_SHL, 0x00000001, 4, 0x00000010},
		{ALU_OP_SHR, 0x10000000, 4, 0x01000000},
		{ALU_OP_ADD, 0x12345678, 0x11111111, 0x23456789},
		{ALU_OP_SUB, 0x23456789, 0x11111111, 0x12345678},
	}

	for _, tt := range tests {
		cpu := NewCpu(64)
		defer cpu.Close()
		cpu.Register[0] = tt.input
		cpu.Ip = 0

		code := MakeCodeAlu(COND_ALWAYS, tt.op, IR_REG_R0, IR_IMMEDIATE_32, uint16(tt.arg>>16), uint16(tt.arg&0xFFFF))
		err := cpu.Execute(code)
		assert.NoError(err)
		assert.Equal(tt.expected, cpu.Register[0], "op=%v input=%08x arg=%08x", tt.op, tt.input, tt.arg)
	}
}

func TestCpu_Execute_ALU_StackFull(t *testing.T) {
	assert := assert.New(t)

	cpu := NewCpu(64)
	defer cpu.Close()
	cpu.Ip = 0
	for i := 0; i < STACK_LIMIT; i++ {
		cpu.Stack.Push(uint32(i))
	}

	code := MakeCodeAlu(COND_ALWAYS, ALU_OP_SET, IR_STACK, IR_CONST_0)
	err := cpu.Execute(code)
	assert.Error(err)
	assert.ErrorIs(err, ErrStackFull)
}

func TestCpu_Execute_ALU_StackPop(t *testing.T) {
	assert := assert.New(t)

	cpu := NewCpu(64)
	defer cpu.Close()
	cpu.Ip = 0
	cpu.Stack.Push(0x12345678)

	code := MakeCodeAlu(COND_ALWAYS, ALU_OP_ADD, IR_STACK, IR_IMMEDIATE_16, 0x1111)
	err := cpu.Execute(code)
	assert.NoError(err)

	val, ok := cpu.Stack.Pop()
	assert.True(ok)
	assert.Equal(uint32(0x12346789), val)
}

func TestCpu_Execute_ALU_SetIP(t *testing.T) {
	assert := assert.New(t)

	cpu := NewCpu(64)
	defer cpu.Close()
	cpu.Ip = 0x100

	code := MakeCodeAlu(COND_ALWAYS, ALU_OP_SET, IR_IP, IR_IMMEDIATE_16, 0x200)
	err := cpu.Execute(code)
	assert.NoError(err)
	assert.Equal(uint32(0x200), cpu.Ip)
}

func TestCpu_Execute_ALU_InvalidDst(t *testing.T) {
	assert := assert.New(t)

	cpu := NewCpu(64)
	defer cpu.Close()
	cpu.Ip = 0

	code := Code{Word: uint16((uint16(OP_ALU) << 11) | (uint16(ALU_OP_SET) << 8) | (0x8 << 4))}
	err := cpu.Execute(code)
	assert.Error(err)
	assert.ErrorIs(err, ErrOpcodeAlu)
}

func TestCpu_Execute_COND_AllOps(t *testing.T) {
	assert := assert.New(t)

	tests := []struct {
		op       CodeCondOp
		a        int32
		b        int32
		expected bool
	}{
		{COND_OP_EQ, 100, 100, true},
		{COND_OP_EQ, 100, 99, false},
		{COND_OP_NE, 100, 99, true},
		{COND_OP_NE, 100, 100, false},
		{COND_OP_LT, 99, 100, true},
		{COND_OP_LT, 100, 99, false},
		{COND_OP_LE, 99, 100, true},
		{COND_OP_LE, 100, 100, true},
		{COND_OP_LE, 101, 100, false},
		{COND_OP_LT, -1, 0, true},
		{COND_OP_LE, -100, -100, true},
	}

	for _, tt := range tests {
		cpu := NewCpu(64)
		defer cpu.Close()
		cpu.Ip = 0

		code := MakeCodeCond(COND_ALWAYS, tt.op, IR_IMMEDIATE_32, IR_IMMEDIATE_32,
			uint16(uint32(tt.a)>>16), uint16(uint32(tt.a)&0xFFFF),
			uint16(uint32(tt.b)>>16), uint16(uint32(tt.b)&0xFFFF))
		err := cpu.Execute(code)
		assert.NoError(err)
		assert.Equal(tt.expected, cpu.Cond, "op=%v a=%d b=%d", tt.op, tt.a, tt.b)
	}
}

func TestCpu_Execute_COND_InvalidOp(t *testing.T) {
	assert := assert.New(t)

	cpu := NewCpu(64)
	defer cpu.Close()
	cpu.Ip = 0

	code := Code{Word: uint16((uint16(OP_COND) << 11) | (0x7 << 8))}
	err := cpu.Execute(code)
	assert.Error(err)
	assert.ErrorIs(err, ErrOpcodeCond)
}

func TestCpu_Execute_CAPP_SetSwap_Error(t *testing.T) {
	assert := assert.New(t)

	cpu := NewCpu(64)
	defer cpu.Close()
	cpu.Ip = 0

	code := MakeCodeCapp(COND_ALWAYS, CAPP_OP_SET_SWAP, IR_CONST_0, IR_CONST_0)
	err := cpu.Execute(code)
	assert.Error(err)
	assert.ErrorIs(err, ErrOpcodeCapp)
}

func TestCpu_Execute_CAPP_ListAll(t *testing.T) {
	assert := assert.New(t)

	cpu := NewCpu(64)
	defer cpu.Close()
	cpu.Ip = 0
	cpu.Capp.Action(capp.SET_OF, ARENA_FREE, ARENA_MASK)
	cpu.Capp.Action(capp.LIST_ALL, 0, 0)
	cpu.Capp.Action(capp.WRITE_FIRST, ARENA_IO|0x100, 0xffffffff)
	cpu.Capp.Action(capp.LIST_NEXT, 0, 0)
	cpu.Capp.Action(capp.WRITE_FIRST, ARENA_IO|0x200, 0xffffffff)
	cpu.Capp.Action(capp.LIST_NEXT, 0, 0)
	cpu.Capp.Action(capp.LIST_NOT, 0, 0)

	code := MakeCodeCapp(COND_ALWAYS, CAPP_OP_LIST_ALL, IR_CONST_0, IR_CONST_0)
	err := cpu.Execute(code)
	assert.NoError(err)
	assert.Equal(uint(64), cpu.Capp.Count())
}

func TestCpu_Execute_CAPP_ListNot(t *testing.T) {
	assert := assert.New(t)

	cpu := NewCpu(64)
	defer cpu.Close()
	cpu.Ip = 0
	cpu.Capp.Action(capp.SET_OF, ARENA_FREE, ARENA_MASK)
	cpu.Capp.Action(capp.LIST_ALL, 0, 0)
	cpu.Capp.Action(capp.WRITE_FIRST, ARENA_IO|0x100, 0xffffffff)
	cpu.Capp.Action(capp.LIST_NOT, 0, 0)

	code := MakeCodeCapp(COND_ALWAYS, CAPP_OP_LIST_NOT, IR_CONST_0, IR_CONST_0)
	err := cpu.Execute(code)
	assert.NoError(err)
}

func TestCpu_Execute_CAPP_ListNext(t *testing.T) {
	assert := assert.New(t)

	cpu := NewCpu(64)
	defer cpu.Close()
	cpu.Ip = 0
	cpu.Capp.Action(capp.SET_OF, ARENA_FREE, ARENA_MASK)
	cpu.Capp.Action(capp.LIST_ALL, 0, 0)
	cpu.Capp.Action(capp.WRITE_LIST, ARENA_IO|0x100, 0xffffffff)
	cpu.Capp.Action(capp.LIST_NEXT, 0, 0)
	cpu.Capp.Action(capp.WRITE_LIST, ARENA_IO|0x200, 0xffffffff)
	cpu.Capp.Action(capp.LIST_NEXT, 0, 0)
	cpu.Capp.Action(capp.LIST_NOT, 0, 0)

	code := MakeCodeCapp(COND_ALWAYS, CAPP_OP_LIST_NEXT, IR_CONST_0, IR_CONST_0)
	err := cpu.Execute(code)
	assert.NoError(err)
	assert.Equal(uint(1), cpu.Capp.Count())
}

func TestCpu_Execute_CAPP_ListOnly(t *testing.T) {
	assert := assert.New(t)

	cpu := NewCpu(64)
	defer cpu.Close()
	cpu.Ip = 0
	cpu.Capp.Action(capp.SET_OF, ARENA_FREE, ARENA_MASK)
	cpu.Capp.Action(capp.LIST_ALL, 0, 0)
	cpu.Capp.Action(capp.WRITE_FIRST, ARENA_IO|0x100, 0xffffffff)
	cpu.Capp.Action(capp.LIST_NEXT, 0, 0)
	cpu.Capp.Action(capp.WRITE_FIRST, ARENA_IO|0x200, 0xffffffff)
	cpu.Capp.Action(capp.LIST_NOT, 0, 0)

	code := MakeCodeCapp(COND_ALWAYS, CAPP_OP_LIST_ONLY, IR_IMMEDIATE_32, IR_IMMEDIATE_32,
		uint16((ARENA_IO|0x100)>>16), uint16((ARENA_IO|0x100)&0xFFFF),
		0xFFFF, 0xFFFF)
	err := cpu.Execute(code)
	assert.NoError(err)
	assert.Equal(uint(1), cpu.Capp.Count())
	assert.Equal(uint32(ARENA_IO|0x100), cpu.Capp.First())
}

func TestCpu_Execute_CAPP_SetOf(t *testing.T) {
	assert := assert.New(t)

	cpu := NewCpu(64)
	defer cpu.Close()
	cpu.Ip = 0
	cpu.Capp.Action(capp.SET_OF, ARENA_FREE, ARENA_MASK)
	cpu.Capp.Action(capp.LIST_ALL, 0, 0)
	cpu.Capp.Action(capp.WRITE_FIRST, ARENA_IO|0x100, 0xffffffff)
	cpu.Capp.Action(capp.LIST_NOT, 0, 0)

	code := MakeCodeCapp(COND_ALWAYS, CAPP_OP_SET_OF, IR_IMMEDIATE_32, IR_IMMEDIATE_32,
		uint16((ARENA_IO|0x100)>>16), uint16((ARENA_IO|0x100)&0xFFFF),
		0xFFFF, 0xFFFF)
	err := cpu.Execute(code)
	assert.NoError(err)
	assert.Equal(uint32(ARENA_IO|0x100), cpu.Match)
	assert.Equal(uint32(0xffffffff), cpu.Mask)
}

func TestCpu_Execute_CAPP_WriteFirst(t *testing.T) {
	assert := assert.New(t)

	cpu := NewCpu(64)
	defer cpu.Close()
	cpu.Ip = 0
	cpu.Capp.Action(capp.SET_OF, ARENA_FREE, ARENA_MASK)
	cpu.Capp.Action(capp.LIST_ALL, 0, 0)
	cpu.Capp.Action(capp.WRITE_FIRST, ARENA_IO|0x100, 0xffffffff)

	code := MakeCodeCapp(COND_ALWAYS, CAPP_OP_WRITE_FIRST, IR_IMMEDIATE_16, IR_IMMEDIATE_16, 0x202, 0x2FF)
	err := cpu.Execute(code)
	assert.NoError(err)
	assert.Equal(uint32(ARENA_IO|0x302), cpu.Capp.First())
}

func TestCpu_Execute_CAPP_WriteList(t *testing.T) {
	assert := assert.New(t)

	cpu := NewCpu(64)
	defer cpu.Close()
	cpu.Ip = 0
	cpu.Capp.Action(capp.SET_OF, ARENA_FREE, ARENA_MASK)
	cpu.Capp.Action(capp.LIST_ALL, 0, 0)
	cpu.Capp.Action(capp.WRITE_FIRST, ARENA_IO|0x100, 0xffffffff)
	cpu.Capp.Action(capp.LIST_NOT, 0, 0)

	code := MakeCodeCapp(COND_ALWAYS, CAPP_OP_WRITE_LIST, IR_IMMEDIATE_16, IR_IMMEDIATE_16, 0x200, 0xFF)
	err := cpu.Execute(code)
	assert.NoError(err)
}

func TestCpu_Execute_IO_Fetch(t *testing.T) {
	assert := assert.New(t)

	cpu := NewCpu(64)
	defer cpu.Close()
	cpu.Ip = 0

	tape := &io.Tape{}
	tape.Input = bytes.NewReader([]byte{0xFF})
	cpu.SetChannel(CHANNEL_ID_TAPE, tape)

	cpu.Capp.Action(capp.SET_OF, ARENA_FREE, ARENA_MASK)
	cpu.Capp.Action(capp.LIST_ALL, 0, 0)
	cpu.Capp.Action(capp.WRITE_FIRST, ARENA_IO, 0xffffffff)
	cpu.Capp.Action(capp.LIST_NOT, 0, 0)

	code := MakeCodeIo(COND_ALWAYS, IO_OP_FETCH, CHANNEL_ID_TAPE, IR_IMMEDIATE_16, 0xFF)
	err := cpu.Execute(code)
	assert.NoError(err)
}

func TestCpu_Execute_IO_Store(t *testing.T) {
	assert := assert.New(t)

	cpu := NewCpu(64)
	defer cpu.Close()
	cpu.Ip = 0

	buf := &bytes.Buffer{}
	tape := &io.Tape{}
	tape.Output = buf
	cpu.SetChannel(CHANNEL_ID_TAPE, tape)

	cpu.Capp.Action(capp.SET_OF, ARENA_FREE, ARENA_MASK)
	cpu.Capp.Action(capp.LIST_ALL, 0, 0)
	cpu.Capp.Action(capp.WRITE_FIRST, ARENA_IO|0x0AA, 0xffffffff)
	cpu.Capp.Action(capp.LIST_NEXT, 0, 0)
	cpu.Capp.Action(capp.LIST_NOT, 0, 0)

	code := MakeCodeIo(COND_ALWAYS, IO_OP_STORE, CHANNEL_ID_TAPE, IR_IMMEDIATE_16, 0xFF)
	err := cpu.Execute(code)
	assert.NoError(err)
	assert.NotEmpty(buf.Bytes())
}

func TestCpu_Execute_IO_Alert(t *testing.T) {
	assert := assert.New(t)

	cpu := NewCpu(64)
	defer cpu.Close()
	cpu.Ip = 0

	tape := &dummyIo{}
	cpu.SetChannel(CHANNEL_ID_TAPE, tape)

	code := MakeCodeIo(COND_ALWAYS, IO_OP_ALERT, CHANNEL_ID_TAPE, IR_IMMEDIATE_16, 0x1234)
	err := cpu.Execute(code)
	assert.NoError(err)

	alert, ok := <-tape.response
	assert.True(ok)
	assert.Equal(uint32(0x1234), alert)
}

func TestCpu_Execute_IO_Await_Ready(t *testing.T) {
	assert := assert.New(t)

	cpu := NewCpu(64)
	defer cpu.Close()
	cpu.Ip = 0

	tape := &dummyIo{}
	cpu.SetChannel(CHANNEL_ID_TAPE, tape)
	cpu.channel[CHANNEL_ID_TAPE].Response <- 0xABCD

	code := MakeCodeIo(COND_ALWAYS, IO_OP_AWAIT, CHANNEL_ID_TAPE, IR_REG_R0)
	err := cpu.Execute(code)
	assert.NoError(err)
	assert.Equal(uint32(0xABCD), cpu.Register[0])
	assert.Equal(uint32(1), cpu.Ip)
}

func TestCpu_Execute_IO_Await_NotReady(t *testing.T) {
	assert := assert.New(t)

	cpu := NewCpu(64)
	defer cpu.Close()
	cpu.Ip = 5

	tape := &dummyIo{}
	cpu.SetChannel(CHANNEL_ID_TAPE, tape)

	code := MakeCodeIo(COND_ALWAYS, IO_OP_AWAIT, CHANNEL_ID_TAPE, IR_REG_R0)
	err := cpu.Execute(code)
	assert.NoError(err)
	assert.Equal(uint32(5), cpu.Ip)
}

func TestCpu_Execute_IO_Await_ToStack(t *testing.T) {
	assert := assert.New(t)

	cpu := NewCpu(64)
	defer cpu.Close()
	cpu.Ip = 0

	tape := &dummyIo{}
	cpu.SetChannel(CHANNEL_ID_TAPE, tape)
	cpu.channel[CHANNEL_ID_TAPE].Response <- 0x5678

	code := MakeCodeIo(COND_ALWAYS, IO_OP_AWAIT, CHANNEL_ID_TAPE, IR_STACK)
	err := cpu.Execute(code)
	assert.NoError(err)

	val, ok := cpu.Stack.Pop()
	assert.True(ok)
	assert.Equal(uint32(0x5678), val)
}

func TestCpu_Execute_IO_Await_ToIP(t *testing.T) {
	assert := assert.New(t)

	cpu := NewCpu(64)
	defer cpu.Close()
	cpu.Ip = 0

	tape := &dummyIo{}
	cpu.SetChannel(CHANNEL_ID_TAPE, tape)
	cpu.channel[CHANNEL_ID_TAPE].Response <- 0x100

	code := MakeCodeIo(COND_ALWAYS, IO_OP_AWAIT, CHANNEL_ID_TAPE, IR_IP)
	err := cpu.Execute(code)
	assert.NoError(err)
	assert.Equal(uint32(0x100), cpu.Ip)
}

func TestCpu_Execute_IO_Await_InvalidTarget(t *testing.T) {
	assert := assert.New(t)

	cpu := NewCpu(64)
	defer cpu.Close()
	cpu.Ip = 0

	tape := &io.Tape{}
	cpu.SetChannel(CHANNEL_ID_TAPE, tape)

	code := Code{Word: uint16((uint16(OP_IO) << 11) | (uint16(IO_OP_AWAIT) << 8) | (uint16(CHANNEL_ID_TAPE) << 4) | 0xE)}
	err := cpu.Execute(code)
	assert.Error(err)
	assert.ErrorIs(err, ErrOpcodeIo)
}

func TestCpu_Execute_IO_InvalidChannel(t *testing.T) {
	assert := assert.New(t)

	cpu := NewCpu(64)
	defer cpu.Close()
	cpu.Ip = 0

	code := MakeCodeIo(COND_ALWAYS, IO_OP_ALERT, CHANNEL_ID_TAPE, IR_CONST_0)
	err := cpu.Execute(code)
	assert.Error(err)
	assert.ErrorIs(err, ErrChannelInvalid)
}

func TestCpu_Execute_IO_InvalidOp(t *testing.T) {
	assert := assert.New(t)

	cpu := NewCpu(64)
	defer cpu.Close()
	cpu.Ip = 0

	tape := &io.Tape{}
	cpu.SetChannel(CHANNEL_ID_TAPE, tape)

	code := Code{Word: uint16((uint16(OP_IO) << 11) | (0x7 << 8) | (uint16(CHANNEL_ID_TAPE) << 4))}
	err := cpu.Execute(code)
	assert.Error(err)
	assert.ErrorIs(err, ErrOpcodeIo)
}

func TestCpu_Execute_ExtraImmediates(t *testing.T) {
	assert := assert.New(t)

	cpu := NewCpu(64)
	defer cpu.Close()
	cpu.Ip = 0

	code := MakeCodeAlu(COND_ALWAYS, ALU_OP_SET, IR_REG_R0, IR_CONST_0, 0x1234)
	err := cpu.Execute(code)
	assert.Error(err)
	assert.ErrorIs(err, ErrOpcodeImm)
}

func TestCpu_getValue_AllIR(t *testing.T) {
	assert := assert.New(t)

	cpu := NewCpu(64)
	defer cpu.Close()
	cpu.Ip = 100
	cpu.Register[0] = 0x11111111
	cpu.Register[1] = 0x22222222
	cpu.Register[2] = 0x33333333
	cpu.Register[3] = 0x44444444
	cpu.Register[4] = 0x55555555
	cpu.Register[5] = 0x66666666
	cpu.Stack.Push(0x77777777)
	cpu.Match = 0x88888888
	cpu.Mask = 0x99999999
	cpu.Capp.Action(capp.SET_OF, ARENA_FREE, ARENA_MASK)
	cpu.Capp.Action(capp.LIST_ALL, 0, 0)
	cpu.Capp.Action(capp.WRITE_FIRST, ARENA_IO|0xAAA, 0xffffffff)
	cpu.Capp.Action(capp.LIST_NEXT, 0, 0)
	cpu.Capp.Action(capp.LIST_NOT, 0, 0)

	tests := []struct {
		ir       CodeIR
		expected uint32
	}{
		{IR_CONST_0, 0},
		{IR_CONST_FFFFFFFF, 0xFFFFFFFF},
		{IR_REG_R0, 0x11111111},
		{IR_REG_R1, 0x22222222},
		{IR_REG_R2, 0x33333333},
		{IR_REG_R3, 0x44444444},
		{IR_REG_R4, 0x55555555},
		{IR_REG_R5, 0x66666666},
		{IR_IP, 101},
		{IR_STACK, 0x77777777},
		{IR_REG_MATCH, 0x88888888},
		{IR_REG_MASK, 0x99999999},
		{IR_REG_FIRST, ARENA_IO | 0xAAA},
		{IR_REG_COUNT, 1},
	}

	for _, tt := range tests {
		val, _, err := cpu.getValue(tt.ir, nil)
		assert.NoError(err, "ir=%v", tt.ir)
		assert.Equal(tt.expected, val, "ir=%v", tt.ir)
	}
}

func TestCpu_getValue_Immediate16(t *testing.T) {
	assert := assert.New(t)

	cpu := NewCpu(64)
	defer cpu.Close()
	val, _, err := cpu.getValue(IR_IMMEDIATE_16, []uint16{0x1234, 0x5678})
	assert.NoError(err)
	assert.Equal(uint32(0x1234), val)
}

func TestCpu_getValue_Immediate32(t *testing.T) {
	assert := assert.New(t)

	cpu := NewCpu(64)
	defer cpu.Close()
	val, _, err := cpu.getValue(IR_IMMEDIATE_32, []uint16{0x1234, 0x5678})
	assert.NoError(err)
	assert.Equal(uint32(0x12345678), val)
}

func TestCpu_getValue_StackEmpty(t *testing.T) {
	assert := assert.New(t)

	cpu := NewCpu(64)
	defer cpu.Close()
	_, _, err := cpu.getValue(IR_STACK, nil)
	assert.Error(err)
	assert.ErrorIs(err, ErrStackEmpty)
}

func TestCpu_getValue_Immediate16_Missing(t *testing.T) {
	assert := assert.New(t)

	cpu := NewCpu(64)
	defer cpu.Close()
	_, _, err := cpu.getValue(IR_IMMEDIATE_16, nil)
	assert.Error(err)
	assert.ErrorIs(err, ErrOpcodeImm)
}

func TestCpu_getValue_Immediate32_Missing(t *testing.T) {
	assert := assert.New(t)

	cpu := NewCpu(64)
	defer cpu.Close()
	_, _, err := cpu.getValue(IR_IMMEDIATE_32, []uint16{0x1234})
	assert.Error(err)
	assert.ErrorIs(err, ErrOpcodeImm)
}

func TestCpu_listInput(t *testing.T) {
	assert := assert.New(t)

	cpu := NewCpu(64)
	defer cpu.Close()
	cpu.Capp.Action(capp.SET_OF, ARENA_IO, ARENA_MASK)
	cpu.Capp.Action(capp.LIST_ALL, 0, 0)
	cpu.Capp.Action(capp.WRITE_FIRST, ARENA_IO, 0xffffffff)
	cpu.Capp.Action(capp.LIST_NEXT, 0, 0)
	cpu.Capp.Action(capp.WRITE_FIRST, ARENA_IO, 0xffffffff)

	tape := &io.Tape{}
	tape.Input = bytes.NewReader([]byte{0xAB})

	err := cpu.listInput(tape, 0xFF)
	assert.NoError(err)
}

func TestCpu_listInput_ZeroMask(t *testing.T) {
	assert := assert.New(t)

	cpu := NewCpu(64)
	defer cpu.Close()
	tape := &io.Tape{}

	err := cpu.listInput(tape, 0)
	assert.NoError(err)
}

func TestCpu_listOutput(t *testing.T) {
	assert := assert.New(t)

	cpu := NewCpu(64)
	defer cpu.Close()
	cpu.Capp.Action(capp.SET_OF, ARENA_FREE, ARENA_MASK)
	cpu.Capp.Action(capp.LIST_ALL, 0, 0)
	cpu.Capp.Action(capp.WRITE_FIRST, ARENA_IO|0xAB, 0xffffffff)
	cpu.Capp.Action(capp.LIST_NEXT, 0, 0)
	cpu.Capp.Action(capp.LIST_NOT, 0, 0)

	buf := &bytes.Buffer{}
	tape := &io.Tape{}
	tape.Output = buf

	err := cpu.listOutput(tape, 0xFF)
	assert.NoError(err)
	assert.NotEmpty(buf.Bytes())
}

func TestCpu_listOutput_ZeroMask(t *testing.T) {
	assert := assert.New(t)

	cpu := NewCpu(64)
	defer cpu.Close()
	tape := &io.Tape{}

	err := cpu.listOutput(tape, 0)
	assert.NoError(err)
}

func TestCpu_doAlu_ShlClamp(t *testing.T) {
	assert := assert.New(t)

	cpu := NewCpu(64)
	defer cpu.Close()
	result := cpu.doAlu(ALU_OP_SHL, 0x1, 0x20)
	assert.Equal(uint32(0x1), result)

	result = cpu.doAlu(ALU_OP_SHL, 0x1, 0x3F)
	assert.Equal(uint32(0x1<<0x1F), result)
}

func TestCpu_doAlu_ShrClamp(t *testing.T) {
	assert := assert.New(t)

	cpu := NewCpu(64)
	defer cpu.Close()
	result := cpu.doAlu(ALU_OP_SHR, 0xFFFFFFFF, 0x20)
	assert.Equal(uint32(0xFFFFFFFF), result)

	result = cpu.doAlu(ALU_OP_SHR, 0x80000000, 0x3F)
	assert.Equal(uint32(0x1), result)
}

func TestCpu_Verbose(t *testing.T) {
	assert := assert.New(t)

	cpu := NewCpu(64)
	defer cpu.Close()
	cpu.Verbose = true
	cpu.Ip = IP_MODE_REG

	code := MakeCodeAlu(COND_ALWAYS, ALU_OP_SET, IR_REG_R0, IR_CONST_0)
	cpu.Register[0] = uint32(code.Word)

	err := cpu.Execute(code)
	assert.NoError(err)
}

func TestAssembler_Define(t *testing.T) {
	assert := assert.New(t)

	asm := &Assembler{}
	asm.Define("TEST", "0x1234")
	assert.Equal("0x1234", asm.Equate["TEST"])

	asm.Define("TEST", "0x5678")
	assert.Equal("0x5678", asm.Equate["TEST"])
}

func TestAssembler_valueOf(t *testing.T) {
	assert := assert.New(t)

	asm := &Assembler{}

	val, err := asm.valueOf("0x1234")
	assert.NoError(err)
	assert.Equal(uint32(0x1234), val)

	val, err = asm.valueOf("1234")
	assert.NoError(err)
	assert.Equal(uint32(1234), val)

	val, err = asm.valueOf("-1")
	assert.NoError(err)
	assert.Equal(uint32(0xFFFFFFFF), val)

	val, err = asm.valueOf("~0x00FF")
	assert.NoError(err)
	assert.Equal(uint32(0xFFFFFF00), val)

	_, err = asm.valueOf("'xy'")
	assert.Error(err)

	_, err = asm.valueOf("notanumber")
	assert.Error(err)
}

func TestMakeCodeCond(t *testing.T) {
	assert := assert.New(t)

	code := MakeCodeCond(COND_ALWAYS, COND_OP_EQ, IR_REG_R0, IR_REG_R1)
	assert.Equal(COND_ALWAYS, code.Cond())
	assert.Equal(OP_COND, code.Class())

	op, a, b := code.CondDecode()
	assert.Equal(COND_OP_EQ, op)
	assert.Equal(IR_REG_R0, a)
	assert.Equal(IR_REG_R1, b)
}

func TestCode_String_AllClasses(t *testing.T) {
	assert := assert.New(t)

	tests := []struct {
		code Code
	}{
		{MakeCodeAlu(COND_ALWAYS, ALU_OP_SET, IR_REG_R0, IR_CONST_0)},
		{MakeCodeCond(COND_TRUE, COND_OP_EQ, IR_REG_R0, IR_REG_R1)},
		{MakeCodeCapp(COND_FALSE, CAPP_OP_LIST_ALL, IR_CONST_0, IR_CONST_0)},
		{MakeCodeIo(COND_ALWAYS, IO_OP_FETCH, CHANNEL_ID_TAPE, IR_IMMEDIATE_16, 0xFF)},
	}

	for _, tt := range tests {
		str := tt.code.String()
		assert.NotEmpty(str)
	}
}

func TestCode_ImmediateNeed_AllClasses(t *testing.T) {
	assert := assert.New(t)

	tests := []struct {
		code     Code
		expected int
	}{
		{MakeCodeAlu(COND_ALWAYS, ALU_OP_SET, IR_REG_R0, IR_CONST_0), 0},
		{MakeCodeAlu(COND_ALWAYS, ALU_OP_SET, IR_REG_R0, IR_IMMEDIATE_16), 1},
		{MakeCodeAlu(COND_ALWAYS, ALU_OP_SET, IR_REG_R0, IR_IMMEDIATE_32), 2},
		{MakeCodeCond(COND_ALWAYS, COND_OP_EQ, IR_IMMEDIATE_16, IR_IMMEDIATE_16), 2},
		{MakeCodeCond(COND_ALWAYS, COND_OP_EQ, IR_IMMEDIATE_32, IR_IMMEDIATE_32), 4},
		{MakeCodeCapp(COND_ALWAYS, CAPP_OP_SET_OF, IR_IMMEDIATE_16, IR_IMMEDIATE_32), 3},
		{MakeCodeIo(COND_ALWAYS, IO_OP_FETCH, CHANNEL_ID_TAPE, IR_IMMEDIATE_16), 1},
	}

	for _, tt := range tests {
		need := tt.code.ImmediateNeed()
		assert.Equal(tt.expected, need, "code=%v", tt.code)
	}
}

func TestCodeIR_Writable(t *testing.T) {
	assert := assert.New(t)

	assert.True(IR_REG_R0.Writable())
	assert.True(IR_IP.Writable())
	assert.True(IR_STACK.Writable())
	assert.False(IR_REG_MATCH.Writable())
	assert.False(IR_REG_MASK.Writable())
	assert.False(IR_CONST_0.Writable())
}

func TestErrOpcode_Is(t *testing.T) {
	assert := assert.New(t)

	code := MakeCodeAlu(COND_ALWAYS, ALU_OP_SET, IR_REG_R0, IR_CONST_0)
	err := ErrOpcode(code)

	assert.True(errors.Is(err, ErrOpcode(Code{})))
}

func TestErrOpcode_Error(t *testing.T) {
	assert := assert.New(t)

	code := MakeCodeAlu(COND_ALWAYS, ALU_OP_SET, IR_REG_R0, IR_CONST_0)
	err := ErrOpcode(code)

	str := err.Error()
	assert.Contains(str, "0x")
	assert.NotEmpty(str)
}

func TestErrLabelMissing_Error(t *testing.T) {
	assert := assert.New(t)

	err := ErrLabelMissing("MYLABEL")
	str := err.Error()
	assert.Contains(str, "MYLABEL")
}

func TestErrSyntax_Unwrap(t *testing.T) {
	assert := assert.New(t)

	inner := errors.New("inner error")
	err := ErrSyntax{LineNo: 5, Line: "test line", Err: inner}

	assert.Equal(inner, err.Unwrap())
	assert.Contains(err.Error(), "line 5")
	assert.Contains(err.Error(), "test line")
}

func TestErrParseCharacter_Error(t *testing.T) {
	assert := assert.New(t)

	err := ErrParseCharacter("xy")
	str := err.Error()
	assert.Contains(str, "xy")
}

func TestErrParseNumber_Error(t *testing.T) {
	assert := assert.New(t)

	err := ErrParseNumber("notanumber")
	str := err.Error()
	assert.Contains(str, "notanumber")
}

func TestErrParseExpression_Error(t *testing.T) {
	assert := assert.New(t)

	err := ErrParseExpression("1 + ")
	str := err.Error()
	assert.Contains(str, "1 + ")
}

func TestErrMacro_Unwrap(t *testing.T) {
	assert := assert.New(t)

	inner := errors.New("inner error")
	err := ErrMacro{Macro: "MYMACRO", Line: 3, Err: inner}

	assert.Equal(inner, err.Unwrap())
	assert.Contains(err.Error(), "MYMACRO")
	assert.Contains(err.Error(), "3")
}

func TestCpu_PowerAndTicks(t *testing.T) {
	assert := assert.New(t)

	cpu := NewCpu(64)
	defer cpu.Close()
	cpu.Ip = IP_MODE_REG

	code := MakeCodeAlu(COND_ALWAYS, ALU_OP_SET, IR_REG_R0, IR_IMMEDIATE_16, 0x1234)
	cpu.Register[0] = uint32(code.Word)
	cpu.Register[1] = 0x1234

	initialTicks := cpu.Ticks
	initialPower := cpu.Power

	err := cpu.Execute(code)
	assert.NoError(err)

	assert.Greater(cpu.Ticks, initialTicks)
	assert.GreaterOrEqual(cpu.Power, initialPower)
}

func TestProgram_Integration(t *testing.T) {
	assert := assert.New(t)

	asm := &Assembler{}
	program := strings.Join([]string{
		"write r0 0x100",
		"write r1 0x200",
		"alu add r0 r1",
	}, "\n")

	prog, err := asm.Parse(strings.NewReader(program))
	assert.NoError(err)
	assert.Equal(3, len(prog.Opcodes))
}
