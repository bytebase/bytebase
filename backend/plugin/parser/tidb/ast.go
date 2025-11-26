package tidb

import (
	tidbast "github.com/pingcap/tidb/pkg/parser/ast"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// AST is the AST implementation for TiDB parser.
// It implements the base.AST interface.
type AST struct {
	BaseLine int
	Node     tidbast.StmtNode
}

// GetBaseLine implements base.AST interface.
func (a *AST) GetBaseLine() int {
	return a.BaseLine
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
