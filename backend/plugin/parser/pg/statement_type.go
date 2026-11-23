package pg

import (
	"github.com/bytebase/omni/pg/analysis"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// StatementTypeWithPosition contains statement type and its position information.
type StatementTypeWithPosition = base.StatementTypeWithPosition

// GetStatementTypes returns statement types with position information.
// The line numbers are one-based.
func GetStatementTypes(asts []base.AST) ([]StatementTypeWithPosition, error) {
	return base.CollectStatementTypesWithPosition(asts, func(unifiedAST base.AST) ([]storepb.StatementType, string, bool, error) {
		wrapped, ok := unifiedAST.(*AST)
		if !ok {
			return nil, "", false, errors.New("expected AST for PostgreSQL")
		}
		if wrapped.Node == nil {
			return nil, "", false, nil
		}
		return base.OmniStatementTypes(analysis.StatementTypes(wrapped.Node)), wrapped.Text, true, nil
	})
}

// GetStatementTypesForRegistry returns only the statement types.
// This is used for registration with base.RegisterGetStatementTypes.
func GetStatementTypesForRegistry(asts []base.AST) ([]storepb.StatementType, error) {
	results, err := GetStatementTypes(asts)
	if err != nil {
		return nil, err
	}
	return base.ExtractTypesFromPositions(results), nil
}
