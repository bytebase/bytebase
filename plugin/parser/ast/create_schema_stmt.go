package ast

// CreateSchemaStmt is the struct for create schema statement.
type CreateSchemaStmt struct {
	ddl

	Name           string
	IfNotExists    bool
	RoleSpec       *RoleSpec
	SchemaElements []Node
}
