package pg

import (
	"regexp"
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

// extractStatementText extracts a statement from the full statements text using line numbers.
// Handles multi-line statements by extracting all lines from startLine to endLine.
func extractStatementText(statementsText string, startLine, endLine int) string {
	lines := strings.Split(statementsText, "\n")
	if startLine < 1 || startLine > len(lines) {
		return ""
	}

	// Convert to 0-indexed
	startIdx := startLine - 1
	endIdx := endLine - 1

	if endIdx >= len(lines) {
		endIdx = len(lines) - 1
	}

	// Extract the lines for this statement
	var stmtLines []string
	for i := startIdx; i <= endIdx; i++ {
		stmtLines = append(stmtLines, lines[i])
	}

	return strings.TrimSpace(strings.Join(stmtLines, " "))
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

// makeTableKey creates a table key in format "schema"."table" for use in tracking maps.
// This uses %q formatting to properly quote schema and table names.
func makeTableKey(schema, table string) string {
	if schema == "" {
		schema = "public"
	}
	return strings.Join([]string{strconv.Quote(schema), strconv.Quote(table)}, ".")
}

// parseTableKey parses a table key in format "schema"."table" or "schema.table" back to schema and table.
// Returns (schema, table). Defaults to "public" schema if only table name is present.
func parseTableKey(key string) (string, string) {
	// Handle quoted format: "schema"."table"
	if strings.Contains(key, `"`) {
		parts := make([]string, 0, 2)
		start := 0
		inQuote := false

		for i := 0; i < len(key); i++ {
			if key[i] == '"' {
				inQuote = !inQuote
			} else if key[i] == '.' && !inQuote {
				// Remove quotes from the part
				part := key[start:i]
				if len(part) > 0 && part[0] == '"' && part[len(part)-1] == '"' {
					part = part[1 : len(part)-1]
				}
				parts = append(parts, part)
				start = i + 1
			}
		}

		// Add last part
		if start < len(key) {
			part := key[start:]
			if len(part) > 0 && part[0] == '"' && part[len(part)-1] == '"' {
				part = part[1 : len(part)-1]
			}
			parts = append(parts, part)
		}

		if len(parts) == 2 {
			return parts[0], parts[1]
		}
		return "public", key
	}

	// Handle simple format: schema.table
	for i := 0; i < len(key); i++ {
		if key[i] == '.' {
			return key[:i], key[i+1:]
		}
	}
	return "public", key
}

// getDefaultPKConstraintName returns PostgreSQL's default primary key constraint name.
// PostgreSQL uses the convention: tablename_pkey
func getDefaultPKConstraintName(tableName string) string {
	return tableName + "_pkey"
}
