package tsql

import (
	"github.com/bytebase/omni/mssql/analysis"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func GetStatementTypes(asts []base.AST) ([]storepb.StatementType, error) {
	sqlTypeSet := make(map[storepb.StatementType]bool)
	for _, a := range asts {
		node, ok := GetNode(a)
		if !ok {
			return nil, errors.New("expected omni AST for MSSQL")
		}
		sqlTypeSet[convertStatementType(analysis.StatementTypeOf(node))] = true
	}
	var sqlTypes []storepb.StatementType
	for sqlType := range sqlTypeSet {
		sqlTypes = append(sqlTypes, sqlType)
	}
	return sqlTypes, nil
}

func convertStatementType(t analysis.StatementType) storepb.StatementType {
	switch t {
	case analysis.AlterDatabase:
		return storepb.StatementType_ALTER_DATABASE
	case analysis.AlterIndex:
		return storepb.StatementType_ALTER_INDEX
	case analysis.AlterTable:
		return storepb.StatementType_ALTER_TABLE
	case analysis.CreateDatabase:
		return storepb.StatementType_CREATE_DATABASE
	case analysis.CreateIndex:
		return storepb.StatementType_CREATE_INDEX
	case analysis.CreateSchema:
		return storepb.StatementType_CREATE_SCHEMA
	case analysis.CreateTable:
		return storepb.StatementType_CREATE_TABLE
	case analysis.CreateView:
		return storepb.StatementType_CREATE_VIEW
	case analysis.DropDatabase:
		return storepb.StatementType_DROP_DATABASE
	case analysis.DropIndex:
		return storepb.StatementType_DROP_INDEX
	case analysis.DropSchema:
		return storepb.StatementType_DROP_SCHEMA
	case analysis.DropTable:
		return storepb.StatementType_DROP_TABLE
	case analysis.DropView:
		return storepb.StatementType_DROP_VIEW
	case analysis.Truncate:
		return storepb.StatementType_TRUNCATE
	case analysis.Delete:
		return storepb.StatementType_DELETE
	case analysis.Insert:
		return storepb.StatementType_INSERT
	case analysis.Update:
		return storepb.StatementType_UPDATE
	case analysis.Execute:
		return storepb.StatementType_EXECUTE
	default:
		return storepb.StatementType_STATEMENT_TYPE_UNSPECIFIED
	}
}
