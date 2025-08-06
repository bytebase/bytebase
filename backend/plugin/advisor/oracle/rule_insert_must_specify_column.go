// Package oracle is the advisor for oracle database.
package oracle

import (
	"context"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/plsql-parser"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
)

var (
	_ advisor.Advisor = (*InsertMustSpecifyColumnAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_ORACLE, advisor.OracleInsertMustSpecifyColumn, &InsertMustSpecifyColumnAdvisor{})
}

// InsertMustSpecifyColumnAdvisor is the advisor checking for to enforce column specified.
type InsertMustSpecifyColumnAdvisor struct {
}

// Check checks for to enforce column specified.
func (*InsertMustSpecifyColumnAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	tree, ok := checkCtx.AST.(antlr.Tree)
	if !ok {
		return nil, errors.Errorf("failed to convert to Tree")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := NewInsertMustSpecifyColumnRule(level, string(checkCtx.Rule.Type), checkCtx.CurrentDatabase)
	checker := NewGenericChecker([]Rule{rule})

	antlr.ParseTreeWalkerDefault.Walk(checker, tree)

	return checker.GetAdviceList()
}

// InsertMustSpecifyColumnRule is the rule implementation for enforcing column specification in INSERT.
type InsertMustSpecifyColumnRule struct {
	BaseRule

	currentDatabase string
}

// NewInsertMustSpecifyColumnRule creates a new InsertMustSpecifyColumnRule.
func NewInsertMustSpecifyColumnRule(level storepb.Advice_Status, title string, currentDatabase string) *InsertMustSpecifyColumnRule {
	return &InsertMustSpecifyColumnRule{
		BaseRule:        NewBaseRule(level, title, 0),
		currentDatabase: currentDatabase,
	}
}

// Name returns the rule name.
func (*InsertMustSpecifyColumnRule) Name() string {
	return "insert.must-specify-column"
}

// OnEnter is called when the parser enters a rule context.
func (r *InsertMustSpecifyColumnRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	if nodeType == "Insert_into_clause" {
		r.handleInsertIntoClause(ctx.(*parser.Insert_into_clauseContext))
	}
	return nil
}

// OnExit is called when the parser exits a rule context.
func (*InsertMustSpecifyColumnRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

func (r *InsertMustSpecifyColumnRule) handleInsertIntoClause(ctx *parser.Insert_into_clauseContext) {
	if ctx.Paren_column_list() == nil {
		r.AddAdvice(
			r.level,
			advisor.InsertNotSpecifyColumn.Int32(),
			"INSERT statement should specify column name.",
			common.ConvertANTLRLineToPosition(ctx.GetStart().GetLine()),
		)
	}
}
