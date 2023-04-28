package ast

// AlterSequenceStmt is the struct for alter sequence statements.
type AlterSequenceStmt struct {
	ddl

	Name        *SequenceNameDef
	IfExists    bool
	Type        *Integer
	IncrementBy *int32
	NoMinValue  bool
	MinValue    *int32
	NoMaxValue  bool
	MaxValue    *int32
	StartWith   *int32
	RestartWith *int32
	Cache       *int32
	Cycle       *bool
	OwnedByNone bool
	OwnedBy     *ColumnNameDef
}
