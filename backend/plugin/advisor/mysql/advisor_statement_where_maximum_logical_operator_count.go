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
	_ advisor.Advisor = (*StatementWhereMaximumLogicalOperatorCountAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.MySQLStatementWhereMaximumLogicalOperatorCount, &StatementWhereMaximumLogicalOperatorCountAdvisor{})
}

type StatementWhereMaximumLogicalOperatorCountAdvisor struct {
}

func (*StatementWhereMaximumLogicalOperatorCountAdvisor) Check(ctx advisor.Context, _ string) ([]advisor.Advice, error) {
	stmtList, ok := ctx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to mysql parse result")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}
	payload, err := advisor.UnmarshalNumberTypeRulePayload(ctx.Rule.Payload)
	if err != nil {
		return nil, err
	}
	checker := &statementWhereMaximumLogicalOperatorCountChecker{
		level:   level,
		title:   string(ctx.Rule.Type),
		maximum: payload.Number,
	}

	for _, stmt := range stmtList {
		checker.baseLine = stmt.BaseLine
		checker.reported = false
		antlr.ParseTreeWalkerDefault.Walk(checker, stmt.Tree)
		if checker.maxOrCount > checker.maximum {
			checker.adviceList = append(checker.adviceList, advisor.Advice{
				Status:  checker.level,
				Code:    advisor.StatementWhereMaximumLogicalOperatorCount,
				Title:   checker.title,
				Content: fmt.Sprintf("Number of tokens (%d) in the OR predicate operation exceeds limit (%d) in statement %q.", checker.maxOrCount, checker.maximum, checker.text),
				Line:    checker.maxOrCountLine,
			})
		}
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

type statementWhereMaximumLogicalOperatorCountChecker struct {
	*mysql.BaseMySQLParserListener

	baseLine          int
	adviceList        []advisor.Advice
	level             advisor.Status
	title             string
	text              string
	maximum           int
	reported          bool
	depth             int
	inPredicateExprIn bool
	maxOrCount        int
	maxOrCountLine    int
}

func (checker *statementWhereMaximumLogicalOperatorCountChecker) EnterQuery(ctx *mysql.QueryContext) {
	checker.text = ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx)
}

func (checker *statementWhereMaximumLogicalOperatorCountChecker) EnterPredicateExprIn(_ *mysql.PredicateExprInContext) {
	checker.inPredicateExprIn = true
}

func (checker *statementWhereMaximumLogicalOperatorCountChecker) ExitPredicateExprIn(_ *mysql.PredicateExprInContext) {
	checker.inPredicateExprIn = false
}

func (checker *statementWhereMaximumLogicalOperatorCountChecker) EnterExprList(ctx *mysql.ExprListContext) {
	if checker.reported {
		return
	}
	if !checker.inPredicateExprIn {
		return
	}

	count := len(ctx.AllExpr())
	if count > checker.maximum {
		checker.adviceList = append(checker.adviceList, advisor.Advice{
			Status:  checker.level,
			Code:    advisor.StatementWhereMaximumLogicalOperatorCount,
			Title:   checker.title,
			Content: fmt.Sprintf("Number of tokens (%d) in IN predicate operation exceeds limit (%d) in statement %q.", count, checker.maximum, checker.text),
			Line:    checker.baseLine + ctx.GetStart().GetLine(),
		})
	}
}

func (checker *statementWhereMaximumLogicalOperatorCountChecker) EnterExprOr(ctx *mysql.ExprOrContext) {
	checker.depth++
	count := checker.depth + 1
	if count > checker.maxOrCount {
		checker.maxOrCount = count
		checker.maxOrCountLine = checker.baseLine + ctx.GetStart().GetLine()
	}
}

func (checker *statementWhereMaximumLogicalOperatorCountChecker) ExitExprOr(_ *mysql.ExprOrContext) {
	checker.depth--
}
