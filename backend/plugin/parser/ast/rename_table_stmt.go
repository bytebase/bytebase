package ast

// RenameTableStmt is the struct for rename table or view statement.
// For PostgreSQL dialect is ALTER TABLE RENAME and ALTER VIEW RENAME.
type RenameTableStmt struct {
	node

	Table   *TableDef
	NewName string
}
