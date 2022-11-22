package ast

// SequenceNameDef is the name of a sequence.
type SequenceNameDef struct {
	Schema string
	Name   string
}

// SequenceDef is the struct for sequence definition.
type SequenceDef struct {
	node

	SequenceName     SequenceNameDef
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
