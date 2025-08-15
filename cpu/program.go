package cpu

import (
	"iter"
)

type Program struct {
	Opcodes []Opcode
}

type Debug struct {
	*Opcode
	Index int
}

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

func (prog *Program) Binary() (bins []uint32) {
	for ip, code := range prog.Codes() {
		data := ARENA_CODE | (uint32(ip) << 20) | uint32(code)
		bins = append(bins, data)
	}

	return
}

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
