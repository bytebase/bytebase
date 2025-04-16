package cosmosdb

import (
	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/cosmosdb-parser"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

type ParseResult struct {
	Tree   antlr.Tree
	Tokens *antlr.CommonTokenStream
}

func ParseCosmosDBQuery(statement string) ([]*ParseResult, error) {
	lexer := parser.NewCosmosDBLexer(antlr.NewInputStream(statement))
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	p := parser.NewCosmosDBParser(stream)

	lexerErrorListener := &base.ParseErrorListener{
		Statement: statement,
	}
	lexer.RemoveErrorListeners()
	lexer.AddErrorListener(lexerErrorListener)

	parserErrorListener := &base.ParseErrorListener{
		Statement: statement,
	}
	p.RemoveErrorListeners()
	p.AddErrorListener(parserErrorListener)

	p.BuildParseTrees = true

	root := p.Root()

	if lexerErrorListener.Err != nil {
		return nil, lexerErrorListener.Err
	}

	if parserErrorListener.Err != nil {
		return nil, parserErrorListener.Err
	}

	return []*ParseResult{
		{
			Tree:   root,
			Tokens: stream,
		},
	}, nil
}
