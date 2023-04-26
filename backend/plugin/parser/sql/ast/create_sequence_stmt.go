package ast

// CreateSequenceStmt is a statement to create a sequence.
// https://www.postgresql.org/docs/13/sql-createsequence.html
type CreateSequenceStmt struct {
	ddl
	SequenceDef SequenceDef
	IfNotExists bool
}
