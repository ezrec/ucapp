package channel

import (
	"io"
	"iter"
)

type Tape struct {
	AlertChannel
	Input      io.Reader
	LastInput  byte
	ReadIndex  int
	Output     io.Writer
	NextOutput byte
	WriteIndex int
}

func (tc *Tape) Reset() {
	tc.ReadIndex = 0
	tc.WriteIndex = 0
	tc.NextOutput = 0
}

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
