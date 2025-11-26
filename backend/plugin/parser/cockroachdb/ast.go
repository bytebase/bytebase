package cockroachdb

import (
	"github.com/cockroachdb/cockroachdb-parser/pkg/sql/parser/statements"
	"github.com/cockroachdb/cockroachdb-parser/pkg/sql/sem/tree"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// AST is the AST implementation for CockroachDB parser.
// It implements the base.AST interface.
type AST struct {
	BaseLine int
	Stmt     statements.Statement[tree.Statement]
}

// GetBaseLine implements base.AST interface.
func (a *AST) GetBaseLine() int {
	return a.BaseLine
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
