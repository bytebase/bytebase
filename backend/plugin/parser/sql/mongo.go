// Package parser is the parser for SQL statement.
package parser

import (
	"github.com/antlr4-go/antlr/v4"
	mongoparser "github.com/bytebase/mongo-parser"
)

// ParseMongo parses the given statement by using antlr4. Returns the AST and token stream if no error.
func ParseMongo(statement string) (antlr.Tree, error) {
	inputStream := antlr.NewInputStream(statement)
	lexer := mongoparser.NewmongoLexer(inputStream)
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	p := mongoparser.NewmongoParser(stream)

	// Remove default error listener and add our own error listener.
	lexer.RemoveErrorListeners()
	lexerErrorListener := &ParseErrorListener{}
	lexer.AddErrorListener(lexerErrorListener)

	p.RemoveErrorListeners()
	parserErrorListener := &ParseErrorListener{}
	p.AddErrorListener(parserErrorListener)

	p.BuildParseTrees = true

	tree := p.Commands()

	if lexerErrorListener.err != nil {
		return nil, lexerErrorListener.err
	}

	if parserErrorListener.err != nil {
		return nil, parserErrorListener.err
	}

	return tree, nil
}
