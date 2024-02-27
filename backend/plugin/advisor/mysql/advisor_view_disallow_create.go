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
	_ advisor.Advisor = (*ViewDisallowCreateAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.MySQLViewDisallowCreate, &ViewDisallowCreateAdvisor{})
}

// ViewDisallowCreateAdvisor is the advisor checking for disallow creating view.
type ViewDisallowCreateAdvisor struct {
}

// Check checks for disallow creating view.
func (*ViewDisallowCreateAdvisor) Check(ctx advisor.Context, _ string) ([]advisor.Advice, error) {
	stmtList, ok := ctx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to mysql parser result")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}
	checker := &viewDisallowCreateChecker{
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

type viewDisallowCreateChecker struct {
	*mysql.BaseMySQLParserListener

	baseLine   int
	adviceList []advisor.Advice
	level      advisor.Status
	title      string
	text       string
}

func (checker *viewDisallowCreateChecker) EnterQuery(ctx *mysql.QueryContext) {
	checker.text = ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx)
}

func (checker *viewDisallowCreateChecker) EnterCreateView(ctx *mysql.CreateViewContext) {
	code := advisor.Ok
	if ctx.ViewName() != nil {
		code = advisor.DisallowView
	}

	if code != advisor.Ok {
		checker.adviceList = append(checker.adviceList, advisor.Advice{
			Status:  checker.level,
			Code:    code,
			Title:   checker.title,
			Content: fmt.Sprintf("View is forbidden, but \"%s\" creates", checker.text),
			Line:    checker.baseLine + ctx.GetStart().GetLine(),
		})
	}
}
