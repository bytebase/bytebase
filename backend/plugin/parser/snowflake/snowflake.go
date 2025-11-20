package snowflake

import (
	"strings"
	"unicode"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/snowflake"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/utils"
)

func init() {
	base.RegisterParseFunc(storepb.Engine_SNOWFLAKE, parseSnowflakeForRegistry)
}

// parseSnowflakeForRegistry is the ParseFunc for Snowflake.
// Returns []*ParseResult on success.
func parseSnowflakeForRegistry(statement string) (any, error) {
	return ParseSnowSQL(statement)
}

type ParseResult struct {
	Tree     antlr.Tree
	Tokens   *antlr.CommonTokenStream
	BaseLine int
}

// ParseSnowSQL parses the given SQL and returns a list of ParseResult (one per statement).
// Use the Snowflake parser based on antlr4.
func ParseSnowSQL(sql string) ([]*ParseResult, error) {
	stmts, err := SplitSQL(sql)
	if err != nil {
		return nil, err
	}

	var results []*ParseResult
	for _, stmt := range stmts {
		if stmt.Empty {
			continue
		}

		parseResult, err := parseSingleSnowSQL(stmt.Text, stmt.BaseLine)
		if err != nil {
			return nil, err
		}
		results = append(results, parseResult)
	}

	return results, nil
}

// parseSingleSnowSQL parses a single Snowflake statement and returns the ParseResult.
func parseSingleSnowSQL(statement string, baseLine int) (*ParseResult, error) {
	// Trim leading newlines to ensure the first token starts at line 1 in ANTLR
	// This makes baseLine calculations simpler: baseLine + line = original line
	statement = strings.TrimLeftFunc(statement, func(r rune) bool {
		return r == '\n' || r == '\r'
	})
	statement = strings.TrimRightFunc(statement, utils.IsSpaceOrSemicolon) + "\n;"
	inputStream := antlr.NewInputStream(statement)
	lexer := parser.NewSnowflakeLexer(inputStream)
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	p := parser.NewSnowflakeParser(stream)

	// Remove default error listener and add our own error listener.
	lexer.RemoveErrorListeners()
	lexerErrorListener := &base.ParseErrorListener{
		Statement: statement,
		BaseLine:  baseLine,
	}
	lexer.AddErrorListener(lexerErrorListener)

	p.RemoveErrorListeners()
	parserErrorListener := &base.ParseErrorListener{
		Statement: statement,
		BaseLine:  baseLine,
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

	result := &ParseResult{
		Tree:     tree,
		Tokens:   stream,
		BaseLine: baseLine,
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
