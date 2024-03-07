package mysql

import (
	"fmt"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	mysql "github.com/bytebase/mysql-parser"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	_ advisor.Advisor = (*StatementWhereNoEqualNullAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.MySQLStatementWhereNoEqualNull, &StatementWhereNoEqualNullAdvisor{})
}

type StatementWhereNoEqualNullAdvisor struct {
}

func (*StatementWhereNoEqualNullAdvisor) Check(ctx advisor.Context, _ string) ([]advisor.Advice, error) {
	stmtList, ok := ctx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to mysql parse result")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}
	checker := &statementWhereNoEqualNullChecker{
		level: level,
		title: string(ctx.Rule.Type),
	}

	for _, stmt := range stmtList {
		checker.baseLine = stmt.BaseLine
		antlr.ParseTreeWalkerDefault.Walk(checker, stmt.Tree)
	}

	if len(checker.adviceList) == 0 {
		checker.adviceList = append(checker.adviceList, advisor.Advice{
			Status:  advisor.Success,
			Code:    advisor.Ok,
			Title:   "OK",
			Content: "",
		})
	}
	return checker.adviceList, nil
}

type statementWhereNoEqualNullChecker struct {
	*mysql.BaseMySQLParserListener

	baseLine   int
	adviceList []advisor.Advice
	level      advisor.Status
	title      string
	text       string
	isSelect   bool
}

func (checker *statementWhereNoEqualNullChecker) EnterQuery(ctx *mysql.QueryContext) {
	checker.text = ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx)
}

func (checker *statementWhereNoEqualNullChecker) EnterSelectStatement(*mysql.SelectStatementContext) {
	checker.isSelect = true
}

func (checker *statementWhereNoEqualNullChecker) ExitSelectStatement(*mysql.SelectStatementContext) {
	checker.isSelect = false
}

func (checker *statementWhereNoEqualNullChecker) EnterPrimaryExprCompare(ctx *mysql.PrimaryExprCompareContext) {
	if !checker.isSelect {
		return
	}

	compOp := ctx.CompOp()
	// We only check for equal and not equal.
	if compOp == nil || (compOp.EQUAL_OPERATOR() == nil && compOp.NOT_EQUAL_OPERATOR() == nil) {
		return
	}
	if ctx.Predicate() != nil && ctx.Predicate().GetText() == "NULL" {
		checker.adviceList = append(checker.adviceList, advisor.Advice{
			Status:  checker.level,
			Code:    advisor.StatementWhereNoEqualNull,
			Title:   checker.title,
			Content: fmt.Sprintf("WHERE clause contains equal null: %s", checker.text),
			Line:    checker.baseLine + ctx.GetStart().GetLine(),
		})
	}
}
