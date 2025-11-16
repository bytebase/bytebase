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
	_ advisor.Advisor = (*WhereRequirementForSelectAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MSSQL, advisor.SchemaRuleStatementRequireWhereForSelect, &WhereRequirementForSelectAdvisor{})
}

// WhereRequirementForSelectAdvisor is the advisor checking for WHERE clause requirement for SELECT statements.
type WhereRequirementForSelectAdvisor struct {
}

// Check checks for WHERE clause requirement.
func (*WhereRequirementForSelectAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	tree, ok := checkCtx.AST.(antlr.Tree)
	if !ok {
		return nil, errors.Errorf("failed to convert to Tree")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	// Create the rule
	rule := NewWhereRequirementForSelectRule(level, string(checkCtx.Rule.Type))

	// Create the generic checker with the rule
	checker := NewGenericChecker([]Rule{rule})

	antlr.ParseTreeWalkerDefault.Walk(checker, tree)

	return checker.GetAdviceList(), nil
}

// WhereRequirementForSelectRule is the rule for WHERE clause requirement for SELECT statements.
type WhereRequirementForSelectRule struct {
	BaseRule
}

// NewWhereRequirementForSelectRule creates a new WhereRequirementForSelectRule.
func NewWhereRequirementForSelectRule(level storepb.Advice_Status, title string) *WhereRequirementForSelectRule {
	return &WhereRequirementForSelectRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
	}
}

// Name returns the rule name.
func (*WhereRequirementForSelectRule) Name() string {
	return "WhereRequirementForSelectRule"
}

// OnEnter is called when entering a parse tree node.
func (r *WhereRequirementForSelectRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	if nodeType == "Query_specification" {
		r.enterQuerySpecification(ctx.(*parser.Query_specificationContext))
	}
	return nil
}

// OnExit is called when exiting a parse tree node.
func (*WhereRequirementForSelectRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	// This rule doesn't need exit processing
	return nil
}

func (r *WhereRequirementForSelectRule) enterQuerySpecification(ctx *parser.Query_specificationContext) {
	// Allow SELECT queries without a FROM clause to proceed, e.g. SELECT 1.
	if ctx.WHERE() == nil && ctx.From_table_sources() != nil {
		r.AddAdvice(&storepb.Advice{
			Status:        r.level,
			Code:          code.StatementNoWhere.Int32(),
			Title:         r.title,
			Content:       "WHERE clause is required for SELETE statement.",
			StartPosition: common.ConvertANTLRLineToPosition(ctx.GetStart().GetLine()),
		})
	}
}
