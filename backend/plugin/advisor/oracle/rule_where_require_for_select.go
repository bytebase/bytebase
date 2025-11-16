// Package oracle is the advisor for oracle database.
package oracle

import (
	"context"
	"strings"

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
	_ advisor.Advisor = (*WhereRequireForSelectAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_ORACLE, advisor.SchemaRuleStatementRequireWhereForSelect, &WhereRequireForSelectAdvisor{})
}

// WhereRequireForSelectAdvisor is the advisor checking for WHERE clause requirement.
type WhereRequireForSelectAdvisor struct {
}

// Check checks for WHERE clause requirement.
func (*WhereRequireForSelectAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmtList, ok := checkCtx.AST.([]*plsqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to ParseResult")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := NewWhereRequireForSelectRule(level, string(checkCtx.Rule.Type), checkCtx.CurrentDatabase)
	checker := NewGenericChecker([]Rule{rule})

	for _, stmtNode := range stmtList {
		rule.SetBaseLine(stmtNode.BaseLine)
		checker.SetBaseLine(stmtNode.BaseLine)
		antlr.ParseTreeWalkerDefault.Walk(checker, stmtNode.Tree)
	}

	return checker.GetAdviceList()
}

// WhereRequireForSelectRule is the rule implementation for WHERE clause requirement in SELECT.
type WhereRequireForSelectRule struct {
	BaseRule

	currentDatabase string
}

// NewWhereRequireForSelectRule creates a new WhereRequireForSelectRule.
func NewWhereRequireForSelectRule(level storepb.Advice_Status, title string, currentDatabase string) *WhereRequireForSelectRule {
	return &WhereRequireForSelectRule{
		BaseRule:        NewBaseRule(level, title, 0),
		currentDatabase: currentDatabase,
	}
}

// Name returns the rule name.
func (*WhereRequireForSelectRule) Name() string {
	return "where.require-for-select"
}

// OnEnter is called when the parser enters a rule context.
func (r *WhereRequireForSelectRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	if nodeType == "Query_block" {
		r.handleQueryBlock(ctx.(*parser.Query_blockContext))
	}
	return nil
}

// OnExit is called when the parser exits a rule context.
func (*WhereRequireForSelectRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

func (r *WhereRequireForSelectRule) handleQueryBlock(ctx *parser.Query_blockContext) {
	// Allow SELECT queries without a FROM clause to proceed, e.g. SELECT 1.
	if ctx.From_clause() == nil || ctx.From_clause().Table_ref_list() == nil {
		return
	}
	if strings.ToLower(ctx.From_clause().Table_ref_list().GetText()) == "dual" {
		return
	}
	if ctx.Where_clause() == nil {
		r.AddAdvice(
			r.level,
			code.StatementNoWhere.Int32(),
			"WHERE clause is required for SELECT statement.",
			common.ConvertANTLRLineToPosition(r.baseLine+ctx.GetStop().GetLine()),
		)
	}
}
