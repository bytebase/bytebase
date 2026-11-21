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
		return convertStatementTypes(analysis.StatementTypes(wrapped.Node)), wrapped.Text, true, nil
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

// convertStatementTypes maps omni's statement types to storepb's by name and
// drops a name storepb does not know.
func convertStatementTypes(types []analysis.StatementType) []storepb.StatementType {
	var result []storepb.StatementType
	for _, t := range types {
		if value := storepb.StatementType(storepb.StatementType_value[string(t)]); value != storepb.StatementType_STATEMENT_TYPE_UNSPECIFIED {
			result = append(result, value)
		}
	}
	return result
}
