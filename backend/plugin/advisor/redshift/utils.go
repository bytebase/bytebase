package redshift

import (
	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// getANTLRTree extracts a single ANTLR tree from the advisor context.
// Redshift advisors use a single parse tree instead of a list of ParseResults.
func getANTLRTree(checkCtx advisor.Context) (antlr.Tree, error) {
	if checkCtx.ParsedStatements == nil {
		return nil, errors.New("ParsedStatements is not provided in context")
	}
	if len(checkCtx.ParsedStatements) == 0 {
		return nil, errors.New("ParsedStatements is empty")
	}
	// Redshift typically processes a single statement
	stmt := checkCtx.ParsedStatements[0]
	if stmt.AST == nil {
		return nil, errors.New("AST is nil in ParsedStatement")
	}
	antlrAST, ok := base.GetANTLRAST(stmt.AST)
	if !ok {
		return nil, errors.New("AST type mismatch: expected ANTLR-based parser result")
	}
	return antlrAST.Tree, nil
}
