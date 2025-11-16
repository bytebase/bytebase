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
	_ advisor.Advisor = (*WhereRequireForSelectAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_SNOWFLAKE, advisor.SchemaRuleStatementRequireWhereForSelect, &WhereRequireForSelectAdvisor{})
}

// WhereRequireForSelectAdvisor is the advisor checking for WHERE clause requirement for SELECT statement.
type WhereRequireForSelectAdvisor struct {
}

// Check checks for WHERE clause requirement.
func (*WhereRequireForSelectAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	tree, ok := checkCtx.AST.(antlr.Tree)
	if !ok {
		return nil, errors.Errorf("failed to convert to Tree")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := NewWhereRequireForSelectRule(level, string(checkCtx.Rule.Type))
	checker := NewGenericChecker([]Rule{rule})

	antlr.ParseTreeWalkerDefault.Walk(checker, tree)

	return checker.GetAdviceList(), nil
}

// WhereRequireForSelectRule checks for WHERE clause requirement in SELECT statements.
type WhereRequireForSelectRule struct {
	BaseRule
}

// NewWhereRequireForSelectRule creates a new WhereRequireForSelectRule.
func NewWhereRequireForSelectRule(level storepb.Advice_Status, title string) *WhereRequireForSelectRule {
	return &WhereRequireForSelectRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
	}
}

// Name returns the rule name.
func (*WhereRequireForSelectRule) Name() string {
	return "WhereRequireForSelectRule"
}

// OnEnter is called when entering a parse tree node.
func (r *WhereRequireForSelectRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	if nodeType == NodeTypeQueryStatement {
		r.enterQueryStatement(ctx.(*parser.Query_statementContext))
	}
	return nil
}

// OnExit is called when exiting a parse tree node.
func (*WhereRequireForSelectRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	// This rule doesn't need exit processing
	return nil
}

func (r *WhereRequireForSelectRule) enterQueryStatement(ctx *parser.Query_statementContext) {
	if ctx.Select_statement() == nil {
		return
	}
	optional := ctx.Select_statement().Select_optional_clauses()
	if optional == nil {
		return
	}
	// Allow SELECT queries without a FROM clause to proceed, e.g. SELECT 1.
	if optional.Where_clause() == nil && optional.From_clause() != nil {
		r.AddAdvice(&storepb.Advice{
			Status:        r.level,
			Code:          code.StatementNoWhere.Int32(),
			Title:         r.title,
			Content:       "WHERE clause is required for SELECT statement.",
			StartPosition: common.ConvertANTLRLineToPosition(ctx.GetStart().GetLine()),
		})
	}
}
