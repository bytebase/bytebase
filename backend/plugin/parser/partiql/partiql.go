package partiql

import (
	"strings"

	"github.com/antlr4-go/antlr/v4"

	parser "github.com/bytebase/partiql-parser"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/utils"
)

type ParseResult struct {
	Tree   antlr.Tree
	Tokens *antlr.CommonTokenStream
}

// ParsePartiQL parses the given PartiQL statement by using antlr4. Returns the AST and token stream if no error.
func ParsePartiQL(statement string) (*ParseResult, error) {
	statement = strings.TrimRightFunc(statement, utils.IsSpaceOrSemicolon) + "\n;"
	inputStream := antlr.NewInputStream(statement)
	lexer := parser.NewPartiQLLexer(inputStream)
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	p := parser.NewPartiQLParserParser(stream)

	// Remove default error listener and add our own error listener.
	lexer.RemoveErrorListeners()
	lexerErrorListener := &base.ParseErrorListener{
		Statement: statement,
	}
	lexer.AddErrorListener(lexerErrorListener)

	p.RemoveErrorListeners()
	parserErrorListener := &base.ParseErrorListener{
		Statement: statement,
	}
	p.AddErrorListener(parserErrorListener)

	p.BuildParseTrees = true

	tree := p.Script()

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
