package mysql

import (
	"context"
	"fmt"

	"github.com/bytebase/bytebase/backend/plugin/advisor/code"

	"github.com/antlr4-go/antlr/v4"

	"github.com/bytebase/parser/mysql"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
)

var (
	_ advisor.Advisor = (*ColumnDisallowChangingOrderAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_COLUMN_DISALLOW_CHANGING_ORDER, &ColumnDisallowChangingOrderAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_COLUMN_DISALLOW_CHANGING_ORDER, &ColumnDisallowChangingOrderAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_COLUMN_DISALLOW_CHANGING_ORDER, &ColumnDisallowChangingOrderAdvisor{})
}

// ColumnDisallowChangingOrderAdvisor is the advisor checking for disallow changing column order.
type ColumnDisallowChangingOrderAdvisor struct {
}

// Check checks for disallow changing column order.
func (*ColumnDisallowChangingOrderAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	// Create the rule
	rule := NewColumnDisallowChangingOrderRule(level, checkCtx.Rule.Type.String())

	// Create the generic checker with the rule
	checker := NewGenericChecker([]Rule{rule})

	for _, stmt := range checkCtx.ParsedStatements {
		rule.SetBaseLine(stmt.BaseLine())
		checker.SetBaseLine(stmt.BaseLine())
		if stmt.AST == nil {
			continue
		}
		antlrAST, ok := base.GetANTLRAST(stmt.AST)
		if !ok {
			continue
		}
		antlr.ParseTreeWalkerDefault.Walk(checker, antlrAST.Tree)
	}

	return checker.GetAdviceList(), nil
}

// ColumnDisallowChangingOrderRule checks for disallow changing column order.
type ColumnDisallowChangingOrderRule struct {
	BaseRule
	text string
}

// NewColumnDisallowChangingOrderRule creates a new ColumnDisallowChangingOrderRule.
func NewColumnDisallowChangingOrderRule(level storepb.Advice_Status, title string) *ColumnDisallowChangingOrderRule {
	return &ColumnDisallowChangingOrderRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
	}
}

// Name returns the rule name.
func (*ColumnDisallowChangingOrderRule) Name() string {
	return "ColumnDisallowChangingOrderRule"
}

// OnEnter is called when entering a parse tree node.
func (r *ColumnDisallowChangingOrderRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case NodeTypeQuery:
		if queryCtx, ok := ctx.(*mysql.QueryContext); ok {
			r.text = queryCtx.GetParser().GetTokenStream().GetTextFromRuleContext(queryCtx)
		}
	case NodeTypeAlterTable:
		r.checkAlterTable(ctx.(*mysql.AlterTableContext))
	default:
	}
	return nil
}

// OnExit is called when exiting a parse tree node.
func (*ColumnDisallowChangingOrderRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

func (r *ColumnDisallowChangingOrderRule) checkAlterTable(ctx *mysql.AlterTableContext) {
	if !mysqlparser.IsTopMySQLRule(&ctx.BaseParserRuleContext) {
		return
	}
	if ctx.AlterTableActions() == nil {
		return
	}
	if ctx.AlterTableActions().AlterCommandList() == nil {
		return
	}
	if ctx.AlterTableActions().AlterCommandList().AlterList() == nil {
		return
	}

	for _, item := range ctx.AlterTableActions().AlterCommandList().AlterList().AllAlterListItem() {
		if item == nil {
			continue
		}

		switch {
		// modify column.
		case item.MODIFY_SYMBOL() != nil && item.ColumnInternalRef() != nil:
			// do nothing.
		// change column
		case item.CHANGE_SYMBOL() != nil && item.ColumnInternalRef() != nil && item.Identifier() != nil:
			// do nothing.
		default:
			continue
		}

		if item.Place() != nil {
			r.AddAdvice(&storepb.Advice{
				Status:        r.level,
				Code:          code.ChangeColumnOrder.Int32(),
				Title:         r.title,
				Content:       fmt.Sprintf("\"%s\" changes column order", r.text),
				StartPosition: common.ConvertANTLRLineToPosition(r.baseLine + ctx.GetStart().GetLine()),
			})
		}
	}
}
