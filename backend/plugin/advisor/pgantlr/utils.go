package pgantlr

import (
	"strconv"
	"strings"

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

// Normalization helper functions for PostgreSQL identifiers.
// These functions wrap the parser's normalization utilities to provide
// consistent handling of quoted/unquoted identifiers across all advisors.

// normalizeColid normalizes a column identifier.
// Unquoted identifiers are lowercased, quoted identifiers preserve case.
func normalizeColid(ctx parser.IColidContext) string {
	return pg.NormalizePostgreSQLColid(ctx)
}

// normalizeName normalizes a generic name.
func normalizeName(ctx parser.INameContext) string {
	return pg.NormalizePostgreSQLName(ctx)
}

// normalizeQualifiedName normalizes a qualified name and returns components.
// Returns a slice like [schema, table] or [table] depending on the qualified name.
func normalizeQualifiedName(ctx parser.IQualified_nameContext) []string {
	return pg.NormalizePostgreSQLQualifiedName(ctx)
}

// normalizeAnyName normalizes an any_name and returns components.
func normalizeAnyName(ctx parser.IAny_nameContext) []string {
	return pg.NormalizePostgreSQLAnyName(ctx)
}

// extractTableName extracts the table name (last component) from a qualified name.
// Handles both "schema.table" and "table" formats.
func extractTableName(ctx parser.IQualified_nameContext) string {
	parts := normalizeQualifiedName(ctx)
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
}

// normalizeTypeName normalizes a type name for comparison.
// Returns lowercase version to enable case-insensitive comparison.
func normalizeTypeName(typename string) string {
	return strings.ToLower(typename)
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
