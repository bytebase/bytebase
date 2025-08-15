package ast

// DropConstraintStmt is the struct for drop constraint statement.
type DropConstraintStmt struct {
	node

	Table          *TableDef
	ConstraintName string
	IfExists       bool
}
