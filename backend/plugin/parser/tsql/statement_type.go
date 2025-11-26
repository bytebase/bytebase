package tsql

import (
	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/tsql"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func GetStatementTypes(asts []base.AST) ([]string, error) {
	sqlTypeSet := make(map[string]bool)
	for _, ast := range asts {
		antlrAST, ok := base.GetANTLRAST(ast)
		if !ok {
			return nil, errors.New("expected ANTLR AST for MSSQL")
		}
		for _, child := range antlrAST.Tree.GetChildren() {
			_, ok := child.(*antlr.TerminalNodeImpl)
			if ok {
				continue
			}

			t := getStatementType(child)
			sqlTypeSet[t] = true
		}
	}
	var sqlTypes []string
	for sqlType := range sqlTypeSet {
		sqlTypes = append(sqlTypes, sqlType)
	}
	return sqlTypes, nil
}

func getStatementType(node antlr.Tree) string {
	switch ctx := node.(type) {
	case *parser.Tsql_fileContext, *parser.Batch_without_goContext, *parser.Batch_level_statementContext:
		for _, child := range ctx.GetChildren() {
			return getStatementType(child)
		}
	case *parser.Sql_clausesContext:
		for _, child := range ctx.GetChildren() {
			switch ctx := child.(type) {
			case *parser.Ddl_clauseContext:
				for _, child := range ctx.GetChildren() {
					return getStatementType(child)
				}
			case *parser.Dml_clauseContext:
				for _, child := range ctx.GetChildren() {
					return getStatementType(child)
				}
			}
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
	case *parser.Create_schemaContext:
		return "CREATE_SCHEMA"
	case *parser.Create_tableContext:
		return "CREATE_TABLE"
	case *parser.Create_viewContext:
		return "CREATE_VIEW"
	case *parser.Drop_databaseContext:
		return "DROP_DATABASE"
	case *parser.Drop_indexContext:
		return "DROP_INDEX"
	case *parser.Drop_schemaContext:
		return "DROP_SCHEMA"
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
