package pg

import (
	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/postgresql-parser"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// ParsePostgreSQL parses the given SQL and returns the AST tree.
// Use the PostgreSQL parser based on antlr4.
func ParsePostgreSQL(sql string) (antlr.Tree, error) {
	lexer := parser.NewPostgreSQLLexer(antlr.NewInputStream(sql))
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	p := parser.NewPostgreSQLParser(stream)
	lexerErrorListener := &base.ParseErrorListener{}
	lexer.RemoveErrorListeners()
	lexer.AddErrorListener(lexerErrorListener)

	parserErrorListener := &base.ParseErrorListener{}
	p.RemoveErrorListeners()
	p.AddErrorListener(parserErrorListener)

	p.BuildParseTrees = true
	p.SetErrorHandler(antlr.NewBailErrorStrategy())

	tree := p.Root()
	if lexerErrorListener.Err != nil {
		return nil, lexerErrorListener.Err
	}

	if parserErrorListener.Err != nil {
		return nil, parserErrorListener.Err
	}

	return tree, nil
}
