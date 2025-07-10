package mysql

import (
	"context"
	"fmt"

	"github.com/antlr4-go/antlr/v4"
	mysql "github.com/bytebase/mysql-parser"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
)

var (
	_ advisor.Advisor = (*TableDisallowDMLAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.MySQLTableDisallowDML, &TableDisallowDMLAdvisor{})
}

// TableDisallowDMLAdvisor is the advisor checking for disallow DML on specific tables.
type TableDisallowDMLAdvisor struct {
}

func (*TableDisallowDMLAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	list, ok := checkCtx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to mysql parser result")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	payload, err := advisor.UnmarshalStringArrayTypeRulePayload(checkCtx.Rule.Payload)
	if err != nil {
		return nil, err
	}
	checker := &tableDisallowDMLChecker{
		level:        level,
		title:        string(checkCtx.Rule.Type),
		disallowList: payload.List,
	}
	for _, stmt := range list {
		checker.baseLine = stmt.BaseLine
		antlr.ParseTreeWalkerDefault.Walk(checker, stmt.Tree)
	}

	return checker.adviceList, nil
}

type tableDisallowDMLChecker struct {
	*mysql.BaseMySQLParserListener

	baseLine   int
	level      storepb.Advice_Status
	title      string
	adviceList []*storepb.Advice
	// disallowList is the list of table names that disallow DML.
	disallowList []string
}

func (checker *tableDisallowDMLChecker) EnterDeleteStatement(ctx *mysql.DeleteStatementContext) {
	_, tableName := mysqlparser.NormalizeMySQLTableRef(ctx.TableRef())
	if tableName == "" {
		return
	}
	checker.checkTableName(tableName, ctx.GetStart().GetLine())
}

func (checker *tableDisallowDMLChecker) EnterInsertStatement(ctx *mysql.InsertStatementContext) {
	_, tableName := mysqlparser.NormalizeMySQLTableRef(ctx.TableRef())
	if tableName == "" {
		return
	}
	checker.checkTableName(tableName, ctx.GetStart().GetLine())
}

func (checker *tableDisallowDMLChecker) EnterSelectStatementWithInto(ctx *mysql.SelectStatementWithIntoContext) {
	// Only check text string literal for now.
	if ctx.IntoClause() == nil || ctx.IntoClause().TextStringLiteral() == nil {
		return
	}
	tableName := ctx.IntoClause().TextStringLiteral().GetText()
	checker.checkTableName(tableName, ctx.GetStart().GetLine())
}

func (checker *tableDisallowDMLChecker) EnterUpdateStatement(ctx *mysql.UpdateStatementContext) {
	if ctx.TableReferenceList() == nil {
		return
	}
	tables, err := extractTableReferenceList(ctx.TableReferenceList())
	if err != nil {
		return
	}
	for _, table := range tables {
		checker.checkTableName(table.table, ctx.GetStart().GetLine())
	}
}

func (checker *tableDisallowDMLChecker) checkTableName(tableName string, line int) {
	for _, disallow := range checker.disallowList {
		if tableName == disallow {
			checker.adviceList = append(checker.adviceList, &storepb.Advice{
				Status:        checker.level,
				Code:          advisor.TableDisallowDML.Int32(),
				Title:         checker.title,
				Content:       fmt.Sprintf("DML is disallowed on table %s.", tableName),
				StartPosition: common.ConvertANTLRLineToPosition(line),
			})
			return
		}
	}
}

type table struct {
	database string
	table    string
}

func extractTableReference(ctx mysql.ITableReferenceContext) ([]table, error) {
	if ctx.TableFactor() == nil {
		return nil, nil
	}
	res, err := extractTableFactor(ctx.TableFactor())
	if err != nil {
		return nil, err
	}
	for _, joinedTableCtx := range ctx.AllJoinedTable() {
		tables, err := extractJoinedTable(joinedTableCtx)
		if err != nil {
			return nil, err
		}
		res = append(res, tables...)
	}

	return res, nil
}

func extractTableRef(ctx mysql.ITableRefContext) ([]table, error) {
	if ctx == nil {
		return nil, nil
	}
	databaseName, tableName := mysqlparser.NormalizeMySQLTableRef(ctx)
	return []table{
		{
			database: databaseName,
			table:    tableName,
		},
	}, nil
}

func extractTableReferenceList(ctx mysql.ITableReferenceListContext) ([]table, error) {
	var res []table
	for _, tableRefCtx := range ctx.AllTableReference() {
		tables, err := extractTableReference(tableRefCtx)
		if err != nil {
			return nil, err
		}
		res = append(res, tables...)
	}
	return res, nil
}

func extractTableReferenceListParens(ctx mysql.ITableReferenceListParensContext) ([]table, error) {
	if ctx.TableReferenceList() != nil {
		return extractTableReferenceList(ctx.TableReferenceList())
	}
	if ctx.TableReferenceListParens() != nil {
		return extractTableReferenceListParens(ctx.TableReferenceListParens())
	}
	return nil, nil
}

func extractTableFactor(ctx mysql.ITableFactorContext) ([]table, error) {
	switch {
	case ctx.SingleTable() != nil:
		return extractSingleTable(ctx.SingleTable())
	case ctx.SingleTableParens() != nil:
		return extractSingleTableParens(ctx.SingleTableParens())
	case ctx.DerivedTable() != nil:
		return nil, nil
	case ctx.TableReferenceListParens() != nil:
		return extractTableReferenceListParens(ctx.TableReferenceListParens())
	case ctx.TableFunction() != nil:
		return nil, nil
	default:
		return nil, nil
	}
}

func extractSingleTable(ctx mysql.ISingleTableContext) ([]table, error) {
	return extractTableRef(ctx.TableRef())
}

func extractSingleTableParens(ctx mysql.ISingleTableParensContext) ([]table, error) {
	if ctx.SingleTable() != nil {
		return extractSingleTable(ctx.SingleTable())
	}
	if ctx.SingleTableParens() != nil {
		return extractSingleTableParens(ctx.SingleTableParens())
	}
	return nil, nil
}

func extractJoinedTable(ctx mysql.IJoinedTableContext) ([]table, error) {
	if ctx.TableFactor() != nil {
		return extractTableFactor(ctx.TableFactor())
	}
	if ctx.TableReference() != nil {
		return extractTableReference(ctx.TableReference())
	}
	return nil, nil
}
