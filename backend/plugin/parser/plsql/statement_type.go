package plsql

import (
	"github.com/bytebase/omni/oracle/analysis"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func GetStatementTypes(asts []base.AST) ([]storepb.StatementType, error) {
	sqlTypeSet := make(map[storepb.StatementType]bool)
	for _, ast := range asts {
		node, ok := GetNode(ast)
		if ok {
			sqlTypeSet[convertStatementType(analysis.StatementTypeOf(node))] = true
			continue
		}

		return nil, errors.New("expected Oracle omni AST")
	}
	var sqlTypes []storepb.StatementType
	for sqlType := range sqlTypeSet {
		sqlTypes = append(sqlTypes, sqlType)
	}
	return sqlTypes, nil
}

func convertStatementType(t analysis.StatementType) storepb.StatementType {
	switch t {
	case analysis.CreateDatabase:
		return storepb.StatementType_CREATE_DATABASE
	case analysis.CreateTable:
		return storepb.StatementType_CREATE_TABLE
	case analysis.CreateIndex:
		return storepb.StatementType_CREATE_INDEX
	case analysis.CreateView:
		return storepb.StatementType_CREATE_VIEW
	case analysis.CreateSequence:
		return storepb.StatementType_CREATE_SEQUENCE
	case analysis.CreateSchema:
		return storepb.StatementType_CREATE_SCHEMA
	case analysis.CreateFunction:
		return storepb.StatementType_CREATE_FUNCTION
	case analysis.CreateTrigger:
		return storepb.StatementType_CREATE_TRIGGER
	case analysis.CreateProcedure:
		return storepb.StatementType_CREATE_PROCEDURE
	case analysis.CreateType:
		return storepb.StatementType_CREATE_TYPE
	case analysis.DropDatabase:
		return storepb.StatementType_DROP_DATABASE
	case analysis.DropTable:
		return storepb.StatementType_DROP_TABLE
	case analysis.DropView:
		return storepb.StatementType_DROP_VIEW
	case analysis.DropIndex:
		return storepb.StatementType_DROP_INDEX
	case analysis.DropSequence:
		return storepb.StatementType_DROP_SEQUENCE
	case analysis.DropFunction:
		return storepb.StatementType_DROP_FUNCTION
	case analysis.DropTrigger:
		return storepb.StatementType_DROP_TRIGGER
	case analysis.DropProcedure:
		return storepb.StatementType_DROP_PROCEDURE
	case analysis.DropType:
		return storepb.StatementType_DROP_TYPE
	case analysis.AlterDatabase:
		return storepb.StatementType_ALTER_DATABASE
	case analysis.AlterTable:
		return storepb.StatementType_ALTER_TABLE
	case analysis.AlterIndex:
		return storepb.StatementType_ALTER_INDEX
	case analysis.AlterView:
		return storepb.StatementType_ALTER_VIEW
	case analysis.AlterSequence:
		return storepb.StatementType_ALTER_SEQUENCE
	case analysis.AlterType:
		return storepb.StatementType_ALTER_TYPE
	case analysis.Truncate:
		return storepb.StatementType_TRUNCATE
	case analysis.Rename:
		return storepb.StatementType_RENAME
	case analysis.Comment:
		return storepb.StatementType_COMMENT
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
