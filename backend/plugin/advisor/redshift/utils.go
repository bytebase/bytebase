package redshift

import (
	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
)

// getANTLRTree extracts a single ANTLR tree from the advisor context.
// Redshift advisors use a single parse tree instead of a list of ParseResults.
func getANTLRTree(checkCtx advisor.Context) (antlr.Tree, error) {
	if checkCtx.AST == nil {
		return nil, errors.New("AST is not provided in context")
	}
	if len(checkCtx.AST) == 0 {
		return nil, errors.New("AST is empty")
	}
	// Redshift typically processes a single statement
	unifiedAST := checkCtx.AST[0]
	antlrData, ok := unifiedAST.GetANTLRTree()
	if !ok {
		return nil, errors.New("AST type mismatch: expected ANTLR-based parser result")
	}
	return antlrData.Tree, nil
}
