package pg

import (
	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
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
	parseResults, ok := asts.([]*base.ParseResult)
	if !ok {
		return nil, errors.Errorf("invalid ast type %T, expected []*base.ParseResult", asts)
	}

	if len(parseResults) == 0 {
		return []StatementTypeWithPosition{}, nil
	}

	var allResults []StatementTypeWithPosition
	for _, parseResult := range parseResults {
		if parseResult == nil || parseResult.Tree == nil {
			return nil, errors.New("invalid parse result")
		}

		collector := &statementTypeCollectorWithPosition{
			tokens:   parseResult.Tokens,
			baseLine: parseResult.BaseLine,
		}

		antlr.ParseTreeWalkerDefault.Walk(collector, parseResult.Tree)
		allResults = append(allResults, collector.results...)
	}

	return allResults, nil
}
