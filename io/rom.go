package io

import (
	"fmt"
	"iter"
	"maps"
)

const (
	// ARENA_ID_PROGRAM is the arena ID for program memory space.
	ARENA_ID_PROGRAM = uint32(2 << 30)
	// ROM_OP_TRAP sets up the trap notification channel for the ROM.
	ROM_OP_TRAP = uint32(0)
)

var _rom_defines = map[string]string{
	"ARENA_ID_PROGRAM": fmt.Sprintf("0x%x", ARENA_ID_PROGRAM),
	"ROM_OP_TRAP":      fmt.Sprintf("0x%x", ROM_OP_TRAP),
}

// Rom represents read-only memory that can issue trap notifications.
// It contains program data as 32-bit words and supports bit-level reading
// but not writing (writes return ErrChannelFull).
type Rom struct {
	Data []uint32

	trapChannel chan uint32
}

var _ Channel = (*Rom)(nil)

// Defines returns an iter of defines for the channel.
func (rc *Rom) Defines() iter.Seq2[string, string] {
	return maps.All(_rom_defines)
}

// Trap sends a trap notification to the registered trap channel if one is set.
func (rc *Rom) Trap() {
	// Issue a trap
	if rc.trapChannel != nil {
		rc.trapChannel <- 0
	}
}

// Rewind is a no-op for ROM as it is read-only and stateless.
func (rc *Rom) Rewind() {
	// Nothing to do.
}

// Receive returns an iterator that yields all bits from the ROM data,
// reading each 32-bit word LSB first.
func (rc *Rom) Receive() iter.Seq[bool] {
	return func(yield func(value bool) bool) {
		for _, data := range rc.Data {
			for bitpos := range 32 {
				bit := (data & (1 << bitpos)) != 0
				if !yield(bit) {
					return
				}
			}
		}
	}
}

// Send always returns ErrChannelFull as ROM is read-only.
func (rc *Rom) Send(value bool) error {
	return ErrChannelFull
}

// Alert handles ROM control operations, currently only supporting trap channel registration.
func (rc *Rom) Alert(request uint32, response chan uint32) {
	switch request {
	case ROM_OP_TRAP:
		rc.trapChannel = response
	default:
		response <- ^uint32(0)
	}
}
