package io

import (
	"bytes"
	"io"
	"iter"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRom_Receive(t *testing.T) {
	assert := assert.New(t)

	rom := &Rom{
		Data: []uint32{0x00000001, 0x80000000, 0xFFFFFFFF},
	}

	seq := rom.Receive()
	var bits []bool
	for bit := range seq {
		bits = append(bits, bit)
	}

	assert.Len(bits, 96) // 3 words * 32 bits

	// First word: 0x00000001
	assert.True(bits[0])
	for i := 1; i < 32; i++ {
		assert.False(bits[i])
	}

	// Second word: 0x80000000
	for i := 32; i < 63; i++ {
		assert.False(bits[i])
	}
	assert.True(bits[63])

	// Third word: 0xFFFFFFFF
	for i := 64; i < 96; i++ {
		assert.True(bits[i])
	}
}

func TestRom_Receive_Empty(t *testing.T) {
	assert := assert.New(t)

	rom := &Rom{Data: []uint32{}}

	seq := rom.Receive()
	count := 0
	for range seq {
		count++
	}

	assert.Equal(0, count)
}

func TestRom_Receive_EarlyStop(t *testing.T) {
	assert := assert.New(t)

	rom := &Rom{Data: []uint32{0xFFFFFFFF, 0xAAAAAAAA}}

	seq := rom.Receive()
	count := 0
	for range seq {
		count++
		if count == 10 {
			break
		}
	}

	assert.Equal(10, count)
}

func TestRom_Send(t *testing.T) {
	assert := assert.New(t)

	rom := &Rom{}

	err := rom.Send(true)
	assert.Equal(ErrChannelFull, err)

	err = rom.Send(false)
	assert.Equal(ErrChannelFull, err)
}

func TestTape_Rewind(t *testing.T) {
	assert := assert.New(t)

	input := bytes.NewBuffer([]byte{0x55, 0xAA, 0xFF})
	tape := &Tape{Input: input}
	tape.Rewind()

	count := 0
	for range ReceiveAsUint8(tape) {
		count++
	}
	assert.Equal(3, count)

	count = 0
	for range ReceiveAsUint8(tape) {
		count++
	}
	assert.Equal(0, count)

	// Rewind is not possible on a tape.
	tape.Rewind()

	count = 0
	for range ReceiveAsUint8(tape) {
		count++
	}
	assert.Equal(0, count)
}

func TestTape_Receive(t *testing.T) {
	assert := assert.New(t)

	input := bytes.NewBuffer([]byte{0x55, 0xAA, 0xFF})
	tape := &Tape{Input: input}
	tape.Rewind()

	seq := tape.Receive()
	var bits []bool
	count := 0
	for bit := range seq {
		bits = append(bits, bit)
		count++
		if count >= 24 {
			break
		}
	}

	assert.Len(bits, 24)

	// 0x55 = 0101 0101
	expected1 := []bool{true, false, true, false, true, false, true, false}
	assert.Equal(expected1, bits[0:8])

	// 0xAA = 1010 1010
	expected2 := []bool{false, true, false, true, false, true, false, true}
	assert.Equal(expected2, bits[8:16])

	// 0xFF = 1111 1111
	expected3 := []bool{true, true, true, true, true, true, true, true}
	assert.Equal(expected3, bits[16:24])
}

func TestTape_Receive_EOF(t *testing.T) {
	assert := assert.New(t)

	input := bytes.NewBuffer([]byte{0x01})
	tape := &Tape{Input: input}
	tape.Rewind()

	seq := tape.Receive()
	count := 0
	for range seq {
		count++
	}

	assert.Equal(8, count)
}

func TestTape_Receive_ReadError(t *testing.T) {
	assert := assert.New(t)

	// Use a reader that returns an error
	tape := &Tape{Input: &errorReader{}}
	tape.Rewind()

	seq := tape.Receive()
	count := 0
	for range seq {
		count++
	}

	assert.Equal(0, count)
}

type errorReader struct{}

func (er *errorReader) Read(p []byte) (n int, err error) {
	return 0, io.ErrUnexpectedEOF
}

func TestTape_Send(t *testing.T) {
	assert := assert.New(t)

	output := &bytes.Buffer{}
	tape := &Tape{Output: output}
	tape.Rewind()

	// Send 0x55 (0101 0101)
	bits := []bool{true, false, true, false, true, false, true, false}
	for _, bit := range bits {
		err := tape.Send(bit)
		assert.NoError(err)
	}

	assert.Equal([]byte{0x55}, output.Bytes())

	// Send 0xAA (1010 1010)
	bits2 := []bool{false, true, false, true, false, true, false, true}
	for _, bit := range bits2 {
		err := tape.Send(bit)
		assert.NoError(err)
	}

	assert.Equal([]byte{0x55, 0xAA}, output.Bytes())
}

func TestTape_Send_PartialByte(t *testing.T) {
	assert := assert.New(t)

	output := &bytes.Buffer{}
	tape := &Tape{Output: output}
	tape.Rewind()

	// Send only 3 bits
	err := tape.Send(true)
	assert.NoError(err)
	err = tape.Send(true)
	assert.NoError(err)
	err = tape.Send(false)
	assert.NoError(err)

	// Nothing written yet since we haven't completed a byte
	assert.Equal(0, output.Len())

	// Send 5 more bits
	err = tape.Send(true)
	assert.NoError(err)
	err = tape.Send(true)
	assert.NoError(err)
	err = tape.Send(false)
	assert.NoError(err)
	err = tape.Send(true)
	assert.NoError(err)
	err = tape.Send(false)
	assert.NoError(err)

	assert.Equal(1, output.Len())
	assert.Equal(uint8(0x5b), output.Bytes()[0])

}

func TestTemporary_Rewind(t *testing.T) {
	assert := assert.New(t)

	temp := &Temporary{
		Capacity:   10,
		ReadIndex:  3,
		WriteIndex: 7,
		Size:       4,
		Data:       []bool{true, false, true},
	}

	temp.Rewind()

	assert.Equal(0, temp.ReadIndex)
	assert.Equal(0, temp.WriteIndex)
	assert.Equal(0, temp.Size)
	assert.Len(temp.Data, 10)
}

func TestTemporary_Send_Receive(t *testing.T) {
	assert := assert.New(t)

	temp := &Temporary{Capacity: 8}
	temp.Rewind()

	// Send some bits
	err := temp.Send(true)
	assert.NoError(err)
	err = temp.Send(false)
	assert.NoError(err)
	err = temp.Send(true)
	assert.NoError(err)
	err = temp.Send(true)
	assert.NoError(err)

	assert.Equal(4, temp.Size)

	// Receive them back
	seq := temp.Receive()
	var bits []bool
	for bit := range seq {
		bits = append(bits, bit)
	}

	assert.Equal([]bool{true, false, true, true}, bits)
	assert.Equal(0, temp.Size)
}

func TestTemporary_Send_CapacityFull(t *testing.T) {
	assert := assert.New(t)

	temp := &Temporary{Capacity: 3}
	temp.Rewind()

	err := temp.Send(true)
	assert.NoError(err)
	err = temp.Send(false)
	assert.NoError(err)
	err = temp.Send(true)
	assert.NoError(err)

	// Should be full
	err = temp.Send(false)
	assert.Equal(ErrChannelFull, err)
}

func TestTemporary_WrapAround(t *testing.T) {
	assert := assert.New(t)

	temp := &Temporary{Capacity: 4}
	temp.Rewind()

	// Fill up
	temp.Send(true)
	temp.Send(false)
	temp.Send(true)
	temp.Send(false)

	// Read some
	seq := temp.Receive()
	pull, stop := iter.Pull(seq)
	bit, ok := pull()
	assert.True(ok)
	assert.True(bit)
	bit, ok = pull()
	assert.True(ok)
	assert.False(bit)
	stop()

	// Now we have space, write more
	err := temp.Send(true)
	assert.NoError(err)
	err = temp.Send(true)
	assert.NoError(err)

	// Should have wrapped around
	assert.Equal(2, temp.WriteIndex)
	assert.Equal(2, temp.ReadIndex)
	assert.Equal(4, temp.Size)

	// Read to trigger wrap-around of ReadIndex
	seq = temp.Receive()
	pull, stop = iter.Pull(seq)

	// Read the two remaining from first batch
	bit, ok = pull()
	assert.True(ok)
	assert.True(bit)
	bit, ok = pull()
	assert.True(ok)
	assert.False(bit)

	// Now ReadIndex should wrap around to 0 and read the new data
	bit, ok = pull()
	assert.True(ok)
	assert.True(bit)
	bit, ok = pull()
	assert.True(ok)
	assert.True(bit)

	stop()
}

func TestTemporary_Receive_EarlyStop(t *testing.T) {
	assert := assert.New(t)

	temp := &Temporary{Capacity: 8}
	temp.Rewind()

	temp.Send(true)
	temp.Send(false)
	temp.Send(true)
	temp.Send(false)

	seq := temp.Receive()
	count := 0
	for range seq {
		count++
		if count == 2 {
			break
		}
	}

	assert.Equal(2, count)
	assert.Equal(2, temp.Size) // Should still have 2 bits left
}
