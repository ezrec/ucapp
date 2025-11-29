// Copyright 2024, Jason S. McMullan <jason.mcmullan@gmail.com>

package capp

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCapp(t *testing.T) {
	const size = 128
	assert := assert.New(t)

	table := [](struct {
		Action Action
		Match  uint32
		Mask   uint32
		Data   uint32
		Count  uint
	}){
		{Action: WRITE_LIST, Match: 0, Mask: 0xffffffff, Count: size, Data: 0}, // Zero the CAPP
		{Action: LIST_NOT}, // Remove all tags.
		{Action: SET_OF, Match: 0xffffffff, Mask: 0xffffffff},                        // Select nothing.
		{Action: WRITE_FIRST, Match: 0b1101},                                         // No-op
		{Action: LIST_ALL, Data: 0, Count: 0},                                        // Nothing selected.
		{Action: LIST_NOT, Data: 0, Count: 0},                                        // Nothing selected, so no-op.
		{Action: SET_OF, Mask: 0b1111, Match: 0, Data: 0, Count: 0},                  // Select all
		{Action: LIST_ALL, Data: 0, Count: 128},                                      // Everything is tagged.
		{Action: WRITE_FIRST, Mask: 0, Match: 0b1100, Count: 128},                    /// Write to first.
		{Action: LIST_ALL, Data: 0, Count: 128},                                      // Selection has not changed.
		{Action: WRITE_LIST, Data: 0b0010, Mask: 0b0011, Match: 0b0010, Count: 128},  // Update lower 2 bits to 10
		{Action: LIST_NEXT, Data: 0b0010, Count: 127},                                // Unselect first tag
		{Action: LIST_ALL, Data: 0b0010, Count: 128},                                 // Re-tag from selection
		{Action: SET_OF, Mask: 0b0011, Match: 0b0000, Count: 0},                      // Select non-existent data
		{Action: SET_OF, Data: 0b0010, Mask: 0b0011, Match: 0b0010, Count: 128},      // Select matching and tagged data
		{Action: LIST_ONLY, Data: 0b0010, Mask: 0b0011, Match: 0b0010, Count: 128},   // Winnow none.
		{Action: LIST_ONLY, Data: 0b0000, Mask: 0b0011, Match: 0b0001, Count: 0},     // Winnow all.
		{Action: LIST_NOT, Data: 0b0010, Count: 128},                                 // Re-tag all
		{Action: WRITE_FIRST, Data: 0b1001, Mask: 0b1111, Match: 0b1001, Count: 128}, // Write to first.
		{Action: LIST_NEXT, Data: 0b0010, Count: 127},                                // Clear tag on first
		{Action: WRITE_FIRST, Data: 0b1010, Mask: 0b1111, Match: 0b1010, Count: 127}, // Write to first.
		{Action: LIST_NOT, Data: 0b1001, Count: 1},
		{Action: LIST_NOT, Data: 0b1010, Count: 127},
		{Action: LIST_NEXT, Data: 0b0010, Count: 126},
		{Action: WRITE_FIRST, Data: 0b1011, Mask: 0b1111, Match: 0b1011, Count: 126},
		{Action: LIST_NEXT, Data: 0b0010, Count: 125},
		{Action: LIST_NOT, Data: 0b1001, Count: 3}, // Complement, then verify the first three entries.
		{Action: LIST_NEXT, Data: 0b1010, Count: 2},
		{Action: LIST_NEXT, Data: 0b1011, Count: 1},
		{Action: LIST_NEXT, Data: 0b0000, Count: 0},
	}

	cp := NewCapp(size)

	for _, testcase := range table {
		cp.Action(testcase.Action, testcase.Match, testcase.Mask)
		assert.Equal(testcase.Data, cp.First(), fmt.Sprintf("%+v", testcase))
		assert.Equal(testcase.Count, cp.Count(), fmt.Sprintf("%+v", testcase))
	}
}

// TestList verifies the List iterator functionality.
func TestList(t *testing.T) {
	assert := assert.New(t)
	cp := NewCapp(10)

	cp.Action(WRITE_LIST, 0, 0xffffffff)
	for i := uint32(0); i < 10; i++ {
		cp.Action(WRITE_FIRST, i, 0xffffffff)
		cp.Action(LIST_NEXT, 0, 0)
	}
	cp.Action(LIST_ALL, 0, 0)

	collected := []uint32{}
	for data := range cp.List {
		collected = append(collected, data)
	}

	assert.Equal([]uint32{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}, collected)
}

// TestListEarlyBreak verifies that the List iterator stops when yield returns false.
func TestListEarlyBreak(t *testing.T) {
	assert := assert.New(t)
	cp := NewCapp(10)

	cp.Action(WRITE_LIST, 0, 0xffffffff)
	for i := uint32(0); i < 10; i++ {
		cp.Action(WRITE_FIRST, i, 0xffffffff)
		cp.Action(LIST_NEXT, 0, 0)
	}
	cp.Action(LIST_ALL, 0, 0)

	collected := []uint32{}
	for data := range cp.List {
		collected = append(collected, data)
		if data == 4 {
			break
		}
	}

	assert.Equal([]uint32{0, 1, 2, 3, 4}, collected)
}

// TestListEmpty verifies List works with no tagged cells.
func TestListEmpty(t *testing.T) {
	assert := assert.New(t)
	cp := NewCapp(10)
	cp.Action(WRITE_LIST, 0, 0xffffffff)
	cp.Action(SET_OF, 0xffffffff, 0xffffffff)
	cp.Action(LIST_ALL, 0, 0)

	collected := []uint32{}
	for data := range cp.List {
		collected = append(collected, data)
	}

	assert.Equal([]uint32{}, collected)
	assert.Equal(uint(0), cp.Count())
}

// TestRandomize verifies that Randomize sets cell properties.
func TestRandomize(t *testing.T) {
	assert := assert.New(t)
	cp := NewCapp(100)

	cp.Randomize(42)

	foundTagTrue := false
	foundTagFalse := false
	foundSet0True := false
	foundSet0False := false
	foundSet1True := false
	foundSet1False := false
	foundNonFF := false

	for i := range cp.Cell {
		cell := &cp.Cell[i]
		if cell.Tag {
			foundTagTrue = true
		} else {
			foundTagFalse = true
		}
		if cell.Set[0] {
			foundSet0True = true
		} else {
			foundSet0False = true
		}
		if cell.Set[1] {
			foundSet1True = true
		} else {
			foundSet1False = true
		}
		if cell.Data != 0xffffffff {
			foundNonFF = true
		}
	}

	assert.True(foundTagTrue, "Should have at least one tagged cell")
	assert.True(foundTagFalse, "Should have at least one untagged cell")
	assert.True(foundSet0True, "Should have at least one Set[0] true")
	assert.True(foundSet0False, "Should have at least one Set[0] false")
	assert.True(foundSet1True, "Should have at least one Set[1] true")
	assert.True(foundSet1False, "Should have at least one Set[1] false")
	assert.True(foundNonFF, "Data should be randomized")
}

// TestImport verifies Import correctly clones and evaluates cells.
func TestImport(t *testing.T) {
	assert := assert.New(t)

	sourceCells := []Cell{
		{Set: [2]bool{true, false}, Tag: true, Data: 0x1111},
		{Set: [2]bool{true, false}, Tag: true, Data: 0x2222},
		{Set: [2]bool{false, false}, Tag: false, Data: 0x3333},
		{Set: [2]bool{true, false}, Tag: true, Data: 0x4444},
	}

	cp := NewCapp(1)
	cp.Import(sourceCells)

	assert.Equal(uint(3), cp.Count())
	assert.Equal(uint32(0x1111), cp.First())

	collected := []uint32{}
	for data := range cp.List {
		collected = append(collected, data)
	}
	assert.Equal([]uint32{0x1111, 0x2222, 0x4444}, collected)

	sourceCells[0].Data = 0x9999
	assert.Equal(uint32(0x1111), cp.Cell[0].Data, "Import should clone, not reference")
	assert.Equal(0, cp.BitsFlipped, "BitsFlipped should be reset after Import")
}

// TestSetSwap verifies SET_SWAP action toggles between set banks.
func TestSetSwap(t *testing.T) {
	assert := assert.New(t)
	cp := NewCapp(10)

	cp.Action(WRITE_LIST, 0, 0xffffffff)
	cp.Action(LIST_ALL, 0, 0)

	for i := uint32(0); i < 5; i++ {
		cp.Action(WRITE_FIRST, 100+i, 0xffffffff)
		cp.Cell[i].Set[1] = false
		cp.Action(LIST_NEXT, 0, 0)
	}
	for i := uint32(5); i < 10; i++ {
		cp.Action(WRITE_FIRST, 200+i, 0xffffffff)
		cp.Cell[i].Set[1] = true
		cp.Action(LIST_NEXT, 0, 0)
	}

	cp.Action(LIST_ALL, 0, 0)
	assert.Equal(uint(10), cp.Count())

	cp.Action(SET_SWAP, 0, 0)
	assert.True(cp.SetsSwapped)
	assert.Equal(uint(5), cp.Count())

	cp.Action(SET_SWAP, 0, 0)
	assert.False(cp.SetsSwapped)
	assert.Equal(uint(10), cp.Count())
}

// TestBitsFlipped verifies BitsFlipped counter tracks changes.
func TestBitsFlipped(t *testing.T) {
	assert := assert.New(t)
	cp := NewCapp(4)

	cp.Action(WRITE_LIST, 0, 0xffffffff)
	cp.Action(LIST_ALL, 0, 0)
	cp.BitsFlipped = 0

	cp.Action(WRITE_FIRST, 0b1111, 0xffffffff)
	assert.Equal(4, cp.BitsFlipped, "Writing 4 bits should flip 4 bits")

	cp.BitsFlipped = 0
	cp.Action(WRITE_FIRST, 0b11111111, 0xff)
	assert.Equal(4, cp.BitsFlipped, "Flipping 4 more bits in first cell")

	cp.BitsFlipped = 0
	cp.Action(LIST_NOT, 0, 0)
	assert.Equal(4, cp.BitsFlipped, "Toggling 4 tag bits")

	cp.BitsFlipped = 0
	cp.Action(SET_OF, 0b11111111, 0xffffffff)
	assert.Equal(3, cp.BitsFlipped, "Changing 3 Set bits (first cell unchanged)")
}

// TestCellChanged verifies the Changed flag is set appropriately.
func TestCellChanged(t *testing.T) {
	assert := assert.New(t)
	cp := NewCapp(5)

	cp.Action(WRITE_LIST, 0, 0xffffffff)
	for i := uint32(0); i < 5; i++ {
		cp.Action(WRITE_FIRST, i, 0xffffffff)
		cp.Action(LIST_NEXT, 0, 0)
	}
	cp.Action(LIST_ALL, 0, 0)

	cp.Action(WRITE_FIRST, 100, 0xffffffff)
	assert.True(cp.Cell[0].Changed, "First cell should be marked as changed")
	assert.False(cp.Cell[1].Changed, "Other cells should not be marked as changed")

	cp.Action(WRITE_LIST, 200, 0xffffffff)
	changedCount := 0
	for i := range cp.Cell {
		if cp.Cell[i].Changed {
			changedCount++
		}
	}
	assert.Equal(5, changedCount, "All tagged cells should be changed")
}

// TestVerboseMode verifies verbose mode doesn't crash.
func TestVerboseMode(t *testing.T) {
	cp := NewCapp(4)
	cp.Verbose = true

	cp.Action(WRITE_LIST, 0, 0xffffffff)
	cp.Action(LIST_ALL, 0, 0)
	cp.Action(WRITE_FIRST, 42, 0xffffffff)
}

// TestLargeVerboseList verifies verbose output handles large lists.
func TestLargeVerboseList(t *testing.T) {
	cp := NewCapp(20)
	cp.Verbose = true

	cp.Action(WRITE_LIST, 0, 0xffffffff)
	cp.Action(LIST_ALL, 0, 0)
	cp.Action(WRITE_LIST, 0xAAAA, 0xffffffff)
}

// TestReset verifies Reset properly initializes CAPP state.
func TestReset(t *testing.T) {
	assert := assert.New(t)
	cp := NewCapp(5)

	cp.Action(WRITE_LIST, 0x12345678, 0xffffffff)
	cp.BitsFlipped = 999

	cp.Reset()

	assert.Equal(0, cp.BitsFlipped)
	assert.Equal(uint(5), cp.Count())
	for i := range cp.Cell {
		assert.Equal(uint32(0xffffffff), cp.Cell[i].Data)
	}
}

// TestActionString verifies Action.String() method.
func TestActionString(t *testing.T) {
	assert := assert.New(t)

	assert.Equal("SET_SWAP", SET_SWAP.String())
	assert.Equal("LIST_ALL", LIST_ALL.String())
	assert.Equal("LIST_NOT", LIST_NOT.String())
	assert.Equal("LIST_NEXT", LIST_NEXT.String())
	assert.Equal("LIST_ONLY", LIST_ONLY.String())
	assert.Equal("SET_OF", SET_OF.String())
	assert.Equal("WRITE_FIRST", WRITE_FIRST.String())
	assert.Equal("WRITE_LIST", WRITE_LIST.String())

	invalidAction := Action(999)
	assert.Equal("Action(999)", invalidAction.String())
	invalidNegAction := Action(-1)
	assert.Equal("Action(-1)", invalidNegAction.String())
}

// TestFirstEmptyCapp verifies First returns 0 when no cells are tagged.
func TestFirstEmptyCapp(t *testing.T) {
	assert := assert.New(t)
	cp := NewCapp(5)
	cp.Action(WRITE_LIST, 0, 0xffffffff)
	cp.Action(SET_OF, 0xffffffff, 0xffffffff)
	cp.Action(LIST_ALL, 0, 0)

	assert.Equal(uint32(0), cp.First())
	assert.Equal(uint(0), cp.Count())
}

// TestWriteFirstNoChange verifies WRITE_FIRST with matching data doesn't increment BitsFlipped.
func TestWriteFirstNoChange(t *testing.T) {
	assert := assert.New(t)
	cp := NewCapp(3)

	cp.Action(WRITE_LIST, 0x1234, 0xffffffff)
	cp.Action(LIST_ALL, 0, 0)
	cp.BitsFlipped = 0

	cp.Action(WRITE_FIRST, 0x1234, 0xffffffff)
	assert.Equal(0, cp.BitsFlipped)
}

// TestWriteListNoChange verifies WRITE_LIST with matching data doesn't increment BitsFlipped.
func TestWriteListNoChange(t *testing.T) {
	assert := assert.New(t)
	cp := NewCapp(3)

	cp.Action(WRITE_LIST, 0xABCD, 0xffffffff)
	cp.Action(LIST_ALL, 0, 0)
	cp.BitsFlipped = 0

	cp.Action(WRITE_LIST, 0xABCD, 0xffffffff)
	assert.Equal(0, cp.BitsFlipped)
}

// TestSwappedSetsEvaluation verifies evaluateAll works correctly with swapped sets.
func TestSwappedSetsEvaluation(t *testing.T) {
	assert := assert.New(t)
	cp := NewCapp(5)

	cp.Action(WRITE_LIST, 0, 0xffffffff)
	cp.Cell[0].Set[1] = true
	cp.Cell[1].Set[1] = true
	cp.Cell[2].Set[1] = false
	cp.Cell[0].Tag = true
	cp.Cell[1].Tag = true
	cp.Cell[2].Tag = true

	cp.Action(SET_SWAP, 0, 0)
	assert.Equal(uint(2), cp.Count())
	assert.Equal(uint32(0), cp.First())
}
