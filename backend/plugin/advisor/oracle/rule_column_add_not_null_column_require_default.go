// Package oracle is the advisor for oracle database.
package oracle

import (
	"context"
	"fmt"

	"github.com/bytebase/bytebase/backend/plugin/advisor/code"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/plsql"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	plsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/plsql"
)

var (
	_ advisor.Advisor = (*ColumnAddNotNullColumnRequireDefaultAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_ORACLE, advisor.SchemaRuleAddNotNullColumnRequireDefault, &ColumnAddNotNullColumnRequireDefaultAdvisor{})
}

// ColumnAddNotNullColumnRequireDefaultAdvisor is the advisor checking for adding not null column requires default.
type ColumnAddNotNullColumnRequireDefaultAdvisor struct {
}

// Check checks for adding not null column requires default.
func (*ColumnAddNotNullColumnRequireDefaultAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmtList, ok := checkCtx.AST.([]*plsqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to ParseResult")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := NewColumnAddNotNullColumnRequireDefaultRule(level, string(checkCtx.Rule.Type), checkCtx.CurrentDatabase)
	checker := NewGenericChecker([]Rule{rule})

	for _, stmtNode := range stmtList {
		rule.SetBaseLine(stmtNode.BaseLine)
		checker.SetBaseLine(stmtNode.BaseLine)
		antlr.ParseTreeWalkerDefault.Walk(checker, stmtNode.Tree)
	}

	return checker.GetAdviceList()
}

// ColumnAddNotNullColumnRequireDefaultRule is the rule implementation for adding not null column requires default.
type ColumnAddNotNullColumnRequireDefaultRule struct {
	BaseRule

	currentDatabase string
	tableName       string
	isNotNull       bool
}

// NewColumnAddNotNullColumnRequireDefaultRule creates a new ColumnAddNotNullColumnRequireDefaultRule.
func NewColumnAddNotNullColumnRequireDefaultRule(level storepb.Advice_Status, title string, currentDatabase string) *ColumnAddNotNullColumnRequireDefaultRule {
	return &ColumnAddNotNullColumnRequireDefaultRule{
		BaseRule:        NewBaseRule(level, title, 0),
		currentDatabase: currentDatabase,
	}
}

// Name returns the rule name.
func (*ColumnAddNotNullColumnRequireDefaultRule) Name() string {
	return "column.add-not-null-column-require-default"
}

// OnEnter is called when the parser enters a rule context.
func (r *ColumnAddNotNullColumnRequireDefaultRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case "Alter_table":
		r.handleAlterTable(ctx.(*parser.Alter_tableContext))
	case "Column_definition":
		r.handleColumnDefinitionEnter(ctx.(*parser.Column_definitionContext))
	case "Inline_constraint":
		r.handleInlineConstraint(ctx.(*parser.Inline_constraintContext))
	default:
	}
	return nil
}

// OnExit is called when the parser exits a rule context.
func (r *ColumnAddNotNullColumnRequireDefaultRule) OnExit(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case "Alter_table":
		r.handleAlterTableExit()
	case "Column_definition":
		r.handleColumnDefinitionExit(ctx.(*parser.Column_definitionContext))
	default:
	}
	return nil
}

func (r *ColumnAddNotNullColumnRequireDefaultRule) handleAlterTable(ctx *parser.Alter_tableContext) {
	r.tableName = normalizeIdentifier(ctx.Tableview_name(), r.currentDatabase)
}

func (r *ColumnAddNotNullColumnRequireDefaultRule) handleAlterTableExit() {
	r.tableName = ""
}

func (r *ColumnAddNotNullColumnRequireDefaultRule) handleInlineConstraint(ctx *parser.Inline_constraintContext) {
	if ctx.NOT() != nil {
		r.isNotNull = true
	}
}

func (r *ColumnAddNotNullColumnRequireDefaultRule) handleColumnDefinitionEnter(_ *parser.Column_definitionContext) {
	r.isNotNull = false
}

func (r *ColumnAddNotNullColumnRequireDefaultRule) handleColumnDefinitionExit(ctx *parser.Column_definitionContext) {
	if r.tableName == "" || !r.isNotNull {
		return
	}

	if ctx.DEFAULT() == nil {
		r.AddAdvice(
			r.level,
			code.NotNullColumnWithNoDefault.Int32(),
			fmt.Sprintf("Adding not null column %q requires default.", normalizeIdentifier(ctx.Column_name(), r.currentDatabase)),
			common.ConvertANTLRLineToPosition(r.baseLine+ctx.GetStart().GetLine()),
		)
	}
}
