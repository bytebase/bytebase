// Package parser is the parser for SQL statement.
package parser

import (
	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/mysql-parser"
)

// ParseMySQL parses the given SQL statement and returns the AST.
func ParseMySQL(statement string, charset string, collation string) (antlr.Tree, error) {
	lexer := parser.NewMySQLLexer(antlr.NewInputStream(statement))
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	p := parser.NewMySQLParser(stream)

	lexerErrorListener := &ParseErrorListener{}
	lexer.RemoveErrorListeners()
	lexer.AddErrorListener(lexerErrorListener)

	parserErrorListener := &ParseErrorListener{}
	p.RemoveErrorListeners()
	p.AddErrorListener(parserErrorListener)

	p.BuildParseTrees = true

	tree := p.Query()

	if lexerErrorListener.err != nil {
		return nil, lexerErrorListener.err
	}

	if parserErrorListener.err != nil {
		return nil, parserErrorListener.err
	}

	return tree, nil
}
