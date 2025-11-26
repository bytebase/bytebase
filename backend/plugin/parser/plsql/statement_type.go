package plsql

import (
	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/plsql"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func GetStatementTypes(asts []base.AST) ([]string, error) {
	sqlTypeSet := make(map[string]bool)
	for _, ast := range asts {
		antlrAST, ok := base.GetANTLRAST(ast)
		if !ok {
			return nil, errors.New("expected ANTLR AST for Oracle")
		}
		t := getStatementType(antlrAST.Tree)
		sqlTypeSet[t] = true
	}
	var sqlTypes []string
	for sqlType := range sqlTypeSet {
		sqlTypes = append(sqlTypes, sqlType)
	}
	return sqlTypes, nil
}

func getStatementType(node antlr.Tree) string {
	switch ctx := node.(type) {
	case *parser.Sql_scriptContext, *parser.Unit_statementContext, *parser.Data_manipulation_language_statementsContext:
		for _, child := range ctx.GetChildren() {
			return getStatementType(child)
		}
	case *parser.Alter_databaseContext:
		return "ALTER_DATABASE"
	case *parser.Alter_indexContext:
		return "ALTER_INDEX"
	case *parser.Alter_tableContext:
		return "ALTER_TABLE"
	case *parser.Create_databaseContext:
		return "CREATE_DATABASE"
	case *parser.Create_indexContext:
		return "CREATE_INDEX"
	case *parser.Create_tableContext:
		return "CREATE_TABLE"
	case *parser.Create_viewContext:
		return "CREATE_VIEW"
	case *parser.Drop_databaseContext:
		return "DROP_DATABASE"
	case *parser.Drop_indexContext:
		return "DROP_INDEX"
	case *parser.Drop_tableContext:
		return "DROP_TABLE"
	case *parser.Drop_viewContext:
		return "DROP_VIEW"
	case *parser.Truncate_tableContext:
		return "TRUNCATE_TABLE"
	case *parser.Delete_statementContext:
		return "DELETE"
	case *parser.Insert_statementContext:
		return "INSERT"
	case *parser.Update_statementContext:
		return "UPDATE"
	}
	return "UNKNOWN"
}
