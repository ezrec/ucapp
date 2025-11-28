package io

import (
	"iter"
)

const ARENA_ID_PROGRAM = uint32(2 << 30)

type Rom struct {
	AlertChannel
	Data []uint32
}

var _ Channel = (*Rom)(nil)

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

func (rc *Rom) Send(value bool) error {
	return ErrChannelFull
}
