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
	_ advisor.Advisor = (*SelectNoSelectAllAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_ORACLE, advisor.OracleNoSelectAll, &SelectNoSelectAllAdvisor{})
	advisor.Register(storepb.Engine_DM, advisor.OracleNoSelectAll, &SelectNoSelectAllAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE_ORACLE, advisor.OracleNoSelectAll, &SelectNoSelectAllAdvisor{})
}

// SelectNoSelectAllAdvisor is the advisor checking for no select all.
type SelectNoSelectAllAdvisor struct {
}

// Check checks for no select all.
func (*SelectNoSelectAllAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	tree, ok := checkCtx.AST.(antlr.Tree)
	if !ok {
		return nil, errors.Errorf("failed to convert to Tree")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := NewSelectNoSelectAllRule(level, string(checkCtx.Rule.Type), checkCtx.CurrentDatabase)
	checker := NewGenericChecker([]Rule{rule})

	antlr.ParseTreeWalkerDefault.Walk(checker, tree)

	return checker.GetAdviceList()
}

// SelectNoSelectAllRule is the rule implementation for no select all.
type SelectNoSelectAllRule struct {
	BaseRule

	currentDatabase string
}

// NewSelectNoSelectAllRule creates a new SelectNoSelectAllRule.
func NewSelectNoSelectAllRule(level storepb.Advice_Status, title string, currentDatabase string) *SelectNoSelectAllRule {
	return &SelectNoSelectAllRule{
		BaseRule:        NewBaseRule(level, title, 0),
		currentDatabase: currentDatabase,
	}
}

// Name returns the rule name.
func (*SelectNoSelectAllRule) Name() string {
	return "select.no-select-all"
}

// OnEnter is called when the parser enters a rule context.
func (r *SelectNoSelectAllRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	if nodeType == "Selected_list" {
		r.handleSelectedList(ctx.(*parser.Selected_listContext))
	}
	return nil
}

// OnExit is called when the parser exits a rule context.
func (*SelectNoSelectAllRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

func (r *SelectNoSelectAllRule) handleSelectedList(ctx *parser.Selected_listContext) {
	if ctx.ASTERISK() != nil {
		r.AddAdvice(
			r.level,
			advisor.StatementSelectAll.Int32(),
			"Avoid using SELECT *.",
			common.ConvertANTLRLineToPosition(ctx.GetStart().GetLine()),
		)
	}
}
