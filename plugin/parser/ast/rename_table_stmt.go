package ast

// RenameTableStmt is the struct for rename table statement.
// For PostgreSQL dialect is ALTER TABLE RENAME.
type RenameTableStmt struct {
	node

	Table   *TableDef
	NewName string
}
