package ast

// CreateIndexStmt is the struct for create index statement.
// TODO(rebelice): fully support CREATE INDEX statements.
// Currently, only support:
// ```
// CREATE [ UNIQUE ] INDEX [ CONCURRENTLY ] [ [ IF NOT EXISTS ] name ] ON table_name [ USING method ]
// ( { column_name | ( expression ) } [ ASC | DESC ] [ NULLS { FIRST | LAST } ] [, ...] )
// ```.
type CreateIndexStmt struct {
	ddl

	Index        *IndexDef
	IfNotExists  bool
	Concurrently bool
}
