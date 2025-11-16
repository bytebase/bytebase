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
	_ advisor.Advisor = (*ColumnTypeDisallowListAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_ORACLE, advisor.SchemaRuleColumnTypeDisallowList, &ColumnTypeDisallowListAdvisor{})
}

// ColumnTypeDisallowListAdvisor is the advisor checking for column type disallow list.
type ColumnTypeDisallowListAdvisor struct {
}

// Check checks for column type disallow list.
func (*ColumnTypeDisallowListAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmtList, ok := checkCtx.AST.([]*plsqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to ParseResult")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	payload, err := advisor.UnmarshalStringArrayTypeRulePayload(checkCtx.Rule.Payload)
	if err != nil {
		return nil, err
	}

	rule := NewColumnTypeDisallowListRule(level, string(checkCtx.Rule.Type), checkCtx.CurrentDatabase, payload.List)
	checker := NewGenericChecker([]Rule{rule})

	for _, stmtNode := range stmtList {
		rule.SetBaseLine(stmtNode.BaseLine)
		checker.SetBaseLine(stmtNode.BaseLine)
		antlr.ParseTreeWalkerDefault.Walk(checker, stmtNode.Tree)
	}

	return checker.GetAdviceList()
}

// ColumnTypeDisallowListRule is the rule implementation for column type disallow list.
type ColumnTypeDisallowListRule struct {
	BaseRule

	currentDatabase string
	disallowList    []string
}

// NewColumnTypeDisallowListRule creates a new ColumnTypeDisallowListRule.
func NewColumnTypeDisallowListRule(level storepb.Advice_Status, title string, currentDatabase string, disallowList []string) *ColumnTypeDisallowListRule {
	return &ColumnTypeDisallowListRule{
		BaseRule:        NewBaseRule(level, title, 0),
		currentDatabase: currentDatabase,
		disallowList:    disallowList,
	}
}

// Name returns the rule name.
func (*ColumnTypeDisallowListRule) Name() string {
	return "column.type-disallow-list"
}

// OnEnter is called when the parser enters a rule context.
func (r *ColumnTypeDisallowListRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case "Column_definition":
		r.handleColumnDefinition(ctx.(*parser.Column_definitionContext))
	case "Modify_col_properties":
		r.handleModifyColProperties(ctx.(*parser.Modify_col_propertiesContext))
	default:
		// Ignore other node types
	}
	return nil
}

// OnExit is called when the parser exits a rule context.
func (*ColumnTypeDisallowListRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

func (r *ColumnTypeDisallowListRule) isDisallowType(tp parser.IDatatypeContext) bool {
	if tp == nil {
		return false
	}
	for _, disallowType := range r.disallowList {
		if equivalent, err := plsqlparser.EquivalentType(tp, disallowType); err == nil && equivalent {
			return true
		}
	}
	return false
}

func (r *ColumnTypeDisallowListRule) handleColumnDefinition(ctx *parser.Column_definitionContext) {
	if r.isDisallowType(ctx.Datatype()) {
		r.AddAdvice(
			r.level,
			code.DisabledColumnType.Int32(),
			fmt.Sprintf("Disallow column type %s but column \"%s\" is", ctx.Datatype().GetText(), normalizeIdentifier(ctx.Column_name(), r.currentDatabase)),
			common.ConvertANTLRLineToPosition(r.baseLine+ctx.Datatype().GetStart().GetLine()),
		)
	}
	if ctx.Regular_id() != nil {
		for _, tp := range r.disallowList {
			if ctx.Regular_id().GetText() == tp {
				r.AddAdvice(
					r.level,
					code.DisabledColumnType.Int32(),
					fmt.Sprintf("Disallow column type %s but column \"%s\" is", ctx.Regular_id().GetText(), normalizeIdentifier(ctx.Column_name(), r.currentDatabase)),
					common.ConvertANTLRLineToPosition(r.baseLine+ctx.Regular_id().GetStart().GetLine()),
				)
				break
			}
		}
	}
}

func (r *ColumnTypeDisallowListRule) handleModifyColProperties(ctx *parser.Modify_col_propertiesContext) {
	if r.isDisallowType(ctx.Datatype()) {
		r.AddAdvice(
			r.level,
			code.DisabledColumnType.Int32(),
			fmt.Sprintf("Disallow column type %s but column \"%s\" is", ctx.Datatype().GetText(), normalizeIdentifier(ctx.Column_name(), r.currentDatabase)),
			common.ConvertANTLRLineToPosition(r.baseLine+ctx.Datatype().GetStart().GetLine()),
		)
	}
}
