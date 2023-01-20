package ast

// RenameIndexStmt is the struct for the rename index statement.
type RenameIndexStmt struct {
	ddl

	// For PostgreSQL, we only use TableDef.Schema.
	// If this rename index statement doesn't contain schema name, Table will be not nil and the Schema name is empty string.
	Table     *TableDef
	IndexName string
	NewName   string
}
