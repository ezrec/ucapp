package io

import (
	"io"
	"iter"
	"maps"
)

// Tape provides sequential I/O operations for reading and writing byte streams.
// It wraps an io.Reader for input and io.Writer for output, converting between
// bit-level Channel operations and byte-level I/O.
type Tape struct {
	Input  io.Reader
	Output io.Writer

	readIndex int
	hasInput  bool
	lastInput byte

	nextOutput byte
	writeIndex int
}

// Defines returns an iter of defines for the channel.
func (tc *Tape) Defines() iter.Seq2[string, string] {
	return maps.All(map[string]string{})
}

// Rewind is not possible on a tape.
func (tc *Tape) Rewind() {
}

// Receive returns an iterator that yields bits from the input stream,
// reading bytes as needed and yielding them LSB first.
func (tc *Tape) Receive() iter.Seq[bool] {
	return func(yield func(value bool) bool) {
		for {
			if tc.readIndex == 0 && !tc.hasInput {
				var err error
				var one [1]byte
				_, err = tc.Input.Read(one[:])
				if err != nil {
					return
				}
				tc.lastInput = one[0]
				tc.hasInput = true
			}
			bit := ((tc.lastInput >> tc.readIndex) & 1) != 0
			if !yield(bit) {
				return
			}
			tc.readIndex++
			if tc.readIndex == 8 {
				tc.readIndex = 0
				tc.hasInput = false
			}
		}
	}
}

// Send writes a bit to the output stream, buffering bits until a complete
// byte is assembled, then writing it.
func (tc *Tape) Send(value bool) (err error) {
	if value {
		tc.nextOutput |= 1 << tc.writeIndex
	}

	tc.writeIndex++

	for tc.writeIndex == 8 {
		tc.Output.Write([]byte{tc.nextOutput})
		tc.nextOutput = 0
		tc.writeIndex = 0
	}

	return
}

// Alert returns an error response for all requests as Tape does not support
// control operations.
func (tc *Tape) Alert(request uint32, response chan uint32) {
	response <- ^uint32(0)
}
