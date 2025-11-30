package io

import (
	"iter"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDepot_Rewind(t *testing.T) {
	assert := assert.New(t)

	drum1 := &Drum{Rings: map[uint8](*Ring){
		0: &Ring{},
	}}
	drum1.Rewind()
	drum1.Rings[0].WriteIndex = 10

	drum2 := &Drum{Rings: map[uint8](*Ring){
		0: &Ring{},
	}}
	drum2.Rewind()
	drum2.Rings[0].WriteIndex = 20

	depot := &Depot{
		Drums: map[uint32](*Drum){
			1: drum1,
			2: drum2,
		},
	}

	depot.Rewind()

	// Should reset all drums (reset sets ReadIndex to 0)
	assert.Equal(0, depot.Drums[1].Rings[0].ReadIndex)
	assert.Equal(0, depot.Drums[2].Rings[0].ReadIndex)

	// Should have no drum selected.
	assert.Nil(depot.Drum)

	// Drums 1 and 2 should still exist
	assert.NotNil(depot.Drums[1])
	assert.NotNil(depot.Drums[2])
}

func TestDepot_selectDrum(t *testing.T) {
	assert := assert.New(t)

	depot := &Depot{}

	// Select drum that doesn't exist
	var value uint32
	awaitResponse := make(chan uint32, 1)
	defer close(awaitResponse)

	depot.Alert(5, awaitResponse)
	value = <-awaitResponse
	assert.Nil(depot.Drum)
	assert.Equal(^uint32(0), value)

	// Create a drum.
	depot.Drums = make(map[uint32](*Drum))
	depot.Drums[5] = &Drum{}
	existingDrum := depot.Drums[5]
	depot.Alert(5, awaitResponse)
	value = <-awaitResponse
	assert.NotNil(depot.Drum)
	assert.Equal(uint32(0), value)
	assert.Equal(existingDrum, depot.Drums[5])
}

func TestDepot_Alert_Select(t *testing.T) {
	assert := assert.New(t)

	depot := &Depot{}
	depot.Rewind()

	var response uint32
	awaitResponse := make(chan uint32, 1)
	defer close(awaitResponse)

	// Create and select drum 1
	depot.Drums = make(map[uint32](*Drum))
	depot.Drums[1] = &Drum{}
	depot.Alert(DEPOT_OP_SELECT|1, awaitResponse)
	response = <-awaitResponse
	assert.Equal(uint32(0), response)
	assert.NotNil(depot.Drum)

	SendAsUint8(depot, 0x42)

	// Create and select drum 2
	depot.Drums[2] = &Drum{}
	depot.Alert(DEPOT_OP_SELECT|2, awaitResponse)
	response = <-awaitResponse
	assert.Equal(uint32(0), response)
	SendAsUint8(depot, 0x99)

	// Switch back to drum 1
	depot.Alert(DEPOT_OP_SELECT|1, awaitResponse)
	response = <-awaitResponse
	assert.Equal(uint32(0), response)

	// Should read from drum 1
	seq := ReceiveAsUint8(depot)
	pull, stop := iter.Pull(seq)
	value, ok := pull()
	assert.True(ok)
	assert.Equal(uint8(0x42), value)
	stop()

	// Switch to drum 2
	depot.Alert(DEPOT_OP_SELECT|2, awaitResponse)
	response = <-awaitResponse
	assert.Equal(uint32(0), response)

	// Should read from drum 2
	seq = ReceiveAsUint8(depot)
	pull, stop = iter.Pull(seq)
	value, ok = pull()
	assert.True(ok)
	assert.Equal(uint8(0x99), value)
	stop()
}

func TestDepot_Alert_SelectNonExistent(t *testing.T) {
	assert := assert.New(t)

	depot := &Depot{}
	depot.Rewind()

	var value uint32
	awaitResponse := make(chan uint32, 1)
	defer close(awaitResponse)

	// After Rewind, depot.Drum should point to drum 0
	depot.Drum = depot.Drums[0]

	// Delete drum 99 to ensure it doesn't exist
	delete(depot.Drums, 99)

	depot.Alert(DEPOT_OP_SELECT|99, awaitResponse)
	value = <-awaitResponse

	// Should send error signal
	assert.Equal(^uint32(0), value)
}

func TestDepot_Alert_DrumOperation(t *testing.T) {
	assert := assert.New(t)

	depot := &Depot{}
	depot.Rewind()

	var value uint32
	awaitResponse := make(chan uint32, 1)
	defer close(awaitResponse)

	// Select drum 1.
	depot.Drums = make(map[uint32](*Drum))
	depot.Drums[1] = &Drum{}
	depot.Alert(DEPOT_OP_SELECT|1, awaitResponse)
	<-awaitResponse

	assert.Equal(depot.Drums[1], depot.Drum)

	// Write some data to default drum
	SendAsUint8(depot, 0x55)

	// Send drum operation to select ring 1
	depot.Alert(DEPOT_OP_DRUM|DRUM_OP_SELECT|1, awaitResponse)
	value = <-awaitResponse

	// Should get response from drum
	assert.Equal(uint32(0), value) // Empty ring

	// Write to ring 1
	SendAsUint8(depot, 0xAA)

	// Select ring 0 again
	depot.Alert(DEPOT_OP_DRUM|DRUM_OP_SELECT, awaitResponse)
	value = <-awaitResponse
	assert.Equal(uint32(1), value) // Has 1 byte

	// Should read from ring 0
	seq := ReceiveAsUint8(depot)
	pull, stop := iter.Pull(seq)
	readValue, ok := pull()
	assert.True(ok)
	assert.Equal(uint8(0x55), readValue)
	stop()
}

func TestDepot_Alert_DrumRewindRead(t *testing.T) {
	assert := assert.New(t)

	depot := &Depot{}
	depot.Rewind()
	depot.Drums = make(map[uint32](*Drum))
	depot.Drums[0] = &Drum{}
	depot.Drum = depot.Drums[0]

	var response uint32
	awaitResponse := make(chan uint32, 1)
	defer close(awaitResponse)

	SendAsUint8(depot, 0x12)
	SendAsUint8(depot, 0x34)

	// Read first byte
	seq := ReceiveAsUint8(depot)
	pull, stop := iter.Pull(seq)
	value, ok := pull()
	assert.True(ok)
	assert.Equal(uint8(0x12), value)
	stop()

	// Rewind read pointer via drum operation
	depot.Alert(DEPOT_OP_DRUM|DRUM_OP_RING|RING_OP_REWIND_READ, awaitResponse)
	response = <-awaitResponse
	assert.Equal(uint32(0), response)

	// Should read first byte again
	seq = ReceiveAsUint8(depot)
	pull, stop = iter.Pull(seq)
	value, ok = pull()
	assert.True(ok)
	assert.Equal(uint8(0x12), value)
	stop()
}

func TestDrum_Alert_RingOperation(t *testing.T) {
	assert := assert.New(t)

	drum := &Drum{}
	drum.Rewind()

	awaitResponse := make(chan uint32, 1)
	defer close(awaitResponse)

	// Write to default ring
	SendAsUint8(drum, 0x01)

	// Rewind write index via ring operation
	drum.Alert(DRUM_OP_RING|RING_OP_REWIND_WRITE, awaitResponse)
	value := <-awaitResponse
	assert.Equal(uint32(0), value)

	// WriteIndex is now 0, but data is still [0x01]
	// Verify that we can still write and get ring operation response
	assert.Equal(0, drum.Ring.WriteIndex)
}

func TestDrum_selectRing_ExistingRing(t *testing.T) {
	assert := assert.New(t)

	existingRing := &Ring{WriteIndex: 42}
	drum := &Drum{
		Rings: map[uint8](*Ring){
			5: existingRing,
		},
	}

	drum.selectRing(5)
	assert.Equal(existingRing, drum.Ring)
	assert.Equal(42, drum.Ring.WriteIndex)
}
