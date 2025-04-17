// Package trino provides SQL parser for Trino.
package trino

import (
	"strings"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/trino-parser"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// ParseResult is the result of parsing a Trino statement.
type ParseResult struct {
	Tree   antlr.Tree
	Tokens *antlr.CommonTokenStream
}

// ParseTrino parses the given SQL and returns the ParseResult.
// Use the Trino parser based on antlr4.
func ParseTrino(sql string) (*ParseResult, error) {
	lexer := parser.NewTrinoLexer(antlr.NewInputStream(sql))
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	p := parser.NewTrinoParser(stream)
	lexerErrorListener := &base.ParseErrorListener{
		Statement: sql,
	}
	lexer.RemoveErrorListeners()
	lexer.AddErrorListener(lexerErrorListener)

	parserErrorListener := &base.ParseErrorListener{
		Statement: sql,
	}
	p.RemoveErrorListeners()
	p.AddErrorListener(parserErrorListener)

	p.BuildParseTrees = true

	tree := p.SingleStatement()
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

// NormalizeTrinoIdentifier normalizes the identifier for Trino.
func NormalizeTrinoIdentifier(ident string) string {
	// Trino identifiers are case-insensitive unless quoted.
	if strings.HasPrefix(ident, "\"") && strings.HasSuffix(ident, "\"") {
		// Remove quotes for quoted identifiers
		return ident[1 : len(ident)-1]
	}
	return strings.ToLower(ident)
}
