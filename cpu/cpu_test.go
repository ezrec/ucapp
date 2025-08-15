package cpu

import (
	"testing"

	"github.com/ezrec/ucapp/channel"
	"github.com/stretchr/testify/assert"
)

func TestImmediate(t *testing.T) {
	assert := assert.New(t)

	table := [](struct {
		name      string
		program   []Code
		immediate uint64
	}){
		{"imm_a_lo", []Code{0x1_1234}, 0x1234},
		{"imm_a_hi", []Code{0x2_1234, 0x3_5678}, 0x12345678},
		{"imm_a_wr", []Code{0x2_1234, 0x3_5678, 0x1_00aa}, 0x12345678000000aa},
		{"imm_a_wr", []Code{0x3_1234, 0x3_5678, 0x2_00aa, 0x3_c0ed}, 0x567800aac0ed},
	}

	for _, entry := range table {
		cpu := NewCpu(4)

		rom := &channel.Rom{}

		cpu.SetChannel(CHANNEL_ID_MONITOR, rom)

		rom.Data = nil
		for n, code := range entry.program {
			data := ARENA_CODE | (uint32(n) << 20) | uint32(code)
			rom.Data = append(rom.Data, data)
		}
		cpu.Reset()

		var err error
		for {
			err = cpu.Tick()
			if err == ErrIpEmpty {
				break
			}
			assert.NoError(err, entry.name)
		}

		assert.Equal(entry.immediate, cpu.Immediate, entry.name)
	}
}
