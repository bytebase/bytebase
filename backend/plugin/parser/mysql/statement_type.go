package mysql

import (
	"github.com/bytebase/parser/mysql"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func GetStatementTypes(asts []base.AST) ([]storepb.StatementType, error) {
	sqlTypeSet := make(map[storepb.StatementType]bool)
	for _, ast := range asts {
		antlrAST, ok := base.GetANTLRAST(ast)
		if !ok {
			return nil, errors.New("expected ANTLR AST for MySQL")
		}
		t := getStatementType(antlrAST)
		sqlTypeSet[t] = true
	}
	var sqlTypes []storepb.StatementType
	for sqlType := range sqlTypeSet {
		sqlTypes = append(sqlTypes, sqlType)
	}
	return sqlTypes, nil
}

// GetStatementType return the type of statement.
func getStatementType(stmt *base.ANTLRAST) storepb.StatementType {
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
									return storepb.StatementType_CREATE_DATABASE
								case *mysql.CreateIndexContext:
									return storepb.StatementType_CREATE_INDEX
								case *mysql.CreateTableContext:
									return storepb.StatementType_CREATE_TABLE
								case *mysql.CreateViewContext:
									return storepb.StatementType_CREATE_VIEW
								case *mysql.CreateEventContext:
									return storepb.StatementType_CREATE_EVENT
								case *mysql.CreateTriggerContext:
									return storepb.StatementType_CREATE_TRIGGER
								case *mysql.CreateFunctionContext:
									return storepb.StatementType_CREATE_FUNCTION
								case *mysql.CreateProcedureContext:
									return storepb.StatementType_CREATE_PROCEDURE
								}
							}
						case *mysql.DropStatementContext:
							for _, child := range ctx.GetChildren() {
								switch child.(type) {
								case *mysql.DropIndexContext:
									return storepb.StatementType_DROP_INDEX
								case *mysql.DropTableContext:
									return storepb.StatementType_DROP_TABLE
								case *mysql.DropDatabaseContext:
									return storepb.StatementType_DROP_DATABASE
								case *mysql.DropViewContext:
									return storepb.StatementType_DROP_VIEW
								case *mysql.DropTriggerContext:
									return storepb.StatementType_DROP_TRIGGER
								case *mysql.DropEventContext:
									return storepb.StatementType_DROP_EVENT
								case *mysql.DropFunctionContext:
									return storepb.StatementType_DROP_FUNCTION
								case *mysql.DropProcedureContext:
									return storepb.StatementType_DROP_PROCEDURE
								}
							}
						case *mysql.AlterStatementContext:
							for _, child := range ctx.GetChildren() {
								switch child.(type) {
								case *mysql.AlterTableContext:
									return storepb.StatementType_ALTER_TABLE
								case *mysql.AlterDatabaseContext:
									return storepb.StatementType_ALTER_DATABASE
								case *mysql.AlterViewContext:
									return storepb.StatementType_ALTER_VIEW
								case *mysql.AlterEventContext:
									return storepb.StatementType_ALTER_EVENT
								}
							}
						case *mysql.TruncateTableStatementContext:
							return storepb.StatementType_TRUNCATE
						case *mysql.RenameTableStatementContext:
							return storepb.StatementType_RENAME

						// dml.
						case *mysql.DeleteStatementContext:
							return storepb.StatementType_DELETE
						case *mysql.InsertStatementContext:
							return storepb.StatementType_INSERT
						case *mysql.UpdateStatementContext:
							return storepb.StatementType_UPDATE
						default:
						}
					}
				default:
				}
			}
		default:
		}
	}
	return storepb.StatementType_STATEMENT_TYPE_UNSPECIFIED
}
