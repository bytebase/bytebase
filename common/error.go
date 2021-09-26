package common

import (
	"errors"
	"fmt"
)

type Code int

// Application error codes.
const (
	// 0 ~ 99 general error
	Ok             Code = 0
	Internal       Code = 1
	NotAuthorized  Code = 2
	Invalid        Code = 3
	NotFound       Code = 4
	Conflict       Code = 5
	NotImplemented Code = 6

	// 101 ~ 199 db error
	DbConnectionFailure    Code = 101
	DbStatementSyntaxError Code = 102
	DbExecutionError       Code = 103
)

// Error represents an application-specific error. Application errors can be
// unwrapped by the caller to extract out the code & message.
//
// Any non-application error (such as a disk error) should be reported as an
// EINTERNAL error and the human user should only see "Internal error" as the
// message. These low-level internal error details should only be logged and
// reported to the operator of the application (not the end user).
type Error struct {
	// Machine-readable error code.
	Code Code

	// Human-readable error message.
	Message string
}

// Error implements the error interface. Not used by the application otherwise.
func (e *Error) Error() string {
	return fmt.Sprintf("bytebase error: code=%d message=%s", e.Code, e.Message)
}

// ErrorCode unwraps an application error and returns its code.
// Non-application errors always return EINTERNAL.
func ErrorCode(err error) Code {
	var e *Error
	if err == nil {
		return Ok
	} else if errors.As(err, &e) {
		return e.Code
	}
	return Internal
}

// ErrorMessage unwraps an application error and returns its message.
// Non-application errors always return "Internal error".
func ErrorMessage(err error) string {
	var e *Error
	if err == nil {
		return ""
	} else if errors.As(err, &e) {
		return e.Message
	}
	return "Internal error."
}

// Errorf is a helper function to return an Error with a given code and formatted message.
func Errorf(code Code, format string, args ...interface{}) *Error {
	return &Error{
		Code:    code,
		Message: fmt.Sprintf(format, args...),
	}
}
