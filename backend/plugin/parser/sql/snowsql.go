// Package parser is the parser for SQL statement.
package parser

import (
	"strings"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/snowsql-parser"
)

// ParseSnowSQL parses the given SQL statement by using antlr4. Returns the AST and token stream if no error.
func ParseSnowSQL(statement string) (antlr.Tree, error) {
	statement = strings.TrimRight(statement, " \t\n\r\f;") + "\n;"
	inputStream := antlr.NewInputStream(statement)
	lexer := parser.NewSnowflakeLexer(inputStream)
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	p := parser.NewSnowflakeParser(stream)

	// Remove default error listener and add our own error listener.
	lexer.RemoveErrorListeners()
	lexerErrorListener := &ParseErrorListener{}
	lexer.AddErrorListener(lexerErrorListener)

	p.RemoveErrorListeners()
	parserErrorListener := &ParseErrorListener{}
	p.AddErrorListener(parserErrorListener)

	p.BuildParseTrees = true

	tree := p.Snowflake_file()

	if lexerErrorListener.err != nil {
		return nil, lexerErrorListener.err
	}

	if parserErrorListener.err != nil {
		return nil, parserErrorListener.err
	}

	return tree, nil
}

var snowflakeKeyword = map[string]bool{
	"ACCOUNT": true,
	"ALL":     true,
	"ALTER":   true,
	"AND":     true,
	"ANY":     true,
	"AS":      true,

	"BETWEEN": true,
	"BY":      true,

	"CASE":              true,
	"CAST":              true,
	"CHECK":             true,
	"COLUMN":            true,
	"CONNECT":           true,
	"CONNECTION":        true,
	"CONSTRAINT":        true,
	"CREATE":            true,
	"CROSS":             true,
	"CURRENT":           true,
	"CURRENT_DATE":      true,
	"CURRENT_TIME":      true,
	"CURRENT_TIMESTAMP": true,
	"CURRENT_USER":      true,

	"DATABASE": true,
	"DELETE":   true,
	"DISTINCT": true,
	"DROP":     true,

	"ELSE":   true,
	"EXISTS": true,

	"FALSE":     true,
	"FOLLOWING": true,
	"FOR":       true,
	"FROM":      true,
	"FULL":      true,

	"GRANT":     true,
	"GROUP":     true,
	"GSCLUSTER": true,

	"HAVING": true,

	"ILIKE":     true,
	"IN":        true,
	"INCREMENT": true,
	"INNER":     true,
	"INSERT":    true,
	"INTERSECT": true,
	"INTO":      true,
	"IS":        true,
	"ISSUE":     true,

	"JOIN": true,

	"LATERAL":        true,
	"LEFT":           true,
	"LIKE":           true,
	"LOCALTIME":      true,
	"LOCALTIMESTAMP": true,

	"MINUS": true,

	"NATURAL": true,
	"NOT":     true,
	"NULL":    true,

	"OF":           true,
	"ON":           true,
	"OR":           true,
	"ORDER":        true,
	"ORGANIZATION": true,

	"QUALIFY": true,

	"REGEXP": true,
	"REVOKE": true,
	"RIGHT":  true,
	"RLIKE":  true,
	"ROW":    true,
	"ROWS":   true,

	"SAMPLE": true,
	"SCHEMA": true,
	"SELECT": true,
	"SET":    true,
	"SOME":   true,
	"START":  true,

	"TABLE":       true,
	"TABLESAMPLE": true,
	"THEN":        true,
	"TO":          true,
	"TRIGGER":     true,
	"TRUE":        true,
	"TRY_CAST":    true,

	"UNION":  true,
	"UNIQUE": true,
	"UPDATE": true,
	"USING":  true,

	"VALUES": true,
	"VIEW":   true,

	"WHEN":     true,
	"WHENEVER": true,
	"WHERE":    true,
	"WITH":     true,
}

// IsSnowflakeKeyword returns true if the given string is a snowflake keyword.
// Follows https://docs.snowflake.com/en/sql-reference/reserved-keywords.
func IsSnowflakeKeyword(s string, caseSensitive bool) bool {
	if !caseSensitive {
		s = strings.ToUpper(s)
	}
	return snowflakeKeyword[s]
}
