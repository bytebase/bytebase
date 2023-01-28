package ast

// ReferentialActionType is the type for referential action.
type ReferentialActionType int

const (
	// ReferentialActionTypeNoAction is the referential action type which will produce an error indicating that the deletion or update would create a foreign key constraint violation.
	// If the constraint is deferred, this error will be produced at constraint check time if there still exist any referencing rows. This is the default action.
	ReferentialActionTypeNoAction ReferentialActionType = iota
	// ReferentialActionTypeRestrict is the referential action type which will produce an error indicating that the deletion or update would create a foreign key constraint violation.
	// This is the same as NO ACTION except that the check is not deferrable.
	ReferentialActionTypeRestrict
	// ReferentialActionTypeCascade is the referential action type which will delete any rows referencing the deleted row, or update the values of the referencing column(s) to the new values of the referenced columns, respectively.
	ReferentialActionTypeCascade
	// ReferentialActionTypeSetNull is the referential action type which will set all of the referencing columns, or a specified subset of the referencing columns(pg 15), to null. A subset of columns can only be specified for ON DELETE actions.
	ReferentialActionTypeSetNull
	// ReferentialActionTypeSetDefault is the referential action type which will set all of the referencing columns, or a specified subset of the referencing columns(pg 15), to their default values.
	// A subset of columns can only be specified for ON DELETE actions.
	// There must be a row in the referenced table matching the default values, if they are not null, or the operation will fail.
	ReferentialActionTypeSetDefault
)

// ReferentialActionDef is the struct for referential actions.
// See https://www.postgresql.org/docs/current/sql-createtable.html for more details.
//
//	In the PostgreSQL 15, users can specify the subset of the referencing columns to set null or default.
//	This parser only support PostgreSQL 13 currently. But we also define the ReferentialAction as a struct instead of an int.
//	It's for supporting PostgreSQL 15 in the future.
type ReferentialActionDef struct {
	node

	Type ReferentialActionType
}
