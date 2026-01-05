package mysql

import (
	"context"
	"fmt"

	"github.com/antlr4-go/antlr/v4"
	"github.com/bytebase/parser/mysql"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

var (
	_ advisor.Advisor = (*DisallowOrderByAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_STATEMENT_DISALLOW_ORDER_BY, &DisallowOrderByAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_STATEMENT_DISALLOW_ORDER_BY, &DisallowOrderByAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_STATEMENT_DISALLOW_ORDER_BY, &DisallowOrderByAdvisor{})
}

// DisallowOrderByAdvisor is the advisor checking for no ORDER BY clause in DELETE/UPDATE statements.
type DisallowOrderByAdvisor struct {
}

// Check checks for no ORDER BY clause in DELETE/UPDATE statements.
func (*DisallowOrderByAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	// Create the rule
	rule := NewDisallowOrderByRule(level, checkCtx.Rule.Type.String())

	// Create the generic checker with the rule
	checker := NewGenericChecker([]Rule{rule})

	for _, stmt := range checkCtx.ParsedStatements {
		rule.SetBaseLine(stmt.BaseLine())
		checker.SetBaseLine(stmt.BaseLine())
		if stmt.AST == nil {
			continue
		}
		antlrAST, ok := base.GetANTLRAST(stmt.AST)
		if !ok {
			continue
		}
		antlr.ParseTreeWalkerDefault.Walk(checker, antlrAST.Tree)
	}

	return checker.GetAdviceList(), nil
}

// DisallowOrderByRule checks for no ORDER BY clause in DELETE/UPDATE statements.
type DisallowOrderByRule struct {
	BaseRule
	text string
}

// NewDisallowOrderByRule creates a new DisallowOrderByRule.
func NewDisallowOrderByRule(level storepb.Advice_Status, title string) *DisallowOrderByRule {
	return &DisallowOrderByRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
	}
}

// Name returns the rule name.
func (*DisallowOrderByRule) Name() string {
	return "DisallowOrderByRule"
}

// OnEnter is called when entering a parse tree node.
func (r *DisallowOrderByRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case NodeTypeQuery:
		queryCtx, ok := ctx.(*mysql.QueryContext)
		if !ok {
			return nil
		}
		r.text = queryCtx.GetParser().GetTokenStream().GetTextFromRuleContext(queryCtx)
	case NodeTypeDeleteStatement:
		r.checkDeleteStatement(ctx.(*mysql.DeleteStatementContext))
	case NodeTypeUpdateStatement:
		r.checkUpdateStatement(ctx.(*mysql.UpdateStatementContext))
	default:
	}
	return nil
}

// OnExit is called when exiting a parse tree node.
func (*DisallowOrderByRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

func (r *DisallowOrderByRule) checkDeleteStatement(ctx *mysql.DeleteStatementContext) {
	if ctx.OrderClause() != nil && ctx.OrderClause().ORDER_SYMBOL() != nil {
		r.handleOrderByClause(code.DeleteUseOrderBy, ctx.GetStart().GetLine())
	}
}

func (r *DisallowOrderByRule) checkUpdateStatement(ctx *mysql.UpdateStatementContext) {
	if ctx.OrderClause() != nil && ctx.OrderClause().ORDER_SYMBOL() != nil {
		r.handleOrderByClause(code.UpdateUseOrderBy, ctx.GetStart().GetLine())
	}
}

func (r *DisallowOrderByRule) handleOrderByClause(code code.Code, lineNumber int) {
	r.AddAdvice(&storepb.Advice{
		Status:        r.level,
		Code:          code.Int32(),
		Title:         r.title,
		Content:       fmt.Sprintf("ORDER BY clause is forbidden in DELETE and UPDATE statements, but \"%s\" uses", r.text),
		StartPosition: common.ConvertANTLRLineToPosition(r.baseLine + lineNumber),
	})
}
