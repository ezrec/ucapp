package channel

import (
	"iter"
)

type Temporary struct {
	AlertChannel
	Capacity int // Capacity in bits.

	ReadIndex  int
	WriteIndex int
	Size       int
	Data       []bool
}

var _ Channel = (*Temporary)(nil)

func (temp *Temporary) Reset() {
	temp.ReadIndex = 0
	temp.WriteIndex = 0
	temp.Size = 0
	temp.Data = make([]bool, temp.Capacity)
}

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
