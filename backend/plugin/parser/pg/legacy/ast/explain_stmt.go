package ast

// ExplainStmt is the struct for explain statement.
type ExplainStmt struct {
	node

	// For PostgreSQL, the Statement can be SELECT, INSERT, UPDATE, DELETE, VALUES, EXECUTE, DECLARE, CREATE TABLE AS, or CREATE MATERIALIZED VIEW AS statement.
	// Here we only support SELECT now.
	// TODO(rebelice): support more.
	Statement Node
	Analyze   bool
}
