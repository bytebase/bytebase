package mysql

import (
	"fmt"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	mysql "github.com/bytebase/mysql-parser"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	_ advisor.Advisor = (*NoLeadingWildcardLikeAdvisor)(nil)
)

func init() {
	// only for mysqlwip test.
	advisor.Register(storepb.Engine_ENGINE_UNSPECIFIED, advisor.MySQLNoLeadingWildcardLike, &NoLeadingWildcardLikeAdvisor{})
}

// NoLeadingWildcardLikeAdvisor is the advisor checking for no leading wildcard LIKE.
type NoLeadingWildcardLikeAdvisor struct {
}

// Check checks for no leading wildcard LIKE.
func (*NoLeadingWildcardLikeAdvisor) Check(ctx advisor.Context, _ string) ([]advisor.Advice, error) {
	root, ok := ctx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to mysql parse result")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}

	checker := &noLeadingWildcardLikeChecker{
		title: string(ctx.Rule.Type),
		level: level,
	}
	for _, stmtNode := range root {
		checker.baseLine = stmtNode.BaseLine
		antlr.ParseTreeWalkerDefault.Walk(checker, stmtNode.Tree)
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

type noLeadingWildcardLikeChecker struct {
	*mysql.BaseMySQLParserListener

	baseLine   int
	title      string
	adviceList []advisor.Advice
	level      advisor.Status
	text       string
}

func (checker *noLeadingWildcardLikeChecker) EnterQuery(ctx *mysql.QueryContext) {
	checker.text = ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx)
}

// EnterPredicateExprLike is called when production predicateExprLike is entered.
func (checker *noLeadingWildcardLikeChecker) EnterPredicateExprLike(ctx *mysql.PredicateExprLikeContext) {
	if ctx.LIKE_SYMBOL() == nil {
		return
	}

	for _, expr := range ctx.AllSimpleExpr() {
		pattern := expr.GetText()
		if (strings.HasPrefix(pattern, "'%") && strings.HasSuffix(pattern, "'")) || (strings.HasPrefix(pattern, "\"%") && strings.HasSuffix(pattern, "\"")) {
			checker.adviceList = append(checker.adviceList, advisor.Advice{
				Status:  checker.level,
				Code:    advisor.StatementLeadingWildcardLike,
				Title:   checker.title,
				Content: fmt.Sprintf("\"%s\" uses leading wildcard LIKE", checker.text),
				Line:    checker.baseLine + ctx.GetStart().GetLine(),
			})
		}
	}
}
