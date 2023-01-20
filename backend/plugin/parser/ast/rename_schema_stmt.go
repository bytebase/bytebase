package ast

// RenameSchemaStmt is the struct for the rename schema statement.
type RenameSchemaStmt struct {
	ddl

	Schema  string
	NewName string
}
