package redshift

import (
	"github.com/bytebase/omni/redshift/analysis"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// StatementTypeWithPosition contains statement type and its position information.
type StatementTypeWithPosition = base.StatementTypeWithPosition

// GetStatementTypesWithPosition returns statement types with position information.
// The line numbers are one-based.
func GetStatementTypesWithPosition(asts []base.AST) ([]StatementTypeWithPosition, error) {
	return base.CollectSingleStatementTypesWithPosition(asts, "expected Redshift omni AST", false, func(a *AST) (storepb.StatementType, string, bool) {
		if a == nil || a.Node == nil {
			return 0, "", false
		}
		return base.OmniStatementType(analysis.StatementTypeOf(a.Node)), a.Text, true
	})
}

// GetStatementTypes returns statement types for registration with base.RegisterGetStatementTypes.
func GetStatementTypes(asts []base.AST) ([]storepb.StatementType, error) {
	results, err := GetStatementTypesWithPosition(asts)
	if err != nil {
		return nil, err
	}
	return base.ExtractTypesFromPositions(results), nil
}
