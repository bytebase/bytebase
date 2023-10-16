package snowflake

import (
	"strings"
	"unicode"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/snowsql-parser"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

type ParseResult struct {
	Tree   antlr.Tree
	Tokens *antlr.CommonTokenStream
}

// ParseSnowSQL parses the given SQL statement by using antlr4. Returns the AST and token stream if no error.
func ParseSnowSQL(statement string) (*ParseResult, error) {
	statement = strings.TrimRight(statement, " \t\n\r\f;") + "\n;"
	inputStream := antlr.NewInputStream(statement)
	lexer := parser.NewSnowflakeLexer(inputStream)
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	p := parser.NewSnowflakeParser(stream)

	// Remove default error listener and add our own error listener.
	lexer.RemoveErrorListeners()
	lexerErrorListener := &base.ParseErrorListener{}
	lexer.AddErrorListener(lexerErrorListener)

	p.RemoveErrorListeners()
	parserErrorListener := &base.ParseErrorListener{}
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
		Tree:   tree,
		Tokens: stream,
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
