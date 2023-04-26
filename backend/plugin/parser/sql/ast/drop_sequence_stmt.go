package ast

// DropSequenceStmt is the struct for drop sequence statement.
type DropSequenceStmt struct {
	ddl

	IfExists         bool
	SequenceNameList []*SequenceNameDef
	Behavior         DropBehavior
}
