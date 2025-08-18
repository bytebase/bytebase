package ast

// CreateExtensionStmt is the struct for create extension statement.
type CreateExtensionStmt struct {
	ddl

	Name        string
	Schema      string
	IfNotExists bool
}
