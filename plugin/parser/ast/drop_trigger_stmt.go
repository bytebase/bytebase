package ast

// DropTriggerStmt is the definition of drop trigger statements.
type DropTriggerStmt struct {
	ddl

	Trigger  *TriggerDef
	IfExists bool
	Behavior DropBehavior
}
