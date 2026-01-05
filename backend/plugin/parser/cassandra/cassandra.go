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
	base.RegisterParseStatementsFunc(storepb.Engine_CASSANDRA, parseCassandraStatements)
}

// parseCassandraForRegistry is the ParseFunc for Cassandra.
// Returns []base.AST with *ANTLRAST instances.
func parseCassandraForRegistry(statement string) ([]base.AST, error) {
	parseResults, err := ParseCassandraSQL(statement)
	if err != nil {
		return nil, err
	}
	asts := make([]base.AST, len(parseResults))
	for i, r := range parseResults {
		asts[i] = r
	}
	return asts, nil
}

// parseCassandraStatements is the ParseStatementsFunc for Cassandra.
// Returns []ParsedStatement with both text and AST populated.
func parseCassandraStatements(statement string) ([]base.ParsedStatement, error) {
	// First split to get Statement with text and positions
	stmts, err := SplitSQL(statement)
	if err != nil {
		return nil, err
	}

	// Then parse to get ASTs
	parseResults, err := ParseCassandraSQL(statement)
	if err != nil {
		return nil, err
	}

	// Combine: Statement provides text/positions, ParseResult provides AST
	var result []base.ParsedStatement
	astIndex := 0
	for _, stmt := range stmts {
		ps := base.ParsedStatement{
			Statement: stmt,
		}
		if !stmt.Empty && astIndex < len(parseResults) {
			ps.AST = parseResults[astIndex]
			astIndex++
		}
		result = append(result, ps)
	}

	return result, nil
}

// ParseCassandraSQL parses the given CQL statement by using antlr4. Returns a list of AST and token stream if no error.
func ParseCassandraSQL(statement string) ([]*base.ANTLRAST, error) {
	stmts, err := SplitSQL(statement)
	if err != nil {
		return nil, err
	}

	var result []*base.ANTLRAST
	for _, stmt := range stmts {
		if stmt.Empty {
			continue
		}

		parseResult, err := parseSingleCassandraSQL(stmt.Text, stmt.BaseLine())
		if err != nil {
			return nil, err
		}
		result = append(result, parseResult)
	}

	return result, nil
}

func parseSingleCassandraSQL(statement string, baseLine int) (*base.ANTLRAST, error) {
	statement = strings.TrimRightFunc(statement, utils.IsSpaceOrSemicolon) + "\n;"
	inputStream := antlr.NewInputStream(statement)
	lexer := cql.NewCqlLexer(inputStream)
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	p := cql.NewCqlParser(stream)

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
		StartPosition: &storepb.Position{Line: int32(baseLine) + 1},
		Tree:          tree,
		Tokens:        stream,
	}

	return result, nil
}
