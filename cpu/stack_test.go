package cpu

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStack_Push(t *testing.T) {
	assert := assert.New(t)

	s := &Stack{}
	assert.True(s.Empty())
	assert.False(s.Full())

	s.Push(0x12345678)
	assert.False(s.Empty())
	assert.Equal(1, len(s.Data))
	assert.Equal(uint32(0x12345678), s.Data[0])
}

func TestStack_Pop(t *testing.T) {
	assert := assert.New(t)

	s := &Stack{}
	s.Push(0x12345678)
	s.Push(0xABCDEF01)

	val, ok := s.Pop()
	assert.True(ok)
	assert.Equal(uint32(0xABCDEF01), val)
	assert.Equal(1, len(s.Data))

	val, ok = s.Pop()
	assert.True(ok)
	assert.Equal(uint32(0x12345678), val)
	assert.Equal(0, len(s.Data))
}

func TestStack_Pop_Empty(t *testing.T) {
	assert := assert.New(t)

	s := &Stack{}
	val, ok := s.Pop()
	assert.False(ok)
	assert.Equal(uint32(0), val)
}

func TestStack_Peek(t *testing.T) {
	assert := assert.New(t)

	s := &Stack{}
	s.Push(0x12345678)
	s.Push(0xABCDEF01)

	val, ok := s.Peek()
	assert.True(ok)
	assert.Equal(uint32(0xABCDEF01), val)
	assert.Equal(2, len(s.Data))
}

func TestStack_Peek_Empty(t *testing.T) {
	assert := assert.New(t)

	s := &Stack{}
	val, ok := s.Peek()
	assert.False(ok)
	assert.Equal(uint32(0), val)
}

func TestStack_Full(t *testing.T) {
	assert := assert.New(t)

	s := &Stack{}
	assert.False(s.Full())

	for i := 0; i < STACK_LIMIT; i++ {
		s.Push(uint32(i))
	}

	assert.True(s.Full())
	assert.False(s.Empty())
}

func TestStack_Reset(t *testing.T) {
	assert := assert.New(t)

	s := &Stack{}
	s.Push(0x12345678)
	s.Push(0xABCDEF01)
	assert.Equal(2, len(s.Data))

	s.Reset()
	assert.True(s.Empty())
	assert.Equal(0, len(s.Data))
}

func TestStack_Reset_Empty(t *testing.T) {
	assert := assert.New(t)

	s := &Stack{}
	s.Reset()
	assert.True(s.Empty())
}

func TestStack_Empty(t *testing.T) {
	assert := assert.New(t)

	s := &Stack{}
	assert.True(s.Empty())

	s.Push(1)
	assert.False(s.Empty())

	s.Pop()
	assert.True(s.Empty())
}

func TestStack_Capacity(t *testing.T) {
	assert := assert.New(t)

	s := &Stack{}

	for i := 0; i < STACK_LIMIT; i++ {
		assert.False(s.Full())
		s.Push(uint32(i))
	}

	assert.True(s.Full())
	assert.Equal(STACK_LIMIT, len(s.Data))
}
