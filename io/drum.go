package io

const (
	DRUM_OP_MASK        = (1 << 8)
	DRUM_OP_SELECT      = (0 << 8)
	DRUM_OP_SELECT_MASK = ((1 << 8) - 1)
	DRUM_OP_RING        = (1 << 8)
)

type Drum struct {
	*Ring
	Rings map[uint8](*Ring)
}

func (dc *Drum) Reset() {
	for _, ring := range dc.Rings {
		ring.Reset()
	}
	dc.selectRing(0)
}

func (dc *Drum) selectRing(selected uint8) {
	ring, ok := dc.Rings[selected]
	if !ok {
		if dc.Rings == nil {
			dc.Rings = make(map[uint8](*Ring))
		}
		ring = &Ring{}
		ring.Reset()
		dc.Rings[selected] = ring
	}
	dc.Ring = ring
}

func (dc *Drum) Alert(request uint32) {
	// Ring request
	switch request & DRUM_OP_MASK {
	case DRUM_OP_SELECT:
		selected := uint8(request & DRUM_OP_SELECT_MASK)
		dc.selectRing(selected)
		dc.Ring.SendAwait(uint32(len(dc.Ring.Data)))
	case DRUM_OP_RING:
		dc.Ring.Alert(request)
	}
}
