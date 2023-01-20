package ast

// SubqueryDef is the struct for subquery definition.
type SubqueryDef struct {
	expression

	Select *SelectStmt
}
