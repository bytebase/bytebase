package base

import (
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// Statement represents a single SQL statement with both text and parsed AST.
// This is the unified type that combines SingleSQL (text-based) and AST (tree-based).
type Statement struct {
	// Text content
	Text  string
	Empty bool

	// Position tracking (1-based)
	StartPosition *storepb.Position
	EndPosition   *storepb.Position

	// Byte offsets for execution tracking
	ByteOffsetStart int
	ByteOffsetEnd   int

	// Parsed tree (always present after Parse)
	AST AST
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

// SingleSQLToStatement converts a SingleSQL to a Statement without AST.
// The AST field will be nil. Use this for incremental migration.
func SingleSQLToStatement(sql SingleSQL) Statement {
	return Statement{
		Text:            sql.Text,
		Empty:           sql.Empty,
		StartPosition:   sql.Start,
		EndPosition:     sql.End,
		ByteOffsetStart: sql.ByteOffsetStart,
		ByteOffsetEnd:   sql.ByteOffsetEnd,
		AST:             nil,
	}
}

// SingleSQLsToStatements converts a slice of SingleSQL to Statements without AST.
func SingleSQLsToStatements(sqls []SingleSQL) []Statement {
	result := make([]Statement, len(sqls))
	for i, sql := range sqls {
		result[i] = SingleSQLToStatement(sql)
	}
	return result
}

// StatementToSingleSQL converts a Statement back to SingleSQL.
// The AST is discarded. Use this for backward compatibility.
func StatementToSingleSQL(stmt Statement) SingleSQL {
	return SingleSQL{
		Text:            stmt.Text,
		Empty:           stmt.Empty,
		Start:           stmt.StartPosition,
		End:             stmt.EndPosition,
		ByteOffsetStart: stmt.ByteOffsetStart,
		ByteOffsetEnd:   stmt.ByteOffsetEnd,
		// Note: BaseLine is not preserved as Statement uses 1-based StartPosition
	}
}
