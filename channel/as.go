package channel

import (
	"iter"
)

func SendAsUint8(ch Channel, value uint8) (err error) {
	for n := range 8 {
		err = ch.Send(((value >> n) & 1) == 1)
		if err != nil {
			return
		}
	}
	return
}

func ReceiveAsUint8(ch Channel) iter.Seq[uint8] {
	return func(yield func(value uint8) bool) {
		var n int
		var value uint8
		for bit := range ch.Receive() {
			if bit {
				value |= (1 << n)
			}
			if n == 7 {
				if !yield(value) {
					return
				}
				value = 0
				n = 0
			} else {
				n++
			}
		}
		if n != 0 {
			yield(value)
		}
	}
}

func SendAsUint16(ch Channel, value uint16) (err error) {
	for n := range 16 {
		err = ch.Send(((value >> n) & 1) == 1)
		if err != nil {
			return
		}
	}
	return
}

func ReceiveAsUint16(ch Channel) iter.Seq[uint16] {
	return func(yield func(value uint16) bool) {
		var n int
		var value uint16
		for bit := range ch.Receive() {
			if bit {
				value |= (1 << n)
			}
			if n == 15 {
				if !yield(value) {
					return
				}
				value = 0
				n = 0
			} else {
				n++
			}
		}
		if n != 0 {
			yield(value)
		}
	}
}
