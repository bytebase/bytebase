package cosmosdb

import (
	"strings"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/cosmosdb"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/utils"
)

// ParseCosmosDBQuery parses the given CosmosDB SQL statement by using antlr4. Returns a list of AST and token stream if no error.
// Note: CosmosDB only supports single SELECT statements, so this will typically return a list with one element.
func ParseCosmosDBQuery(statement string) ([]*base.ANTLRAST, error) {
	stmts, err := SplitSQL(statement)
	if err != nil {
		return nil, err
	}

	var result []*base.ANTLRAST
	for _, stmt := range stmts {
		if stmt.Empty {
			continue
		}

		parseResult, err := parseSingleCosmosDBQuery(stmt.Text, stmt.BaseLine())
		if err != nil {
			return nil, err
		}
		result = append(result, parseResult)
	}

	return result, nil
}

func parseSingleCosmosDBQuery(statement string, baseLine int) (*base.ANTLRAST, error) {
	statement = strings.TrimRightFunc(statement, utils.IsSpaceOrSemicolon)
	inputStream := antlr.NewInputStream(statement)
	lexer := parser.NewCosmosDBLexer(inputStream)
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	p := parser.NewCosmosDBParser(stream)

	// Remove default error listener and add our own error listener.
	startPosition := &storepb.Position{Line: int32(baseLine) + 1}
	lexer.RemoveErrorListeners()
	lexerErrorListener := &base.ParseErrorListener{
		Statement:     statement,
		StartPosition: startPosition,
	}
	lexer.AddErrorListener(lexerErrorListener)

	p.RemoveErrorListeners()
	parserErrorListener := &base.ParseErrorListener{
		Statement:     statement,
		StartPosition: startPosition,
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

	result := &base.ANTLRAST{
		StartPosition: startPosition,
		Tree:          tree,
		Tokens:        stream,
	}

	return result, nil
}
