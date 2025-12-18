package pg

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/postgresql"
	"github.com/pkg/errors"

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

// getTextFromTokens extracts the original text for a rule context from the token stream.
// Uses GetTextFromRuleContext to include hidden channel tokens (whitespace, comments).
// Returns clean text without leading/trailing whitespace.
func getTextFromTokens(tokens *antlr.CommonTokenStream, ctx antlr.ParserRuleContext) string {
	if tokens == nil || ctx == nil {
		return ""
	}
	text := tokens.GetTextFromRuleContext(ctx)
	return strings.TrimSpace(text)
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

// extractSchemaName extracts the schema name (first component) from a qualified name.
// Returns empty string if only table name is provided (implying default schema).
func extractSchemaName(ctx parser.IQualified_nameContext) string {
	parts := pg.NormalizePostgreSQLQualifiedName(ctx)
	if len(parts) <= 1 {
		return ""
	}
	return parts[0]
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

// getTemplateRegexp generates a regex pattern by replacing tokens in the template with actual values.
// Used by naming convention advisors to dynamically build patterns based on metadata.
func getTemplateRegexp(template string, templateList []string, tokens map[string]string) (*regexp.Regexp, error) {
	for _, key := range templateList {
		if token, ok := tokens[key]; ok {
			template = strings.ReplaceAll(template, key, token)
		}
	}

	return regexp.Compile(template)
}

// normalizeSchemaName normalizes empty schema names to "public" (PostgreSQL default schema).
func normalizeSchemaName(schemaName string) string {
	if schemaName == "" {
		return "public"
	}
	return schemaName
}
