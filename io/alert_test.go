package io

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAlertChannel_Reset(t *testing.T) {
	assert := assert.New(t)

	ac := &AlertChannel{
		FromDevice: []uint32{1, 2, 3},
		ToDevice:   []uint32{4, 5, 6},
	}

	ac.Reset()

	assert.Nil(ac.FromDevice)
	assert.Nil(ac.ToDevice)
}

func TestAlertChannel_Send(t *testing.T) {
	assert := assert.New(t)

	ac := &AlertChannel{}
	err := ac.Send(true)
	assert.Equal(ErrChannelFull, err)

	err = ac.Send(false)
	assert.Equal(ErrChannelFull, err)
}

func TestAlertChannel_Receive(t *testing.T) {
	ac := &AlertChannel{}
	seq := ac.Receive()

	if seq != nil {
		for range seq {
			t.Fatal("Expected no items from AlertChannel.Receive()")
		}
	}
}

func TestAlertChannel_Alert(t *testing.T) {
	assert := assert.New(t)

	ac := &AlertChannel{}
	ac.Alert(42)

	assert.Len(ac.ToDevice, 1)
	assert.Equal(uint32(42), ac.ToDevice[0])
}

func TestAlertChannel_Await(t *testing.T) {
	assert := assert.New(t)

	ac := &AlertChannel{}

	// Empty queue
	value, ok := ac.Await()
	assert.False(ok)
	assert.Equal(uint32(0), value)

	// With items
	ac.FromDevice = []uint32{10, 20, 30}
	value, ok = ac.Await()
	assert.True(ok)
	assert.Equal(uint32(10), value)
	assert.Len(ac.FromDevice, 2)

	value, ok = ac.Await()
	assert.True(ok)
	assert.Equal(uint32(20), value)
	assert.Len(ac.FromDevice, 1)
}

func TestAlertChannel_GetAlert(t *testing.T) {
	assert := assert.New(t)

	ac := &AlertChannel{}

	// Empty queue
	alert, ok := ac.GetAlert()
	assert.False(ok)
	assert.Equal(uint32(0), alert)

	// With items
	ac.ToDevice = []uint32{100, 200}
	alert, ok = ac.GetAlert()
	assert.True(ok)
	assert.Equal(uint32(100), alert)
	assert.Len(ac.ToDevice, 1)

	alert, ok = ac.GetAlert()
	assert.True(ok)
	assert.Equal(uint32(200), alert)
	assert.Len(ac.ToDevice, 0)
}

func TestAlertChannel_SetAlert(t *testing.T) {
	assert := assert.New(t)

	ac := &AlertChannel{}
	ac.SetAlert(77)
	ac.SetAlert(88)

	assert.Len(ac.ToDevice, 2)
	assert.Equal(uint32(77), ac.ToDevice[0])
	assert.Equal(uint32(88), ac.ToDevice[1])
}

func TestAlertChannel_SendAwait(t *testing.T) {
	assert := assert.New(t)

	ac := &AlertChannel{}
	ac.SendAwait(55)
	ac.SendAwait(66)

	assert.Len(ac.FromDevice, 2)
	assert.Equal(uint32(55), ac.FromDevice[0])
	assert.Equal(uint32(66), ac.FromDevice[1])
}
