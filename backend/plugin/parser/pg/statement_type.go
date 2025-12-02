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
			baseLine: base.GetLineOffset(antlrAST.StartPosition),
		}

		antlr.ParseTreeWalkerDefault.Walk(collector, antlrAST.Tree)
		allResults = append(allResults, collector.results...)
	}

	return allResults, nil
}

// GetStatementTypesForRegistry returns only the statement types as strings.
// This is used for registration with base.RegisterGetStatementTypes.
func GetStatementTypesForRegistry(asts []base.AST) ([]string, error) {
	results, err := GetStatementTypes(asts)
	if err != nil {
		return nil, err
	}
	types := make([]string, len(results))
	for i, r := range results {
		types[i] = r.Type
	}
	return types, nil
}
