package mysql

import (
	"github.com/bytebase/parser/mysql"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func GetStatementTypes(asts []base.AST) ([]string, error) {
	sqlTypeSet := make(map[string]bool)
	for _, ast := range asts {
		antlrAST, ok := base.GetANTLRAST(ast)
		if !ok {
			return nil, errors.New("expected ANTLR AST for MySQL")
		}
		t := getStatementType(antlrAST)
		sqlTypeSet[t] = true
	}
	var sqlTypes []string
	for sqlType := range sqlTypeSet {
		sqlTypes = append(sqlTypes, sqlType)
	}
	return sqlTypes, nil
}

// GetStatementType return the type of statement.
func getStatementType(stmt *base.ANTLRAST) string {
	for _, child := range stmt.Tree.GetChildren() {
		switch ctx := child.(type) {
		case *mysql.QueryContext:
			for _, child := range ctx.GetChildren() {
				switch ctx := child.(type) {
				case *mysql.SimpleStatementContext:
					for _, child := range ctx.GetChildren() {
						switch ctx := child.(type) {
						case *mysql.CreateStatementContext:
							for _, child := range ctx.GetChildren() {
								switch child.(type) {
								case *mysql.CreateDatabaseContext:
									return "CREATE_DATABASE"
								case *mysql.CreateIndexContext:
									return "CREATE_INDEX"
								case *mysql.CreateTableContext:
									return "CREATE_TABLE"
								case *mysql.CreateViewContext:
									return "CREATE_VIEW"
								case *mysql.CreateEventContext:
									return "CREATE_EVENT"
								case *mysql.CreateTriggerContext:
									return "CREATE_TRIGGER"
								case *mysql.CreateFunctionContext:
									return "CREATE_FUNCTION"
								case *mysql.CreateProcedureContext:
									return "CREATE_PROCEDURE"
								}
							}
						case *mysql.DropStatementContext:
							for _, child := range ctx.GetChildren() {
								switch child.(type) {
								case *mysql.DropIndexContext:
									return "DROP_INDEX"
								case *mysql.DropTableContext:
									return "DROP_TABLE"
								case *mysql.DropDatabaseContext:
									return "DROP_DATABASE"
								case *mysql.DropViewContext:
									return "DROP_VIEW"
								case *mysql.DropTriggerContext:
									return "DROP_TRIGGER"
								case *mysql.DropEventContext:
									return "DROP_EVENT"
								case *mysql.DropFunctionContext:
									return "DROP_FUNCTION"
								case *mysql.DropProcedureContext:
									return "DROP_PROCEDURE"
								}
							}
						case *mysql.AlterStatementContext:
							for _, child := range ctx.GetChildren() {
								switch child.(type) {
								case *mysql.AlterTableContext:
									return "ALTER_TABLE"
								case *mysql.AlterDatabaseContext:
									return "ALTER_DATABASE"
								case *mysql.AlterViewContext:
									return "ALTER_VIEW"
								case *mysql.AlterEventContext:
									return "ALTER_EVENT"
								}
							}
						case *mysql.TruncateTableStatementContext:
							return "TRUNCATE"
						case *mysql.RenameTableStatementContext:
							return "RENAME"

						// dml.
						case *mysql.DeleteStatementContext:
							return "DELETE"
						case *mysql.InsertStatementContext:
							return "INSERT"
						case *mysql.UpdateStatementContext:
							return "UPDATE"
						}
					}
				default:
				}
			}
		default:
		}
	}
	return "UNKNOWN"
}
