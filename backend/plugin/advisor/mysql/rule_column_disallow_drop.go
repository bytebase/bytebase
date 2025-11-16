package mysql

import (
	"context"
	"fmt"

	"github.com/bytebase/bytebase/backend/plugin/advisor/code"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	"github.com/bytebase/parser/mysql"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
)

var (
	_ advisor.Advisor = (*ColumnDisallowDropAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_OCEANBASE, advisor.SchemaRuleColumnDisallowDrop, &ColumnDisallowDropAdvisor{})
}

// ColumnDisallowDropAdvisor is the advisor checking for disallow DROP COLUMN statement.
type ColumnDisallowDropAdvisor struct {
}

// Check checks for disallow DROP COLUMN statement.
func (*ColumnDisallowDropAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmtList, ok := checkCtx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to mysql parser result")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	// Create the rule
	rule := NewColumnDisallowDropRule(level, string(checkCtx.Rule.Type))

	// Create the generic checker with the rule
	checker := NewGenericChecker([]Rule{rule})

	for _, stmtNode := range stmtList {
		rule.SetBaseLine(stmtNode.BaseLine)
		checker.SetBaseLine(stmtNode.BaseLine)
		antlr.ParseTreeWalkerDefault.Walk(checker, stmtNode.Tree)
	}

	return checker.GetAdviceList(), nil
}

// ColumnDisallowDropRule checks for disallow DROP COLUMN statement.
type ColumnDisallowDropRule struct {
	BaseRule
}

// NewColumnDisallowDropRule creates a new ColumnDisallowDropRule.
func NewColumnDisallowDropRule(level storepb.Advice_Status, title string) *ColumnDisallowDropRule {
	return &ColumnDisallowDropRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
	}
}

// Name returns the rule name.
func (*ColumnDisallowDropRule) Name() string {
	return "ColumnDisallowDropRule"
}

// OnEnter is called when entering a parse tree node.
func (r *ColumnDisallowDropRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	if nodeType == NodeTypeAlterTable {
		r.checkAlterTable(ctx.(*mysql.AlterTableContext))
	}
	return nil
}

// OnExit is called when exiting a parse tree node.
func (*ColumnDisallowDropRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

func (r *ColumnDisallowDropRule) checkAlterTable(ctx *mysql.AlterTableContext) {
	if ctx.AlterTableActions() == nil {
		return
	}
	if ctx.AlterTableActions().AlterCommandList() == nil {
		return
	}
	if ctx.AlterTableActions().AlterCommandList().AlterList() == nil {
		return
	}

	_, tableName := mysqlparser.NormalizeMySQLTableRef(ctx.TableRef())
	for _, item := range ctx.AlterTableActions().AlterCommandList().AlterList().AllAlterListItem() {
		if item == nil || item.DROP_SYMBOL() == nil || item.ColumnInternalRef() == nil {
			continue
		}

		columnName := mysqlparser.NormalizeMySQLColumnInternalRef(item.ColumnInternalRef())
		r.AddAdvice(&storepb.Advice{
			Status:        r.level,
			Code:          code.DropColumn.Int32(),
			Title:         r.title,
			Content:       fmt.Sprintf("drops column \"%s\" of table \"%s\"", columnName, tableName),
			StartPosition: common.ConvertANTLRLineToPosition(r.baseLine + item.GetStart().GetLine()),
		})
	}
}
