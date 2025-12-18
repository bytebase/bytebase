package oceanbase

import (
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// getANTLRTree extracts MySQL ANTLR parse trees from the advisor context.
// OceanBase uses the MySQL parser.
func getANTLRTree(checkCtx advisor.Context) ([]*base.ParseResult, error) {
	if checkCtx.ParsedStatements == nil {
		return nil, errors.New("ParsedStatements is not provided in context")
	}

	var parseResults []*base.ParseResult
	for _, stmt := range checkCtx.ParsedStatements {
		if stmt.AST == nil {
			continue
		}
		antlrAST, ok := base.GetANTLRAST(stmt.AST)
		if !ok {
			return nil, errors.New("AST type mismatch: expected ANTLR-based parser result")
		}
		parseResults = append(parseResults, &base.ParseResult{
			Tree:     antlrAST.Tree,
			Tokens:   antlrAST.Tokens,
			BaseLine: stmt.BaseLine,
		})
	}
	return parseResults, nil
}
