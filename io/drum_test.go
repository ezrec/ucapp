package io

import (
	"iter"
	"testing"

	"github.com/stretchr/testify/assert"
)

func slum(size int, items ...uint8) (arr []uint8) {
	arr = make([]uint8, len(items), size)
	copy(arr, items)
	return
}

func TestDrum(t *testing.T) {
	assert := assert.New(t)

	drum := &Drum{}

	drum.Rings = map[uint8](*Ring){
		// 0 does not exist,
		1: &Ring{Data: slum(8)},
		3: &Ring{WriteIndex: 4 * 8, Data: slum(8, 1, 2, 3, 4)},
	}

	drum.Rewind()

	// Simple write to default ring.
	SendAsUint8(drum, 0x08)
	SendAsUint8(drum, 0x19)
	SendAsUint8(drum, 0x2a)

	var ring uint32
	awaitResponse := make(chan uint32, 1)
	defer close(awaitResponse)

	// Select new ring (empty)
	drum.Alert(1, awaitResponse)
	ring = <-awaitResponse
	assert.Equal(uint32(0), ring)

	// Select new ring (nil)
	drum.Alert(2, awaitResponse)
	ring = <-awaitResponse
	assert.Equal(uint32(0), ring)

	// Select new ring (has content)
	drum.Alert(3, awaitResponse)
	ring = <-awaitResponse
	assert.Equal(uint32(4), ring)

	seq := ReceiveAsUint8(drum)
	get, stop := iter.Pull(seq)
	for n := range 3 {
		v, ok := get()
		assert.True(ok)
		assert.Equal(uint8(n+1), v)
	}
	stop()

	// append to the end of the drum
	SendAsUint8(drum, 0x07)
	SendAsUint8(drum, 0x0b)

	expecting := map[uint8](*Ring){
		0: &Ring{Capacity: RING_DEFAULT_CAPACITY, WriteIndex: 24, Data: slum(8, 0x08, 0x19, 0x2a)},
		1: &Ring{Capacity: 64, Data: slum(8)},
		2: &Ring{Capacity: RING_DEFAULT_CAPACITY, Data: []uint8{}},
		3: &Ring{Capacity: 64, WriteIndex: 48, ReadIndex: 24, Data: slum(8, 1, 2, 3, 4, 7, 11)},
	}

	assert.Equal(expecting, drum.Rings)
}
