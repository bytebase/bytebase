package pgantlr

import (
	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/postgresql"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/parser/pg"
)

// isTopLevel checks if the context is at the top level of the parse tree.
// Top level contexts are: RootContext, StmtblockContext, StmtmultiContext, or StmtContext.
func isTopLevel(ctx antlr.Tree) bool {
	if ctx == nil {
		return true
	}

	switch ctx := ctx.(type) {
	case *parser.RootContext, *parser.StmtblockContext:
		return true
	case *parser.StmtmultiContext, *parser.StmtContext:
		return isTopLevel(ctx.GetParent())
	default:
		return false
	}
}

// splitIdentifier splits a qualified identifier by dots, handling quoted parts.
// For example:
//   - "public.table" -> ["public", "table"]
//   - "\"public\".\"table\"" -> ["public", "table"]
//   - "table" -> ["table"]
func splitIdentifier(s string) []string {
	var parts []string
	var current string
	inQuote := false

	for i := 0; i < len(s); i++ {
		ch := s[i]
		if ch == '"' {
			inQuote = !inQuote
		} else if ch == '.' && !inQuote {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(ch)
		}
	}

	if current != "" {
		parts = append(parts, current)
	}

	return parts
}

// getANTLRTree extracts the ANTLR parse tree from the advisor context.
// The AST must be pre-parsed and passed via checkCtx.AST (e.g., in tests or by the framework).
// This enforces proper AST caching and makes any missing cache obvious.
func getANTLRTree(checkCtx advisor.Context) (*pg.ParseResult, error) {
	if checkCtx.AST == nil {
		return nil, errors.New("AST is not provided in context - must be parsed before calling advisor")
	}

	parseResult, ok := checkCtx.AST.(*pg.ParseResult)
	if !ok {
		return nil, errors.Errorf("AST type mismatch: expected *pg.ParseResult, got %T", checkCtx.AST)
	}

	return parseResult, nil
}
