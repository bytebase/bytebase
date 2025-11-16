// Package oracle is the advisor for oracle database.
package oracle

import (
	"context"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/plsql"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	plsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/plsql"
)

var (
	_ advisor.Advisor = (*WhereRequireForUpdateDeleteAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_ORACLE, advisor.SchemaRuleStatementRequireWhereForUpdateDelete, &WhereRequireForUpdateDeleteAdvisor{})
}

// WhereRequireForUpdateDeleteAdvisor is the advisor checking for WHERE clause requirement.
type WhereRequireForUpdateDeleteAdvisor struct {
}

// Check checks for WHERE clause requirement.
func (*WhereRequireForUpdateDeleteAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmtList, ok := checkCtx.AST.([]*plsqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to ParseResult")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := NewWhereRequireForUpdateDeleteRule(level, string(checkCtx.Rule.Type), checkCtx.CurrentDatabase)
	checker := NewGenericChecker([]Rule{rule})

	for _, stmtNode := range stmtList {
		rule.SetBaseLine(stmtNode.BaseLine)
		checker.SetBaseLine(stmtNode.BaseLine)
		antlr.ParseTreeWalkerDefault.Walk(checker, stmtNode.Tree)
	}

	return checker.GetAdviceList()
}

// WhereRequireForUpdateDeleteRule is the rule implementation for WHERE clause requirement in UPDATE/DELETE.
type WhereRequireForUpdateDeleteRule struct {
	BaseRule

	currentDatabase string
}

// NewWhereRequireForUpdateDeleteRule creates a new WhereRequireForUpdateDeleteRule.
func NewWhereRequireForUpdateDeleteRule(level storepb.Advice_Status, title string, currentDatabase string) *WhereRequireForUpdateDeleteRule {
	return &WhereRequireForUpdateDeleteRule{
		BaseRule:        NewBaseRule(level, title, 0),
		currentDatabase: currentDatabase,
	}
}

// Name returns the rule name.
func (*WhereRequireForUpdateDeleteRule) Name() string {
	return "where.require-for-update-delete"
}

// OnEnter is called when the parser enters a rule context.
func (r *WhereRequireForUpdateDeleteRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case "Update_statement":
		r.handleUpdateStatement(ctx.(*parser.Update_statementContext))
	case "Delete_statement":
		r.handleDeleteStatement(ctx.(*parser.Delete_statementContext))
	default:
		// Ignore other node types
	}
	return nil
}

// OnExit is called when the parser exits a rule context.
func (*WhereRequireForUpdateDeleteRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

func (r *WhereRequireForUpdateDeleteRule) handleUpdateStatement(ctx *parser.Update_statementContext) {
	if ctx.Where_clause() == nil {
		r.AddAdvice(
			r.level,
			code.StatementNoWhere.Int32(),
			"WHERE clause is required for UPDATE statement.",
			common.ConvertANTLRLineToPosition(r.baseLine+ctx.GetStop().GetLine()),
		)
	}
}

func (r *WhereRequireForUpdateDeleteRule) handleDeleteStatement(ctx *parser.Delete_statementContext) {
	if ctx.Where_clause() == nil {
		r.AddAdvice(
			r.level,
			code.StatementNoWhere.Int32(),
			"WHERE clause is required for DELETE statement.",
			common.ConvertANTLRLineToPosition(r.baseLine+ctx.GetStop().GetLine()),
		)
	}
}
