package io

import ()

const (
	DEPOT_OP_MASK        = (1 << 23)
	DEPOT_OP_SELECT      = (0 << 23)
	DEPOT_OP_DRUM        = (1 << 23)
	DEPOT_OP_SELECT_MASK = ((1 << 23) - 1)
)

type Depot struct {
	*Drum
	Drums map[uint32](*Drum)
}

var _ Channel = &Depot{}

func (depot *Depot) selectDrum(index uint32) {
	if depot.Drums == nil {
		depot.Drums = make(map[uint32](*Drum))
	}
	_, ok := depot.Drums[index]
	if !ok {
		drum := &Drum{}
		drum.Reset()
		depot.Drums[index] = drum
	}
}

func (depot *Depot) Reset() {
	for _, drum := range depot.Drums {
		drum.Reset()
	}

	depot.selectDrum(0)
}

func (depot *Depot) Alert(request uint32) {
	switch request & DEPOT_OP_MASK {
	case DEPOT_OP_SELECT:
		drum_id := request & DEPOT_OP_SELECT_MASK
		drum, ok := depot.Drums[drum_id]
		if !ok {
			depot.SendAwait(^uint32(0))
			return
		}
		depot.Drum = drum
	case DEPOT_OP_DRUM:
		depot.Drum.Alert(request)
	}
}
