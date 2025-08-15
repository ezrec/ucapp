package cpu

import (
	"fmt"
)

type CodeCond int

//go:generate go tool stringer -linecomment -type=CodeCond
const (
	COND_ALWAYS = CodeCond(0) // .
	COND_TRUE   = CodeCond(1) // ?
	COND_FALSE  = CodeCond(2) // !
	COND_NEVER  = CodeCond(3) // ~
)

type CodeClass int

//go:generate go tool stringer -linecomment -type=CodeClass
const (
	OP_ALU  = CodeClass(0) // alu
	OP_COND = CodeClass(1) // if
	OP_CAPP = CodeClass(2) // list
	OP_IO   = CodeClass(3) // io
	OP_IMM  = CodeClass(4) // imm
)

type CodeImmOp int

//go:generate go tool stringer -linecomment -type=CodeImmOp
const (
	IMM_OP_LO32 = CodeImmOp(1) // lo32
	IMM_OP_HI32 = CodeImmOp(2) // hi32
	IMM_OP_OR16 = CodeImmOp(3) // or16
)

type CodeAluOp int

//go:generate go tool stringer -linecomment -type=CodeAluOp
const (
	ALU_OP_SET = CodeAluOp(0) // set
	ALU_OP_XOR = CodeAluOp(1) // xor
	ALU_OP_AND = CodeAluOp(2) // and
	ALU_OP_OR  = CodeAluOp(3) // or
	ALU_OP_SHL = CodeAluOp(4) // shl
	ALU_OP_SHR = CodeAluOp(5) // shr
	ALU_OP_ADD = CodeAluOp(6) // add
	ALU_OP_SUB = CodeAluOp(7) // sub
)

type CodeCondOp int

//go:generate go tool stringer -linecomment -type=CodeCondOp
const (
	COND_OP_EQ       = CodeCondOp(0) // eq
	COND_OP_LT       = CodeCondOp(1) // lt
	COND_OP_GT       = CodeCondOp(2) // gt
	COND_OP_UNUSED_3 = CodeCondOp(3) // [3]
	COND_OP_NE       = CodeCondOp(4) // ne
	COND_OP_GE       = CodeCondOp(5) // ge
	COND_OP_LE       = CodeCondOp(6) // le
	COND_OP_UNUSED_4 = CodeCondOp(7) // [7]
)

type CodeCappOp int

//go:generate go tool stringer -linecomment -type=CodeCappOp
const (
	CAPP_OP_SET_SWAP    = CodeCappOp(0) // swap
	CAPP_OP_LIST_ALL    = CodeCappOp(1) // all
	CAPP_OP_LIST_NOT    = CodeCappOp(2) // not
	CAPP_OP_LIST_NEXT   = CodeCappOp(3) // next
	CAPP_OP_LIST_ONLY   = CodeCappOp(4) // only
	CAPP_OP_SET_OF      = CodeCappOp(5) // of
	CAPP_OP_WRITE_FIRST = CodeCappOp(6) // wfirst
	CAPP_OP_WRITE_LIST  = CodeCappOp(7) // wlist
)

type CodeIoOp int

//go:generate go tool stringer -linecomment -type=CodeIoOp
const (
	IO_OP_FETCH = CodeIoOp(0) // fetch
	IO_OP_STORE = CodeIoOp(1) // store
	IO_OP_AWAIT = CodeIoOp(2) // await
	IO_OP_ALERT = CodeIoOp(3) // alert
)

type CodeIR int

//go:generate go tool stringer -linecomment -type=CodeIR
const (
	IR_REG_R0         = CodeIR(0)  // r0
	IR_REG_R1         = CodeIR(1)  // r1
	IR_REG_R2         = CodeIR(2)  // r2
	IR_REG_R3         = CodeIR(3)  // r3
	IR_REG_R4         = CodeIR(4)  // r4
	IR_REG_R5         = CodeIR(5)  // r5
	IR_IP             = CodeIR(6)  // ip
	IR_STACK          = CodeIR(7)  // stack
	IR_REG_MATCH      = CodeIR(8)  // match
	IR_REG_MASK       = CodeIR(9)  // mask
	IR_REG_FIRST      = CodeIR(10) // first
	IR_REG_COUNT      = CodeIR(11) // count
	IR_CONST_0        = CodeIR(12) // immz
	IR_CONST_1        = CodeIR(13) // imm1
	IR_CONST_FFFFFFFF = CodeIR(14) // immnz
	IR_IMMEDIATE_32   = CodeIR(15) // imm
)

func (ir CodeIR) Writable() bool {
	return ir < IR_REG_MATCH
}

type CodeChannel int

//go:generate go tool stringer -linecomment -type=CodeChannel
const (
	CHANNEL_ID_TEMP    = CodeChannel(0) // temp
	CHANNEL_ID_DEPOT   = CodeChannel(1) // depot
	CHANNEL_ID_TAPE    = CodeChannel(2) // tape
	CHANNEL_ID_VT      = CodeChannel(3) // vt
	CHANNEL_ID_MONITOR = CodeChannel(7) // monitor
)

type Opcode struct {
	LineNo    int
	Ip        int
	Words     []string
	Codes     []Code
	LinkLabel string
}

type Code uint32

func MakeCodeImmediate(cond CodeCond, op CodeImmOp, value uint16) Code {
	return Code((uint32(cond) << 18) | (uint32(op) << 16) | uint32(value))
}

func makeCond(cond CodeCond, op uint32) Code {
	return Code((uint32(cond) << 18) | op)
}

func MakeCodeExit(cond CodeCond) Code {
	return MakeCodeAlu(cond, ALU_OP_SET, IR_IP, IR_CONST_FFFFFFFF, IR_CONST_FFFFFFFF)
}

func MakeCodeCapp(cond CodeCond, op CodeCappOp, src_v, src_m CodeIR) Code {
	return makeCond(cond, (uint32(OP_CAPP)<<14)|(uint32(op)<<11)|(uint32(src_v)<<4)|(uint32(src_m)<<0))
}

func MakeCodeIo(cond CodeCond, op CodeIoOp, channel CodeChannel, src_v, src_m CodeIR) Code {
	return makeCond(cond, (uint32(OP_IO)<<14)|(uint32(op)<<11)|(uint32(channel)<<8)|(uint32(src_v)<<4)|(uint32(src_m)<<0))
}

func MakeCodeAlu(cond CodeCond, op CodeAluOp, target, src_v, src_m CodeIR) Code {
	return makeCond(cond, (uint32(OP_ALU)<<14)|(uint32(op)<<11)|((uint32(target)&7)<<8)|(uint32(src_v)<<4)|(uint32(src_m)<<0))
}

func MakeCodeCond(cond CodeCond, op CodeCondOp, src_a, src_b CodeIR) Code {
	return makeCond(cond, (uint32(OP_COND)<<14)|(uint32(op)<<11)|(uint32(src_a)<<4)|(uint32(src_b)<<0))
}

func (code Code) Cond() CodeCond {
	return CodeCond((uint32(code) >> 18) & 0x3)
}

func (code Code) Class() CodeClass {
	if ((uint32(code) >> 16) & 0x3) != 0 {
		return OP_IMM
	}

	return CodeClass((uint32(code) >> 14) & 0x3)
}

func (code Code) ImmOp() CodeImmOp {
	return CodeImmOp((uint32(code) >> 16) & 0x3)
}

func (code Code) AluOp() CodeAluOp {
	return CodeAluOp((uint32(code) >> 11) & 0x7)
}

func (code Code) CondOp() CodeCondOp {
	return CodeCondOp((uint32(code) >> 11) & 0x7)
}

func (code Code) CappOp() CodeCappOp {
	return CodeCappOp((uint32(code) >> 11) & 0x7)
}

func (code Code) IoOp() CodeIoOp {
	return CodeIoOp((uint32(code) >> 11) & 0x7)
}

func (code Code) Channel() CodeChannel {
	return CodeChannel((uint32(code) >> 8) & 0x7)
}

func (code Code) Target() CodeIR {
	return CodeIR((uint32(code) >> 8) & 0x7)
}

func (code Code) Value() CodeIR {
	return CodeIR((uint32(code) >> 4) & 0xf)
}

func (code Code) Match() CodeIR {
	return CodeIR((uint32(code) >> 4) & 0xf)
}

func (code Code) Mask() CodeIR {
	return CodeIR((uint32(code) >> 0) & 0xf)
}

func (code Code) String() (out string) {
	cond := code.Cond()
	class := code.Class()
	val := code.Value()
	msk := code.Mask()

	var op_str string
	var dst_str string

	switch class {
	case OP_IMM:
		op := code.ImmOp()
		op_str = op.String()
		out = fmt.Sprintf(".imm.%v.0%04x", op_str, uint32(code)&0xffff)
		return
	case OP_CAPP:
		dst_str = "-"
		op_str = code.CappOp().String()
	case OP_IO:
		dst_str = code.Channel().String()
		op_str = code.IoOp().String()
	case OP_ALU:
		dst_str = code.Target().String()
		op_str = code.AluOp().String()
	case OP_COND:
		dst_str = "-"
		op_str = code.CondOp().String()
	}

	out = fmt.Sprintf("%v%v.%v.%v.%v.%v", cond.String(), class.String(), op_str, dst_str, val.String(), msk.String())

	return
}
