// Package parser is the parser for SQL statement.
package parser

import (
	"strings"
	"unicode"

	"github.com/antlr4-go/antlr/v4"
	tsqlparser "github.com/bytebase/tsql-parser"
	"github.com/pkg/errors"
)

// ParseTSQL parses the given SQL statement by using antlr4. Returns the AST and token stream if no error.
func ParseTSQL(statement string) (antlr.Tree, error) {
	statement = strings.TrimRight(statement, " \t\n\r\f;") + "\n;"
	inputStream := antlr.NewInputStream(statement)
	lexer := tsqlparser.NewTSqlLexer(inputStream)
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	p := tsqlparser.NewTSqlParser(stream)

	// Remove default error listener and add our own error listener.
	lexer.RemoveErrorListeners()
	lexerErrorListener := &ParseErrorListener{}
	lexer.AddErrorListener(lexerErrorListener)

	p.RemoveErrorListeners()
	parserErrorListener := &ParseErrorListener{}
	p.AddErrorListener(parserErrorListener)

	p.BuildParseTrees = true

	tree := p.Tsql_file()

	if lexerErrorListener.err != nil {
		return nil, lexerErrorListener.err
	}

	if parserErrorListener.err != nil {
		return nil, parserErrorListener.err
	}

	return tree, nil
}

// NormalizedTSqlTableNamePart returns the normalized table name part.
// https://learn.microsoft.com/zh-cn/sql/relational-databases/databases/database-identifiers?view=sql-server-ver15
// TODO(zp): currently, we returns the lower case of the part, we may need to get the CI/CS from the server/database.
func NormalizedTSqlTableNamePart(part tsqlparser.IId_Context) (string, error) {
	if part == nil {
		return "", nil
	}
	text := part.GetText()
	if text == "" {
		return "", nil
	}
	if text[0] == '[' && text[len(text)-1] == ']' {
		text = text[1 : len(text)-1]
	}
	var sb strings.Builder
	for _, r := range text {
		if _, err := sb.WriteRune(unicode.ToLower(r)); err != nil {
			return "", errors.Wrapf(err, "failed to write rune: %q", r)
		}
	}
	return sb.String(), nil
}
