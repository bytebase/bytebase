package mysql

import (
	"context"

	"github.com/antlr4-go/antlr/v4"
	mysql "github.com/bytebase/mysql-parser"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// GetStatementType return the type of statement.
func GetStatementType(stmt *ParseResult) string {
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

type AffectedRowsListener struct {
	*mysql.BaseMySQLParserListener

	ctx                        context.Context
	text                       string
	getAffectedRowsByQueryFunc base.GetAffectedRowsCountByQueryFunc
	getTableDataSizeFunc       base.GetTableDataSizeFunc
	affectedRows               int64
	err                        error
}

// GetAffectedRows return the rows count affected by the sql.
func GetAffectedRows(ctx context.Context, stmt *ParseResult, getAffectedRowsByQuery base.GetAffectedRowsCountByQueryFunc, getTableDataSizeFunc base.GetTableDataSizeFunc) (int64, error) {
	listener := &AffectedRowsListener{
		ctx:                        ctx,
		getAffectedRowsByQueryFunc: getAffectedRowsByQuery,
		getTableDataSizeFunc:       getTableDataSizeFunc,
	}
	antlr.ParseTreeWalkerDefault.Walk(listener, stmt.Tree)
	if listener.err != nil {
		return 0, listener.err
	}

	return listener.affectedRows, nil
}

func (l *AffectedRowsListener) EnterQuery(ctx *mysql.QueryContext) {
	l.text = ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx)
}

// EnterInsertStatement is called when production insertStatement is entered.
func (l *AffectedRowsListener) EnterInsertStatement(ctx *mysql.InsertStatementContext) {
	if ctx.GetParent() == nil {
		return
	}
	simpleCtx, ok := ctx.GetParent().(*mysql.SimpleStatementContext)
	if !ok || simpleCtx.GetParent() == nil {
		return
	}
	if _, ok := simpleCtx.GetParent().(*mysql.QueryContext); !ok {
		return
	}

	if ctx.InsertQueryExpression() != nil && l.getAffectedRowsByQueryFunc != nil {
		affectedRows, err := l.getAffectedRowsByQueryFunc(l.ctx, l.text)
		if err != nil {
			l.err = err
			return
		}
		l.affectedRows = affectedRows
		return
	}

	if ctx.InsertFromConstructor() == nil || ctx.InsertFromConstructor().InsertValues() == nil || ctx.InsertFromConstructor().InsertValues().ValueList() == nil {
		return
	}

	l.affectedRows = int64(len(ctx.InsertFromConstructor().InsertValues().ValueList().AllValues()))
}

// EnterUpdateStatement is called when production updateStatement is entered.
func (l *AffectedRowsListener) EnterUpdateStatement(ctx *mysql.UpdateStatementContext) {
	if ctx.GetParent() == nil {
		return
	}
	simpleCtx, ok := ctx.GetParent().(*mysql.SimpleStatementContext)
	if !ok || simpleCtx.GetParent() == nil {
		return
	}
	if _, ok := simpleCtx.GetParent().(*mysql.QueryContext); !ok {
		return
	}
	if l.getAffectedRowsByQueryFunc == nil {
		return
	}
	affectedRows, err := l.getAffectedRowsByQueryFunc(l.ctx, l.text)
	if err != nil {
		l.err = err
	}
	l.affectedRows = affectedRows
}

// EnterDeleteStatement is called when production deleteStatement is entered.
func (l *AffectedRowsListener) EnterDeleteStatement(ctx *mysql.DeleteStatementContext) {
	if ctx.GetParent() == nil {
		return
	}
	simpleCtx, ok := ctx.GetParent().(*mysql.SimpleStatementContext)
	if !ok || simpleCtx.GetParent() == nil {
		return
	}
	if _, ok := simpleCtx.GetParent().(*mysql.QueryContext); !ok {
		return
	}
	if l.getAffectedRowsByQueryFunc == nil {
		return
	}
	affectedRows, err := l.getAffectedRowsByQueryFunc(l.ctx, l.text)
	if err != nil {
		l.err = err
	}
	l.affectedRows = affectedRows
}

// EnterAlterTable is called when production alterTable is entered.
func (l *AffectedRowsListener) EnterAlterTable(ctx *mysql.AlterTableContext) {
	if ctx.GetParent() == nil {
		return
	}
	alertCtx, ok := ctx.GetParent().(*mysql.AlterStatementContext)
	if !ok || alertCtx.GetParent() == nil {
		return
	}
	simpleCtx, ok := alertCtx.GetParent().(*mysql.SimpleStatementContext)
	if !ok || simpleCtx.GetParent() == nil {
		return
	}
	if _, ok := simpleCtx.GetParent().(*mysql.QueryContext); !ok {
		return
	}

	if ctx.TableRef() == nil {
		return
	}
	databaseName, tableName := NormalizeMySQLTableRef(ctx.TableRef())
	if l.getTableDataSizeFunc == nil {
		return
	}
	l.affectedRows = l.getTableDataSizeFunc(databaseName, tableName)
}

// EnterDropTable is called when production dropTable is entered.
func (l *AffectedRowsListener) EnterDropTable(ctx *mysql.DropTableContext) {
	if ctx.GetParent() == nil {
		return
	}

	dropCtx, ok := ctx.GetParent().(*mysql.DropStatementContext)
	if !ok || dropCtx.GetParent() == nil {
		return
	}
	simpleCtx, ok := dropCtx.GetParent().(*mysql.SimpleStatementContext)
	if !ok || simpleCtx.GetParent() == nil {
		return
	}
	if _, ok := simpleCtx.GetParent().(*mysql.QueryContext); !ok {
		return
	}

	if ctx.TableRefList() == nil {
		return
	}
	for _, tableRef := range ctx.TableRefList().AllTableRef() {
		if tableRef == nil {
			continue
		}
		databaseName, tableName := NormalizeMySQLTableRef(tableRef)
		if l.getTableDataSizeFunc == nil {
			return
		}
		l.affectedRows += l.getTableDataSizeFunc(databaseName, tableName)
	}
}
