package snowflake

import (
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// getANTLRTree extracts the ANTLR parse trees from the advisor context.
func getANTLRTree(checkCtx advisor.Context) ([]*base.ParseResult, error) {
	if checkCtx.AST == nil {
		return nil, errors.New("AST is not provided in context")
	}
	var parseResults []*base.ParseResult
	for _, unifiedAST := range checkCtx.AST {
		antlrData, ok := unifiedAST.GetANTLRTree()
		if !ok {
			return nil, errors.Errorf("AST type mismatch: expected ANTLR-based parser result, got engine %s", unifiedAST.GetEngine())
		}
		parseResults = append(parseResults, &base.ParseResult{
			Tree:     antlrData.Tree,
			Tokens:   antlrData.Tokens,
			BaseLine: unifiedAST.GetBaseLine(),
		})
	}
	return parseResults, nil
}
