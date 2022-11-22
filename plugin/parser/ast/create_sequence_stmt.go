package ast

// SequenceNameDef is the name of a sequence.
type SequenceNameDef struct {
	Schema string
	Name   string
}

// CreateSequenceStmt is a statement to create a sequence.
// https://www.postgresql.org/docs/13/sql-createsequence.html
type CreateSequenceStmt struct {
	ddl

	SequenceName     SequenceNameDef
	IfNotExists      bool
	SequenceDataType *Integer
	// No cycle is default
	Cycle       bool
	StartWith   *int32
	IncrementBy *int32
	MinValue    *int32
	MaxValue    *int32
	Cache       *int32
	OwnedBy     *ColumnNameDef
}
