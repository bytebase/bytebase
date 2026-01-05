// Package oracle is the advisor for oracle database.
package oracle

import (
	"context"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/plsql"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

var (
	_ advisor.Advisor = (*WhereNoLeadingWildcardLikeAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_ORACLE, storepb.SQLReviewRule_STATEMENT_WHERE_NO_LEADING_WILDCARD_LIKE, &WhereNoLeadingWildcardLikeAdvisor{})
}

// WhereNoLeadingWildcardLikeAdvisor is the advisor checking for no leading wildcard LIKE.
type WhereNoLeadingWildcardLikeAdvisor struct {
}

// Check checks for no leading wildcard LIKE.
func (*WhereNoLeadingWildcardLikeAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := NewWhereNoLeadingWildcardLikeRule(level, checkCtx.Rule.Type.String(), checkCtx.CurrentDatabase)
	checker := NewGenericChecker([]Rule{rule})

	for _, stmt := range checkCtx.ParsedStatements {
		if stmt.AST == nil {
			continue
		}
		antlrAST, ok := base.GetANTLRAST(stmt.AST)
		if !ok {
			continue
		}
		rule.SetBaseLine(stmt.BaseLine())
		checker.SetBaseLine(stmt.BaseLine())
		antlr.ParseTreeWalkerDefault.Walk(checker, antlrAST.Tree)
	}

	return checker.GetAdviceList()
}

// WhereNoLeadingWildcardLikeRule is the rule implementation for no leading wildcard LIKE.
type WhereNoLeadingWildcardLikeRule struct {
	BaseRule

	currentDatabase string
}

// NewWhereNoLeadingWildcardLikeRule creates a new WhereNoLeadingWildcardLikeRule.
func NewWhereNoLeadingWildcardLikeRule(level storepb.Advice_Status, title string, currentDatabase string) *WhereNoLeadingWildcardLikeRule {
	return &WhereNoLeadingWildcardLikeRule{
		BaseRule:        NewBaseRule(level, title, 0),
		currentDatabase: currentDatabase,
	}
}

// Name returns the rule name.
func (*WhereNoLeadingWildcardLikeRule) Name() string {
	return "where.no-leading-wildcard-like"
}

// OnEnter is called when the parser enters a rule context.
func (r *WhereNoLeadingWildcardLikeRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	if nodeType == "Compound_expression" {
		r.handleCompoundExpression(ctx.(*parser.Compound_expressionContext))
	}
	return nil
}

// OnExit is called when the parser exits a rule context.
func (*WhereNoLeadingWildcardLikeRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

func (r *WhereNoLeadingWildcardLikeRule) handleCompoundExpression(ctx *parser.Compound_expressionContext) {
	if ctx.LIKE() == nil && ctx.LIKE2() == nil && ctx.LIKE4() == nil && ctx.LIKEC() == nil {
		return
	}

	if ctx.Concatenation(1) == nil {
		return
	}

	text := ctx.Concatenation(1).GetText()
	if strings.HasPrefix(text, "'%") && strings.HasSuffix(text, "'") {
		r.AddAdvice(
			r.level,
			code.StatementLeadingWildcardLike.Int32(),
			"Avoid using leading wildcard LIKE.",
			common.ConvertANTLRLineToPosition(r.baseLine+ctx.GetStart().GetLine()),
		)
	}
}
