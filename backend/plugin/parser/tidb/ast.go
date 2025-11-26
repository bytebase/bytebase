package tidb

import (
	tidbast "github.com/pingcap/tidb/pkg/parser/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// AST is the AST implementation for TiDB parser.
// It implements the base.AST interface.
type AST struct {
	// StartPosition is the 1-based position where this statement starts.
	StartPosition *storepb.Position
	Node          tidbast.StmtNode
}

// ASTStartPosition implements base.AST interface.
func (a *AST) ASTStartPosition() *storepb.Position {
	return a.StartPosition
}

// GetTiDBAST extracts the TiDB AST from a base.AST.
// Returns the AST and true if it is a TiDB AST, nil and false otherwise.
func GetTiDBAST(a base.AST) (*AST, bool) {
	if a == nil {
		return nil, false
	}
	tidbAST, ok := a.(*AST)
	return tidbAST, ok
}
