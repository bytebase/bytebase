package ast

// CreateIndexStmt is the struct for create index statement.
type CreateIndexStmt struct {
	ddl

	Index *IndexDef
}
