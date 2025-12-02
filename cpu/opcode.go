package cpu

import (
	"fmt"
)

// CodeCond is a condition code.
type CodeCond int

//go:generate go tool stringer -linecomment -type=CodeCond
const (
	COND_ALWAYS = CodeCond(0) // .
	COND_TRUE   = CodeCond(1) // +
	COND_FALSE  = CodeCond(2) // -
	COND_NEVER  = CodeCond(3) // ~
)

// CodeClass is the type of opcode class.
type CodeClass int

//go:generate go tool stringer -linecomment -type=CodeClass
const (
	OP_ALU  = CodeClass(0) // alu
	OP_COND = CodeClass(1) // if
	OP_CAPP = CodeClass(2) // list
	OP_IO   = CodeClass(3) // io
)

// CodeAluOp is an ALU operation type.
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

// CodeCondOp is a conditional operation type.
type CodeCondOp int

//go:generate go tool stringer -linecomment -type=CodeCondOp
const (
	COND_OP_EQ = CodeCondOp(0) // eq
	COND_OP_NE = CodeCondOp(1) // ne
	COND_OP_LT = CodeCondOp(2) // lt
	COND_OP_LE = CodeCondOp(3) // le
)

// CodeCappOp is a CAPP operation type.
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

// CodeIoOp is an IO operation type.
type CodeIoOp int

//go:generate go tool stringer -linecomment -type=CodeIoOp
const (
	IO_OP_FETCH = CodeIoOp(0) // fetch
	IO_OP_STORE = CodeIoOp(1) // store
	IO_OP_AWAIT = CodeIoOp(2) // await
	IO_OP_ALERT = CodeIoOp(3) // alert
)

// CodeIR is an Immediate-or-Register decode type.
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
	IR_CONST_FFFFFFFF = CodeIR(13) // immnz
	IR_IMMEDIATE_16   = CodeIR(14) // imm16
	IR_IMMEDIATE_32   = CodeIR(15) // imm32
)

// Writable returns true if the CodeIR represents a writable destination.
func (ir CodeIR) Writable() bool {
	return ir < IR_REG_MATCH
}

// CodeChannel is an IO channel index type.
type CodeChannel int

//go:generate go tool stringer -linecomment -type=CodeChannel
const (
	CHANNEL_ID_TEMP    = CodeChannel(0) // temp
	CHANNEL_ID_DEPOT   = CodeChannel(1) // depot
	CHANNEL_ID_TAPE    = CodeChannel(2) // tape
	CHANNEL_ID_VT      = CodeChannel(3) // vt
	CHANNEL_ID_MONITOR = CodeChannel(7) // monitor
)

// Opcode represents a line of assembled code with its source location and generated instructions.
type Opcode struct {
	LineNo    int
	Ip        int
	Words     []string
	Codes     []Code
	LinkLabel string
}

// Code represents a single instruction word with optional immediate values.
type Code struct {
	Word       uint16
	Immediates []uint16
}

// makeCond creates an instruction with the specified condition code.
func makeCond(cond CodeCond, op uint16, imms ...uint16) Code {
	return Code{
		Word:       (uint16(cond) << 14) | op,
		Immediates: imms,
	}
}

// MakeCodeExit creates an end-of-program instruction that sets IP to 0xffffffff.
func MakeCodeExit(cond CodeCond) Code {
	return MakeCodeAlu(cond, ALU_OP_SET, IR_IP, IR_CONST_FFFFFFFF)
}

// MakeCodeCapp creates a CAPP operation instruction.
func MakeCodeCapp(cond CodeCond, op CodeCappOp, src_v, src_m CodeIR, imms ...uint16) Code {
	return makeCond(cond, (uint16(OP_CAPP)<<11)|(uint16(op)<<8)|(uint16(src_v)<<4)|(uint16(src_m)<<0), imms...)
}

// MakeCodeIo creates an I/O operation instruction.
func MakeCodeIo(cond CodeCond, op CodeIoOp, channel CodeChannel, arg CodeIR, imms ...uint16) Code {
	return makeCond(cond, (uint16(OP_IO)<<11)|(uint16(op)<<8)|(uint16(channel)<<4)|(uint16(arg)<<0), imms...)
}

// MakeCodeAlu creates an ALU operation instruction.
func MakeCodeAlu(cond CodeCond, op CodeAluOp, target, arg CodeIR, imms ...uint16) Code {
	return makeCond(cond, (uint16(OP_ALU)<<11)|(uint16(op)<<8)|((uint16(target)&7)<<4)|(uint16(arg)<<0), imms...)
}

// MakeCodeCond creates a conditional comparison instruction.
func MakeCodeCond(cond CodeCond, op CodeCondOp, arg_a, arg_b CodeIR, imms ...uint16) Code {
	return makeCond(cond, (uint16(OP_COND)<<11)|(uint16(op)<<8)|(uint16(arg_a)<<4)|(uint16(arg_b)<<0), imms...)
}

// Cond returns the condition code from the instruction word.
func (code Code) Cond() CodeCond {
	word := uint16(code.Word)
	return CodeCond((word >> 14) & 0x3)
}

// Class returns the operation class (ALU, COND, CAPP, or IO) from the instruction word.
func (code Code) Class() CodeClass {
	word := uint16(code.Word)
	return CodeClass((word >> 11) & 0x3)
}

// AluDecode decodes and returns the ALU operation, target register, and argument.
func (code Code) AluDecode() (op CodeAluOp, target, arg CodeIR) {
	word := uint16(code.Word)
	op = CodeAluOp((word >> 8) & 0x7)
	target = CodeIR((word >> 4) & 0xf)
	arg = CodeIR((word >> 0) & 0xf)
	return
}

// CondDecode decodes and returns the conditional operation and its two arguments.
func (code Code) CondDecode() (op CodeCondOp, arg1, arg2 CodeIR) {
	word := uint16(code.Word)
	op = CodeCondOp((word >> 8) & 0x7)
	arg1 = CodeIR((word >> 4) & 0xf)
	arg2 = CodeIR((word >> 0) & 0xf)
	return
}

// CappDecode decodes and returns the CAPP operation, match value, and mask.
func (code Code) CappDecode() (op CodeCappOp, match, mask CodeIR) {
	word := uint16(code.Word)
	op = CodeCappOp((word >> 8) & 0x7)
	match = CodeIR((word >> 4) & 0xf)
	mask = CodeIR((word >> 0) & 0xf)
	return
}

// IoDecode decodes and returns the I/O operation, channel, and argument.
func (code Code) IoDecode() (op CodeIoOp, channel CodeChannel, arg CodeIR) {
	word := uint16(code.Word)
	op = CodeIoOp((word >> 8) & 0x7)
	channel = CodeChannel((word >> 4) & 0xf)
	arg = CodeIR((word >> 0) & 0xf)
	return
}

// ImmediateNeed returns the number of 16-bit immediate values required by this instruction.
func (code Code) ImmediateNeed() int {
	class := code.Class()

	a := IR_CONST_0
	b := IR_CONST_0

	switch class {
	case OP_CAPP:
		_, a, b = code.CappDecode()
	case OP_IO:
		_, _, a = code.IoDecode()
	case OP_ALU:
		_, _, a = code.AluDecode()
	case OP_COND:
		_, a, b = code.CondDecode()
	}

	need := 0
	if a == IR_IMMEDIATE_16 {
		need += 1
	}
	if b == IR_IMMEDIATE_16 {
		need += 1
	}
	if a == IR_IMMEDIATE_32 {
		need += 2
	}
	if b == IR_IMMEDIATE_32 {
		need += 2
	}

	return need
}

// String returns the assembly language representation of this instruction.
func (code Code) String() (out string) {
	cond := code.Cond()
	class := code.Class()

	var str string

	switch class {
	case OP_CAPP:
		op, match, mask := code.CappDecode()
		str = fmt.Sprintf("%v.%v.%v", op.String(), match.String(), mask.String())
	case OP_IO:
		op, channel, arg := code.IoDecode()
		str = fmt.Sprintf("%v.%v.%v", op.String(), channel.String(), arg.String())
	case OP_ALU:
		op, target, arg := code.AluDecode()
		str = fmt.Sprintf("%v.%v.%v", op.String(), target.String(), arg.String())
	case OP_COND:
		op, arg1, arg2 := code.CondDecode()
		str = fmt.Sprintf("%v.%v.%v", op.String(), arg1.String(), arg2.String())
	}

	out = fmt.Sprintf("%v%v.%v imm:%#v", cond.String(), class.String(), str, code.Immediates)

	return
}
