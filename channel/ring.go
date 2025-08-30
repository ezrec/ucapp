package channel

import (
	"iter"
)

const (
	RING_OP_MASK        = (1 << 7) - 1
	RING_OP_RESET_READ  = 0
	RING_OP_RESET_WRITE = 1

	RING_DEFAULT_CAPACITY = 65536
)

type Ring struct {
	AlertChannel
	Capacity int

	Readable   bool
	Writable   bool
	Executable bool

	WriteIndex int
	ReadIndex  int
	Data       []uint8
}

var _ Channel = (*Ring)(nil)

func (ring *Ring) Reset() {
	if ring.Data == nil {
		if ring.Capacity == 0 {
			ring.Capacity = RING_DEFAULT_CAPACITY
		}
		ring.Data = make([]byte, 0, (ring.Capacity+7)/8)
	} else {
		ring.Capacity = cap(ring.Data) * 8
	}

	ring.ReadIndex = 0
}

func (ring *Ring) Receive() iter.Seq[bool] {
	return func(yield func(value bool) bool) {
		for ring.ReadIndex < ring.WriteIndex {
			value := ring.Data[ring.ReadIndex/8]
			for n := ring.ReadIndex % 8; n < 8; n++ {
				bit := ((value >> n) & 1) != 0
				ring.ReadIndex++
				if !yield(bit) {
					return
				}
			}
		}
	}
}

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
		ring.Data = append(ring.Data, 0)
	}

	var bitmask uint8
	if value {
		bitmask |= 1 << (ring.WriteIndex % 8)
	}
	ring.Data[ring.WriteIndex/8] |= bitmask

	ring.WriteIndex++

	return
}

func (ring *Ring) Alert(request uint32) {
	switch request & RING_OP_MASK {
	case RING_OP_RESET_READ:
		ring.ReadIndex = 0
		ring.SendAwait(0)
	case RING_OP_RESET_WRITE:
		ring.WriteIndex = 0
		ring.SendAwait(0)
	default:
		ring.SendAwait(^uint32(0))
	}
}
