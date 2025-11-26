package cockroachdb

import (
	"github.com/cockroachdb/cockroachdb-parser/pkg/sql/parser/statements"
	"github.com/cockroachdb/cockroachdb-parser/pkg/sql/sem/tree"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// AST is the AST implementation for CockroachDB parser.
// It implements the base.AST interface.
type AST struct {
	// StartPosition is the 1-based position where this statement starts.
	StartPosition *storepb.Position
	Stmt          statements.Statement[tree.Statement]
}

// ASTStartPosition implements base.AST interface.
func (a *AST) ASTStartPosition() *storepb.Position {
	return a.StartPosition
}

// GetCockroachDBAST extracts the CockroachDB AST from a base.AST.
// Returns the AST and true if it is a CockroachDB AST, nil and false otherwise.
func GetCockroachDBAST(a base.AST) (*AST, bool) {
	if a == nil {
		return nil, false
	}
	crAST, ok := a.(*AST)
	return crAST, ok
}
