package redshift

import (
	"strings"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/redshift"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/utils"
)

func init() {
	base.RegisterParseFunc(storepb.Engine_REDSHIFT, parseRedshiftForRegistry)
	base.RegisterParseStatementsFunc(storepb.Engine_REDSHIFT, parseRedshiftStatements)
	base.RegisterGetStatementTypes(storepb.Engine_REDSHIFT, GetStatementTypes)
}

// parseRedshiftForRegistry is the ParseFunc for Redshift.
// Returns []base.AST with *ANTLRAST instances.
func parseRedshiftForRegistry(statement string) ([]base.AST, error) {
	parseResults, err := ParseRedshift(statement)
	if err != nil {
		return nil, err
	}
	asts := make([]base.AST, len(parseResults))
	for i, r := range parseResults {
		asts[i] = r
	}
	return asts, nil
}

// parseRedshiftStatements is the ParseStatementsFunc for Redshift.
// Returns []ParsedStatement with both text and AST populated.
func parseRedshiftStatements(statement string) ([]base.ParsedStatement, error) {
	// First split to get Statement with text and positions
	stmts, err := SplitSQL(statement)
	if err != nil {
		return nil, err
	}

	// Then parse to get ASTs
	parseResults, err := ParseRedshift(statement)
	if err != nil {
		return nil, err
	}

	// Combine: Statement provides text/positions, ANTLRAST provides AST
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

// ParseRedshift parses the given SQL and returns a list of ANTLRAST (one per statement).
// Use the Redshift parser based on antlr4.
func ParseRedshift(sql string) ([]*base.ANTLRAST, error) {
	stmts, err := SplitSQL(sql)
	if err != nil {
		return nil, errors.Wrap(err, "failed to split SQL")
	}

	var results []*base.ANTLRAST
	for _, stmt := range stmts {
		if stmt.Empty {
			continue
		}

		parseResult, err := parseSingleRedshift(stmt.Text, stmt.BaseLine())
		if err != nil {
			return nil, err
		}
		results = append(results, parseResult)
	}

	return results, nil
}

// parseSingleRedshift parses a single Redshift statement and returns the ANTLRAST.
func parseSingleRedshift(statement string, baseLine int) (*base.ANTLRAST, error) {
	statement = strings.TrimRightFunc(statement, utils.IsSpaceOrSemicolon) + ";"
	inputStream := antlr.NewInputStream(statement)
	lexer := parser.NewRedshiftLexer(inputStream)
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	p := parser.NewRedshiftParser(stream)

	// Remove default error listener and add our own error listener.
	startPosition := &storepb.Position{Line: int32(baseLine) + 1}
	lexer.RemoveErrorListeners()
	lexerErrorListener := &base.ParseErrorListener{
		Statement:     statement,
		StartPosition: startPosition,
	}
	lexer.AddErrorListener(lexerErrorListener)

	parserErrorListener := &base.ParseErrorListener{
		Statement:     statement,
		StartPosition: startPosition,
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

	result := &base.ANTLRAST{
		StartPosition: &storepb.Position{Line: int32(baseLine) + 1},
		Tree:          tree,
		Tokens:        stream,
	}

	return result, nil
}
