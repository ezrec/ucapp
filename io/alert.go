package io

import (
	"iter"
)

type AlertChannel struct {
	FromDevice []uint32
	ToDevice   []uint32
}

func (ac *AlertChannel) Reset() {
	ac.FromDevice = nil
	ac.ToDevice = nil
}

func (ac *AlertChannel) Send(value bool) (err error) {
	err = ErrChannelFull
	return
}

func (ac *AlertChannel) Receive() (seq iter.Seq[bool]) {
	return
}

func (ac *AlertChannel) Alert(value uint32) {
	ac.SetAlert(value)
}

func (ac *AlertChannel) Await() (value uint32, ok bool) {
	if len(ac.FromDevice) > 0 {
		ok = true
		value = ac.FromDevice[0]
		ac.FromDevice = ac.FromDevice[1:]
	}
	return
}

func (ac *AlertChannel) GetAlert() (alert uint32, ok bool) {
	if len(ac.ToDevice) > 0 {
		ok = true
		alert = ac.ToDevice[0]
		ac.ToDevice = ac.ToDevice[1:]
	}

	return
}

func (ac *AlertChannel) SetAlert(alert uint32) {
	ac.ToDevice = append(ac.ToDevice, alert)
}

func (ac *AlertChannel) SendAwait(value uint32) {
	ac.FromDevice = append(ac.FromDevice, value)
}
