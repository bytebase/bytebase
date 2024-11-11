package mysql

import (
	"fmt"

	"github.com/antlr4-go/antlr/v4"
	mysql "github.com/bytebase/mysql-parser"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
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

func (*TableDisallowDMLAdvisor) Check(ctx advisor.Context, _ string) ([]*storepb.Advice, error) {
	list, ok := ctx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to mysql parser result")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}
	payload, err := advisor.UnmarshalStringArrayTypeRulePayload(ctx.Rule.Payload)
	if err != nil {
		return nil, err
	}
	checker := &tableDisallowDMLChecker{
		level:        level,
		title:        string(ctx.Rule.Type),
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
				Status:  checker.level,
				Code:    advisor.TableDisallowDML.Int32(),
				Title:   checker.title,
				Content: fmt.Sprintf("DML is disallowed on table %s.", tableName),
				StartPosition: &storepb.Position{
					Line: int32(line),
				},
			})
			return
		}
	}
}
