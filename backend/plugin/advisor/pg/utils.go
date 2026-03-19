package pg

import (
	"regexp"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/postgresql"

	"github.com/bytebase/bytebase/backend/plugin/parser/pg"
)

// appendSessionPreExecutionStatements tracks top-level SET statements that affect
// how subsequent DML statements should be planned/executed in the same script.
func appendSessionPreExecutionStatements(preExecutions []string, tokens *antlr.CommonTokenStream, ctx *parser.VariablesetstmtContext) []string {
	if ctx == nil {
		return preExecutions
	}

	if !isRoleOrSearchPathSetStatement(ctx) {
		return preExecutions
	}

	return append(preExecutions, getTextFromTokens(tokens, ctx))
}

func isRoleOrSearchPathSetStatement(ctx *parser.VariablesetstmtContext) bool {
	if ctx == nil {
		return false
	}

	setRest := ctx.Set_rest()
	if setRest == nil {
		return false
	}
	setRestMore := setRest.Set_rest_more()
	if setRestMore == nil {
		return false
	}

	// Covers SET ROLE / SET SESSION ROLE syntax.
	if setRestMore.ROLE() != nil {
		return true
	}

	genericSet := setRestMore.Generic_set()
	if genericSet == nil {
		return false
	}
	varName := genericSet.Var_name()
	if varName == nil || len(varName.AllColid()) != 1 {
		return false
	}

	name := pg.NormalizePostgreSQLColid(varName.Colid(0))
	return strings.EqualFold(name, "role") || strings.EqualFold(name, "search_path")
}

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

// nolint:unused
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
