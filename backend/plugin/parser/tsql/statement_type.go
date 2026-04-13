package tsql

import (
	"github.com/bytebase/omni/mssql/ast"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func GetStatementTypes(asts []base.AST) ([]storepb.StatementType, error) {
	sqlTypeSet := make(map[storepb.StatementType]bool)
	for _, a := range asts {
		node, ok := GetOmniNode(a)
		if !ok {
			return nil, errors.New("expected omni AST for MSSQL")
		}
		sqlTypeSet[getStatementType(node)] = true
	}
	var sqlTypes []storepb.StatementType
	for sqlType := range sqlTypeSet {
		sqlTypes = append(sqlTypes, sqlType)
	}
	return sqlTypes, nil
}

func getStatementType(node ast.Node) storepb.StatementType {
	switch n := node.(type) {
	case *ast.AlterDatabaseStmt:
		return storepb.StatementType_ALTER_DATABASE
	case *ast.AlterIndexStmt:
		return storepb.StatementType_ALTER_INDEX
	case *ast.AlterTableStmt:
		return storepb.StatementType_ALTER_TABLE
	case *ast.CreateDatabaseStmt:
		return storepb.StatementType_CREATE_DATABASE
	case *ast.CreateIndexStmt:
		return storepb.StatementType_CREATE_INDEX
	case *ast.CreateSchemaStmt:
		return storepb.StatementType_CREATE_SCHEMA
	case *ast.CreateTableStmt:
		return storepb.StatementType_CREATE_TABLE
	case *ast.CreateViewStmt:
		return storepb.StatementType_CREATE_VIEW
	case *ast.DropStmt:
		switch n.ObjectType {
		case ast.DropDatabase:
			return storepb.StatementType_DROP_DATABASE
		case ast.DropIndex:
			return storepb.StatementType_DROP_INDEX
		case ast.DropSchema:
			return storepb.StatementType_DROP_SCHEMA
		case ast.DropTable:
			return storepb.StatementType_DROP_TABLE
		case ast.DropView:
			return storepb.StatementType_DROP_VIEW
		default:
		}
	case *ast.TruncateStmt:
		return storepb.StatementType_TRUNCATE
	case *ast.DeleteStmt:
		return storepb.StatementType_DELETE
	case *ast.InsertStmt:
		return storepb.StatementType_INSERT
	case *ast.UpdateStmt:
		return storepb.StatementType_UPDATE
	default:
	}
	return storepb.StatementType_STATEMENT_TYPE_UNSPECIFIED
}
