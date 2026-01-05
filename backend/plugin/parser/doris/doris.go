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
	antlrASTs, err := ParseDorisSQL(statement)
	if err != nil {
		return nil, err
	}
	var asts []base.AST
	for _, ast := range antlrASTs {
		asts = append(asts, ast)
	}
	return asts, nil
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
	antlrASTs, err := ParseDorisSQL(statement)
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
		if !stmt.Empty && astIndex < len(antlrASTs) {
			ps.AST = antlrASTs[astIndex]
			astIndex++
		}
		result = append(result, ps)
	}

	return result, nil
}

// ParseDorisSQL parses the given SQL statement by using antlr4. Returns a list of AST and token stream if no error.
func ParseDorisSQL(statement string) ([]*base.ANTLRAST, error) {
	stmts, err := SplitSQL(statement)
	if err != nil {
		return nil, err
	}

	var result []*base.ANTLRAST
	for _, stmt := range stmts {
		if stmt.Empty {
			continue
		}

		antlrAST, err := parseSingleDorisSQL(stmt.Text, stmt.BaseLine())
		if err != nil {
			return nil, err
		}
		result = append(result, antlrAST)
	}

	return result, nil
}

func parseSingleDorisSQL(statement string, baseLine int) (*base.ANTLRAST, error) {
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

	result := &base.ANTLRAST{
		StartPosition: startPosition,
		Tree:          tree,
		Tokens:        stream,
	}

	return result, nil
}
