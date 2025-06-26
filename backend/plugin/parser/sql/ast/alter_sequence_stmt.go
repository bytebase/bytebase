package ast

// AlterSequenceStmt is the struct for alter sequence statements.
type AlterSequenceStmt struct {
	ddl

	Name        *SequenceNameDef
	IfExists    bool
	Type        *Integer
	IncrementBy *int64
	NoMinValue  bool
	MinValue    *int64
	NoMaxValue  bool
	MaxValue    *int64
	StartWith   *int64
	RestartWith *int64
	Cache       *int64
	Cycle       *bool
	OwnedByNone bool
	OwnedBy     *ColumnNameDef
}
