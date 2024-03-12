package mysql

import (
	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	mysql "github.com/bytebase/mysql-parser"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	_ advisor.Advisor = (*StatementQueryMinumumPlanLevelAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.MySQLStatementQueryMinumumPlanLevel, &StatementQueryMinumumPlanLevelAdvisor{})
}

type StatementQueryMinumumPlanLevelAdvisor struct {
}

func (*StatementQueryMinumumPlanLevelAdvisor) Check(ctx advisor.Context, _ string) ([]advisor.Advice, error) {
	stmtList, ok := ctx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to mysql parse result")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}
	checker := &statementQueryMinumumPlanLevelChecker{
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

// TODO(sql-review): implement me please.
type statementQueryMinumumPlanLevelChecker struct {
	*mysql.BaseMySQLParserListener

	baseLine   int
	adviceList []advisor.Advice
	level      advisor.Status
	title      string
	text       string
}

func (checker *statementQueryMinumumPlanLevelChecker) EnterQuery(ctx *mysql.QueryContext) {
	checker.text = ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx)
}
