package common

import (
	"errors"

	pkgerrors "github.com/pkg/errors"
)

// Code is the error code.
type Code int

// Application error codes.
const (
	// 0 ~ 99 general error.
	Ok             Code = 0
	Internal       Code = 1
	NotAuthorized  Code = 2
	Invalid        Code = 3
	NotFound       Code = 4
	Conflict       Code = 5
	NotImplemented Code = 6

	// 101 ~ 199 db error.
	DbConnectionFailure Code = 101
	DbExecutionError    Code = 102

	// 201 db migration error
	// Db migration is a core feature, so we separate it from the db error.
	MigrationSchemaMissing  Code = 201
	MigrationAlreadyApplied Code = 202
	MigrationOutOfOrder     Code = 203
	// MigrationBaselineMissing is no longer used.
	// MigrationBaselineMissing Code = 204.
	MigrationPending Code = 205
	MigrationFailed  Code = 206

	// 301 task error.
	TaskTimingNotAllowed Code = 301

	// 401 task sql type error.
	TaskTypeNotDML         Code = 401
	TaskTypeNotDDL         Code = 402
	TaskTypeDropDatabase   Code = 403
	TaskTypeCreateDatabase Code = 404
	TaskTypeDropTable      Code = 405
	TaskTypeDropIndex      Code = 406
	TaskTypeDropColumn     Code = 407
	TaskTypeDropPrimaryKey Code = 408
	TaskTypeDropForeignKey Code = 409
	TaskTypeDropCheck      Code = 410
)

// Int returns the int type of code.
func (c Code) Int() int {
	return int(c)
}

// Int64 returns the int64 type of code.
func (c Code) Int64() int64 {
	return int64(c)
}

// Error represents an application-specific error. Application errors can be
// unwrapped by the caller to extract out the code & message.
//
// Any non-application error (such as a disk error) should be reported as an
// Internal error and the human user should only see "Internal error" as the
// message. These low-level internal error details should only be logged and
// reported to the operator of the application (not the end user).
type Error struct {
	// Machine-readable error code.
	Code Code

	// Embedded error.
	Err error
}

// Error implements the error interface. Not used by the application otherwise.
func (e *Error) Error() string {
	return e.Err.Error()
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
		return e.Err.Error()
	}
	return "Internal error."
}

// Wrapf is a helper function to wrap an Error with given code and formatted message.
func Wrapf(err error, code Code, format string, args ...any) *Error {
	return &Error{
		Code: code,
		Err:  pkgerrors.Wrapf(err, format, args...),
	}
}

// Errorf is a helper function to create an Error with given code and formatted message.
func Errorf(code Code, format string, args ...any) *Error {
	return &Error{
		Code: code,
		Err:  pkgerrors.Errorf(format, args...),
	}
}

// Wrap is a helper function to wrap an Error with given code.
func Wrap(err error, code Code) *Error {
	return &Error{
		Code: code,
		Err:  err,
	}
}

// FormatDBErrorEmptyRowWithQuery formats database error that query returns empty row.
func FormatDBErrorEmptyRowWithQuery(query string) error {
	return Errorf(DbExecutionError, "query %q returned empty row", query)
}
