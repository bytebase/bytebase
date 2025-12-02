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
