package io

import (
	"iter"
)

// SendAsUint8 sends an 8-bit unsigned integer as 8 bits to the channel,
// LSB first.
func SendAsUint8(ch Channel, value uint8) (err error) {
	for n := range 8 {
		err = ch.Send(((value >> n) & 1) == 1)
		if err != nil {
			return
		}
	}
	return
}

// ReceiveAsUint8 returns an iterator that reads bits from the channel and
// yields complete 8-bit unsigned integers, LSB first.
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

// SendAsUint16 sends a 16-bit unsigned integer as 16 bits to the channel,
// LSB first.
func SendAsUint16(ch Channel, value uint16) (err error) {
	for n := range 16 {
		err = ch.Send(((value >> n) & 1) == 1)
		if err != nil {
			return
		}
	}
	return
}

// ReceiveAsUint16 returns an iterator that reads bits from the channel and
// yields complete 16-bit unsigned integers, LSB first.
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

// SendAsUint32 sends a 32-bit unsigned integer as 32 bits to the channel,
// LSB first.
func SendAsUint32(ch Channel, value uint32) (err error) {
	for n := range 32 {
		err = ch.Send(((value >> n) & 1) == 1)
		if err != nil {
			return
		}
	}
	return
}

// ReceiveAsUint32 returns an iterator that reads bits from the channel and
// yields complete 32-bit unsigned integers, LSB first.
func ReceiveAsUint32(ch Channel) iter.Seq[uint32] {
	return func(yield func(value uint32) bool) {
		var n int
		var value uint32
		for bit := range ch.Receive() {
			if bit {
				value |= (1 << n)
			}
			if n == 31 {
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
