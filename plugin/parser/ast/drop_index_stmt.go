package ast

// DropIndexStmt is the struct for drop index statement.
type DropIndexStmt struct {
	ddl

	// Here use IndexDef because the drop index statement needs the schema name for PostgreSQL.
	// If the drop index statement doesn't contain schema name, the Table of this index is nil.
	IndexList []*IndexDef
}
