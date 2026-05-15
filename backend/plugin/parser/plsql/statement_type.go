package plsql

import (
	"strings"

	"github.com/antlr4-go/antlr/v4"
	oracleast "github.com/bytebase/omni/oracle/ast"
	parser "github.com/bytebase/parser/plsql"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func GetStatementTypes(asts []base.AST) ([]storepb.StatementType, error) {
	sqlTypeSet := make(map[storepb.StatementType]bool)
	for _, ast := range asts {
		node, ok := GetOmniNode(ast)
		if ok {
			sqlTypeSet[classifyOmniStatementType(node)] = true
			continue
		}

		antlrAST, ok := base.GetANTLRAST(ast)
		if !ok {
			return nil, errors.New("expected Oracle AST")
		}
		t := getStatementType(antlrAST.Tree)
		sqlTypeSet[t] = true
	}
	var sqlTypes []storepb.StatementType
	for sqlType := range sqlTypeSet {
		sqlTypes = append(sqlTypes, sqlType)
	}
	return sqlTypes, nil
}

func classifyOmniStatementType(node oracleast.Node) storepb.StatementType {
	switch n := node.(type) {
	case *oracleast.CreateTableStmt:
		return storepb.StatementType_CREATE_TABLE
	case *oracleast.CreateIndexStmt:
		return storepb.StatementType_CREATE_INDEX
	case *oracleast.CreateViewStmt:
		return storepb.StatementType_CREATE_VIEW
	case *oracleast.CreateSequenceStmt:
		return storepb.StatementType_CREATE_SEQUENCE
	case *oracleast.CreateSchemaStmt:
		return storepb.StatementType_CREATE_SCHEMA
	case *oracleast.CreateFunctionStmt:
		return storepb.StatementType_CREATE_FUNCTION
	case *oracleast.CreateTriggerStmt:
		return storepb.StatementType_CREATE_TRIGGER
	case *oracleast.CreateProcedureStmt:
		return storepb.StatementType_CREATE_PROCEDURE
	case *oracleast.CreateTypeStmt:
		return storepb.StatementType_CREATE_TYPE
	case *oracleast.DropStmt:
		return classifyOmniDropStatementType(n.ObjectType)
	case *oracleast.AdminDDLStmt:
		return classifyOmniAdminDDLStatementType(n.Action, n.ObjectType)
	case *oracleast.AlterTableStmt:
		return storepb.StatementType_ALTER_TABLE
	case *oracleast.AlterIndexStmt:
		return storepb.StatementType_ALTER_INDEX
	case *oracleast.AlterViewStmt:
		return storepb.StatementType_ALTER_VIEW
	case *oracleast.AlterSequenceStmt:
		return storepb.StatementType_ALTER_SEQUENCE
	case *oracleast.AlterTypeStmt:
		return storepb.StatementType_ALTER_TYPE
	case *oracleast.TruncateStmt:
		return storepb.StatementType_TRUNCATE
	case *oracleast.RenameStmt:
		return storepb.StatementType_RENAME
	case *oracleast.CommentStmt:
		return storepb.StatementType_COMMENT
	case *oracleast.InsertStmt:
		return storepb.StatementType_INSERT
	case *oracleast.UpdateStmt:
		return storepb.StatementType_UPDATE
	case *oracleast.DeleteStmt:
		return storepb.StatementType_DELETE
	default:
		return storepb.StatementType_STATEMENT_TYPE_UNSPECIFIED
	}
}

func classifyOmniAdminDDLStatementType(action string, objectType oracleast.ObjectType) storepb.StatementType {
	if objectType != oracleast.OBJECT_DATABASE {
		return storepb.StatementType_STATEMENT_TYPE_UNSPECIFIED
	}
	switch strings.ToUpper(action) {
	case "CREATE":
		return storepb.StatementType_CREATE_DATABASE
	case "ALTER":
		return storepb.StatementType_ALTER_DATABASE
	case "DROP":
		return storepb.StatementType_DROP_DATABASE
	default:
		return storepb.StatementType_STATEMENT_TYPE_UNSPECIFIED
	}
}

func classifyOmniDropStatementType(objectType oracleast.ObjectType) storepb.StatementType {
	switch objectType {
	case oracleast.OBJECT_DATABASE:
		return storepb.StatementType_DROP_DATABASE
	case oracleast.OBJECT_TABLE:
		return storepb.StatementType_DROP_TABLE
	case oracleast.OBJECT_VIEW, oracleast.OBJECT_MATERIALIZED_VIEW:
		return storepb.StatementType_DROP_VIEW
	case oracleast.OBJECT_INDEX:
		return storepb.StatementType_DROP_INDEX
	case oracleast.OBJECT_SEQUENCE:
		return storepb.StatementType_DROP_SEQUENCE
	case oracleast.OBJECT_FUNCTION:
		return storepb.StatementType_DROP_FUNCTION
	case oracleast.OBJECT_TRIGGER:
		return storepb.StatementType_DROP_TRIGGER
	case oracleast.OBJECT_PROCEDURE:
		return storepb.StatementType_DROP_PROCEDURE
	case oracleast.OBJECT_TYPE, oracleast.OBJECT_TYPE_BODY:
		return storepb.StatementType_DROP_TYPE
	default:
		return storepb.StatementType_STATEMENT_TYPE_UNSPECIFIED
	}
}

func getStatementType(node antlr.Tree) storepb.StatementType {
	switch ctx := node.(type) {
	case *parser.Sql_scriptContext, *parser.Unit_statementContext, *parser.Data_manipulation_language_statementsContext:
		for _, child := range ctx.GetChildren() {
			return getStatementType(child)
		}
	case *parser.Alter_databaseContext:
		return storepb.StatementType_ALTER_DATABASE
	case *parser.Alter_indexContext:
		return storepb.StatementType_ALTER_INDEX
	case *parser.Alter_tableContext:
		return storepb.StatementType_ALTER_TABLE
	case *parser.Create_databaseContext:
		return storepb.StatementType_CREATE_DATABASE
	case *parser.Create_indexContext:
		return storepb.StatementType_CREATE_INDEX
	case *parser.Create_tableContext:
		return storepb.StatementType_CREATE_TABLE
	case *parser.Create_viewContext:
		return storepb.StatementType_CREATE_VIEW
	case *parser.Drop_databaseContext:
		return storepb.StatementType_DROP_DATABASE
	case *parser.Drop_indexContext:
		return storepb.StatementType_DROP_INDEX
	case *parser.Drop_tableContext:
		return storepb.StatementType_DROP_TABLE
	case *parser.Drop_viewContext:
		return storepb.StatementType_DROP_VIEW
	case *parser.Truncate_tableContext:
		return storepb.StatementType_TRUNCATE
	case *parser.Delete_statementContext:
		return storepb.StatementType_DELETE
	case *parser.Insert_statementContext:
		return storepb.StatementType_INSERT
	case *parser.Update_statementContext:
		return storepb.StatementType_UPDATE
	default:
	}
	return storepb.StatementType_STATEMENT_TYPE_UNSPECIFIED
}
