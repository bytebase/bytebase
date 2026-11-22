package oceanbase

import (
	"github.com/bytebase/omni/oceanbase/analysis"
	"github.com/bytebase/omni/oceanbase/ast"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func GetStatementTypes(asts []base.AST) ([]storepb.StatementType, error) {
	return base.CollectUniqueStatementTypes(asts, func(a base.AST) (storepb.StatementType, error) {
		node, ok := GetNode(a)
		if !ok {
			return 0, errors.New("expected AST for OceanBase")
		}
		return classifyStatementType(node), nil
	})
}

// StatementTypeWithPosition contains a statement type and its position information.
type StatementTypeWithPosition = base.StatementTypeWithPosition

func classifyStatementType(node ast.Node) storepb.StatementType {
	switch analysis.StatementTypeOf(node) {
	case analysis.CreateDatabase:
		return storepb.StatementType_CREATE_DATABASE
	case analysis.CreateTable:
		return storepb.StatementType_CREATE_TABLE
	case analysis.CreateView:
		return storepb.StatementType_CREATE_VIEW
	case analysis.CreateIndex:
		return storepb.StatementType_CREATE_INDEX
	case analysis.CreateEvent:
		return storepb.StatementType_CREATE_EVENT
	case analysis.CreateTrigger:
		return storepb.StatementType_CREATE_TRIGGER
	case analysis.CreateFunction:
		return storepb.StatementType_CREATE_FUNCTION
	case analysis.CreateProcedure:
		return storepb.StatementType_CREATE_PROCEDURE
	case analysis.DropDatabase:
		return storepb.StatementType_DROP_DATABASE
	case analysis.DropTable:
		return storepb.StatementType_DROP_TABLE
	case analysis.DropView:
		return storepb.StatementType_DROP_VIEW
	case analysis.DropIndex:
		return storepb.StatementType_DROP_INDEX
	case analysis.DropEvent:
		return storepb.StatementType_DROP_EVENT
	case analysis.DropTrigger:
		return storepb.StatementType_DROP_TRIGGER
	case analysis.DropFunction:
		return storepb.StatementType_DROP_FUNCTION
	case analysis.DropProcedure:
		return storepb.StatementType_DROP_PROCEDURE
	case analysis.AlterDatabase:
		return storepb.StatementType_ALTER_DATABASE
	case analysis.AlterTable:
		return storepb.StatementType_ALTER_TABLE
	case analysis.AlterView:
		return storepb.StatementType_ALTER_VIEW
	case analysis.AlterEvent:
		return storepb.StatementType_ALTER_EVENT
	case analysis.Truncate:
		return storepb.StatementType_TRUNCATE
	case analysis.Rename:
		return storepb.StatementType_RENAME
	case analysis.Insert:
		return storepb.StatementType_INSERT
	case analysis.Update:
		return storepb.StatementType_UPDATE
	case analysis.Delete:
		return storepb.StatementType_DELETE
	case analysis.Execute:
		return storepb.StatementType_EXECUTE
	default:
		return storepb.StatementType_STATEMENT_TYPE_UNSPECIFIED
	}
}
