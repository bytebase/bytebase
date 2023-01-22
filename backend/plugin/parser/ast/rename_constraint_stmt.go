package ast

// RenameConstraintStmt is the struct for rename constraint statement.
type RenameConstraintStmt struct {
	node

	Table          *TableDef
	ConstraintName string
	NewName        string
}
