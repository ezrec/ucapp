package cpu

import (
	"iter"
)

// Program is a list of opcodes.
type Program struct {
	Opcodes []Opcode
}

// Debug information.
type Debug struct {
	*Opcode     // Opcode for the index.
	Index   int // Index into the Program's Opcodes array.
}

// Debug returns the information about the program listing at the IP.
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

// Binary returns the instruction list version of the program.
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

// Codes returns an interator over all of IPs and instruction codes of the program.
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
