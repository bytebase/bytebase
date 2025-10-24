package pgantlr

import (
	"strconv"

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

// extractTableName extracts the table name (last component) from a qualified name.
// Handles both "schema.table" and "table" formats.
func extractTableName(ctx parser.IQualified_nameContext) string {
	parts := pg.NormalizePostgreSQLQualifiedName(ctx)
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
}

// extractIntegerConstant extracts an integer value from an Iconst context.
// Returns the integer value and an error if parsing fails.
func extractIntegerConstant(ctx parser.IIconstContext) (int, error) {
	if ctx == nil {
		return 0, errors.New("iconst context is nil")
	}
	text := ctx.GetText()
	val, err := strconv.Atoi(text)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to parse integer constant: %s", text)
	}
	return val, nil
}

// extractStringConstant extracts a string value from an Sconst context.
// Removes surrounding quotes and handles basic escape sequences.
func extractStringConstant(ctx parser.ISconstContext) string {
	if ctx == nil {
		return ""
	}

	text := ctx.GetText()
	// Remove surrounding single quotes
	if len(text) >= 2 && text[0] == '\'' && text[len(text)-1] == '\'' {
		return text[1 : len(text)-1]
	}
	return text
}
