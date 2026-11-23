package mariadb

import (
	"github.com/bytebase/omni/mariadb/analysis"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func GetStatementTypes(asts []base.AST) ([]storepb.StatementType, error) {
	return base.CollectUniqueStatementTypes(asts, func(a base.AST) (storepb.StatementType, error) {
		node, ok := GetNode(a)
		if !ok {
			return 0, errors.New("expected AST for MariaDB")
		}
		return base.OmniStatementType(analysis.StatementTypeOf(node)), nil
	})
}

// StatementTypeWithPosition contains a statement type and its position information.
type StatementTypeWithPosition = base.StatementTypeWithPosition
