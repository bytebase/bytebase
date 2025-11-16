package mysql

import (
	"context"
	"fmt"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/bytebase/parser/mysql"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
)

var (
	_ advisor.Advisor = (*NoLeadingWildcardLikeAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.SchemaRuleStatementNoLeadingWildcardLike, &NoLeadingWildcardLikeAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, advisor.SchemaRuleStatementNoLeadingWildcardLike, &NoLeadingWildcardLikeAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, advisor.SchemaRuleStatementNoLeadingWildcardLike, &NoLeadingWildcardLikeAdvisor{})
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

	// Create the rule
	rule := NewNoLeadingWildcardLikeRule(level, string(checkCtx.Rule.Type))

	// Create the generic checker with the rule
	checker := NewGenericChecker([]Rule{rule})

	for _, stmtNode := range root {
		rule.SetBaseLine(stmtNode.BaseLine)
		checker.SetBaseLine(stmtNode.BaseLine)
		antlr.ParseTreeWalkerDefault.Walk(checker, stmtNode.Tree)
	}

	return checker.GetAdviceList(), nil
}

// NoLeadingWildcardLikeRule checks for no leading wildcard LIKE.
type NoLeadingWildcardLikeRule struct {
	BaseRule
	text string
}

// NewNoLeadingWildcardLikeRule creates a new NoLeadingWildcardLikeRule.
func NewNoLeadingWildcardLikeRule(level storepb.Advice_Status, title string) *NoLeadingWildcardLikeRule {
	return &NoLeadingWildcardLikeRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
	}
}

// Name returns the rule name.
func (*NoLeadingWildcardLikeRule) Name() string {
	return "NoLeadingWildcardLikeRule"
}

// OnEnter is called when entering a parse tree node.
func (r *NoLeadingWildcardLikeRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case NodeTypeQuery:
		queryCtx, ok := ctx.(*mysql.QueryContext)
		if !ok {
			return nil
		}
		r.text = queryCtx.GetParser().GetTokenStream().GetTextFromRuleContext(queryCtx)
	case NodeTypePredicateExprLike:
		r.checkPredicateExprLike(ctx.(*mysql.PredicateExprLikeContext))
	default:
	}
	return nil
}

// OnExit is called when exiting a parse tree node.
func (*NoLeadingWildcardLikeRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

func (r *NoLeadingWildcardLikeRule) checkPredicateExprLike(ctx *mysql.PredicateExprLikeContext) {
	if ctx.LIKE_SYMBOL() == nil {
		return
	}

	for _, expr := range ctx.AllSimpleExpr() {
		pattern := expr.GetText()
		if (strings.HasPrefix(pattern, "'%") && strings.HasSuffix(pattern, "'")) || (strings.HasPrefix(pattern, "\"%") && strings.HasSuffix(pattern, "\"")) {
			r.AddAdvice(&storepb.Advice{
				Status:        r.level,
				Code:          code.StatementLeadingWildcardLike.Int32(),
				Title:         r.title,
				Content:       fmt.Sprintf("\"%s\" uses leading wildcard LIKE", r.text),
				StartPosition: common.ConvertANTLRLineToPosition(r.baseLine + ctx.GetStart().GetLine()),
			})
		}
	}
}
