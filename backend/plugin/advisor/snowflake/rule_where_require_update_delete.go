// Package snowflake is the advisor for snowflake database.
package snowflake

import (
	"context"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/snowflake"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*WhereRequireForUpdateDeleteAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_SNOWFLAKE, advisor.SchemaRuleStatementRequireWhereForUpdateDelete, &WhereRequireForUpdateDeleteAdvisor{})
}

// WhereRequireForUpdateDeleteAdvisor is the advisor checking for WHERE clause requirement for UPDATE and DELETE statement.
type WhereRequireForUpdateDeleteAdvisor struct {
}

// Check checks for WHERE clause requirement.
func (*WhereRequireForUpdateDeleteAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	tree, ok := checkCtx.AST.(antlr.Tree)
	if !ok {
		return nil, errors.Errorf("failed to convert to Tree")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := NewWhereRequireForUpdateDeleteRule(level, string(checkCtx.Rule.Type))
	checker := NewGenericChecker([]Rule{rule})

	antlr.ParseTreeWalkerDefault.Walk(checker, tree)

	return checker.GetAdviceList(), nil
}

// WhereRequireForUpdateDeleteRule checks for WHERE clause requirement in UPDATE and DELETE statements.
type WhereRequireForUpdateDeleteRule struct {
	BaseRule
}

// NewWhereRequireForUpdateDeleteRule creates a new WhereRequireForUpdateDeleteRule.
func NewWhereRequireForUpdateDeleteRule(level storepb.Advice_Status, title string) *WhereRequireForUpdateDeleteRule {
	return &WhereRequireForUpdateDeleteRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
	}
}

// Name returns the rule name.
func (*WhereRequireForUpdateDeleteRule) Name() string {
	return "WhereRequireForUpdateDeleteRule"
}

// OnEnter is called when entering a parse tree node.
func (r *WhereRequireForUpdateDeleteRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case NodeTypeUpdateStatement:
		r.enterUpdateStatement(ctx.(*parser.Update_statementContext))
	case NodeTypeDeleteStatement:
		r.enterDeleteStatement(ctx.(*parser.Delete_statementContext))
	default:
		// Ignore other node types
	}
	return nil
}

// OnExit is called when exiting a parse tree node.
func (*WhereRequireForUpdateDeleteRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	// This rule doesn't need exit processing
	return nil
}

func (r *WhereRequireForUpdateDeleteRule) enterUpdateStatement(ctx *parser.Update_statementContext) {
	if ctx.WHERE() == nil {
		r.AddAdvice(&storepb.Advice{
			Status:        r.level,
			Code:          code.StatementNoWhere.Int32(),
			Title:         r.title,
			Content:       "WHERE clause is required for UPDATE statement.",
			StartPosition: common.ConvertANTLRLineToPosition(ctx.GetStart().GetLine()),
		})
	}
}

func (r *WhereRequireForUpdateDeleteRule) enterDeleteStatement(ctx *parser.Delete_statementContext) {
	if ctx.WHERE() == nil {
		r.AddAdvice(&storepb.Advice{
			Status:        r.level,
			Code:          code.StatementNoWhere.Int32(),
			Title:         r.title,
			Content:       "WHERE clause is required for DELETE statement.",
			StartPosition: common.ConvertANTLRLineToPosition(ctx.GetStart().GetLine()),
		})
	}
}
