package mysql

import (
	"github.com/bytebase/omni/mysql/analysis"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func GetStatementTypes(asts []base.AST) ([]storepb.StatementType, error) {
	return base.CollectUniqueStatementTypes(asts, func(a base.AST) (storepb.StatementType, error) {
		node, ok := GetNode(a)
		if !ok {
			return 0, errors.New("expected AST for MySQL")
		}
		return base.OmniStatementType(analysis.StatementTypeOf(node)), nil
	})
}

// StatementTypeWithPosition contains a statement type and its position information.
type StatementTypeWithPosition = base.StatementTypeWithPosition

// GetStatementTypesWithPositions returns statement types with position information,
// preserving statement order. Line numbers are one-based.
func GetStatementTypesWithPositions(asts []base.AST) ([]StatementTypeWithPosition, error) {
	// STATEMENT_TYPE_UNSPECIFIED entries are included so the SDL gate fails closed.
	return base.CollectSingleStatementTypesWithPosition(asts, "expected AST for MySQL", true, func(a *AST) (storepb.StatementType, string, bool) {
		if a == nil || a.Node == nil {
			return 0, "", false
		}
		return base.OmniStatementType(analysis.StatementTypeOf(a.Node)), a.Text, true
	})
}
