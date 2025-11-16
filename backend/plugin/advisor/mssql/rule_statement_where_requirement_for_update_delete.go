package mssql

import (
	"context"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/tsql"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*WhereRequirementForUpdateDeleteAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MSSQL, advisor.SchemaRuleStatementRequireWhereForUpdateDelete, &WhereRequirementForUpdateDeleteAdvisor{})
}

// WhereRequirementForUpdateDeleteAdvisor is the advisor checking for WHERE clause requirement for UPDATE/DELETE statements.
type WhereRequirementForUpdateDeleteAdvisor struct {
}

// Check checks for WHERE clause requirement.
func (*WhereRequirementForUpdateDeleteAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	tree, ok := checkCtx.AST.(antlr.Tree)
	if !ok {
		return nil, errors.Errorf("failed to convert to Tree")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	// Create the rule
	rule := NewWhereRequirementForUpdateDeleteRule(level, string(checkCtx.Rule.Type))

	// Create the generic checker with the rule
	checker := NewGenericChecker([]Rule{rule})

	antlr.ParseTreeWalkerDefault.Walk(checker, tree)

	return checker.GetAdviceList(), nil
}

// WhereRequirementForUpdateDeleteRule is the rule for WHERE clause requirement for UPDATE/DELETE statements.
type WhereRequirementForUpdateDeleteRule struct {
	BaseRule
}

// NewWhereRequirementForUpdateDeleteRule creates a new WhereRequirementForUpdateDeleteRule.
func NewWhereRequirementForUpdateDeleteRule(level storepb.Advice_Status, title string) *WhereRequirementForUpdateDeleteRule {
	return &WhereRequirementForUpdateDeleteRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
	}
}

// Name returns the rule name.
func (*WhereRequirementForUpdateDeleteRule) Name() string {
	return "WhereRequirementForUpdateDeleteRule"
}

// OnEnter is called when entering a parse tree node.
func (r *WhereRequirementForUpdateDeleteRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case NodeTypeDeleteStatement:
		r.enterDeleteStatement(ctx.(*parser.Delete_statementContext))
	case NodeTypeUpdateStatement:
		r.enterUpdateStatement(ctx.(*parser.Update_statementContext))
	default:
		// Ignore other node types
	}
	return nil
}

// OnExit is called when exiting a parse tree node.
func (*WhereRequirementForUpdateDeleteRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	// This rule doesn't need exit processing
	return nil
}

func (r *WhereRequirementForUpdateDeleteRule) enterDeleteStatement(ctx *parser.Delete_statementContext) {
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

func (r *WhereRequirementForUpdateDeleteRule) enterUpdateStatement(ctx *parser.Update_statementContext) {
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
