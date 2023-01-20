package ast

// TriggerDef is the definition of trigger.
type TriggerDef struct {
	node

	Name  string
	Table *TableDef
}
