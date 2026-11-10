package tidb

import (
	omniast "github.com/bytebase/omni/tidb/ast"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func GetStatementTypes(asts []base.AST) ([]storepb.StatementType, error) {
	sqlTypeSet := make(map[storepb.StatementType]bool)
	for _, ast := range asts {
		node, ok := GetNode(ast)
		if !ok {
			return nil, errors.New("expected TiDB AST")
		}
		t := getStatementType(node)
		sqlTypeSet[t] = true
	}
	var sqlTypes []storepb.StatementType
	for sqlType := range sqlTypeSet {
		sqlTypes = append(sqlTypes, sqlType)
	}
	return sqlTypes, nil
}

// getStatementType returns the type of statement.
func getStatementType(stmt omniast.Node) storepb.StatementType {
	switch stmt.(type) {
	// DDL statements
	case *omniast.CreateDatabaseStmt:
		return storepb.StatementType_CREATE_DATABASE
	case *omniast.CreateTableStmt:
		return storepb.StatementType_CREATE_TABLE
	case *omniast.CreateViewStmt:
		return storepb.StatementType_CREATE_VIEW
	case *omniast.CreateIndexStmt:
		return storepb.StatementType_CREATE_INDEX

	case *omniast.DropDatabaseStmt:
		return storepb.StatementType_DROP_DATABASE
	case *omniast.DropTableStmt:
		return storepb.StatementType_DROP_TABLE
	case *omniast.DropViewStmt:
		return storepb.StatementType_DROP_VIEW
	case *omniast.DropIndexStmt:
		return storepb.StatementType_DROP_INDEX

	case *omniast.AlterDatabaseStmt:
		return storepb.StatementType_ALTER_DATABASE
	case *omniast.AlterTableStmt:
		return storepb.StatementType_ALTER_TABLE

	case *omniast.TruncateStmt:
		return storepb.StatementType_TRUNCATE
	case *omniast.RenameTableStmt:
		return storepb.StatementType_RENAME

	// DML statements
	case *omniast.DeleteStmt:
		return storepb.StatementType_DELETE
	case *omniast.InsertStmt:
		return storepb.StatementType_INSERT
	case *omniast.UpdateStmt:
		return storepb.StatementType_UPDATE

	// A procedure call or a prepared statement runs code the statement text does
	// not show. PREPARE only defines the statement, so it stays unclassified.
	case *omniast.CallStmt, *omniast.ExecuteStmt:
		return storepb.StatementType_EXECUTE
	}

	return storepb.StatementType_STATEMENT_TYPE_UNSPECIFIED
}
