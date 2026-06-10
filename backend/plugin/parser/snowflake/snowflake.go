package snowflake

import (
	"strings"
	"unicode"

	"github.com/antlr4-go/antlr/v4"
	omniast "github.com/bytebase/omni/snowflake/ast"
	omniparser "github.com/bytebase/omni/snowflake/parser"
	parser "github.com/bytebase/parser/snowflake"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/utils"
)

func init() {
	base.RegisterParseStatementsFunc(storepb.Engine_SNOWFLAKE, parseSnowflakeStatements)
}

// parseSnowflakeStatements is the ParseStatementsFunc for Snowflake.
// Returns []ParsedStatement with both text and AST populated.
func parseSnowflakeStatements(statement string) ([]base.ParsedStatement, error) {
	// First split to get Statement with text and positions
	stmts, err := SplitSQL(statement)
	if err != nil {
		return nil, err
	}

	// Then parse to get ASTs
	parseResults, err := ParseSnowSQL(statement)
	if err != nil {
		return nil, err
	}

	// Combine: Statement provides text/positions, ANTLRAST provides AST
	var result []base.ParsedStatement
	astIndex := 0
	for _, stmt := range stmts {
		ps := base.ParsedStatement{
			Statement: stmt,
		}
		if !stmt.Empty && astIndex < len(parseResults) {
			ps.AST = parseResults[astIndex]
			astIndex++
		}
		result = append(result, ps)
	}

	return result, nil
}

// ParseSnowSQL parses the given SQL and returns a list of ANTLRAST (one per statement).
// Use the Snowflake parser based on antlr4.
func ParseSnowSQL(sql string) ([]*base.ANTLRAST, error) {
	stmts, err := SplitSQL(sql)
	if err != nil {
		return nil, err
	}

	var results []*base.ANTLRAST
	for _, stmt := range stmts {
		if stmt.Empty {
			continue
		}

		parseResult, err := parseSingleSnowSQL(stmt.Text, stmt.BaseLine())
		if err != nil {
			return nil, err
		}
		results = append(results, parseResult)
	}

	return results, nil
}

// parseSingleSnowSQL parses a single Snowflake statement and returns the ANTLRAST.
func parseSingleSnowSQL(statement string, baseLine int) (*base.ANTLRAST, error) {
	statement = strings.TrimRightFunc(statement, utils.IsSpaceOrSemicolon) + "\n;"
	inputStream := antlr.NewInputStream(statement)
	lexer := parser.NewSnowflakeLexer(inputStream)
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	p := parser.NewSnowflakeParser(stream)

	// Remove default error listener and add our own error listener.
	startPosition := &storepb.Position{Line: int32(baseLine) + 1}
	lexer.RemoveErrorListeners()
	lexerErrorListener := &base.ParseErrorListener{
		Statement:     statement,
		StartPosition: startPosition,
	}
	lexer.AddErrorListener(lexerErrorListener)

	p.RemoveErrorListeners()
	parserErrorListener := &base.ParseErrorListener{
		Statement:     statement,
		StartPosition: startPosition,
	}
	p.AddErrorListener(parserErrorListener)

	p.BuildParseTrees = true

	tree := p.Snowflake_file()

	if lexerErrorListener.Err != nil {
		return nil, lexerErrorListener.Err
	}

	if parserErrorListener.Err != nil {
		return nil, parserErrorListener.Err
	}

	result := &base.ANTLRAST{
		StartPosition: &storepb.Position{Line: int32(baseLine) + 1},
		Tree:          tree,
		Tokens:        stream,
	}

	return result, nil
}

// NormalizeSnowSQLObjectNamePart normalizes the object name part.
func NormalizeSnowSQLObjectNamePart(part parser.IId_Context) string {
	if part == nil {
		return ""
	}
	return ExtractSnowSQLOrdinaryIdentifier(part.GetText())
}

// ExtractSnowSQLOrdinaryIdentifier extracts the ordinary object name from a string. It follows the following rules:
//
// 1. If there are no double quotes on either side, it will be converted to uppercase.
//
// 2. If there are double quotes on both sides, the case will not change, the double quotes on both sides will be removed, and `""` in content will be converted to `"`.
//
// Caller MUST ensure the identifier is VALID.
func ExtractSnowSQLOrdinaryIdentifier(identifier string) string {
	quoted := strings.HasPrefix(identifier, `"`) && strings.HasSuffix(identifier, `"`)
	if quoted {
		identifier = identifier[1 : len(identifier)-1]
	}
	runeObjectName := []rune(identifier)
	var result []rune
	for i := 0; i < len(runeObjectName); i++ {
		newRune := runeObjectName[i]
		if i+1 < len(runeObjectName) && runeObjectName[i] == '"' && runeObjectName[i+1] == '"' && quoted {
			newRune = '"'
			i++
		} else if !quoted {
			newRune = unicode.ToUpper(newRune)
		}
		result = append(result, newRune)
	}
	return string(result)
}

// NormalizeSnowSQLSchemaName normalizes the given schema name.
func NormalizeSnowSQLSchemaName(schemaName parser.ISchema_nameContext, fallbackDatabaseName string) string {
	ids := schemaName.AllId_()

	var parts []string
	database := fallbackDatabaseName
	if len(ids) == 2 {
		normalizedD := NormalizeSnowSQLObjectNamePart(ids[0])
		if normalizedD != "" {
			database = normalizedD
		}
	}
	parts = append(parts, database)

	var schema string
	if len(ids) == 2 {
		normalizedS := NormalizeSnowSQLObjectNamePart(ids[1])
		if normalizedS != "" {
			schema = normalizedS
		}
	} else {
		normalizedS := NormalizeSnowSQLObjectNamePart(ids[0])
		if normalizedS != "" {
			schema = normalizedS
		}
	}
	parts = append(parts, schema)
	return strings.Join(parts, ".")
}

// NormalizeSnowSQLObjectName normalizes the given object name.
func NormalizeSnowSQLObjectName(objectName parser.IObject_nameContext, fallbackDatabaseName, fallbackSchemaName string) string {
	var parts []string

	database := fallbackDatabaseName
	if d := objectName.GetD(); d != nil {
		normalizedD := NormalizeSnowSQLObjectNamePart(d)
		if normalizedD != "" {
			database = normalizedD
		}
	}
	parts = append(parts, database)

	schema := fallbackSchemaName
	if s := objectName.GetS(); s != nil {
		normalizedS := NormalizeSnowSQLObjectNamePart(s)
		if normalizedS != "" {
			schema = normalizedS
		}
	}
	parts = append(parts, schema)

	if o := objectName.GetO(); o != nil {
		normalizedO := NormalizeSnowSQLObjectNamePart(o)
		if normalizedO != "" {
			parts = append(parts, normalizedO)
		}
	}
	return strings.Join(parts, ".")
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
// These mirror the legacy ANTLR helpers (NormalizeSnowSQLObjectName /
// NormalizeSnowSQLObjectNamePart / NormalizeSnowSQLSchemaName) but operate on
// omni's hand-written AST nodes (*omniast.ObjectName, omniast.Ident) instead of
// ANTLR contexts, applying the SAME Snowflake identifier-folding rules
// (unquoted → uppercase, quoted → verbatim). They are the shared base the
// query-span extractor and (next) the advisors build on once they move onto the
// omni parser. The legacy ANTLR helpers above are kept ADDITIVELY because the
// not-yet-migrated advisors still consume them.
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
// surrounding quotes already stripped by the lexer). This is the omni-AST analog
// of ExtractSnowSQLOrdinaryIdentifier and delegates to omni's own Ident.Normalize
// so the folding stays in lock-step with the parser.
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

// normalizeSnowflakeSchemaName normalizes an omni *ObjectName that names a schema
// (a 1- or 2-part name: [database.]schema), returning the "database.schema"
// resource key with the fallback database filled in when the name is 1-part.
// It is the omni-AST analog of NormalizeSnowSQLSchemaName.
func normalizeSnowflakeSchemaName(schemaName *omniast.ObjectName, fallbackDatabaseName string) string {
	if schemaName == nil {
		return fallbackDatabaseName + "."
	}
	database := fallbackDatabaseName
	// A 2-part schema name parses as Schema.Name (database.schema); a 1-part
	// schema name parses as just Name (schema). Prefer the explicit database part
	// when present.
	if d := normalizeSnowflakeIdentifier(schemaName.Schema); d != "" {
		database = d
	}
	schema := normalizeSnowflakeIdentifier(schemaName.Name)
	return strings.Join([]string{database, schema}, ".")
}
