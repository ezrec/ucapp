package io

import (
	"io"
	"iter"
)

const (
	// RING_OP_MASK masks the ring operation type from an alert request.
	RING_OP_MASK = (1 << 7) - 1
	// RING_OP_REWIND_READ resets the read position to the start.
	RING_OP_REWIND_READ = 0
	// RING_OP_REWIND_WRITE resets the write position to the start.
	RING_OP_REWIND_WRITE = 1

	// RING_DEFAULT_CAPACITY is the default capacity in bits for a new ring.
	RING_DEFAULT_CAPACITY = 65536
)

// Ring represents a circular buffer storage device with separate read and write
// positions. It stores up to 64KB of data and supports sequential bit-level I/O.
type Ring struct {
	Capacity int

	Readable   bool
	Writable   bool
	Executable bool

	WriteIndex int
	ReadIndex  int
	Data       []uint8
}

var _ Channel = (*Ring)(nil)

// Rewind resets the ring's read position to the start and write position to the end
// of existing data. Initializes the data buffer if not already allocated.
func (ring *Ring) Rewind() {
	if ring.Data == nil {
		if ring.Capacity == 0 {
			ring.Capacity = RING_DEFAULT_CAPACITY
		}
		ring.Data = make([]byte, 0, (ring.Capacity+7)/8)
	} else {
		ring.Capacity = cap(ring.Data) * 8
	}

	ring.ReadIndex = 0
	ring.WriteIndex = len(ring.Data) * 8
}

// Unmarshal loads ring data from a reader, replacing any existing data.
func (ring *Ring) Unmarshal(file io.Reader) (err error) {
	data, err := io.ReadAll(file)
	if err != nil {
		return
	}

	ring.Data = data
	ring.ReadIndex = 0
	ring.WriteIndex = len(ring.Data) * 8

	return
}

// Marshal writes the ring's data to a writer up to the current write position.
func (ring *Ring) Marshal(file io.Writer) (err error) {
	_, err = file.Write(ring.Data[0 : (ring.WriteIndex+7)/8])

	return
}

// Receive returns an iterator that yields bits from the ring starting at the
// current read position up to the write position.
func (ring *Ring) Receive() iter.Seq[bool] {
	if ring == nil {
		return func(func(bool) bool) {}
	}

	return func(yield func(value bool) bool) {
		for ring.ReadIndex < ring.WriteIndex {
			value := ring.Data[ring.ReadIndex/8]
			n := ring.ReadIndex % 8
			bit := ((value >> n) & 1) != 0
			ring.ReadIndex++
			if !yield(bit) {
				return
			}
		}
	}
}

// Send writes a bit to the ring at the current write position.
// Returns ErrChannelFull if the ring has reached capacity.
func (ring *Ring) Send(value bool) (err error) {
	if ring == nil {
		err = ErrChannelFull
		return
	}

	if ring.WriteIndex >= ring.Capacity {
		err = ErrChannelFull
		return
	}

	for (ring.WriteIndex / 8) >= len(ring.Data) {
		ring.Data = append(ring.Data, 0xff)
	}

	bitmask := ring.Data[ring.WriteIndex/8]
	if value {
		bitmask |= 1 << (ring.WriteIndex % 8)
	} else {
		bitmask &= ^(1 << (ring.WriteIndex % 8))
	}
	ring.Data[ring.WriteIndex/8] = bitmask

	ring.WriteIndex++

	return
}

// Alert handles ring control operations including resetting read and write positions.
func (ring *Ring) Alert(request uint32, response chan uint32) {
	if ring == nil {
		response <- ^uint32(0)
		return
	}

	switch request & RING_OP_MASK {
	case RING_OP_REWIND_READ:
		ring.ReadIndex = 0
		response <- 0
	case RING_OP_REWIND_WRITE:
		ring.WriteIndex = 0
		response <- 0
	default:
		response <- ^uint32(0)
	}
}
