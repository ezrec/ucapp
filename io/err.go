package io

import (
	"errors"

	"github.com/ezrec/ucapp/translate"
)

var f = translate.From

var (
	// Channel errors
	ErrChannelFull = errors.New(f("channel full"))
)
