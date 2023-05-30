package ast

// SetSchemaStmt is the struct for set schema statement.
type SetSchemaStmt struct {
	node

	Table     *TableDef
	NewSchema string
}
