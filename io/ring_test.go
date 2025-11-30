package io

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRing_Alert(t *testing.T) {
	assert := assert.New(t)

	ring := &Ring{}
	ring.Rewind()
	ring.WriteIndex = 42
	ring.ReadIndex = 10

	var value uint32
	awaitResponse := make(chan uint32, 1)
	defer close(awaitResponse)

	// Test RING_OP_REWIND_READ
	ring.Alert(RING_OP_REWIND_READ, awaitResponse)
	value = <-awaitResponse
	assert.Equal(0, ring.ReadIndex)
	assert.Equal(42, ring.WriteIndex)
	assert.Equal(uint32(0), value)

	// Test RING_OP_REWIND_WRITE
	ring.Alert(RING_OP_REWIND_WRITE, awaitResponse)
	value = <-awaitResponse
	assert.Equal(0, ring.WriteIndex)

	assert.Equal(uint32(0), value)

	// Test invalid operation
	ring.Alert(99, awaitResponse)
	value = <-awaitResponse
	assert.Equal(^uint32(0), value)
}

func TestRing_Send_CapacityFull(t *testing.T) {
	assert := assert.New(t)

	ring := &Ring{Capacity: 3}
	ring.Rewind()

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
