package mysql

import (
	"context"
	"fmt"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	mysql "github.com/bytebase/mysql-parser"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	_ advisor.Advisor = (*NoLeadingWildcardLikeAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.MySQLNoLeadingWildcardLike, &NoLeadingWildcardLikeAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, advisor.MySQLNoLeadingWildcardLike, &NoLeadingWildcardLikeAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, advisor.MySQLNoLeadingWildcardLike, &NoLeadingWildcardLikeAdvisor{})
}

// NoLeadingWildcardLikeAdvisor is the advisor checking for no leading wildcard LIKE.
type NoLeadingWildcardLikeAdvisor struct {
}

// Check checks for no leading wildcard LIKE.
func (*NoLeadingWildcardLikeAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	root, ok := checkCtx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to mysql parse result")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	checker := &noLeadingWildcardLikeChecker{
		title: string(checkCtx.Rule.Type),
		level: level,
	}
	for _, stmtNode := range root {
		checker.baseLine = stmtNode.BaseLine
		antlr.ParseTreeWalkerDefault.Walk(checker, stmtNode.Tree)
	}

	return checker.adviceList, nil
}

type noLeadingWildcardLikeChecker struct {
	*mysql.BaseMySQLParserListener

	baseLine   int
	title      string
	adviceList []*storepb.Advice
	level      storepb.Advice_Status
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
			checker.adviceList = append(checker.adviceList, &storepb.Advice{
				Status:        checker.level,
				Code:          advisor.StatementLeadingWildcardLike.Int32(),
				Title:         checker.title,
				Content:       fmt.Sprintf("\"%s\" uses leading wildcard LIKE", checker.text),
				StartPosition: common.ConvertANTLRLineToPosition(checker.baseLine + ctx.GetStart().GetLine()),
			})
		}
	}
}
