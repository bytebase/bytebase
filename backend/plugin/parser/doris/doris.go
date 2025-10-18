package doris

import (
	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/doris-parser"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	RegisterParser()
}

// RegisterParser registers the Doris parser.
// Returns *ParseResult on success.
func RegisterParser() {
	base.RegisterParseFunc(storepb.Engine_DORIS, func(statement string) (any, error) {
		result, err := ParseDorisSQL(statement)
		if err != nil {
			return nil, err
		}
		return result, nil
	})
}

// ParseResult is the result of parsing a MySQL statement.
type ParseResult struct {
	Tree     antlr.Tree
	Tokens   *antlr.CommonTokenStream
	BaseLine int
}

func ParseDorisSQL(statement string) (*ParseResult, error) {
	lexer := parser.NewDorisSQLLexer(antlr.NewInputStream(statement))
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	p := parser.NewDorisSQLParser(stream)
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

	tree := p.SqlStatements()
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
