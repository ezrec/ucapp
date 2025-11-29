package emulator

import (
	"github.com/ezrec/ucapp/translate"
)

var f = translate.From

// ErrRuntime indicates the location of a runtime error.
type ErrRuntime struct {
	LineNo int
	Err    error
}

func (err *ErrRuntime) Error() string {
	return f("line %d %v", err.LineNo, err.Err)
}

func (err *ErrRuntime) Unwrap() error {
	return err.Err
}
