package io

import (
	"errors"

	"github.com/ezrec/ucapp/translate"
)

var f = translate.From

// ErrChannelFull is returned when attempting to write to a full channel.
var ErrChannelFull = errors.New(f("channel full"))

// ErrDrumMissing is returned when attempting to access a drum that doesn't exist.
var ErrDrumMissing = errors.New(f("drum missing"))
