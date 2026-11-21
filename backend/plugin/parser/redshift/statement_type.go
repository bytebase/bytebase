package redshift

import (
	"github.com/bytebase/omni/redshift/analysis"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// StatementTypeWithPosition contains statement type and its position information.
type StatementTypeWithPosition = base.StatementTypeWithPosition

// GetStatementTypesWithPosition returns statement types with position information.
// The line numbers are one-based.
func GetStatementTypesWithPosition(asts []base.AST) ([]StatementTypeWithPosition, error) {
	return base.CollectStatementTypesWithPosition(asts, func(unifiedAST base.AST) ([]storepb.StatementType, string, bool, error) {
		wrapped, ok := unifiedAST.(*AST)
		if !ok {
			return nil, "", false, errors.New("expected Redshift omni AST")
		}
		if wrapped.Node == nil {
			return nil, "", false, nil
		}
		stmtType := convertStatementType(analysis.StatementTypeOf(wrapped.Node))
		if stmtType == storepb.StatementType_STATEMENT_TYPE_UNSPECIFIED {
			return nil, "", false, nil
		}
		return []storepb.StatementType{stmtType}, wrapped.Text, true, nil
	})
}

// GetStatementTypes returns only the statement types as strings.
// This is used for registration with base.RegisterGetStatementTypes.
func GetStatementTypes(asts []base.AST) ([]storepb.StatementType, error) {
	results, err := GetStatementTypesWithPosition(asts)
	if err != nil {
		return nil, err
	}
	return base.ExtractTypesFromPositions(results), nil
}

func convertStatementType(t analysis.StatementType) storepb.StatementType {
	switch t {
	case analysis.CreateDatabase:
		return storepb.StatementType_CREATE_DATABASE
	case analysis.CreateTable:
		return storepb.StatementType_CREATE_TABLE
	case analysis.CreateView:
		return storepb.StatementType_CREATE_VIEW
	case analysis.CreateIndex:
		return storepb.StatementType_CREATE_INDEX
	case analysis.CreateSequence:
		return storepb.StatementType_CREATE_SEQUENCE
	case analysis.CreateSchema:
		return storepb.StatementType_CREATE_SCHEMA
	case analysis.CreateFunction:
		return storepb.StatementType_CREATE_FUNCTION
	case analysis.DropDatabase:
		return storepb.StatementType_DROP_DATABASE
	case analysis.DropTable:
		return storepb.StatementType_DROP_TABLE
	case analysis.DropView:
		return storepb.StatementType_DROP_VIEW
	case analysis.DropSchema:
		return storepb.StatementType_DROP_SCHEMA
	case analysis.DropIndex:
		return storepb.StatementType_DROP_INDEX
	case analysis.DropSequence:
		return storepb.StatementType_DROP_SEQUENCE
	case analysis.DropFunction:
		return storepb.StatementType_DROP_FUNCTION
	case analysis.AlterDatabase:
		return storepb.StatementType_ALTER_DATABASE
	case analysis.AlterTable:
		return storepb.StatementType_ALTER_TABLE
	case analysis.AlterView:
		return storepb.StatementType_ALTER_VIEW
	case analysis.AlterIndex:
		return storepb.StatementType_ALTER_INDEX
	case analysis.AlterSequence:
		return storepb.StatementType_ALTER_SEQUENCE
	case analysis.RenameIndex:
		return storepb.StatementType_RENAME_INDEX
	case analysis.RenameSchema:
		return storepb.StatementType_RENAME_SCHEMA
	case analysis.RenameSequence:
		return storepb.StatementType_RENAME_SEQUENCE
	case analysis.Comment:
		return storepb.StatementType_COMMENT
	case analysis.Truncate:
		return storepb.StatementType_TRUNCATE
	case analysis.Insert:
		return storepb.StatementType_INSERT
	case analysis.Update:
		return storepb.StatementType_UPDATE
	case analysis.Delete:
		return storepb.StatementType_DELETE
	case analysis.Merge:
		return storepb.StatementType_MERGE
	case analysis.Execute:
		return storepb.StatementType_EXECUTE
	default:
		return storepb.StatementType_STATEMENT_TYPE_UNSPECIFIED
	}
}
