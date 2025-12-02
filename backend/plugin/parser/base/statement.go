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
