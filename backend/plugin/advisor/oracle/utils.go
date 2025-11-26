package oracle

import (
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// getANTLRTree extracts PL/SQL ANTLR parse trees from the advisor context.
func getANTLRTree(checkCtx advisor.Context) ([]*base.ParseResult, error) {
	if checkCtx.AST == nil {
		return nil, errors.New("AST is not provided in context")
	}
	var parseResults []*base.ParseResult
	for _, unifiedAST := range checkCtx.AST {
		antlrAST, ok := base.GetANTLRAST(unifiedAST)
		if !ok {
			return nil, errors.New("AST type mismatch: expected ANTLR-based parser result")
		}
		parseResults = append(parseResults, &base.ParseResult{
			Tree:     antlrAST.Tree,
			Tokens:   antlrAST.Tokens,
			BaseLine: antlrAST.BaseLine,
		})
	}
	return parseResults, nil
}
