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
	return toAST(parseResults), nil
}

// toAST converts []*ParseResult to []base.AST.
func toAST(results []*base.ParseResult) []base.AST {
	var asts []base.AST
	for _, r := range results {
		asts = append(asts, &base.ANTLRAST{
			StartPosition: &storepb.Position{Line: int32(r.BaseLine) + 1},
			Tree:          r.Tree,
			Tokens:        r.Tokens,
		})
	}
	return asts
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

	// Combine: Statement provides text/positions, ParseResult provides AST
	var result []base.ParsedStatement
	astIndex := 0
	for _, stmt := range stmts {
		ps := base.ParsedStatement{
			Statement: stmt,
		}
		if !stmt.Empty && astIndex < len(parseResults) {
			ps.AST = &base.ANTLRAST{
				StartPosition: &storepb.Position{Line: int32(parseResults[astIndex].BaseLine) + 1},
				Tree:          parseResults[astIndex].Tree,
				Tokens:        parseResults[astIndex].Tokens,
			}
			astIndex++
		}
		result = append(result, ps)
	}

	return result, nil
}

// ParseRedshift parses the given SQL statement and returns a list of ParseResults.
// Each ParseResult represents one statement with its AST, tokens, and base line offset.
func ParseRedshift(sql string) ([]*base.ParseResult, error) {
	stmts, err := SplitSQL(sql)
	if err != nil {
		return nil, errors.Wrap(err, "failed to split SQL")
	}

	var results []*base.ParseResult
	for _, stmt := range stmts {
		if stmt.Empty {
			continue
		}

		parseResult, err := parseSingleRedshift(stmt.Text, stmt.BaseLine)
		if err != nil {
			return nil, err
		}
		results = append(results, parseResult)
	}

	return results, nil
}

// parseSingleRedshift parses a single Redshift statement by using antlr4. Returns the AST and token stream if no error.
func parseSingleRedshift(statement string, baseLine int) (*base.ParseResult, error) {
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

	result := &base.ParseResult{
		Tree:     tree,
		Tokens:   stream,
		BaseLine: baseLine,
	}

	return result, nil
}
