package parser

import "fmt"

// ConvertError is the conversion error.
type ConvertError struct {
	err error
}

// Error implements the error interface
func (e *ConvertError) Error() string {
	return e.err.Error()
}

// NewConvertErrorf returns the new conversion error.
func NewConvertErrorf(format string, a ...interface{}) *ConvertError {
	return &ConvertError{err: fmt.Errorf(format, a...)}
}
