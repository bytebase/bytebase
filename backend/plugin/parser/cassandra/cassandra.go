package cassandra

import (
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/bytebase/parser/cql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/utils"
)

func init() {
	base.RegisterParseFunc(storepb.Engine_CASSANDRA, parseCassandraForRegistry)
}

// parseCassandraForRegistry is the ParseFunc for Cassandra.
// Returns []*base.ParseResult on success.
func parseCassandraForRegistry(statement string) (any, error) {
	result, err := ParseCassandraSQL(statement)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// ParseCassandraSQL parses the given CQL statement by using antlr4. Returns a list of AST and token stream if no error.
func ParseCassandraSQL(statement string) ([]*base.ParseResult, error) {
	stmts, err := SplitSQL(statement)
	if err != nil {
		return nil, err
	}

	var result []*base.ParseResult
	for _, stmt := range stmts {
		if stmt.Empty {
			continue
		}

		parseResult, err := parseSingleCassandraSQL(stmt.Text, stmt.BaseLine)
		if err != nil {
			return nil, err
		}
		result = append(result, parseResult)
	}

	return result, nil
}

func parseSingleCassandraSQL(statement string, baseLine int) (*base.ParseResult, error) {
	statement = strings.TrimRightFunc(statement, utils.IsSpaceOrSemicolon) + "\n;"
	inputStream := antlr.NewInputStream(statement)
	lexer := cql.NewCqlLexer(inputStream)
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	p := cql.NewCqlParser(stream)

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

	result := &base.ParseResult{
		Tree:     tree,
		Tokens:   stream,
		BaseLine: baseLine,
	}

	return result, nil
}
