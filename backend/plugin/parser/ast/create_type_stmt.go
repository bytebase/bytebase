package ast

// CreateTypeStmt is the struct for CREATE TYPE statements.
type CreateTypeStmt struct {
	ddl

	Type UserDefinedType
}
