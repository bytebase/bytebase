package doris

import (
	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/doris"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterParseFunc(storepb.Engine_DORIS, parseDorisForRegistry)
	base.RegisterParseStatementsFunc(storepb.Engine_DORIS, parseDorisStatements)
}

// parseDorisForRegistry is the ParseFunc for Doris.
// Returns []base.AST with *ANTLRAST instances.
func parseDorisForRegistry(statement string) ([]base.AST, error) {
	parseResults, err := ParseDorisSQL(statement)
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

// parseDorisStatements is the ParseStatementsFunc for Doris.
// Returns []ParsedStatement with both text and AST populated.
func parseDorisStatements(statement string) ([]base.ParsedStatement, error) {
	// First split to get Statement with text and positions
	stmts, err := SplitSQL(statement)
	if err != nil {
		return nil, err
	}

	// Then parse to get ASTs
	parseResults, err := ParseDorisSQL(statement)
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

// ParseDorisSQL parses the given SQL statement by using antlr4. Returns a list of AST and token stream if no error.
func ParseDorisSQL(statement string) ([]*base.ParseResult, error) {
	stmts, err := SplitSQL(statement)
	if err != nil {
		return nil, err
	}

	var result []*base.ParseResult
	for _, stmt := range stmts {
		if stmt.Empty {
			continue
		}

		parseResult, err := parseSingleDorisSQL(stmt.Text, stmt.BaseLine)
		if err != nil {
			return nil, err
		}
		result = append(result, parseResult)
	}

	return result, nil
}

func parseSingleDorisSQL(statement string, baseLine int) (*base.ParseResult, error) {
	lexer := parser.NewDorisLexer(antlr.NewInputStream(statement))
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	p := parser.NewDorisParser(stream)
	startPosition := &storepb.Position{Line: int32(baseLine) + 1}
	lexerErrorListener := &base.ParseErrorListener{
		Statement:     statement,
		StartPosition: startPosition,
	}
	lexer.RemoveErrorListeners()
	lexer.AddErrorListener(lexerErrorListener)

	parserErrorListener := &base.ParseErrorListener{
		Statement:     statement,
		StartPosition: startPosition,
	}
	p.RemoveErrorListeners()
	p.AddErrorListener(parserErrorListener)

	p.BuildParseTrees = true

	tree := p.MultiStatements()
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
