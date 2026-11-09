package tsql

import (
	"strings"

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
	case *ast.ExecStmt:
		if isProcedureCall(n) {
			return storepb.StatementType_EXECUTE
		}
	default:
	}
	return storepb.StatementType_STATEMENT_TYPE_UNSPECIFIED
}

// isProcedureCall reports whether an EXEC runs a user procedure by name. A
// sql_type rule can route EXECUTE to a lighter approval path, so EXEC of a SQL
// string stays unclassified because it can run any statement. System procedures
// stay unclassified for the same reason: sp_executesql runs dynamic SQL, and
// others such as sp_rename and xp_cmdshell change schema or reach the host.
// They are matched by the sp_ and xp_ prefixes in any schema, so a user
// procedure named with those prefixes is also left unclassified.
func isProcedureCall(n *ast.ExecStmt) bool {
	if n.ExecString != nil || n.Name == nil {
		return false
	}
	if strings.EqualFold(n.Name.Schema, "sys") {
		return false
	}
	object := strings.ToLower(n.Name.Object)
	return !strings.HasPrefix(object, "sp_") && !strings.HasPrefix(object, "xp_")
}
