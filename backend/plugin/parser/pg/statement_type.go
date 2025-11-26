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
func GetStatementTypes(asts []base.AST) ([]StatementTypeWithPosition, error) {
	if len(asts) == 0 {
		return []StatementTypeWithPosition{}, nil
	}

	var allResults []StatementTypeWithPosition
	for _, unifiedAST := range asts {
		antlrAST, ok := base.GetANTLRAST(unifiedAST)
		if !ok {
			return nil, errors.New("expected ANTLR AST for PostgreSQL")
		}
		if antlrAST.Tree == nil {
			return nil, errors.New("ANTLR tree is nil")
		}

		collector := &statementTypeCollectorWithPosition{
			tokens:   antlrAST.Tokens,
			baseLine: antlrAST.BaseLine,
		}

		antlr.ParseTreeWalkerDefault.Walk(collector, antlrAST.Tree)
		allResults = append(allResults, collector.results...)
	}

	return allResults, nil
}
