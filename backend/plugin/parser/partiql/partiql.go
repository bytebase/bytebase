package partiql

import (
	"strings"

	"github.com/antlr4-go/antlr/v4"

	parser "github.com/bytebase/parser/partiql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/utils"
)

func init() {
	base.RegisterParseFunc(storepb.Engine_DYNAMODB, parsePartiQLForRegistry)
}

// parsePartiQLForRegistry is the ParseFunc for PartiQL.
// Returns []*ParseResult on success.
func parsePartiQLForRegistry(statement string) (any, error) {
	result, err := ParsePartiQL(statement)
	if err != nil {
		return nil, err
	}
	return result, nil
}

type ParseResult struct {
	Tree     antlr.Tree
	Tokens   *antlr.CommonTokenStream
	BaseLine int
}

// ParsePartiQL parses the given PartiQL statement by using antlr4. Returns a list of AST and token stream if no error.
func ParsePartiQL(statement string) ([]*ParseResult, error) {
	stmts, err := SplitSQL(statement)
	if err != nil {
		return nil, err
	}

	var result []*ParseResult
	for _, stmt := range stmts {
		if stmt.Empty {
			continue
		}

		parseResult, err := parseSinglePartiQL(stmt.Text, stmt.BaseLine)
		if err != nil {
			return nil, err
		}
		result = append(result, parseResult)
	}

	return result, nil
}

func parseSinglePartiQL(statement string, baseLine int) (*ParseResult, error) {
	statement = strings.TrimRightFunc(statement, utils.IsSpaceOrSemicolon) + "\n;"
	inputStream := antlr.NewInputStream(statement)
	lexer := parser.NewPartiQLLexer(inputStream)
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	p := parser.NewPartiQLParserParser(stream)

	// Remove default error listener and add our own error listener.
	lexer.RemoveErrorListeners()
	lexerErrorListener := &base.ParseErrorListener{
		Statement: statement,
		BaseLine:  baseLine,
	}
	lexer.AddErrorListener(lexerErrorListener)

	p.RemoveErrorListeners()
	parserErrorListener := &base.ParseErrorListener{
		Statement: statement,
		BaseLine:  baseLine,
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
		Tree:     tree,
		Tokens:   stream,
		BaseLine: baseLine,
	}

	return result, nil
}
