package io

import (
	"io"
	"iter"
)

// Tape provides sequential I/O operations for reading and writing byte streams.
// It wraps an io.Reader for input and io.Writer for output, converting between
// bit-level Channel operations and byte-level I/O.
type Tape struct {
	Input      io.Reader
	LastInput  byte
	ReadIndex  int
	Output     io.Writer
	NextOutput byte
	WriteIndex int
}

// Rewind resets the tape's read and write indices to zero and clears buffered output.
func (tc *Tape) Rewind() {
	tc.ReadIndex = 0
	tc.WriteIndex = 0
	tc.NextOutput = 0
}

// Receive returns an iterator that yields bits from the input stream,
// reading bytes as needed and yielding them LSB first.
func (tc *Tape) Receive() iter.Seq[bool] {
	return func(yield func(value bool) bool) {
		for {
			if tc.ReadIndex == 0 {
				var err error
				var one [1]byte
				_, err = tc.Input.Read(one[:])
				if err != nil {
					return
				}
				tc.LastInput = one[0]
			}
			for ; tc.ReadIndex < 8; tc.ReadIndex++ {
				bit := ((tc.LastInput >> tc.ReadIndex) & 1) != 0
				if !yield(bit) {
					return
				}
			}
			tc.ReadIndex = 0
		}
	}
}

// Send writes a bit to the output stream, buffering bits until a complete
// byte is assembled, then writing it.
func (tc *Tape) Send(value bool) (err error) {
	if value {
		tc.NextOutput |= 1 << tc.WriteIndex
	}

	tc.WriteIndex++

	for tc.WriteIndex == 8 {
		tc.Output.Write([]byte{tc.NextOutput})
		tc.NextOutput = 0
		tc.WriteIndex = 0
	}

	return
}

// Alert returns an error response for all requests as Tape does not support
// control operations.
func (tc *Tape) Alert(request uint32, response chan uint32) {
	response <- ^uint32(0)
}
