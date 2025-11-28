package tidb

import (
	tidbast "github.com/pingcap/tidb/pkg/parser/ast"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func GetStatementTypes(asts []base.AST) ([]string, error) {
	sqlTypeSet := make(map[string]bool)
	for _, ast := range asts {
		tidbAST, ok := GetTiDBAST(ast)
		if !ok {
			return nil, errors.New("expected TiDB AST")
		}
		node := tidbAST.Node
		t := getStatementType(node)
		sqlTypeSet[t] = true
	}
	var sqlTypes []string
	for sqlType := range sqlTypeSet {
		sqlTypes = append(sqlTypes, sqlType)
	}
	return sqlTypes, nil
}

// getStatementType returns the type of statement.
func getStatementType(stmt tidbast.StmtNode) string {
	switch v := stmt.(type) {
	// DDL statements
	case *tidbast.CreateDatabaseStmt:
		return "CREATE_DATABASE"
	case *tidbast.CreateTableStmt:
		return "CREATE_TABLE"
	case *tidbast.CreateViewStmt:
		return "CREATE_VIEW"
	case *tidbast.CreateIndexStmt:
		return "CREATE_INDEX"

	case *tidbast.DropDatabaseStmt:
		return "DROP_DATABASE"
	case *tidbast.DropTableStmt:
		if v.IsView {
			return "DROP_VIEW"
		}
		return "DROP_TABLE"
	case *tidbast.DropIndexStmt:
		return "DROP_INDEX"

	case *tidbast.AlterDatabaseStmt:
		return "ALTER_DATABASE"
	case *tidbast.AlterTableStmt:
		return "ALTER_TABLE"

	case *tidbast.TruncateTableStmt:
		return "TRUNCATE"
	case *tidbast.RenameTableStmt:
		return "RENAME"

	// DML statements
	case *tidbast.DeleteStmt:
		return "DELETE"
	case *tidbast.InsertStmt:
		return "INSERT"
	case *tidbast.UpdateStmt:
		return "UPDATE"
	}

	return "UNKNOWN"
}
