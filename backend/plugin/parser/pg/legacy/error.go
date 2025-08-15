package pg

import "github.com/pkg/errors"

// ConvertError is the conversion error.
// Because this module is still in its early stages and lacks sufficient testing to ensure compatibility,
// we add ConvertError to differentiate between syntax errors and conversion errors.
// We will remove it in the future.
// TODO(rebelice): remove it.
type ConvertError struct {
	err error
}

// Error implements the error interface.
func (e *ConvertError) Error() string {
	return e.err.Error()
}

// NewConvertErrorf returns the new conversion error.
func NewConvertErrorf(format string, a ...any) *ConvertError {
	return &ConvertError{err: errors.Errorf(format, a...)}
}
