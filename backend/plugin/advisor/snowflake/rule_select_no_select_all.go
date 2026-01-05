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
	_ advisor.Advisor = (*SelectNoSelectAllAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_SNOWFLAKE, storepb.SQLReviewRule_STATEMENT_SELECT_NO_SELECT_ALL, &SelectNoSelectAllAdvisor{})
}

// SelectNoSelectAllAdvisor is the advisor checking for no select all.
type SelectNoSelectAllAdvisor struct {
}

// Check checks for no select all.
func (*SelectNoSelectAllAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := NewSelectNoSelectAllRule(level, checkCtx.Rule.Type.String())
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

// SelectNoSelectAllRule checks for no select all.
type SelectNoSelectAllRule struct {
	BaseRule
}

// NewSelectNoSelectAllRule creates a new SelectNoSelectAllRule.
func NewSelectNoSelectAllRule(level storepb.Advice_Status, title string) *SelectNoSelectAllRule {
	return &SelectNoSelectAllRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
	}
}

// Name returns the rule name.
func (*SelectNoSelectAllRule) Name() string {
	return "SelectNoSelectAllRule"
}

// OnEnter is called when entering a parse tree node.
func (r *SelectNoSelectAllRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	if nodeType == NodeTypeSelectListElem {
		r.enterSelectListElem(ctx.(*parser.Select_list_elemContext))
	}
	return nil
}

// OnExit is called when exiting a parse tree node.
func (*SelectNoSelectAllRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	// This rule doesn't need exit processing
	return nil
}

func (r *SelectNoSelectAllRule) enterSelectListElem(ctx *parser.Select_list_elemContext) {
	if v := ctx.Column_elem(); v != nil {
		if v.STAR() != nil {
			r.AddAdvice(&storepb.Advice{
				Status:        r.level,
				Code:          code.StatementSelectAll.Int32(),
				Title:         r.title,
				Content:       "Avoid using SELECT *.",
				StartPosition: common.ConvertANTLRLineToPosition(r.baseLine + v.STAR().GetSymbol().GetLine()),
			})
		}
	}
}
