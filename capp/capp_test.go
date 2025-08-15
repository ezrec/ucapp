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
