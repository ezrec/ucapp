package cpu

import (
	"errors"

	"github.com/ezrec/ucapp/translate"
)

var f = translate.From

var (
	// Cpu errors
	ErrIpEmpty        = errors.New(f("ip empty"))
	ErrIpMultiple     = errors.New(f("ip multiple"))
	ErrIpTrap         = errors.New(f("ip trap"))
	ErrIpKey          = errors.New(f("ip key unknown"))
	ErrStackEmpty     = errors.New(f("stack empty"))
	ErrStackFull      = errors.New(f("stack full"))
	ErrChannelInvalid = errors.New(f("channel invalid"))
	ErrChannelPartial = errors.New(f("partial channel read"))
	ErrChannelFull    = errors.New(f("channel full"))

	// Instruction decode errors
	ErrOpcodeDecode = errors.New(f("decode"))
	ErrOpcodeAlu    = errors.New(f("alu"))
	ErrOpcodeCond   = errors.New(f("cond"))
	ErrOpcodeCapp   = errors.New(f("capp"))
	ErrOpcodeIo     = errors.New(f("io"))
	ErrOpcodeOp     = errors.New(f("op"))
	ErrOpcodeArg1   = errors.New(f("arg1"))
	ErrOpcodeArg2   = errors.New(f("arg2"))
	ErrOpcodeImm    = errors.New(f("imm"))

	// Assembler errors
	ErrEquateSyntax       = errors.New(f(".equ syntax"))
	ErrEquateDuplicate    = errors.New(f(".equ duplicated"))
	ErrLabelDuplicate     = errors.New(f("label duplicated"))
	ErrMacroSyntax        = errors.New(f(".macro syntax"))
	ErrMacroNesting       = errors.New(f(".macro in .macro prohibited"))
	ErrMacroDuplicate     = errors.New(f(".macro duplicated"))
	ErrMacroLonely        = errors.New(f(".macro wihtout .endm"))
	ErrMacroLonelyEndm    = errors.New(f(".endm without .macro"))
	ErrOpcodeExtraArgs    = errors.New(f("excessive arguments"))
	ErrOpcodeMissing      = errors.New(f("opcode missing"))
	ErrOpcodeValueMissing = errors.New(f("value missing"))
	ErrOpcodeInvalid      = errors.New(f("opcode invalid"))
	ErrRegisterInvalid    = errors.New(f("register invalid"))
	ErrTargetMissing      = errors.New(f("target missing"))
	ErrTargetInvalid      = errors.New(f("target invalid"))
	ErrInstructionInvalid = errors.New(f("instruction invalid"))
)

// ErrLabelMissing indicates a missing jump label.
type ErrLabelMissing string

func (el ErrLabelMissing) Error() string {
	return f("label %v missing", string(el))
}

// ErrOpcode indicates an invalid opcode.
type ErrOpcode Code

func (eo ErrOpcode) Error() string {
	return f("bad opcode 0x%04x %v", uint16(eo.Word), Code(eo).String())
}

func (eo ErrOpcode) Is(err error) (ok bool) {
	_, ok = err.(ErrOpcode)
	return
}

// ErrSyntax locates an error in the assembly listing.
type ErrSyntax struct {
	LineNo int
	Line   string
	Err    error
}

func (err ErrSyntax) Error() string {
	return f("line %d \"%v\": %v", err.LineNo, err.Line, err.Err)
}

func (err ErrSyntax) Unwrap() error {
	return err.Err
}

// ErrParseCharacter indicates a character ('x') parsing failure.
type ErrParseCharacter string

func (err ErrParseCharacter) Error() string {
	return f("'%v' is not a single byte character", string(err))
}

// ErrParseNumber indicates a numeric parsing failure.
type ErrParseNumber string

func (err ErrParseNumber) Error() string {
	return f("%v is not a number", string(err))
}

// ErrParseExpression indicates a $(..) expression parsing failure.
type ErrParseExpression string

func (err ErrParseExpression) Error() string {
	return f("$(%v) is not a valid expression", string(err))
}

// ErrMacro locates a macro error in the assembly listing.
type ErrMacro struct {
	Macro string
	Line  int
	Err   error
}

func (err ErrMacro) Error() string {
	return f("macro %v line %v %v", err.Macro, err.Line, err.Err.Error())
}

func (err ErrMacro) Unwrap() error {
	return err.Err
}
