package pg

import (
	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"
)

// StatementTypeWithPosition contains statement type and its position information.
type StatementTypeWithPosition struct {
	Type string
	// Line is the one-based line number where the statement ends.
	Line int
	Text string
}

// GetStatementTypes returns statement types with position information.
// The line numbers are one-based.
func GetStatementTypes(asts any) ([]StatementTypeWithPosition, error) {
	parseResult, ok := asts.(*ParseResult)
	if !ok {
		return nil, errors.Errorf("invalid ast type %T, expected *ParseResult", asts)
	}

	if parseResult == nil || parseResult.Tree == nil {
		return nil, errors.New("invalid parse result")
	}

	collector := &statementTypeCollectorWithPosition{
		tokens: parseResult.Tokens,
	}

	antlr.ParseTreeWalkerDefault.Walk(collector, parseResult.Tree)

	return collector.results, nil
}
