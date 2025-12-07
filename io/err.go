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

// ErrNameRuneInvalid is returned when the dirent name has invalid characters.
var ErrNameRuneInvalid = errors.New(f("name has an unknown rune"))

// ErrNameTooShort is returned when the dirent name is empty.
var ErrNameTooShort = errors.New(f("name length too short"))

// ErrNameTooLong is returned when the name is too long.
var ErrNameTooLong = errors.New(f("name length too long"))
