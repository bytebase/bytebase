package snowflake

import (
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// getANTLRTree extracts the ANTLR parse trees from the advisor context.
// The AST must be pre-parsed and passed via checkCtx.AST (e.g., in tests or by the framework).
// This enforces proper AST caching and makes any missing cache obvious.
// Returns all parse results for multi-statement SQL review.
func getANTLRTree(checkCtx advisor.Context) ([]*base.ParseResult, error) {
	if checkCtx.AST == nil {
		return nil, errors.New("AST is not provided in context - must be parsed before calling advisor")
	}

	parseResults, ok := checkCtx.AST.([]*base.ParseResult)
	if !ok {
		return nil, errors.Errorf("AST type mismatch: expected []*base.ParseResult, got %T", checkCtx.AST)
	}

	return parseResults, nil
}
