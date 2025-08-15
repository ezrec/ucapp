package channel

import (
	"iter"
)

type Channel interface {
	Reset()
	Receive() iter.Seq[bool] // Iteration of bits in the channel.
	Send(value bool) error   // Send a bit to the channel.
	Alert(value uint32)      // Notify channel with value
	GetAlert() (value uint32, ok bool)
	Await() (value uint32, ok bool) // Await a value
}
