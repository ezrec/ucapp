// Copyright 2024, Jason S. McMullan <jason.mcmullan@gmail.com>

package capp

import (
	"fmt"
	"log"
	"math/bits"
	"math/rand"
	"slices"
)

// Individual CAPP cell
type Cell struct {
	Set     [2]bool
	Tag     bool
	Data    uint32
	Next    *Cell
	Changed bool
}

// Computational Associative Parallel Processor
type Capp struct {
	Cell        []Cell
	Verbose     bool
	count       uint
	firstCell   *Cell
	BitsFlipped int
	SetsSwapped bool
}

// NewCapp creates a new CAPP.
func NewCapp(count uint) (cp *Capp) {
	cp = &Capp{
		Cell: make([]Cell, count),
	}

	cp.Reset()

	return
}

// Reset Capp
func (cp *Capp) Reset() {
	for n := range cp.Cell {
		cell := &cp.Cell[n]
		cell.Data = 0xffffffff
	}

	// Put all data into the set.
	cp.Action(SET_OF, 0xffffffff, 0xffffffff)
	cp.Action(LIST_ALL, 0, 0)

	cp.BitsFlipped = 0
}

// First gets the data of the first tagged item.
func (cp *Capp) First() (value uint32) {
	if cp.firstCell != nil {
		value = cp.firstCell.Data
	}
	return
}

// Count of the number of active (selected & tagged) cells.
func (cp *Capp) Count() (count uint) {
	count = cp.count
	return
}

// List returns the iterator for the list.
func (cp *Capp) List(yield func(data uint32) bool) {
	for cell := cp.firstCell; cell != nil; cell = cell.Next {
		if !yield(cell.Data) {
			break
		}
	}
}

// Randomize the flags and contents.
func (cp *Capp) Randomize(seed int) {
	rands := rand.New(rand.NewSource(int64(seed)))
	for n := range cp.Cell {
		cell := &cp.Cell[n]
		cell.Tag = (rands.Uint32() & 1) != 0
		cell.Set[0] = (rands.Uint32() & 1) != 0
		cell.Set[1] = (rands.Uint32() & 1) != 0
		cell.Data = rands.Uint32()
	}
}

func (cp *Capp) evaluateAll(eval func(cell *Cell)) {
	var set int = 0
	if cp.SetsSwapped {
		set = 1
	}

	cp.firstCell = nil
	cp.count = 0
	var current *Cell
	for n := range cp.Cell {
		cell := &cp.Cell[n]
		cell.Next = nil
		cell.Changed = false
		old_cell := *cell
		eval(cell)
		old_flipped := cp.BitsFlipped
		if old_cell.Set[set] != cell.Set[set] {
			cp.BitsFlipped++
		}
		if old_cell.Tag != cell.Tag {
			cp.BitsFlipped++
		}
		cp.BitsFlipped += bits.OnesCount32(cell.Data ^ old_cell.Data)
		cell.Changed = cp.BitsFlipped != old_flipped
		if cell.Set[set] && cell.Tag {
			if current == nil {
				current = cell
				cp.firstCell = cell
			} else {
				current.Next = cell
				current = cell
			}
			cp.count += 1
		}
	}
}

// Import an external set of cells.
func (cp *Capp) Import(cells []Cell) {
	cp.Cell = slices.Clone(cells)
	cp.evaluateAll(func(_ *Cell) {})
	cp.BitsFlipped = 0
}

// Action performs a Capp action on the memory.
func (cp *Capp) Action(action Action, match uint32, mask uint32) {
	if cp.Verbose {
		log.Printf("%-16v match:0x%08x mask:0x%08x\n", action, match, mask)
	}

	var set int = 0
	if cp.SetsSwapped {
		set = 1
	}

	switch action {
	case SET_SWAP:
		cp.SetsSwapped = !cp.SetsSwapped
		set = set ^ 1
		cp.evaluateAll(func(cell *Cell) {})
	case SET_OF:
		// Select only cells where bits set in mask match bits in word.
		cp.evaluateAll(func(cell *Cell) {
			cell.Set[set] = (cell.Data & mask) == (match & mask)
		})
		// Tag manipulation operations.
	case LIST_ALL:
		cp.evaluateAll(func(cell *Cell) {
			if cell.Set[set] {
				cell.Tag = true
			}
		})
	case LIST_NEXT:
		if cp.firstCell != nil {
			cp.firstCell.Tag = false
			cp.firstCell = cp.firstCell.Next
			cp.count -= 1
			cp.BitsFlipped++
			if cp.firstCell != nil {
				cp.firstCell.Changed = true
				for cell := cp.firstCell.Next; cell != nil; cell = cell.Next {
					cell.Changed = false
				}
			}
		}
	case LIST_NOT:
		cp.evaluateAll(func(cell *Cell) {
			if cell.Set[set] {
				cell.Tag = !cell.Tag
			}
		})
	case LIST_ONLY:
		// Keep only tagged cells where bits set in mask match bits in word.
		cp.evaluateAll(func(cell *Cell) {
			if cell.Tag && cell.Set[set] {
				cell.Tag = (cell.Data & mask) == (match & mask)
			}
		})
	case WRITE_LIST:
		// Update the bits set in mask with the bits in match.
		for cell := cp.firstCell; cell != nil; cell = cell.Next {
			old_data := cell.Data
			cell.Data = (cell.Data & ^mask) | (match & mask)
			old_flipped := cp.BitsFlipped
			cp.BitsFlipped += bits.OnesCount32(cell.Data ^ old_data)
			cell.Changed = old_flipped != cp.BitsFlipped
		}
	case WRITE_FIRST:
		if cp.firstCell != nil {
			cell := cp.firstCell
			// Update the bits set in mask with the bits in match.
			old_data := cell.Data
			cell.Data = (cell.Data & ^mask) | (match & mask)
			old_flipped := cp.BitsFlipped
			cp.BitsFlipped += bits.OnesCount32(cell.Data ^ old_data)
			cell.Changed = old_flipped != cp.BitsFlipped
			for cell := cp.firstCell.Next; cell != nil; cell = cell.Next {
				cell.Changed = false
			}
		}
	}

	if cp.Verbose {
		n := 0
		shown := 0
		for cell := cp.firstCell; cell != nil; cell = cell.Next {
			var context string
			if n == 0 {
				context = "first   "
			} else {
				context = fmt.Sprintf("next[%2d]", n-1)
			}
			n += 1
			changed := " "
			if cell.Changed {
				changed = "*"
			}
			shown += 1
			log.Printf("%v %v0x%04x", context, changed, cell.Data)
			if shown >= 8 {
				log.Printf(" ... (%d)", cp.Count())
				break
			}
		}
	}
}
