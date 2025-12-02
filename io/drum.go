package io

import (
	"fmt"
	"io"
	"io/fs"
	"iter"
	"maps"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/ezrec/ucapp/internal"
)

const (
	// DRUM_OP_MASK masks the drum operation type from an alert request.
	DRUM_OP_MASK = (1 << 8)
	// DRUM_OP_SELECT indicates a ring selection operation.
	DRUM_OP_SELECT = (0 << 8)
	// DRUM_OP_SELECT_MASK masks the ring ID from a select operation.
	DRUM_OP_SELECT_MASK = 0xff
	// DRUM_OP_RING indicates a ring-level operation.
	DRUM_OP_RING = (1 << 8)
)

var _drum_defines = map[string]string{
	"DRUM_OP_MASK":        fmt.Sprintf("0x%x", DRUM_OP_MASK),
	"DRUM_OP_SELECT":      fmt.Sprintf("0x%x", DRUM_OP_SELECT),
	"DRUM_OP_SELECT_MASK": fmt.Sprintf("0x%x", DRUM_OP_SELECT_MASK),
	"DRUM_OP_RING":        fmt.Sprintf("0x%x", DRUM_OP_RING),
}

// Drum represents a collection of up to 256 rings, providing persistent storage
// similar to a drum memory device. It implements the Channel interface by
// forwarding operations to the currently selected ring.
type Drum struct {
	*Ring
	Rings map[uint8](*Ring)
}

// Defines returns an iter of defines for the channel.
func (dc *Drum) Defines() iter.Seq2[string, string] {
	return internal.IterSeq2Concat(maps.All(_drum_defines), (&Ring{}).Defines())
}

// Rewind resets all rings in the drum to their initial positions.
func (dc *Drum) Rewind() {
	for _, ring := range dc.Rings {
		ring.Rewind()
	}
}

// Unmarshal loads drum data from a file system by scanning for ring files
// matching the pattern XX.ring (2 hex digits).
func (drum *Drum) Unmarshal(filesys fs.FS) (err error) {
	drum.Rings = map[uint8](*Ring){}

	return fs.WalkDir(filesys, ".", func(path string, d fs.DirEntry, err_in error) (err error) {
		if d.IsDir() {
			return
		}
		name := d.Name()
		ok, err := regexp.MatchString("(?i)[0-9a-f][0-9a-f].ring", name)
		if err != nil {
			return
		}
		if !ok {
			// Skip this file.
			return nil
		}
		ring_index, err := strconv.ParseUint(strings.TrimSuffix(name, filepath.Ext(name)), 16, 8)
		if err != nil {
			return
		}

		// Ensure the ring exists, and unmarshal it.
		ring := &Ring{}
		ring.Rewind()
		drum.Rings[uint8(ring_index)] = ring
		ring_io, err := filesys.Open(name)
		if err != nil {
			return
		}
		defer ring_io.Close()

		ring.Unmarshal(ring_io)

		return
	})
}

// Marshal writes the drum's rings to a file system, creating files named
// XX.ring for each ring.
func (drum *Drum) Marshal(filesys CreateFS) (err error) {
	for index, ring := range drum.Rings {
		ring_name := fmt.Sprintf("%02x.ring", index)
		var ring_file io.WriteCloser
		ring_file, err = filesys.Create(ring_name)
		if err != nil {
			return
		}

		err = ring.Marshal(ring_file)
		ring_file.Close()
		if err != nil {
			return
		}
	}

	return
}

// Receive returns an iterator that yields bits from the currently selected ring.
// If no ring is selected, selects ring 0 by default.
func (dc *Drum) Receive() iter.Seq[bool] {
	if dc == nil {
		return func(func(bool) bool) {}
	}

	if dc.Ring == nil {
		dc.selectRing(0)
	}

	return dc.Ring.Receive()
}

// Send writes a bit to the currently selected ring.
// If no ring is selected, selects ring 0 by default.
func (dc *Drum) Send(value bool) (err error) {
	if dc == nil {
		err = ErrDrumMissing
		return
	}

	if dc.Ring == nil {
		dc.selectRing(0)
	}

	err = dc.Ring.Send(value)
	return
}

func (dc *Drum) selectRing(selected uint8) {
	ring, ok := dc.Rings[selected]
	if !ok {
		if dc.Rings == nil {
			dc.Rings = make(map[uint8](*Ring))
		}
		ring = &Ring{}
		ring.Rewind()
		dc.Rings[selected] = ring
	}
	dc.Ring = ring
}

// Alert handles drum control operations including ring selection and
// forwarding ring-specific operations to the currently selected ring.
func (dc *Drum) Alert(request uint32, response chan uint32) {
	if dc == nil {
		response <- ^uint32(0)
		return
	}

	// Ring request
	switch request & DRUM_OP_MASK {
	case DRUM_OP_SELECT:
		selected := uint8(request & DRUM_OP_SELECT_MASK)
		dc.selectRing(selected)
		response <- uint32(len(dc.Ring.Data))
	case DRUM_OP_RING:
		dc.Ring.Alert(request, response)
	}
}
