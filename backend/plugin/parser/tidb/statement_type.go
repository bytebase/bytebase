package tidb

import (
	tidbast "github.com/pingcap/tidb/pkg/parser/ast"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func GetStatementTypes(asts []base.AST) ([]storepb.StatementType, error) {
	sqlTypeSet := make(map[storepb.StatementType]bool)
	for _, ast := range asts {
		tidbAST, ok := GetTiDBAST(ast)
		if !ok {
			return nil, errors.New("expected TiDB AST")
		}
		node := tidbAST.Node
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
func getStatementType(stmt tidbast.StmtNode) storepb.StatementType {
	switch v := stmt.(type) {
	// DDL statements
	case *tidbast.CreateDatabaseStmt:
		return storepb.StatementType_CREATE_DATABASE
	case *tidbast.CreateTableStmt:
		return storepb.StatementType_CREATE_TABLE
	case *tidbast.CreateViewStmt:
		return storepb.StatementType_CREATE_VIEW
	case *tidbast.CreateIndexStmt:
		return storepb.StatementType_CREATE_INDEX

	case *tidbast.DropDatabaseStmt:
		return storepb.StatementType_DROP_DATABASE
	case *tidbast.DropTableStmt:
		if v.IsView {
			return storepb.StatementType_DROP_VIEW
		}
		return storepb.StatementType_DROP_TABLE
	case *tidbast.DropIndexStmt:
		return storepb.StatementType_DROP_INDEX

	case *tidbast.AlterDatabaseStmt:
		return storepb.StatementType_ALTER_DATABASE
	case *tidbast.AlterTableStmt:
		return storepb.StatementType_ALTER_TABLE

	case *tidbast.TruncateTableStmt:
		return storepb.StatementType_TRUNCATE
	case *tidbast.RenameTableStmt:
		return storepb.StatementType_RENAME

	// DML statements
	case *tidbast.DeleteStmt:
		return storepb.StatementType_DELETE
	case *tidbast.InsertStmt:
		return storepb.StatementType_INSERT
	case *tidbast.UpdateStmt:
		return storepb.StatementType_UPDATE
	}

	return storepb.StatementType_STATEMENT_TYPE_UNSPECIFIED
}
