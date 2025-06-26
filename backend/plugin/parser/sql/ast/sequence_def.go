package ast

// SequenceNameDef is the name of a sequence.
type SequenceNameDef struct {
	node

	Schema string
	Name   string
}

// SequenceDef is the struct for sequence definition.
type SequenceDef struct {
	node

	SequenceName     *SequenceNameDef
	SequenceDataType *Integer
	// No cycle is default
	Cycle       bool
	StartWith   *int64
	IncrementBy *int64
	MinValue    *int64
	MaxValue    *int64
	Cache       *int64
	OwnedBy     *ColumnNameDef
}
