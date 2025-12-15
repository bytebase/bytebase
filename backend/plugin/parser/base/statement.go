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

	// BaseLine is the line number of the first line of the SQL in the original SQL.
	// HINT: ZERO based. This is kept for backward compatibility with advisor code.
	BaseLine int

	// Start is the inclusive start position of the SQL in the original SQL (1-based).
	Start *storepb.Position
	// End is the inclusive end position of the SQL in the original SQL (1-based).
	End *storepb.Position

	// ByteOffsetStart is the start byte position of the sql.
	// This field may not be present for every engine.
	// ByteOffsetStart is intended for sql execution log display. It may not represent the actual sql that is sent to the database.
	ByteOffsetStart int
	// ByteOffsetEnd is the end byte position of the sql.
	// This field may not be present for every engine.
	// ByteOffsetEnd is intended for sql execution log display. It may not represent the actual sql that is sent to the database.
	ByteOffsetEnd int
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

// FilterEmptyStatementsWithIndexes removes empty statements and returns original indexes.
func FilterEmptyStatementsWithIndexes(list []Statement) ([]Statement, []int32) {
	var result []Statement
	var originalIndex []int32
	for i, stmt := range list {
		if !stmt.Empty {
			result = append(result, stmt)
			originalIndex = append(originalIndex, int32(i))
		}
	}
	return result, originalIndex
}

// FilterEmptyParsedStatements removes empty parsed statements from the list.
func FilterEmptyParsedStatements(list []ParsedStatement) []ParsedStatement {
	var result []ParsedStatement
	for _, stmt := range list {
		if !stmt.Empty {
			result = append(result, stmt)
		}
	}
	return result
}

// FilterEmptyParsedStatementsWithIndexes removes empty parsed statements and returns original indexes.
func FilterEmptyParsedStatementsWithIndexes(list []ParsedStatement) ([]ParsedStatement, []int32) {
	var result []ParsedStatement
	var originalIndex []int32
	for i, stmt := range list {
		if !stmt.Empty {
			result = append(result, stmt)
			originalIndex = append(originalIndex, int32(i))
		}
	}
	return result, originalIndex
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

// ExtractStatements extracts Statements from a slice of ParsedStatements.
// This is useful when you only need the text/position information without AST.
func ExtractStatements(stmts []ParsedStatement) []Statement {
	result := make([]Statement, len(stmts))
	for i, stmt := range stmts {
		result[i] = stmt.Statement
	}
	return result
}
