package redshift

import (
	"strings"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/redshift-parser"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/utils"
)

// ParseResult is the result of parsing a Redshift statement.
type ParseResult struct {
	Tree   antlr.Tree
	Tokens *antlr.CommonTokenStream
}

// ParseRedshift parses the given SQL statement by using antlr4. Returns the AST and token stream if no error.
func ParseRedshift(statement string) (*ParseResult, error) {
	statement = strings.TrimRightFunc(statement, utils.IsSpaceOrSemicolon) + ";"
	inputStream := antlr.NewInputStream(statement)
	lexer := parser.NewRedshiftLexer(inputStream)
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	p := parser.NewRedshiftParser(stream)

	// Remove default error listener and add our own error listener.
	lexer.RemoveErrorListeners()
	lexerErrorListener := &base.ParseErrorListener{
		Statement: statement,
	}
	lexer.AddErrorListener(lexerErrorListener)

	parserErrorListener := &base.ParseErrorListener{
		Statement: statement,
	}
	p.RemoveErrorListeners()
	p.AddErrorListener(parserErrorListener)

	p.BuildParseTrees = true

	tree := p.Root()

	// Return early if there are any lexer errors
	if lexerErrorListener.Err != nil {
		return nil, lexerErrorListener.Err
	}

	// Return early if there are any parser errors
	if parserErrorListener.Err != nil {
		return nil, parserErrorListener.Err
	}

	result := &ParseResult{
		Tree:   tree,
		Tokens: stream,
	}

	return result, nil
}
