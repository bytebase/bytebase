package plsql

import (
	"github.com/bytebase/omni/oracle/analysis"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func GetStatementTypes(asts []base.AST) ([]storepb.StatementType, error) {
	return base.CollectUniqueStatementTypes(asts, func(ast base.AST) (storepb.StatementType, error) {
		node, ok := GetNode(ast)
		if !ok {
			return 0, errors.New("expected Oracle omni AST")
		}
		return base.OmniStatementType(analysis.StatementTypeOf(node)), nil
	})
}
