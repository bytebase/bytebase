package ast

// SequenceDataType is the type of a sequence data.
type SequenceDataType int

const (
	// SequenceDataTypeUnknown is the unknown sequence data type.
	SequenceDataTypeUnknown SequenceDataType = iota
	// SequenceDataTypeSmallInt is the smallint sequence data type.
	SequenceDataTypeSmallInt
	// SequenceDataTypeInteger is the integer sequence data type.
	SequenceDataTypeInteger
	// SequenceDataTypeBigInt is the bigint sequence data type.
	SequenceDataTypeBigInt
)

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
	SequenceDataType SequenceDataType
	// No cycle is default
	Cycle       bool
	StartWith   *int32
	IncrementBy *int32
	MinValue    *int32
	MaxValue    *int32
	Cache       *int32
	OwnedBy     *ColumnNameDef
}
