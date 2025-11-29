package io

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRing_Alert(t *testing.T) {
	assert := assert.New(t)

	ring := &Ring{}
	ring.Reset()
	ring.WriteIndex = 42
	ring.ReadIndex = 10

	// Test RING_OP_RESET_READ
	ring.Alert(RING_OP_RESET_READ)
	assert.Equal(0, ring.ReadIndex)
	assert.Equal(42, ring.WriteIndex)

	value, ok := ring.Await()
	assert.True(ok)
	assert.Equal(uint32(0), value)

	// Test RING_OP_RESET_WRITE
	ring.Alert(RING_OP_RESET_WRITE)
	assert.Equal(0, ring.WriteIndex)

	value, ok = ring.Await()
	assert.True(ok)
	assert.Equal(uint32(0), value)

	// Test invalid operation
	ring.Alert(99)
	value, ok = ring.Await()
	assert.True(ok)
	assert.Equal(^uint32(0), value)
}

func TestRing_Send_Nil(t *testing.T) {
	assert := assert.New(t)

	var ring *Ring
	err := ring.Send(true)
	assert.Equal(ErrChannelFull, err)
}

func TestRing_Send_CapacityFull(t *testing.T) {
	assert := assert.New(t)

	ring := &Ring{Capacity: 3}
	ring.Reset()

	err := ring.Send(true)
	assert.NoError(err)
	err = ring.Send(false)
	assert.NoError(err)
	err = ring.Send(true)
	assert.NoError(err)

	// Should be full now
	err = ring.Send(false)
	assert.Equal(ErrChannelFull, err)
}
