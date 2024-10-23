package mysql

import (
	"github.com/antlr4-go/antlr/v4"
	mysql "github.com/bytebase/mysql-parser"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func init() {
	base.RegisterGetQueryType(storepb.Engine_MYSQL, GetQueryType)
	base.RegisterGetQueryType(storepb.Engine_MARIADB, GetQueryType)
	base.RegisterGetQueryType(storepb.Engine_OCEANBASE, GetQueryType)
	base.RegisterGetQueryType(storepb.Engine_STARROCKS, GetQueryType)
	base.RegisterGetQueryType(storepb.Engine_DORIS, GetQueryType)
}

func GetQueryType(statement string) (base.QueryType, error) {
	parseResult, err := ParseMySQL(statement)
	if err != nil {
		return base.QueryTypeUnknown, err
	}

	if len(parseResult) == 0 {
		return base.QueryTypeUnknown, nil
	}
	if len(parseResult) > 1 {
		return base.QueryTypeUnknown, errors.Errorf("expecting only one statement, but got %d", len(parseResult))
	}

	tree := parseResult[0].Tree

	listener := &QueryTypeListener{
		result: base.QueryTypeUnknown,
	}
	antlr.ParseTreeWalkerDefault.Walk(listener, tree)
	return listener.result, nil
}

type QueryTypeListener struct {
	*mysql.BaseMySQLParserListener

	result base.QueryType
}

func (l *QueryTypeListener) EnterSimpleStatement(ctx *mysql.SimpleStatementContext) {
	if !isTopLevel(ctx) {
		return
	}

	switch {
	case ctx.AlterStatement() != nil,
		ctx.CreateStatement() != nil,
		ctx.DropStatement() != nil,
		ctx.RenameTableStatement() != nil,
		ctx.TruncateTableStatement() != nil,
		ctx.ImportStatement() != nil:
		l.result = base.DDL
	case ctx.CallStatement() != nil,
		ctx.DeleteStatement() != nil,
		ctx.DoStatement() != nil,
		ctx.HandlerStatement() != nil,
		ctx.InsertStatement() != nil,
		ctx.LoadStatement() != nil,
		ctx.ReplaceStatement() != nil,
		ctx.UpdateStatement() != nil,
		ctx.TransactionOrLockingStatement() != nil,
		ctx.ReplicationStatement() != nil,
		ctx.PreparedStatement() != nil:
		l.result = base.DML
	case ctx.ShowStatement() != nil:
		l.result = base.SelectInfoSchema
	case ctx.UtilityStatement() != nil:
		if ctx.UtilityStatement().DescribeStatement() != nil {
			l.result = base.SelectInfoSchema
		}
		if ctx.UtilityStatement().ExplainStatement() != nil {
			l.result = base.Explain
		}
	}
}

func (l *QueryTypeListener) EnterSelectStatement(ctx *mysql.SelectStatementContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	// MySQL cannot use SELECT ... INTO .. FROM ... syntax to create a new table or insert into an existing table.
	// So we can safely assume it's a SELECT statement.

	accessTables := getAccessTables("", ctx)
	allSystems, _ := isMixedQuery(accessTables, true)
	if allSystems {
		l.result = base.SelectInfoSchema
	} else {
		l.result = base.Select
	}
}
