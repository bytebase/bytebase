package cosmosdb

import (
	"strings"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/cosmosdb"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/utils"
)

type ParseResult struct {
	Tree     antlr.Tree
	Tokens   *antlr.CommonTokenStream
	BaseLine int
}

// ParseCosmosDBQuery parses the given CosmosDB SQL statement by using antlr4. Returns a list of AST and token stream if no error.
// Note: CosmosDB only supports single SELECT statements, so this will typically return a list with one element.
func ParseCosmosDBQuery(statement string) ([]*ParseResult, error) {
	stmts, err := SplitSQL(statement)
	if err != nil {
		return nil, err
	}

	var result []*ParseResult
	for _, stmt := range stmts {
		if stmt.Empty {
			continue
		}

		parseResult, err := parseSingleCosmosDBQuery(stmt.Text, stmt.BaseLine)
		if err != nil {
			return nil, err
		}
		result = append(result, parseResult)
	}

	return result, nil
}

func parseSingleCosmosDBQuery(statement string, baseLine int) (*ParseResult, error) {
	statement = strings.TrimRightFunc(statement, utils.IsSpaceOrSemicolon)
	inputStream := antlr.NewInputStream(statement)
	lexer := parser.NewCosmosDBLexer(inputStream)
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	p := parser.NewCosmosDBParser(stream)

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

	tree := p.Root()

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
