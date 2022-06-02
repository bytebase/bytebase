package common

import (
	"errors"
)

// Code is the error code.
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

	// 201 db migration error
	// Db migration is a core feature, so we separate it from the db error
	MigrationSchemaMissing   Code = 201
	MigrationAlreadyApplied  Code = 202
	MigrationOutOfOrder      Code = 203
	MigrationBaselineMissing Code = 204
	MigrationPending         Code = 205
	MigrationFailed          Code = 206

	// 301 task error
	TaskTimingNotAllowed Code = 301

	// 401 task check error
	TaskCheckEmptySchemaReviewPolicy Code = 401

	// 10001 advisor error code
	CompatibilityDropDatabase  Code = 10001
	CompatibilityRenameTable   Code = 10002
	CompatibilityDropTable     Code = 10003
	CompatibilityRenameColumn  Code = 10004
	CompatibilityDropColumn    Code = 10005
	CompatibilityAddPrimaryKey Code = 10006
	CompatibilityAddUniqueKey  Code = 10007
	CompatibilityAddForeignKey Code = 10008
	CompatibilityAddCheck      Code = 10009
	CompatibilityAlterCheck    Code = 10010
	CompatibilityAlterColumn   Code = 10011

	StatementNoWhere             Code = 10101
	StatementSelectAll           Code = 10102
	StatementLeadingWildcardLike Code = 10103

	// 10201 table naming advisor error code
	NamingTableConventionMismatch Code = 10201
	// 10202 column naming advisor error code
	NamingColumnConventionMismatch Code = 10202
	// 10203 index naming advisor error code
	NamingIndexConventionMismatch Code = 10203
	// 10204 unique key naming advisor error code
	NamingUKConventionMismatch Code = 10204
	// 10205 foreign key naming advisor error code
	NamingFKConventionMismatch Code = 10205

	// 10301 column rule advisor error code
	NoRequiredColumn Code = 10301
	ColumnCanNotNull Code = 10302

	NotInnoDBEngine Code = 10401

	// 10501 table rule advisor error code
	TableNoPK Code = 10501
)

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

// Errorf is a helper function to return an Error with a given code and error.
func Errorf(code Code, err error) *Error {
	return &Error{
		Code: code,
		Err:  err,
	}
}
