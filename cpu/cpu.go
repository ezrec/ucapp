package cpu

import (
	"errors"
	"fmt"
	"log"
	"math/bits"

	"github.com/ezrec/ucapp/capp"
	"github.com/ezrec/ucapp/channel"
)

type Channel channel.Channel

const (
	IP_MODE_CAPP  = uint32(0b00 << 30)
	IP_MODE_STACK = uint32(0b01 << 30)
	IP_MODE_REG   = uint32(0b10 << 30)
	IP_MODE_MASK  = uint32(0b11 << 30)
)

type Cpu struct {
	Verbose bool

	Capp *capp.Capp

	Ip        uint32
	Immediate uint64
	Register  [6]uint32
	Stack     Stack
	Match     uint32
	Mask      uint32
	Cond      bool

	Power int
	Ticks int

	channel [8]Channel
}

func NewCpu(count uint) (cpu *Cpu) {
	cpu = &Cpu{
		Capp: capp.NewCapp(count),
	}

	return
}

func (cpu *Cpu) String() (text string) {
	regs := []string{
		"ip",
		"cond",
		"r0", "r1", "r2", "r3", "r4",
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

func (cpu *Cpu) Reset() (err error) {
	cpu.Immediate = 0
	clear(cpu.Register[:])
	cpu.Stack.Reset()
	cpu.Capp.Reset()
	cpu.Ticks = 0
	cpu.Power = 0

	for _, channel := range cpu.channel {
		if channel == nil {
			continue
		}
		channel.Reset()
	}

	// Set IP to run from registers
	cpu.Ip = IP_MODE_REG

	// r0: .list.of.-.immz.immz        ; Select all of the CAPP
	// r1: .list.all.-.-.-             ; Tag all items
	// r2: .list.write.-.immnz.immnz   ; Replace all values with 0xFFFFFFFF
	// r3: .io.fetch.rom.immz.immnz    ; Load boot ROM into CAPP
	// r4: .list.not.-.-.-             ; Now, only the program is tagged
	// r5: .alu.set.ip.immz.immnz      ; Set IP to 0x00000000 (exec from CAPP)

	bootstrap := [6]Code{
		MakeCodeCapp(COND_ALWAYS, CAPP_OP_SET_OF, IR_CONST_0, IR_CONST_0),
		MakeCodeCapp(COND_ALWAYS, CAPP_OP_LIST_ALL, IR_CONST_0, IR_CONST_0),
		MakeCodeCapp(COND_ALWAYS, CAPP_OP_WRITE_LIST, IR_CONST_FFFFFFFF, IR_CONST_FFFFFFFF),
		MakeCodeIo(COND_ALWAYS, IO_OP_FETCH, CHANNEL_ID_MONITOR, IR_CONST_0, IR_CONST_FFFFFFFF),
		MakeCodeCapp(COND_ALWAYS, CAPP_OP_LIST_NOT, IR_CONST_0, IR_CONST_0),
		MakeCodeAlu(COND_ALWAYS, ALU_OP_SET, IR_IP, IR_CONST_0, IR_CONST_FFFFFFFF),
	}

	for n, code := range bootstrap {
		cpu.Register[n] = uint32(code)
	}

	return
}

func (cpu *Cpu) SetChannel(index CodeChannel, channel Channel) {
	cpu.channel[int(index)] = channel
}

func (cpu *Cpu) GetChannel(ch CodeChannel) (channel Channel, err error) {
	index := int(ch)
	if index >= len(cpu.channel) || cpu.channel[index] == nil {
		err = ErrChannelInvalid
		return
	}

	channel = cpu.channel[index]
	return
}

func (cpu *Cpu) listInput(in Channel, value, mask uint32) (err error) {
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
			cp.Action(capp.WRITE_FIRST, value|inval, mask)
			cp.Action(capp.LIST_NEXT, 0, 0)
			tmp = mask
			inval = 0
		}
	}
	return
}

func (cpu *Cpu) listOutput(out Channel, value, mask uint32) (err error) {
	cp := cpu.Capp

	if mask == 0 {
		return
	}

	for cp.Count() > 0 {
		inval := cp.First() | value
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
		code = Code(cpu.Register[reg])
	case IP_MODE_STACK:
		opcode, ok := cpu.Stack.Pop()
		if !ok {
			log.Printf("Ip 0x%x > stack empty", cpu.Ip)
			err = ErrIpEmpty
			return
		}
		code = Code(opcode)
	case IP_MODE_CAPP:
		cpu.Capp.Verbose = false
		cpu.Capp.Action(capp.SET_SWAP, 0, 0)
		cpu.Capp.Action(capp.SET_OF, ARENA_CODE|(uint32(cpu.Ip&0x3ff)<<20), ARENA_MASK|(0x3ff<<20))
		count := cpu.Capp.Count()
		first := cpu.Capp.First()
		cpu.Capp.Action(capp.SET_SWAP, 0, 0)
		if count != 1 {
			log.Printf("Ip 0x%x > capp empty (%d)", cpu.Ip, count)
			err = ErrIpEmpty
			return
		}
		code = Code(first & ((1 << 20) - 1))
		cpu.Capp.Verbose = cpu.Verbose
	default:
		log.Printf("Ip 0x%x > unknown source", cpu.Ip)
		err = ErrIpEmpty
		return
	}

	return
}

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
	var prog Channel
	prog, err = cpu.GetChannel(CHANNEL_ID_MONITOR)
	if err == ErrChannelInvalid {
		// no debug channel
		err = nil
	} else {
		if err == nil {
			_, ok := prog.GetAlert()
			if ok {
				err = ErrIpTrap
			}
		}
	}

	return
}

func (cpu *Cpu) Execute(first Code) (err error) {
	defer func() {
		if err != nil {
			err = errors.Join(ErrOpcode(first), err)
		}
	}()
	if cpu.Verbose {
		log.Printf("%03x: %v", cpu.Ip, first.String())
	}

	cp := cpu.Capp

	cpu.Capp.BitsFlipped = 0

	next_ip := cpu.Ip + 1

	// Bits flipped by CPU action.
	var prior uint64
	var result uint64

	no_op := MakeCodeAlu(COND_ALWAYS, ALU_OP_OR, IR_REG_R0, IR_CONST_0, IR_CONST_0)

	cond := first.Cond()
	switch cond {
	case COND_ALWAYS:
		// pass
	case COND_NEVER:
		return ErrOpcode(first)
	case COND_TRUE:
		if !cpu.Cond {
			// Convert to no-op
			first = no_op
		}
	case COND_FALSE:
		if cpu.Cond {
			// Convert to no-op
			first = no_op
		}
	}

	switch first.Class() {
	case OP_IMM:
		op := first.ImmOp()
		prior = cpu.Immediate
		switch op {
		case IMM_OP_LO32:
			result = (cpu.Immediate << 32) | uint64(first&0xffff)
		case IMM_OP_HI32:
			result = (cpu.Immediate << 32) | uint64((first&0xffff)<<16)
		case IMM_OP_OR16:
			result = (cpu.Immediate & ^uint64(0xffff)) | uint64(first&0xffff)
		default:
			err = errors.Join(ErrOpcodeImm, ErrOpcodeOp)
			return
		}
		cpu.Immediate = result
	case OP_ALU:
		op := first.AluOp()
		dst := first.Target()
		match := first.Value()
		mask := first.Mask()
		var val uint32
		var msk uint32
		val, err = cpu.getValue(match)
		if err != nil {
			err = errors.Join(ErrOpcodeAlu, ErrOpcodeValue, err)
			return
		}
		msk, err = cpu.getValue(mask)
		if err != nil {
			err = errors.Join(ErrOpcodeAlu, ErrOpcodeMask, err)
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
				err = ErrStackFull
				return
			}
			if op == ALU_OP_SET {
				input = 0
			} else {
				var ok bool
				input, ok = cpu.Stack.Pop()
				if !ok {
					err = ErrStackEmpty
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
			err = errors.Join(ErrOpcodeAlu, ErrOpcodeTarget)
			return
		}
		prior = uint64(input)
		output := cpu.doAlu(op, input, val, msk)
		set_target(output)
		result = uint64(output)
	case OP_COND:
		op := first.CondOp()
		dst := first.Target()
		if dst != 0 {
			err = ErrOpcode(first)
			err = errors.Join(ErrOpcodeCond, ErrOpcodeTarget, err)
			return
		}
		a_ir := first.Value()
		b_ir := first.Mask()
		var a_u uint32
		var b_u uint32
		a_u, err = cpu.getValue(a_ir)
		if err != nil {
			err = errors.Join(ErrOpcodeCond, ErrOpcodeValue, err)
			return
		}
		b_u, err = cpu.getValue(b_ir)
		if err != nil {
			err = errors.Join(ErrOpcodeCond, ErrOpcodeMask, err)
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
		case COND_OP_GT:
			cpu.Cond = a > b
		case COND_OP_LE:
			cpu.Cond = a <= b
		case COND_OP_GE:
			cpu.Cond = a >= b
		default:
			err = errors.Join(ErrOpcodeCond, ErrOpcodeOp)
			return
		}
	case OP_CAPP:
		op := first.CappOp()
		match := first.Match()
		mask := first.Mask()
		dst := first.Target()
		if dst != CodeIR(0) {
			err = errors.Join(ErrOpcodeCapp, ErrOpcodeTarget)
			return
		}
		var val uint32
		var msk uint32
		val, err = cpu.getValue(match)
		if err != nil {
			err = errors.Join(ErrOpcodeCapp, ErrOpcodeValue, err)
			return
		}
		msk, err = cpu.getValue(mask)
		if err != nil {
			err = errors.Join(ErrOpcodeCapp, ErrOpcodeMask, err)
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
				err = errors.Join(ErrOpcodeCapp, ErrOpcodeMask, ErrOpcodeValue)
				return
			}
			cp.Action(capp.LIST_ALL, 0, 0)
		case CAPP_OP_LIST_NOT:
			if match != IR_CONST_0 || mask != IR_CONST_0 {
				err = errors.Join(ErrOpcodeCapp, ErrOpcodeMask, ErrOpcodeValue)
				return
			}
			cp.Action(capp.LIST_NOT, 0, 0)
		case CAPP_OP_LIST_NEXT:
			if match != IR_CONST_0 || mask != IR_CONST_0 {
				err = errors.Join(ErrOpcodeCapp, ErrOpcodeMask, ErrOpcodeValue)
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
		op := first.IoOp()
		dst := first.Channel()
		ir_value := first.Value()
		ir_mask := first.Mask()
		var value uint32
		var mask uint32
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
				err = errors.Join(ErrOpcodeIo, ErrOpcodeTarget)
				return
			}
		} else {
			value, err = cpu.getValue(ir_value)
			if err != nil {
				err = errors.Join(ErrOpcodeIo, ErrOpcodeValue, err)
				return
			}
		}
		mask, err = cpu.getValue(ir_mask)
		if err != nil {
			err = errors.Join(ErrOpcodeIo, ErrOpcodeMask, err)
			return
		}
		var channel Channel
		channel, err = cpu.GetChannel(dst)
		if err != nil {
			err = errors.Join(ErrOpcodeIo, err)
			return
		}
		switch op {
		case IO_OP_FETCH:
			err = cpu.listInput(channel, value, mask)
		case IO_OP_STORE:
			err = cpu.listOutput(channel, value, mask)
		case IO_OP_ALERT:
			channel.Alert(value & mask)
		case IO_OP_AWAIT:
			recv, ok := channel.Await()
			if !ok {
				// Don't advance to next IP.
				next_ip = cpu.Ip
				// Push-back IMM if mask was an IMM
				if ir_mask == IR_IMMEDIATE_32 {
					cpu.Immediate <<= 32
					cpu.Immediate |= uint64(mask)
				}
			} else {
				// Update as requested.
				set_await(recv & mask)
			}
		default:
			err = errors.Join(ErrOpcodeIo, ErrOpcodeOp)
			return
		}
	default:
		err = ErrOpcodeDecode
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

func (cp *Cpu) getValue(src CodeIR) (value uint32, err error) {
	switch src {
	case IR_CONST_0:
		value = 0
	case IR_CONST_1:
		value = 1
	case IR_CONST_FFFFFFFF:
		value = 0xffffffff
	case IR_IMMEDIATE_32:
		value = uint32(cp.Immediate & 0xffffffff)
		cp.Immediate >>= 32
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

func (cp *Cpu) doAlu(op CodeAluOp, input uint32, value uint32, mask uint32) (output uint32) {
	value &= mask

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
