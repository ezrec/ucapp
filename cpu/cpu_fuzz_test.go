package cpu

import (
	"bytes"
	"errors"
	"fmt"
	"math/rand"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ezrec/ucapp/capp"
	"github.com/ezrec/ucapp/io"
)

func shl(input uint32, rot uint32) uint32 {
	rot &= 0x1f
	return input << rot
}

func shr(input uint32, rot uint32) uint32 {
	rot &= 0x1f
	return input >> rot
}

func FuzzCpu(f *testing.F) {
	for rv := range 0xf {
		stack := ((rv >> 20) & 1) == 1
		alerted := ((rv >> 21) & 1) == 1
		inputs := uint8((rv >> 22) & 0x3)
		f.Add(uint16(0), stack, inputs, alerted)
		f.Add(uint16(0xffff), stack, inputs, alerted)
	}

	f.Fuzz(func(t *testing.T, opcode uint16, stack bool, inputs uint8, alerted bool) {
		assert := assert.New(t)

		code := Code{Word: opcode,
			Immediates: []uint16{0x1B2B, 0x3B4B, 0x1A2A, 0x3A4A},
		}

		code.Immediates = code.Immediates[:code.ImmediateNeed()]

		cpu := NewCpu(1024)
		cpu.Capp.Action(capp.SET_OF, ARENA_FREE, ARENA_MASK)
		cpu.Capp.Action(capp.LIST_ALL, 0, 0)
		cpu.Capp.Action(capp.WRITE_FIRST, ARENA_IO|0xcafe30, 0xffffffff)
		cpu.Capp.Action(capp.LIST_NEXT, 0, 0)
		cpu.Capp.Action(capp.WRITE_FIRST, ARENA_IO|0xcafe31, 0xffffffff)
		cpu.Capp.Action(capp.LIST_NEXT, 0, 0)
		cpu.Capp.Action(capp.LIST_NOT, 0, 0)
		cpu.Capp.Action(capp.SET_OF, ARENA_IO, ARENA_MASK)
		cpu.Capp.Action(capp.LIST_ALL, 0, 0)
		cpu.Capp.Action(capp.LIST_NEXT, 0, 0)
		cpu.Mask = ARENA_MASK
		cpu.Match = ARENA_IO

		assert.Equal(uint32(ARENA_IO|0xcafe31), cpu.Capp.First())
		assert.Equal(uint(1), cpu.Capp.Count())

		cpu.Ip = 0x1ab
		cpu.Cond = true
		cpu.Register[0] = 0x50607080
		cpu.Register[1] = 0x51617181
		cpu.Register[2] = 0x52627282
		cpu.Register[3] = 0x53637383
		cpu.Stack.Reset()
		if stack {
			cpu.Stack.Push(0xabcd1234)
		}

		tape := &io.Tape{}

		tape_output := &bytes.Buffer{}
		tape.Output = tape_output

		tape_input := make([]byte, inputs)
		for n := range inputs {
			tape_input[n] = uint8(rand.Uint32() & 0xff)
		}
		tape.Input = bytes.NewReader(tape_input)
		tape_alert := uint32(0x9a128923)
		if alerted {
			tape.SetAlert(tape_alert)
		}
		cpu.SetChannel(CHANNEL_ID_TAPE, tape)

		pre_input := slices.Clone(tape_input)
		pre_value := map[CodeIR]uint32{}
		pre_value[IR_REG_R0] = cpu.Register[0]
		pre_value[IR_REG_R1] = cpu.Register[1]
		pre_value[IR_REG_R2] = cpu.Register[2]
		pre_value[IR_REG_R3] = cpu.Register[3]
		pre_value[IR_REG_R4] = cpu.Register[4]
		pre_value[IR_REG_R5] = cpu.Register[5]
		pre_value[IR_CONST_0] = 0
		pre_value[IR_CONST_FFFFFFFF] = 0xffffffff
		pre_value[IR_IP] = uint32((cpu.Ip + 1) & 0x3ff) // next_ip
		pre_value[IR_STACK] = 0
		if stack {
			pre_value[IR_STACK], _ = cpu.Stack.Peek()
		}
		pre_value[IR_REG_MATCH] = cpu.Match
		pre_value[IR_REG_MASK] = cpu.Mask
		pre_value[IR_REG_FIRST] = cpu.Capp.First()
		pre_value[IR_REG_COUNT] = uint32(cpu.Capp.Count())

		next_ip := cpu.Ip + 1

		assert.Equal(uint32(ARENA_IO|0xcafe31), cpu.Capp.First())
		assert.Equal(uint(1), cpu.Capp.Count())

		err := cpu.Execute(code)

		code_str := fmt.Sprintf("0x%04x (%v)\nimm: %#v stack:%v inputs:%v alerted:%v\ncpu:%v",
			code.Word, code, code.Immediates, stack, inputs, alerted, cpu.String())

		now_value := func(dst CodeIR) (output uint32, squash func(val uint32) uint32) {
			squash = func(val uint32) uint32 { return val }
			switch dst {
			case IR_CONST_0:
				output = 0
				squash = func(val uint32) uint32 { return 0 }
			case IR_REG_R0, IR_REG_R1, IR_REG_R2, IR_REG_R3, IR_REG_R4, IR_REG_R5:
				output = cpu.Register[dst-IR_REG_R0]
			case IR_STACK:
				output, _ = cpu.Stack.Pop()
			case IR_IP:
				output = uint32(cpu.Ip)
			default:
				panic(ErrOpcode(code))
			}
			return
		}

		if err != nil {
			switch {
			case stack == false && errors.Is(err, ErrStackEmpty):
				switch {
				case errors.Is(err, ErrOpcodeArg2):
					switch (code.Word >> 0) & 0xf {
					case 7:
						// expected error
					default:
						assert.NoError(err, code_str)
					}
				case errors.Is(err, ErrOpcodeArg1):
					switch (code.Word >> 4) & 0xf {
					case 7:
						// expected error
					default:
						assert.NoError(err, code_str)
					}
				default:
					assert.NoError(err, code_str)
				}
			case stack == true && errors.Is(err, ErrStackEmpty):
				if (code.Word & 0xff) == 0x77 {
					// expected error
				} else {
					assert.NoError(err, code_str)
				}
			case errors.Is(err, ErrOpcodeCapp):
				op, value, mask := code.CappDecode()
				switch op {
				case CAPP_OP_LIST_ALL, CAPP_OP_LIST_NOT, CAPP_OP_LIST_NEXT:
					switch {
					case errors.Is(err, ErrOpcodeArg1) || errors.Is(err, ErrOpcodeArg2):
						if IR_CONST_0 != mask {
							// expected error
						} else if IR_CONST_0 != value {
							// expected error
						} else {
							assert.NoError(err, code_str)
						}
					default:
						assert.NoError(err, code_str)
					}
				default:
					assert.NoError(err, code_str)
				}
			case errors.Is(err, ErrOpcodeAlu):
				switch {
				case errors.Is(err, ErrOpcodeArg1):
					switch {
					case ((code.Word >> 4) & 0x8) == 0x8:
						// expected error
					default:
						assert.NoError(err, code_str)
					}
				default:
					assert.NoError(err, code_str)
				}
			case errors.Is(err, ErrChannelInvalid):
				switch code.Class() {
				case OP_IO:
					_, channel, _ := code.IoDecode()
					_, err_tmp := cpu.GetChannel(channel)
					if err_tmp == nil {
						assert.NoError(err, code_str)
					} else {
						// expected error
					}
				default:
					assert.NoError(err, code_str)
				}
			case errors.Is(err, ErrChannelPartial):
				switch code.Class() {
				case OP_IO:
					_, channel, _ := code.IoDecode()
					if channel == CHANNEL_ID_TAPE && cpu.Capp.Count() == 0 {
						// expected error
					} else {
						assert.NoError(err, code_str)
					}
				default:
					assert.NoError(err, code_str)
				}
			case errors.Is(err, ErrOpcode(Code{})):
				switch code.Class() {
				case OP_IO:
					op, ch, _ := code.IoDecode()
					switch op {
					case IO_OP_FETCH:
						switch ch {
						case CHANNEL_ID_TAPE:
							assert.NoError(err, code_str)
						}
					case IO_OP_STORE:
						switch ch {
						case CHANNEL_ID_TAPE:
							assert.NoError(err, code_str)
						}
					default:
						// expected error
					}
				case OP_CAPP:
					op, match, mask := code.CappDecode()
					switch {
					case op == CAPP_OP_SET_SWAP:
						// expected error
					case (op & 0xc) == 0x0:
						// No arguments
						if match == IR_CONST_0 && mask == IR_CONST_0 {
							assert.NoError(err, code_str)
						} else {
							// expected error
						}
					case (op & 0xc) == 0x4:
						// V & M args
						if code.Word&0xf00 == 0 {
							assert.NoError(err, code_str)
						} else {
							// expected error
						}
					default:
						// expected error
					}
				case OP_ALU:
					op, dst, _ := code.AluDecode()
					if (op & 0x8) == 0 {
						switch dst {
						case IR_STACK, IR_REG_R0, IR_REG_R1, IR_REG_R2, IR_REG_R3:
							// Valid code
							assert.NoError(err, code_str)
						default:
							// expected error
						}
					} else {
						// expected error
					}
				default:
					// expected error
				}
			default:
				type unwrapper interface {
					Unwrap() []error
				}
				var errstr string
				errset, ok := err.(unwrapper)
				if ok {
					errs := errset.Unwrap()
					for _, e := range errs {
						errstr += ";" + e.Error()
					}
				} else {
					errstr = ";" + err.Error()
				}
				assert.NoError(err, code_str+errstr)
			}
			return
		}

		imms := code.Immediates[:]

		get_value := func(arg CodeIR, imms_in []uint16) (value uint32, imms []uint16) {
			imms = imms_in
			switch arg {
			case IR_IMMEDIATE_16:
				value = uint32(imms[0])
				imms = imms[1:]
			case IR_IMMEDIATE_32:
				value = (uint32(imms[0]) << 16) | uint32(imms[1])
				imms = imms[2:]
			default:
				value = pre_value[arg]
			}

			return
		}

		switch code.Class() {
		case OP_ALU:
			op, dst, arg := code.AluDecode()
			input, imms := get_value(dst, imms)
			value, _ := get_value(arg, imms)
			output, squash := now_value(dst)
			var expected uint32
			switch op {
			case ALU_OP_SET:
				expected = value
			case ALU_OP_XOR:
				expected = value ^ input
			case ALU_OP_AND:
				expected = value & input
			case ALU_OP_OR:
				expected = value | input
			case ALU_OP_SHL:
				expected = shl(input, value)
			case ALU_OP_SHR:
				expected = shr(input, value)
			case ALU_OP_ADD:
				expected = value + input
			case ALU_OP_SUB:
				expected = input + (^value + 1)
			}
			assert.Equal(squash(expected), output, code_str)
			if dst == IR_IP {
				next_ip = expected
			}
		case OP_CAPP:
			op, ir_match, ir_mask := code.CappDecode()
			value, imms := get_value(ir_match, imms)
			mask, _ := get_value(ir_mask, imms)
			expect_first := pre_value[IR_REG_FIRST]
			expect_count := pre_value[IR_REG_COUNT]
			expect_match := pre_value[IR_REG_MATCH]
			expect_mask := pre_value[IR_REG_MASK]
			switch op {
			case CAPP_OP_SET_OF:
				expect_match = value
				expect_mask = mask
				if ((ARENA_IO | 0xcafe31) & mask) == (value & mask) {
					expect_first = ARENA_IO | 0xcafe31
					expect_count = 1
				} else {
					expect_first = 0
					expect_count = 0
				}
			case CAPP_OP_LIST_ALL:
				expect_first = ARENA_IO | 0xcafe30
				expect_count = 2
			case CAPP_OP_LIST_NOT:
				expect_first = ARENA_IO | 0xcafe30
				expect_count = 1
			case CAPP_OP_LIST_NEXT:
				expect_first = 0
				expect_count = 0
			case CAPP_OP_LIST_ONLY:
				if ((ARENA_IO | 0xcafe31) & mask) == (value & mask) {
					expect_first = ARENA_IO | 0xcafe31
					expect_count = 1
				} else {
					expect_first = 0
					expect_count = 0
				}
			case CAPP_OP_WRITE_FIRST:
				expect_first = (expect_first & ^mask) | (value & mask)
			case CAPP_OP_WRITE_LIST:
				expect_first = (expect_first & ^mask) | (value & mask)
			default:
				panic(ErrOpcode(code))
			}
			assert.Equal(expect_first, cpu.Capp.First(), code_str)
			assert.Equal(expect_count, uint32(cpu.Capp.Count()), code_str)
			assert.Equal(expect_match, cpu.Match, code_str)
			assert.Equal(expect_mask, cpu.Mask, code_str)
		case OP_IO:
			op, ch, ir_mask := code.IoDecode()
			mask, _ := get_value(ir_mask, imms)
			expect_first := pre_value[IR_REG_FIRST]
			expect_count := pre_value[IR_REG_COUNT]
			expect_match := pre_value[IR_REG_MATCH]
			expect_mask := pre_value[IR_REG_MASK]
			switch op {
			case IO_OP_FETCH:
				if ch == CHANNEL_ID_TAPE {
					// Input from the tape completes with a LIST_NOT,
					// which will set the FIRST to our original magic.
					expect_count = uint32(1 + len(pre_input))
					expect_first = 0xcafe30
				}
			case IO_OP_STORE:
				if ch == CHANNEL_ID_TAPE {
					tape_bytes := tape_output.Bytes()
					assert.Equal(uint32(len(tape_bytes)), expect_count)
					if expect_count > 0 {
						assert.Equal(tape_bytes[0], (expect_first & mask))
					}
				}
			case IO_OP_AWAIT:
				if ch == CHANNEL_ID_TAPE {
					if alerted {
						output, squash := now_value(ir_mask)
						assert.Equal(squash(tape_alert), output)
					} else {
						next_ip = cpu.Ip
					}
				}
			case IO_OP_ALERT:
				if ch == CHANNEL_ID_TAPE {
					alert, ok := tape.GetAlert()
					if assert.True(ok) {
						assert.Equal(alert, mask)
					}
				}
			default:
				panic(ErrOpcode(code))
			}
			assert.Equal(expect_first, cpu.Capp.First(), code_str)
			assert.Equal(expect_count, uint32(cpu.Capp.Count()), code_str)
			assert.Equal(expect_match, cpu.Match, code_str)
			assert.Equal(expect_mask, cpu.Mask, code_str)
		case OP_COND:
			op, a_ir, b_ir := code.CondDecode()
			a, imms := get_value(a_ir, imms)
			b, _ := get_value(b_ir, imms)
			var cond bool
			switch op {
			case COND_OP_EQ:
				cond = int32(a) == int32(b)
			case COND_OP_NE:
				cond = int32(a) != int32(b)
			case COND_OP_LT:
				cond = int32(a) < int32(b)
			case COND_OP_LE:
				cond = int32(a) <= int32(b)
			}
			assert.Equal(cond, cpu.Cond, code_str)
		default:
			panic(ErrOpcode(code))
		}

		assert.Equal(next_ip, cpu.Ip, code_str)
	})
}
