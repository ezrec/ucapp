package cpu

import (
	"iter"
)

// Program is a list of opcodes.
type Program struct {
	Opcodes []Opcode
}

// Debug contains debugging information for a program instruction pointer.
type Debug struct {
	*Opcode     // Opcode for the index.
	Index   int // Index into the Program's Opcodes array.
}

// Debug returns debugging information for the instruction at the given IP address.
func (prog *Program) Debug(ip uint16) (dbg Debug) {
	for n, op := range prog.Opcodes {
		if ip >= uint16(op.Ip) && ip < uint16(op.Ip)+uint16(len(op.Codes)) {
			index := int(ip - uint16(op.Ip))
			dbg = Debug{
				Opcode: &prog.Opcodes[n],
				Index:  index,
			}
			break
		}
	}

	return
}

// Binary returns the program as a list of 32-bit CAPP memory words.
func (prog *Program) Binary() (bins []uint32) {
	for ip, code := range prog.Codes() {
		var data []uint32
		for _, imm := range code.Immediates {
			data = append(data, ARENA_CODE|(uint32(ip)<<16)|uint32(imm))
		}
		data = append(data, ARENA_CODE|(uint32(ip)<<16)|uint32(code.Word))
		bins = append(bins, data...)
	}

	return
}

// Codes returns an iterator over all instruction pointer addresses and their codes.
func (prog *Program) Codes() iter.Seq2[uint16, Code] {
	return func(yield func(ip uint16, code Code) bool) {
		for _, op := range prog.Opcodes {
			ip := uint16(op.Ip)
			for n, code := range op.Codes {
				if !yield(ip+uint16(n), code) {
					return
				}
			}
		}
	}
}
