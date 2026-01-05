// Package snowflake is the advisor for snowflake database.
package snowflake

import (
	"context"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/snowflake"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

var (
	_ advisor.Advisor = (*WhereRequireForSelectAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_SNOWFLAKE, storepb.SQLReviewRule_STATEMENT_WHERE_REQUIRE_SELECT, &WhereRequireForSelectAdvisor{})
}

// WhereRequireForSelectAdvisor is the advisor checking for WHERE clause requirement for SELECT statement.
type WhereRequireForSelectAdvisor struct {
}

// Check checks for WHERE clause requirement.
func (*WhereRequireForSelectAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := NewWhereRequireForSelectRule(level, checkCtx.Rule.Type.String())
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
			StartPosition: common.ConvertANTLRLineToPosition(r.baseLine + ctx.GetStart().GetLine()),
		})
	}
}
