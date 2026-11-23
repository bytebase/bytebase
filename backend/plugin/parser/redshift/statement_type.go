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
		return statementTypeMap[analysis.StatementTypeOf(a.Node)], a.Text, true
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

var statementTypeMap = map[analysis.StatementType]storepb.StatementType{
	analysis.CreateDatabase: storepb.StatementType_CREATE_DATABASE,
	analysis.CreateTable:    storepb.StatementType_CREATE_TABLE,
	analysis.CreateView:     storepb.StatementType_CREATE_VIEW,
	analysis.CreateIndex:    storepb.StatementType_CREATE_INDEX,
	analysis.CreateSequence: storepb.StatementType_CREATE_SEQUENCE,
	analysis.CreateSchema:   storepb.StatementType_CREATE_SCHEMA,
	analysis.CreateFunction: storepb.StatementType_CREATE_FUNCTION,
	analysis.DropDatabase:   storepb.StatementType_DROP_DATABASE,
	analysis.DropTable:      storepb.StatementType_DROP_TABLE,
	analysis.DropView:       storepb.StatementType_DROP_VIEW,
	analysis.DropSchema:     storepb.StatementType_DROP_SCHEMA,
	analysis.DropIndex:      storepb.StatementType_DROP_INDEX,
	analysis.DropSequence:   storepb.StatementType_DROP_SEQUENCE,
	analysis.DropFunction:   storepb.StatementType_DROP_FUNCTION,
	analysis.AlterDatabase:  storepb.StatementType_ALTER_DATABASE,
	analysis.AlterTable:     storepb.StatementType_ALTER_TABLE,
	analysis.AlterView:      storepb.StatementType_ALTER_VIEW,
	analysis.AlterIndex:     storepb.StatementType_ALTER_INDEX,
	analysis.AlterSequence:  storepb.StatementType_ALTER_SEQUENCE,
	analysis.RenameIndex:    storepb.StatementType_RENAME_INDEX,
	analysis.RenameSchema:   storepb.StatementType_RENAME_SCHEMA,
	analysis.RenameSequence: storepb.StatementType_RENAME_SEQUENCE,
	analysis.Comment:        storepb.StatementType_COMMENT,
	analysis.Truncate:       storepb.StatementType_TRUNCATE,
	analysis.Insert:         storepb.StatementType_INSERT,
	analysis.Update:         storepb.StatementType_UPDATE,
	analysis.Delete:         storepb.StatementType_DELETE,
	analysis.Merge:          storepb.StatementType_MERGE,
	analysis.Execute:        storepb.StatementType_EXECUTE,
}
