package io

import (
	"iter"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDepot_Reset(t *testing.T) {
	assert := assert.New(t)

	drum1 := &Drum{}
	drum1.Reset()
	drum1.Rings[0].WriteIndex = 10

	drum2 := &Drum{}
	drum2.Reset()
	drum2.Rings[0].WriteIndex = 20

	depot := &Depot{
		Drums: map[uint32](*Drum){
			1: drum1,
			2: drum2,
		},
	}

	depot.Reset()

	// Should reset all drums (reset sets ReadIndex to 0)
	assert.Equal(0, depot.Drums[1].Rings[0].ReadIndex)
	assert.Equal(0, depot.Drums[2].Rings[0].ReadIndex)

	// Should have drum 0 selected and created
	assert.NotNil(depot.Drums[0])
}

func TestDepot_selectDrum(t *testing.T) {
	assert := assert.New(t)

	depot := &Depot{}

	// Select drum that doesn't exist
	depot.selectDrum(5)
	assert.NotNil(depot.Drums)
	assert.NotNil(depot.Drums[5])

	// Select same drum again - should not create new
	existingDrum := depot.Drums[5]
	depot.selectDrum(5)
	assert.Equal(existingDrum, depot.Drums[5])
}

func TestDepot_Alert_Select(t *testing.T) {
	assert := assert.New(t)

	depot := &Depot{}
	depot.Reset()

	// Create and select drum 1
	depot.selectDrum(1)
	depot.Alert(DEPOT_OP_SELECT | 1)
	assert.NotNil(depot.Drum)

	SendAsUint8(depot, 0x42)

	// Create and select drum 2
	depot.selectDrum(2)
	depot.Alert(DEPOT_OP_SELECT | 2)
	SendAsUint8(depot, 0x99)

	// Switch back to drum 1
	depot.Alert(DEPOT_OP_SELECT | 1)

	// Should read from drum 1
	seq := ReceiveAsUint8(depot)
	pull, stop := iter.Pull(seq)
	value, ok := pull()
	assert.True(ok)
	assert.Equal(uint8(0x42), value)
	stop()

	// Switch to drum 2
	depot.Alert(DEPOT_OP_SELECT | 2)

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
	depot.Reset()

	// After Reset, depot.Drum should point to drum 0
	depot.Drum = depot.Drums[0]

	// Delete drum 99 to ensure it doesn't exist
	delete(depot.Drums, 99)

	depot.Alert(DEPOT_OP_SELECT | 99)

	// Should send error signal
	value, ok := depot.Await()
	assert.True(ok)
	assert.Equal(^uint32(0), value)
}

func TestDepot_Alert_DrumOperation(t *testing.T) {
	assert := assert.New(t)

	depot := &Depot{}
	depot.Reset()

	// Set depot.Drum to point to the drum created by Reset
	depot.Drum = depot.Drums[0]

	// Write some data to default drum
	SendAsUint8(depot, 0x55)

	// Send drum operation to select ring 1
	depot.Alert(DEPOT_OP_DRUM | DRUM_OP_SELECT | 1)

	// Should get response from drum
	value, ok := depot.Await()
	assert.True(ok)
	assert.Equal(uint32(0), value) // Empty ring

	// Write to ring 1
	SendAsUint8(depot, 0xAA)

	// Select ring 0 again
	depot.Alert(DEPOT_OP_DRUM | DRUM_OP_SELECT)
	value, ok = depot.Await()
	assert.True(ok)
	assert.Equal(uint32(1), value) // Has 1 byte

	// Should read from ring 0
	seq := ReceiveAsUint8(depot)
	pull, stop := iter.Pull(seq)
	readValue, ok := pull()
	assert.True(ok)
	assert.Equal(uint8(0x55), readValue)
	stop()
}

func TestDepot_Alert_DrumResetRead(t *testing.T) {
	assert := assert.New(t)

	depot := &Depot{}
	depot.Reset()
	depot.Drum = depot.Drums[0]

	SendAsUint8(depot, 0x12)
	SendAsUint8(depot, 0x34)

	// Read first byte
	seq := ReceiveAsUint8(depot)
	pull, stop := iter.Pull(seq)
	value, ok := pull()
	assert.True(ok)
	assert.Equal(uint8(0x12), value)
	stop()

	// Reset read pointer via drum operation
	depot.Alert(DEPOT_OP_DRUM | DRUM_OP_RING | RING_OP_RESET_READ)
	resp, ok := depot.Await()
	assert.True(ok)
	assert.Equal(uint32(0), resp)

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
	drum.Reset()

	// Write to default ring
	SendAsUint8(drum, 0x01)

	// Reset write index via ring operation
	drum.Alert(DRUM_OP_RING | RING_OP_RESET_WRITE)
	value, ok := drum.Await()
	assert.True(ok)
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
