// Package io provides I/O channel implementations for the μCAPP emulator.
// It includes various channel types for bit-level I/O operations including
// temporary storage (Temp), persistent drum/ring storage (Depot, Drum, Ring),
// sequential I/O (Tape), and ROM.
package io

import (
	"iter"
)

// Channel defines the interface for all I/O channels in the μCAPP system.
// Channels operate at the bit level and support sequential reading, writing,
// and control operations via alerts.
type Channel interface {
	// Rewind resets the channel to its initial state.
	Rewind()
	// Receive returns an iterator that yields bits from the channel.
	Receive() iter.Seq[bool]
	// Send writes a single bit to the channel.
	Send(value bool) error
	// Alert sends a control message to the channel with a response callback.
	Alert(value uint32, response chan uint32)
}
