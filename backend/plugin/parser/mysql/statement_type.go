package mysql

import (
	"github.com/bytebase/omni/mysql/ast"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func GetStatementTypes(asts []base.AST) ([]storepb.StatementType, error) {
	sqlTypeSet := make(map[storepb.StatementType]bool)
	for _, a := range asts {
		node, ok := GetOmniNode(a)
		if !ok {
			return nil, errors.New("expected OmniAST for MySQL")
		}
		t := classifyStatementType(node)
		sqlTypeSet[t] = true
	}
	var sqlTypes []storepb.StatementType
	for sqlType := range sqlTypeSet {
		sqlTypes = append(sqlTypes, sqlType)
	}
	return sqlTypes, nil
}

func classifyStatementType(node ast.Node) storepb.StatementType {
	switch n := node.(type) {
	// CREATE
	case *ast.CreateDatabaseStmt:
		return storepb.StatementType_CREATE_DATABASE
	case *ast.CreateTableStmt:
		return storepb.StatementType_CREATE_TABLE
	case *ast.CreateIndexStmt:
		return storepb.StatementType_CREATE_INDEX
	case *ast.CreateViewStmt:
		return storepb.StatementType_CREATE_VIEW
	case *ast.CreateEventStmt:
		return storepb.StatementType_CREATE_EVENT
	case *ast.CreateTriggerStmt:
		return storepb.StatementType_CREATE_TRIGGER
	case *ast.CreateFunctionStmt:
		if n.IsProcedure {
			return storepb.StatementType_CREATE_PROCEDURE
		}
		return storepb.StatementType_CREATE_FUNCTION

	// DROP
	case *ast.DropDatabaseStmt:
		return storepb.StatementType_DROP_DATABASE
	case *ast.DropTableStmt:
		return storepb.StatementType_DROP_TABLE
	case *ast.DropIndexStmt:
		return storepb.StatementType_DROP_INDEX
	case *ast.DropViewStmt:
		return storepb.StatementType_DROP_VIEW
	case *ast.DropEventStmt:
		return storepb.StatementType_DROP_EVENT
	case *ast.DropTriggerStmt:
		return storepb.StatementType_DROP_TRIGGER
	case *ast.DropRoutineStmt:
		if n.IsProcedure {
			return storepb.StatementType_DROP_PROCEDURE
		}
		return storepb.StatementType_DROP_FUNCTION

	// ALTER
	case *ast.AlterTableStmt:
		return storepb.StatementType_ALTER_TABLE
	case *ast.AlterDatabaseStmt:
		return storepb.StatementType_ALTER_DATABASE
	case *ast.AlterViewStmt:
		return storepb.StatementType_ALTER_VIEW
	case *ast.AlterEventStmt:
		return storepb.StatementType_ALTER_EVENT

	// OTHER DDL
	case *ast.TruncateStmt:
		return storepb.StatementType_TRUNCATE
	case *ast.RenameTableStmt:
		return storepb.StatementType_RENAME

	// DML
	case *ast.InsertStmt:
		return storepb.StatementType_INSERT
	case *ast.UpdateStmt:
		return storepb.StatementType_UPDATE
	case *ast.DeleteStmt:
		return storepb.StatementType_DELETE

	default:
		return storepb.StatementType_STATEMENT_TYPE_UNSPECIFIED
	}
}
