package cockroachdb

import (
	"github.com/bytebase/omni/cockroachdb/analysis"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// AST is the AST implementation for CockroachDB parser.
// It implements the base.AST interface.
type AST struct {
	// StartPosition is the 1-based position where this statement starts.
	StartPosition *storepb.Position
	Stmt          analysis.ParsedStatement
}

// ASTStartPosition implements base.AST interface.
func (a *AST) ASTStartPosition() *storepb.Position {
	return a.StartPosition
}
