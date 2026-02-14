package plsql

import (
	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/plsql"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func GetStatementTypes(asts []base.AST) ([]storepb.StatementType, error) {
	sqlTypeSet := make(map[storepb.StatementType]bool)
	for _, ast := range asts {
		antlrAST, ok := base.GetANTLRAST(ast)
		if !ok {
			return nil, errors.New("expected ANTLR AST for Oracle")
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
