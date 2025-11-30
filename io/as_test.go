package io

import (
	"iter"
	"testing"

	"github.com/stretchr/testify/assert"
)

type mockChannel struct {
	sendCalls    []bool
	sendError    error
	receiveData  []bool
	receiveCalls int
}

func (mc *mockChannel) Rewind() {}

func (mc *mockChannel) Send(value bool) error {
	mc.sendCalls = append(mc.sendCalls, value)
	return mc.sendError
}

func (mc *mockChannel) Receive() iter.Seq[bool] {
	return func(yield func(bool) bool) {
		mc.receiveCalls++
		for _, bit := range mc.receiveData {
			if !yield(bit) {
				return
			}
		}
	}
}

func (mc *mockChannel) Alert(value uint32, response chan uint32) { response <- 0 }

func TestSendAsUint8(t *testing.T) {
	assert := assert.New(t)

	tests := []struct {
		name     string
		value    uint8
		expected []bool
	}{
		{
			name:     "zero",
			value:    0x00,
			expected: []bool{false, false, false, false, false, false, false, false},
		},
		{
			name:     "all bits set",
			value:    0xFF,
			expected: []bool{true, true, true, true, true, true, true, true},
		},
		{
			name:     "0x55 (0101 0101)",
			value:    0x55,
			expected: []bool{true, false, true, false, true, false, true, false},
		},
		{
			name:     "0xAA (1010 1010)",
			value:    0xAA,
			expected: []bool{false, true, false, true, false, true, false, true},
		},
		{
			name:     "0x01",
			value:    0x01,
			expected: []bool{true, false, false, false, false, false, false, false},
		},
		{
			name:     "0x80",
			value:    0x80,
			expected: []bool{false, false, false, false, false, false, false, true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := &mockChannel{}
			err := SendAsUint8(mc, tt.value)
			assert.NoError(err)
			assert.Equal(tt.expected, mc.sendCalls)
		})
	}
}

func TestSendAsUint8_Error(t *testing.T) {
	assert := assert.New(t)

	mc := &mockChannel{sendError: ErrChannelFull}
	err := SendAsUint8(mc, 0x42)
	assert.Error(err)
	assert.Equal(ErrChannelFull, err)
	assert.Len(mc.sendCalls, 1)
}

func TestReceiveAsUint8(t *testing.T) {
	assert := assert.New(t)

	tests := []struct {
		name     string
		bits     []bool
		expected []uint8
	}{
		{
			name:     "single byte zero",
			bits:     []bool{false, false, false, false, false, false, false, false},
			expected: []uint8{0x00},
		},
		{
			name:     "single byte 0xFF",
			bits:     []bool{true, true, true, true, true, true, true, true},
			expected: []uint8{0xFF},
		},
		{
			name:     "two bytes",
			bits:     []bool{true, false, true, false, true, false, true, false, false, true, false, true, false, true, false, true},
			expected: []uint8{0x55, 0xAA},
		},
		{
			name:     "partial byte",
			bits:     []bool{true, true, false},
			expected: []uint8{0x03},
		},
		{
			name:     "empty",
			bits:     []bool{},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := &mockChannel{receiveData: tt.bits}
			seq := ReceiveAsUint8(mc)

			var result []uint8
			for value := range seq {
				result = append(result, value)
			}

			assert.Equal(tt.expected, result)
		})
	}
}

func TestReceiveAsUint8_EarlyStop(t *testing.T) {
	assert := assert.New(t)

	mc := &mockChannel{
		receiveData: []bool{
			true, false, true, false, true, false, true, false, // 0x55
			false, true, false, true, false, true, false, true, // 0xAA
		},
	}

	seq := ReceiveAsUint8(mc)
	for value := range seq {
		assert.Equal(uint8(0x55), value)
		break // Stop after first value
	}
}

func TestSendAsUint16(t *testing.T) {
	assert := assert.New(t)

	tests := []struct {
		name     string
		value    uint16
		expected []bool
	}{
		{
			name:     "zero",
			value:    0x0000,
			expected: make([]bool, 16),
		},
		{
			name:     "all bits set",
			value:    0xFFFF,
			expected: []bool{true, true, true, true, true, true, true, true, true, true, true, true, true, true, true, true},
		},
		{
			name:     "0x0001",
			value:    0x0001,
			expected: []bool{true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
		},
		{
			name:     "0x8000",
			value:    0x8000,
			expected: []bool{false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, true},
		},
		{
			name:     "0xAA55",
			value:    0xAA55,
			expected: []bool{true, false, true, false, true, false, true, false, false, true, false, true, false, true, false, true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := &mockChannel{}
			err := SendAsUint16(mc, tt.value)
			assert.NoError(err)
			assert.Equal(tt.expected, mc.sendCalls)
		})
	}
}

func TestSendAsUint16_Error(t *testing.T) {
	assert := assert.New(t)

	mc := &mockChannel{sendError: ErrChannelFull}
	err := SendAsUint16(mc, 0x1234)
	assert.Error(err)
	assert.Equal(ErrChannelFull, err)
	assert.Len(mc.sendCalls, 1)
}

func TestReceiveAsUint16(t *testing.T) {
	assert := assert.New(t)

	tests := []struct {
		name     string
		bits     []bool
		expected []uint16
	}{
		{
			name:     "single word zero",
			bits:     make([]bool, 16),
			expected: []uint16{0x0000},
		},
		{
			name:     "single word 0xFFFF",
			bits:     []bool{true, true, true, true, true, true, true, true, true, true, true, true, true, true, true, true},
			expected: []uint16{0xFFFF},
		},
		{
			name:     "0xAA55",
			bits:     []bool{true, false, true, false, true, false, true, false, false, true, false, true, false, true, false, true},
			expected: []uint16{0xAA55},
		},
		{
			name:     "two words",
			bits:     append(make([]bool, 16), []bool{true, true, true, true, true, true, true, true, true, true, true, true, true, true, true, true}...),
			expected: []uint16{0x0000, 0xFFFF},
		},
		{
			name:     "partial word",
			bits:     []bool{true, true, false, true, false},
			expected: []uint16{0x000B},
		},
		{
			name:     "empty",
			bits:     []bool{},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := &mockChannel{receiveData: tt.bits}
			seq := ReceiveAsUint16(mc)

			var result []uint16
			for value := range seq {
				result = append(result, value)
			}

			assert.Equal(tt.expected, result)
		})
	}
}

func TestReceiveAsUint16_EarlyStop(t *testing.T) {
	assert := assert.New(t)

	mc := &mockChannel{
		receiveData: append(make([]bool, 16), make([]bool, 16)...),
	}

	seq := ReceiveAsUint16(mc)
	count := 0
	for range seq {
		count++
		if count == 1 {
			break
		}
	}

	assert.Equal(1, count)
}
