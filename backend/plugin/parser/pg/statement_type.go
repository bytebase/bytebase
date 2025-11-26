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
func GetStatementTypes(asts []*base.AST) ([]StatementTypeWithPosition, error) {
	if len(asts) == 0 {
		return []StatementTypeWithPosition{}, nil
	}

	var allResults []StatementTypeWithPosition
	for _, unifiedAST := range asts {
		antlrData, ok := unifiedAST.GetANTLRTree()
		if !ok {
			return nil, errors.New("expected ANTLR tree for PostgreSQL")
		}
		if antlrData.Tree == nil {
			return nil, errors.New("ANTLR tree is nil")
		}

		collector := &statementTypeCollectorWithPosition{
			tokens:   antlrData.Tokens,
			baseLine: unifiedAST.BaseLine,
		}

		antlr.ParseTreeWalkerDefault.Walk(collector, antlrData.Tree)
		allResults = append(allResults, collector.results...)
	}

	return allResults, nil
}
