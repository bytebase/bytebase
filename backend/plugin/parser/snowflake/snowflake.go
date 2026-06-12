package snowflake

import (
	"strings"

	omniast "github.com/bytebase/omni/snowflake/ast"
	omniparser "github.com/bytebase/omni/snowflake/parser"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterParseStatementsFunc(storepb.Engine_SNOWFLAKE, parseSnowflakeStatements)
}

// parseSnowflakeStatements is the ParseStatementsFunc for Snowflake.
// Returns []ParsedStatement with both text and AST populated.
//
// Each non-empty statement is parsed with the omni parser (the legacy ANTLR
// parser is gone). A parse failure on any statement fails the whole batch
// with a *base.SyntaxError — exactly the contract the legacy path had, so
// callers (e.g. the sheet manager) keep surfacing syntax errors unchanged.
// The omni side is parsed PER STATEMENT (stmt.Text) rather than from the
// whole input, so positions inside one statement stay relative to it and are
// offset by the statement's start position afterwards.
func parseSnowflakeStatements(statement string) ([]base.ParsedStatement, error) {
	// First split to get Statement with text and positions
	stmts, err := SplitSQL(statement)
	if err != nil {
		return nil, err
	}

	var result []base.ParsedStatement
	for _, stmt := range stmts {
		ps := base.ParsedStatement{
			Statement: stmt,
		}
		if !stmt.Empty {
			node, err := parseOmniStatementNode(stmt.Text)
			if err != nil {
				return nil, convertOmniParseError(err, stmt)
			}
			ps.AST = &OmniAST{
				Node: node,
				Text: stmt.Text,
				// Keep the legacy ASTStartPosition shape: 1-based line of the
				// statement start, no column component.
				StartPosition: &storepb.Position{Line: int32(stmt.BaseLine()) + 1},
			}
		}
		result = append(result, ps)
	}

	return result, nil
}

// IsSnowflakeKeyword returns true if the given string is a snowflake keyword.
// Follows https://docs.snowflake.com/en/sql-reference/reserved-keywords.
func IsSnowflakeKeyword(s string, caseSensitive bool) bool {
	if !caseSensitive {
		s = strings.ToUpper(s)
	}
	return snowflakeKeyword[s]
}

// ---------------------------------------------------------------------------
// omni-AST-based normalization helpers
//
// These replace the deleted legacy ANTLR helpers (NormalizeSnowSQLObjectName /
// NormalizeSnowSQLObjectNamePart / NormalizeSnowSQLSchemaName), operating on
// omni's hand-written AST nodes (*omniast.ObjectName, omniast.Ident) instead of
// ANTLR contexts while applying the SAME Snowflake identifier-folding rules
// (unquoted → uppercase, quoted → verbatim). They are the shared base the
// query-span extractor and the advisors build on.
// ---------------------------------------------------------------------------

// parseSnowflakeAST parses a single Snowflake statement with the omni parser and
// returns the resulting *ast.File. It mirrors trino.parseTrinoSQL: a thin
// wrapper over omniparser.Parse so call sites share one parse entry point.
func parseSnowflakeAST(statement string) (*omniast.File, error) {
	return omniparser.Parse(statement)
}

// normalizeSnowflakeIdentifier returns the canonical (folded) form of an omni
// Ident, applying the Snowflake identifier-folding rules: an unquoted identifier
// is upper-cased; a quoted identifier is returned verbatim (case preserved, the
// surrounding quotes already stripped by the lexer). It delegates to omni's own
// Ident.Normalize so the folding stays in lock-step with the parser.
func normalizeSnowflakeIdentifier(id omniast.Ident) string {
	if id.IsEmpty() {
		return ""
	}
	return id.Normalize()
}

// normalizeSnowflakeObjectName splits an omni *ObjectName into its normalized
// (database, schema, table) parts, filling in the supplied fallbacks for any
// part the name omits. It is the omni-AST analog of the extractor's
// normalizedObjectName helper (which took parser.IObject_nameContext). An empty
// part in the source leaves the corresponding fallback in place.
func normalizeSnowflakeObjectName(objectName *omniast.ObjectName, fallbackDatabaseName, fallbackSchemaName string) (database, schema, table string) {
	if objectName == nil {
		return "", "", ""
	}

	database = fallbackDatabaseName
	if d := normalizeSnowflakeIdentifier(objectName.Database); d != "" {
		database = d
	}

	schema = fallbackSchemaName
	if s := normalizeSnowflakeIdentifier(objectName.Schema); s != "" {
		schema = s
	}

	table = normalizeSnowflakeIdentifier(objectName.Name)
	return database, schema, table
}
