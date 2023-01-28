package ast

// CreateTriggerStmt is the definition of create trigger statements.
type CreateTriggerStmt struct {
	ddl

	Trigger *TriggerDef
}
