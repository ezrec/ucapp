package cpu

import (
	"errors"
	"fmt"
	"iter"
	"log"
	"maps"
	"math/bits"

	"github.com/ezrec/ucapp/capp"
	"github.com/ezrec/ucapp/io"
)

// Channel is an I/O channel interface.
type Channel io.Channel

// Instruction Pointer execution mode constants.
// The upper 2 bits of the IP determine the source for instruction fetch.
const (
	IP_MODE_CAPP  = uint32(0b00 << 30) // Execute from CAPP
	IP_MODE_STACK = uint32(0b01 << 30) // Execute from stack
	IP_MODE_REG   = uint32(0b10 << 30) // Execute from register bank
	IP_MODE_MASK  = uint32(0b11 << 30) // Mask of execute modes.
)

var _cpu_defines = map[string]string{
	"IP_MODE_CAPP":  fmt.Sprintf("0x%x", IP_MODE_CAPP),
	"IP_MODE_STACK": fmt.Sprintf("0x%x", IP_MODE_STACK),
	"IP_MODE_REG":   fmt.Sprintf("0x%x", IP_MODE_REG),
	"IP_MODE_MASK":  fmt.Sprintf("0x%x", IP_MODE_MASK),
}

// CpuChannel represents an I/O channel attached to the CPU with its response channel.
type CpuChannel struct {
	Channel  Channel
	Response chan uint32
}

// Cpu is the simulation context for the control CPU attached to the CAPP
type Cpu struct {
	Verbose bool // Set to enable verbose logging.

	Capp *capp.Capp // Reference to the CAPP simulation.

	Ip       uint32    // Current instruction pointer.
	Register [6]uint32 // Register bank.
	Stack    Stack     // Stack simulation.
	Match    uint32    // Match value sent to the CAPP.
	Mask     uint32    // Mask value sent to the CAPP.
	Cond     bool      // Current conditional execution state.

	Power int // Power (bits flipped) counter.
	Ticks int // CPU ticks counter.

	channel [8](*CpuChannel) // IO channels.
}

// NewCpu creates a new CPU with a specifically sized CAPP.
func NewCpu(count uint) (cpu *Cpu) {
	cpu = &Cpu{
		Capp: capp.NewCapp(count),
	}

	return
}

// Defines for the cpu
func (cpu *Cpu) Defines() iter.Seq2[string, string] {
	return maps.All(_cpu_defines)
}

// Close closes all I/O channels associated with the CPU.
func (cpu *Cpu) Close() (err error) {
	for _, ch := range cpu.channel {
		if ch != nil {
			close(ch.Response)
		}
	}

	return
}

// String returns the current CPU state as a string.
func (cpu *Cpu) String() (text string) {
	regs := []string{
		"ip",
		"cond",
		"r0", "r1", "r2", "r3", "r4", "r5",
		"stack",
		"match", "mask", "first", "count",
	}
	for _, reg := range regs {
		var strval string
		switch reg {
		case "ip":
			val := cpu.Ip
			strval = fmt.Sprintf("%04x_%03X", (val >> 16), val&0x3ff)
		case "cond":
			strval = "false"
			if cpu.Cond {
				strval = "true"
			}
		case "r0", "r1", "r2", "r3", "r4", "r5":
			val := cpu.Register[byte(reg[1]-'0')]
			strval = fmt.Sprintf("%04X_%04X", val>>16, val&0xffff)
		case "stack":
			var val uint32
			val, ok := cpu.Stack.Peek()
			if ok {
				strval = fmt.Sprintf("%04X_%04X", val>>16, val&0xffff)
			} else {
				strval = "----_----"
			}
		case "match":
			val := cpu.Match
			strval = fmt.Sprintf("%01X_%07X", val>>28, val&0xfffffff)
		case "mask":
			val := cpu.Mask
			strval = fmt.Sprintf("%01X_%07X", val>>28, val&0xfffffff)
		case "first":
			val := cpu.Capp.First()
			strval = fmt.Sprintf("%01X_%07X", val>>28, val&0xfffffff)
		case "count":
			val := cpu.Capp.Count()
			strval = fmt.Sprintf("%01X_%07X", val>>28, val&0xfffffff)
		}
		text += fmt.Sprintf("% 5s: %v\n", reg, strval)
	}

	return
}

// Reset the CPU state.
// - Clears the registers, stack, and CAPP.
// - Zeros statistics counters.
// - Resets all IO channels.
// - Installs trampoline to boot from CAPP in register bank.
// - Sets CPU execution mode to boot-from-register bank.
func (cpu *Cpu) Reset(boot CodeChannel) (err error) {
	if cpu.Verbose {
		log.Printf("cpu: reset")
	}

	clear(cpu.Register[:])
	cpu.Stack.Reset()
	cpu.Capp.Reset()
	cpu.Ticks = 0
	cpu.Power = 0

	for _, channel := range cpu.channel {
		if channel == nil {
			continue
		}
		channel.Channel.Rewind()
	}

	// r0: .list.of.immz.immz        ; Select all of the CAPP
	// r1: .list.all.-.-             ; Tag all items
	// r2: .list.write.immnz   ; Replace all values with 0xFFFFFFFF
	// r3: .io.fetch.rom.immnz    ; Load boot channel into CAPP
	// r4: .list.not.-.-             ; Now, only the program is tagged
	// r5: .alu.set.ip.immz      ; Set IP to 0x00000000 (exec from CAPP)

	bootstrap := [6]Code{
		MakeCodeCapp(COND_ALWAYS, CAPP_OP_SET_OF, IR_CONST_0, IR_CONST_0),
		MakeCodeCapp(COND_ALWAYS, CAPP_OP_LIST_ALL, IR_CONST_0, IR_CONST_0),
		MakeCodeCapp(COND_ALWAYS, CAPP_OP_WRITE_LIST, IR_CONST_FFFFFFFF, IR_CONST_FFFFFFFF),
		MakeCodeIo(COND_ALWAYS, IO_OP_FETCH, boot, IR_CONST_FFFFFFFF),
		MakeCodeCapp(COND_ALWAYS, CAPP_OP_LIST_NOT, IR_CONST_0, IR_CONST_0),
		MakeCodeAlu(COND_ALWAYS, ALU_OP_SET, IR_IP, IR_CONST_0),
	}

	for n, code := range bootstrap {
		if len(code.Immediates) != 0 {
			panic("bootstrap cannot use immediates")
		}
		cpu.Register[n] = uint32(code.Word)
	}

	// Set IP to run from registers
	cpu.Ip = IP_MODE_REG

	if cpu.Verbose {
		log.Printf("cpu: boot from channel %v", boot)
	}

	return
}

// SetChannel sets a channel index to a channel simulation model.
func (cpu *Cpu) SetChannel(index CodeChannel, channel Channel) {
	if channel != nil {
		cpu.channel[int(index)] = &CpuChannel{
			Channel:  channel,
			Response: make(chan uint32, 8),
		}
	} else {
		if cpu.channel[int(index)] != nil {
			close(cpu.channel[int(index)].Response)
		}
		cpu.channel[int(index)] = nil
	}
}

// GetChannel gets the channel simulation model by index.
func (cpu *Cpu) GetChannel(ch CodeChannel) (channel Channel, response chan uint32, err error) {
	index := int(ch)
	if index >= len(cpu.channel) || cpu.channel[index] == nil {
		err = ErrChannelInvalid
		return
	}

	channel = cpu.channel[index].Channel
	response = cpu.channel[index].Response
	return
}

// listInput reads from a channel into the active list in the CAPP.
// For each tagged item in the list:
//   - For each set bit in 'mask':
//   - Collect the next bit from the input, starting at LSB.
//   - Replace the masked bits in the list's first entry with read value.
//   - Advance the list to the next entry.
//
// Reads until end of input, or the active list is empty.
// If the read is short, the active list count is the remaining
// available space.
func (cpu *Cpu) listInput(in Channel, mask uint32) (err error) {
	inputs := in.Receive()
	cp := cpu.Capp

	if mask == 0 {
		return
	}

	var n int
	tmp := mask
	var inval uint32
	for input := range inputs {
		if cp.Count() == 0 {
			return
		}
		for (tmp & 1) == 0 {
			n++
			tmp >>= 1
		}
		if input {
			inval |= (1 << n)
		}
		n++
		tmp >>= 1
		if tmp == 0 {
			n = 0
			cp.Action(capp.WRITE_FIRST, inval, mask)
			cp.Action(capp.LIST_NEXT, 0, 0)
			tmp = mask
			inval = 0
		}
	}
	return
}

// listOutput writes to a channel from the active list in the CAPP.
// For each tagged item in the list:
//   - For each set bit in 'mask':
//   - Write the next bit from the list's first entry to the output,
//     starting at LSB.
//   - Advance the list to the next entry.
//
// Writes until the active list is empty.
func (cpu *Cpu) listOutput(out Channel, mask uint32) (err error) {
	cp := cpu.Capp

	if mask == 0 {
		return
	}

	for cp.Count() > 0 {
		inval := cp.First()
		for n := range 32 {
			if (mask & (1 << n)) != 0 {
				bit := ((inval >> n) & 1) != 0
				err = out.Send(bit)
				if err != nil {
					return
				}
			}
		}
		cp.Action(capp.LIST_NEXT, 0, 0)
	}
	return
}

// FetchCode fetches the next instruction to execute based on the IP mode and address.
func (cpu *Cpu) FetchCode() (code Code, err error) {
	if cpu.Ip == 0xffffffff {
		err = ErrIpEmpty
		return
	}

	switch cpu.Ip & IP_MODE_MASK {
	case IP_MODE_REG:
		reg := int(cpu.Ip & ^IP_MODE_MASK)
		if reg >= len(cpu.Register) {
			log.Printf("Ip 0x%x > register len 0x%x", cpu.Ip, len(cpu.Register))
			err = ErrIpEmpty
			return
		}
		code = Code{Word: uint16(cpu.Register[reg])}
	case IP_MODE_STACK:
		opcode, ok := cpu.Stack.Pop()
		if !ok {
			log.Printf("Ip 0x%x > stack empty", cpu.Ip)
			err = ErrIpEmpty
			return
		}
		code = Code{Word: uint16(opcode)}
	case IP_MODE_CAPP:
		cpu.Capp.Verbose = false
		cpu.Capp.Action(capp.SET_SWAP, 0, 0)
		cpu.Capp.Action(capp.SET_OF, ARENA_CODE|(uint32(cpu.Ip&0x3fff)<<16), ARENA_MASK|(0x3fff<<16))
		cpu.Capp.Action(capp.LIST_ALL, 0, 0)

		count := cpu.Capp.Count()
		var imms []uint16
		for count > 1 {
			// save immediates
			imms = append(imms, uint16(cpu.Capp.First()&0xffff))
			cpu.Capp.Action(capp.LIST_NEXT, 0, 0)
			count = cpu.Capp.Count()
		}
		first := cpu.Capp.First()
		cpu.Capp.Action(capp.SET_SWAP, 0, 0)
		if count != 1 {
			err = ErrIpEmpty
			return
		}
		code = Code{Word: uint16(first & 0xffff), Immediates: imms}

		cpu.Capp.Verbose = cpu.Verbose
	default:
		log.Printf("Ip 0x%x > unknown source", cpu.Ip)
		err = ErrIpEmpty
		return
	}

	return
}

// Tick executes a single CPU instruction cycle.
func (cpu *Cpu) Tick() (err error) {
	// Set CAPP verbosity
	cpu.Capp.Verbose = cpu.Verbose

	code, err := cpu.FetchCode()
	if err != nil {
		return
	}

	err = cpu.Execute(code)
	if err != nil {
		return
	}

	// Check for trap on PROGRAM channel.
	_, trap, err := cpu.GetChannel(CHANNEL_ID_MONITOR)
	if err == nil {
		select {
		case _, ok := <-trap:
			if !ok {
				err = ErrChannelInvalid
			} else {
				err = ErrIpTrap
			}
		default:
			err = nil
		}
	} else {
		err = nil
	}

	return
}

// Execute executes a single decoded instruction.
func (cpu *Cpu) Execute(code Code) (err error) {
	defer func() {
		if err != nil {
			err = errors.Join(ErrOpcode(code), err)
		}
	}()
	if cpu.Verbose {
		log.Printf("%03x: %v", cpu.Ip, code)
	}

	cp := cpu.Capp

	cpu.Capp.BitsFlipped = 0

	next_ip := cpu.Ip + 1

	// Bits flipped by CPU action.
	var prior uint64
	var result uint64

	no_op := MakeCodeAlu(COND_ALWAYS, ALU_OP_OR, IR_REG_R0, IR_CONST_0)

	cond := code.Cond()
	switch cond {
	case COND_ALWAYS:
		// pass
	case COND_NEVER:
		return ErrOpcode(code)
	case COND_TRUE:
		if !cpu.Cond {
			// Convert to no-op
			code = no_op
		}
	case COND_FALSE:
		if cpu.Cond {
			// Convert to no-op
			code = no_op
		}
	}

	imms := code.Immediates

	switch code.Class() {
	case OP_ALU:
		op, dst, arg := code.AluDecode()
		var val uint32
		val, imms, err = cpu.getValue(arg, imms)
		if err != nil {
			err = errors.Join(ErrOpcodeAlu, ErrOpcodeArg2, err)
			return
		}
		var input uint32
		var set_target func(value uint32)
		switch dst {
		case IR_IP:
			input = next_ip
			set_target = func(value uint32) { next_ip = value }
		case IR_STACK:
			if cpu.Stack.Full() {
				err = errors.Join(ErrOpcodeAlu, ErrOpcodeArg1, ErrStackFull)
				return
			}
			if op == ALU_OP_SET {
				input = 0
			} else {
				var ok bool
				input, ok = cpu.Stack.Pop()
				if !ok {
					err = errors.Join(ErrOpcodeAlu, ErrOpcodeArg1, ErrStackEmpty)
					return
				}
			}
			// Stask write (push)
			set_target = func(value uint32) { cpu.Stack.Push(value) }
		case IR_REG_R0, IR_REG_R1, IR_REG_R2, IR_REG_R3, IR_REG_R4, IR_REG_R5:
			dst -= IR_REG_R0
			input = cpu.Register[dst]
			set_target = func(value uint32) { cpu.Register[dst] = value }
		default:
			err = errors.Join(ErrOpcodeAlu, ErrOpcodeArg1)
			return
		}
		prior = uint64(input)
		output := cpu.doAlu(op, input, val)
		set_target(output)
		result = uint64(output)
	case OP_COND:
		op, a_ir, b_ir := code.CondDecode()
		var a_u uint32
		var b_u uint32
		a_u, imms, err = cpu.getValue(a_ir, imms)
		if err != nil {
			err = errors.Join(ErrOpcodeCond, ErrOpcodeArg1, err)
			return
		}
		b_u, imms, err = cpu.getValue(b_ir, imms)
		if err != nil {
			err = errors.Join(ErrOpcodeCond, ErrOpcodeArg2, err)
			return
		}
		// Treat as signed.
		a := int32(a_u)
		b := int32(b_u)
		switch op {
		case COND_OP_EQ:
			cpu.Cond = a == b
		case COND_OP_NE:
			cpu.Cond = a != b
		case COND_OP_LT:
			cpu.Cond = a < b
		case COND_OP_LE:
			cpu.Cond = a <= b
		default:
			err = errors.Join(ErrOpcodeCond, ErrOpcodeOp)
			return
		}
	case OP_CAPP:
		op, match, mask := code.CappDecode()
		var val uint32
		var msk uint32
		val, imms, err = cpu.getValue(match, imms)
		if err != nil {
			err = errors.Join(ErrOpcodeCapp, ErrOpcodeArg1, err)
			return
		}
		msk, imms, err = cpu.getValue(mask, imms)
		if err != nil {
			err = errors.Join(ErrOpcodeCapp, ErrOpcodeArg2, err)
			return
		}
		switch op {
		case CAPP_OP_SET_SWAP:
			// Not permitted from the instruction stream - soley used
			// by the CPU during instruction fetch.
			err = ErrOpcodeCapp
			return
		case CAPP_OP_LIST_ALL:
			if match != IR_CONST_0 || mask != IR_CONST_0 {
				err = errors.Join(ErrOpcodeCapp, ErrOpcodeArg1, ErrOpcodeArg2)
				return
			}
			cp.Action(capp.LIST_ALL, 0, 0)
		case CAPP_OP_LIST_NOT:
			if match != IR_CONST_0 || mask != IR_CONST_0 {
				err = errors.Join(ErrOpcodeCapp, ErrOpcodeArg1, ErrOpcodeArg2)
				return
			}
			cp.Action(capp.LIST_NOT, 0, 0)
		case CAPP_OP_LIST_NEXT:
			if match != IR_CONST_0 || mask != IR_CONST_0 {
				err = errors.Join(ErrOpcodeCapp, ErrOpcodeArg1, ErrOpcodeArg2)
				return
			}
			cp.Action(capp.LIST_NEXT, 0, 0)
		case CAPP_OP_LIST_ONLY:
			cp.Action(capp.LIST_ONLY, val, msk)
		case CAPP_OP_SET_OF:
			cpu.Match = val
			cpu.Mask = msk
			cp.Action(capp.SET_OF, val, msk)
		case CAPP_OP_WRITE_FIRST:
			cp.Action(capp.WRITE_FIRST, val, msk)
		case CAPP_OP_WRITE_LIST:
			cp.Action(capp.WRITE_LIST, val, msk)
		default:
			err = errors.Join(ErrOpcodeCapp, ErrOpcodeOp)
			return
		}
	case OP_IO:
		op, dst, ir_value := code.IoDecode()
		var value uint32
		var set_await func(value uint32)
		if op == IO_OP_AWAIT {
			switch ir_value {
			case IR_CONST_0:
				// drop-on-floor
				set_await = func(value uint32) {}
			case IR_REG_R0, IR_REG_R1, IR_REG_R2, IR_REG_R3, IR_REG_R4, IR_REG_R5:
				set_await = func(value uint32) { cpu.Register[ir_value-IR_REG_R0] = value }
			case IR_IP:
				set_await = func(value uint32) {
					next_ip = value
				}
			case IR_STACK:
				set_await = func(value uint32) { cpu.Stack.Push(value) }
			default:
				err = errors.Join(ErrOpcodeIo, ErrOpcodeArg2)
				return
			}
		} else {
			value, imms, err = cpu.getValue(ir_value, imms)
			if err != nil {
				err = errors.Join(ErrOpcodeIo, ErrOpcodeArg2, err)
				return
			}
		}
		var channel Channel
		var response chan uint32
		channel, response, err = cpu.GetChannel(dst)
		if err != nil {
			err = errors.Join(ErrOpcodeIo, err)
			return
		}
		switch op {
		case IO_OP_FETCH:
			err = cpu.listInput(channel, value)
		case IO_OP_STORE:
			err = cpu.listOutput(channel, value)
		case IO_OP_ALERT:
			channel.Alert(value, response)
		case IO_OP_AWAIT:
			select {
			case recv, ok := <-response:
				if !ok {
					err = ErrChannelInvalid
				} else {
					// Update as requested.
					set_await(recv)
				}
			default:
				// Don't advance to next IP.
				next_ip = cpu.Ip
			}
		default:
			err = errors.Join(ErrOpcodeIo, ErrOpcodeOp)
			return
		}
	default:
		err = ErrOpcodeDecode
		return
	}

	if len(imms) != 0 {
		err = ErrOpcodeImm
		return
	}

	cpu.Ip = next_ip

	// only count CAPP ticks against power!
	if (cpu.Ip & IP_MODE_MASK) != IP_MODE_CAPP {
		cpu.Ticks += 1
		cpu.Power += cpu.Capp.BitsFlipped + bits.OnesCount64(prior^result)
	}

	return
}

// getValue gets the value specified by the CodeIR, based on CPU
// state or value of the immediates that followed the opcode.
func (cp *Cpu) getValue(src CodeIR, imms_in []uint16) (value uint32, imms []uint16, err error) {
	imms = imms_in

	switch src {
	case IR_CONST_0:
		value = 0
	case IR_CONST_FFFFFFFF:
		value = 0xffffffff
	case IR_IMMEDIATE_16:
		if len(imms) < 1 {
			err = ErrOpcodeImm
			return
		}
		value = uint32(imms[0])
		imms = imms[1:]
	case IR_IMMEDIATE_32:
		if len(imms) < 2 {
			err = ErrOpcodeImm
			return
		}
		value = (uint32(imms[0]) << 16) | uint32(imms[1])
		imms = imms[2:]
	case IR_IP:
		// next_ip
		value = cp.Ip + 1
	case IR_STACK:
		var ok bool
		value, ok = cp.Stack.Pop()
		if !ok {
			err = ErrStackEmpty
			return
		}
	case IR_REG_R0, IR_REG_R1, IR_REG_R2, IR_REG_R3, IR_REG_R4, IR_REG_R5:
		value = cp.Register[src-IR_REG_R0]
	case IR_REG_MATCH:
		value = cp.Match
	case IR_REG_MASK:
		value = cp.Mask
	case IR_REG_FIRST:
		value = cp.Capp.First()
	case IR_REG_COUNT:
		value = uint32(cp.Capp.Count())
	default:
		panic("unknown IR")
	}

	return
}

// doAlu performs the requested ALU action, and returns the output value.
func (cp *Cpu) doAlu(op CodeAluOp, input uint32, value uint32) (output uint32) {
	switch op {
	case ALU_OP_SET: // set
		output = value
	case ALU_OP_XOR: // xor
		output = input ^ value
	case ALU_OP_AND: // and
		output = input & value
	case ALU_OP_OR: // or
		output = input | value
	case ALU_OP_SHL: // shl
		value &= 0x1f // clamp to 31 bits of shift
		output = input << value
	case ALU_OP_SHR: // shr
		value &= 0x1f // clamp to 31 bits of shift
		output = input >> value
	case ALU_OP_ADD: // add
		output = input + value
	case ALU_OP_SUB: // sub
		output = input + ((^value) + 1)
	}

	return
}
