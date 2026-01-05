package base

import (
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// Statement is the result of splitting SQL (text + positions, no AST).
type Statement struct {
	// Text is the SQL text content.
	Text string
	// Empty indicates if the sql is empty, such as `/* comments */;` or just `;`.
	Empty bool

	// Start is the inclusive start position of the SQL in the original SQL (1-based).
	Start *storepb.Position
	// End is the exclusive end position of the SQL in the original SQL (1-based).
	// It points to the position AFTER the last character of the statement.
	End *storepb.Position

	// Range is the byte offset range of the SQL in the original SQL.
	// This field may not be present for every engine.
	// Range is intended for sql execution log display. It may not represent the actual sql that is sent to the database.
	Range *storepb.Range
}

// BaseLine returns the 0-based line number of the first line of the SQL.
// This is derived from Start.Line - 1 and is used for advisor code compatibility.
func (s *Statement) BaseLine() int {
	if s.Start == nil {
		return 0
	}
	return int(s.Start.Line) - 1
}

// ParsedStatement is the result of parsing SQL (Statement + AST).
// AST is guaranteed to be non-nil after successful parsing.
type ParsedStatement struct {
	Statement     // embedded - access fields directly like ps.Text, ps.Start
	AST       AST // always non-nil after parsing
}

// FilterEmptyStatements removes empty statements from the list.
func FilterEmptyStatements(list []Statement) []Statement {
	var result []Statement
	for _, stmt := range list {
		if !stmt.Empty {
			result = append(result, stmt)
		}
	}
	return result
}

// ExtractASTs extracts non-nil ASTs from a slice of ParsedStatements.
// Empty statements (with nil AST) are skipped.
// Returns nil if no ASTs are found (preserves nil-check compatibility).
// This is useful for backward compatibility when migrating from []AST to []ParsedStatement.
func ExtractASTs(stmts []ParsedStatement) []AST {
	var asts []AST
	for _, stmt := range stmts {
		if stmt.AST != nil {
			asts = append(asts, stmt.AST)
		}
	}
	return asts
}
