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
		return statementTypeMap[analysis.StatementTypeOf(node)], nil
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
		return statementTypeMap[analysis.StatementTypeOf(a.Node)], a.Text, true
	})
}

var statementTypeMap = map[analysis.StatementType]storepb.StatementType{
	analysis.CreateDatabase:  storepb.StatementType_CREATE_DATABASE,
	analysis.CreateTable:     storepb.StatementType_CREATE_TABLE,
	analysis.CreateView:      storepb.StatementType_CREATE_VIEW,
	analysis.CreateIndex:     storepb.StatementType_CREATE_INDEX,
	analysis.CreateEvent:     storepb.StatementType_CREATE_EVENT,
	analysis.CreateTrigger:   storepb.StatementType_CREATE_TRIGGER,
	analysis.CreateFunction:  storepb.StatementType_CREATE_FUNCTION,
	analysis.CreateProcedure: storepb.StatementType_CREATE_PROCEDURE,
	analysis.DropDatabase:    storepb.StatementType_DROP_DATABASE,
	analysis.DropTable:       storepb.StatementType_DROP_TABLE,
	analysis.DropView:        storepb.StatementType_DROP_VIEW,
	analysis.DropIndex:       storepb.StatementType_DROP_INDEX,
	analysis.DropEvent:       storepb.StatementType_DROP_EVENT,
	analysis.DropTrigger:     storepb.StatementType_DROP_TRIGGER,
	analysis.DropFunction:    storepb.StatementType_DROP_FUNCTION,
	analysis.DropProcedure:   storepb.StatementType_DROP_PROCEDURE,
	analysis.AlterDatabase:   storepb.StatementType_ALTER_DATABASE,
	analysis.AlterTable:      storepb.StatementType_ALTER_TABLE,
	analysis.AlterView:       storepb.StatementType_ALTER_VIEW,
	analysis.AlterEvent:      storepb.StatementType_ALTER_EVENT,
	analysis.Truncate:        storepb.StatementType_TRUNCATE,
	analysis.Rename:          storepb.StatementType_RENAME,
	analysis.Insert:          storepb.StatementType_INSERT,
	analysis.Update:          storepb.StatementType_UPDATE,
	analysis.Delete:          storepb.StatementType_DELETE,
	analysis.Execute:         storepb.StatementType_EXECUTE,
}
