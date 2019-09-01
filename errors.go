package cwrap

import (
	"fmt"

	"golang.org/x/xerrors"
)

type Error struct {
	err   error
	msg   string
	frame xerrors.Frame
}

func Wrapf(err error, format string, v ...interface{}) error {
	if err == nil {
		return nil
	}
	return &Error{
		err:   err,
		msg:   fmt.Sprintf(format, v...),
		frame: xerrors.Caller(1),
	}
}

func Wrap(err error) error {
	if err == nil {
		return nil
	}
	return &Error{
		err:   err,
		frame: xerrors.Caller(1),
	}
}

func (e *Error) Error() string {
	return e.err.Error()
}

// Unwrap implements xerrors.Wrapper
func (e *Error) Unwrap() error {
	return e.err
}

// FormatError implements xerrors.Formatter
func (e *Error) FormatError(p xerrors.Printer) error {
	p.Print(e.err)
	p.Print(": ", e.msg)
	e.frame.Format(p)
	return nil
}

// Format implements fmt.Formatter
func (e *Error) Format(f fmt.State, verb rune) {
	xerrors.FormatError(e, f, verb)
}
