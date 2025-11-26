package doris

import (
	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/doris"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterParseFunc(storepb.Engine_DORIS, parseDorisForRegistry)
}

// parseDorisForRegistry is the ParseFunc for Doris.
// Returns []*base.AST with ANTLRResult populated.
func parseDorisForRegistry(statement string) ([]*base.AST, error) {
	parseResults, err := ParseDorisSQL(statement)
	if err != nil {
		return nil, err
	}
	return toAST(parseResults), nil
}

// toAST converts []*ParseResult to []*base.AST.
func toAST(results []*base.ParseResult) []*base.AST {
	var asts []*base.AST
	for _, r := range results {
		asts = append(asts, &base.AST{
			BaseLine: r.BaseLine,
			ANTLRResult: &base.ANTLRParseData{
				Tree:   r.Tree,
				Tokens: r.Tokens,
			},
		})
	}
	return asts
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
	lexer := parser.NewDorisSQLLexer(antlr.NewInputStream(statement))
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	p := parser.NewDorisSQLParser(stream)
	lexerErrorListener := &base.ParseErrorListener{
		Statement: statement,
		BaseLine:  baseLine,
	}
	lexer.RemoveErrorListeners()
	lexer.AddErrorListener(lexerErrorListener)

	parserErrorListener := &base.ParseErrorListener{
		Statement: statement,
		BaseLine:  baseLine,
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

	result := &base.ParseResult{
		Tree:     tree,
		Tokens:   stream,
		BaseLine: baseLine,
	}

	return result, nil
}
