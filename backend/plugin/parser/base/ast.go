package base

import (
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// AST is the interface that all parser AST types must implement.
// Each parser package defines its own concrete AST type with parser-specific fields.
type AST interface {
	// ASTStartPosition returns the 1-based position where this SQL statement starts
	// in the original multi-statement input. Used for error position reporting.
	// Returns nil if position is unknown.
	// Named to avoid collision with protobuf-generated GetStartPosition methods.
	ASTStartPosition() *storepb.Position
}
