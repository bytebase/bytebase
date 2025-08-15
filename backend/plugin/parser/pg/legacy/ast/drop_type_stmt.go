package ast

// DropTypeStmt is the struct for DROP TYPE statements.
type DropTypeStmt struct {
	ddl

	IfExists     bool
	Behavior     DropBehavior
	TypeNameList []*TypeNameDef
}
