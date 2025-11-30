package io

import (
	"iter"
)

// Temporary implements a circular buffer for temporary bit storage.
// It operates as a FIFO queue with a fixed capacity and separate read/write positions.
type Temporary struct {
	Capacity int // Capacity in bits.

	ReadIndex  int
	WriteIndex int
	Size       int
	Data       []bool
}

var _ Channel = (*Temporary)(nil)

// Rewind resets the temporary storage to empty, resetting indices and
// reinitializing the data buffer.
func (temp *Temporary) Rewind() {
	temp.ReadIndex = 0
	temp.WriteIndex = 0
	temp.Size = 0
	temp.Data = make([]bool, temp.Capacity)
}

// Receive returns an iterator that yields bits from the buffer until empty.
// The buffer wraps around at the capacity boundary.
func (temp *Temporary) Receive() iter.Seq[bool] {
	return func(yield func(value bool) bool) {
		for temp.Size > 0 {
			bit := temp.Data[temp.ReadIndex]
			temp.ReadIndex++
			if temp.ReadIndex == temp.Capacity {
				temp.ReadIndex = 0
			}
			temp.Size--
			if !yield(bit) {
				return
			}
		}
	}
}

// Send writes a bit to the buffer at the current write position.
// Returns ErrChannelFull if the buffer has reached capacity.
func (temp *Temporary) Send(value bool) (err error) {
	if temp.Size >= temp.Capacity {
		err = ErrChannelFull
		return
	}

	temp.Data[temp.WriteIndex] = value

	temp.WriteIndex++
	if temp.WriteIndex == temp.Capacity {
		temp.WriteIndex = 0
	}
	temp.Size++

	return
}

// Alert returns an error response for all requests as Temporary does not
// support control operations.
func (temp *Temporary) Alert(request uint32, response chan uint32) {
	response <- ^uint32(0)
}
