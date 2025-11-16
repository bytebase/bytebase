package mssql

import (
	"context"
	"slices"
	"strings"

	"github.com/bytebase/bytebase/backend/plugin/advisor/code"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/tsql"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
)

var (
	_ advisor.Advisor = (*ColumnTypeDisallowListAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MSSQL, advisor.SchemaRuleColumnTypeDisallowList, &ColumnTypeDisallowListAdvisor{})
}

// ColumnTypeDisallowListAdvisor is the advisor checking for disallowed types for column.
type ColumnTypeDisallowListAdvisor struct {
}

func (*ColumnTypeDisallowListAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	tree, ok := checkCtx.AST.(antlr.Tree)
	if !ok {
		return nil, errors.Errorf("failed to convert to Tree")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	payload, err := advisor.UnmarshalStringArrayTypeRulePayload(checkCtx.Rule.Payload)
	if err != nil {
		return nil, err
	}

	disallowTypes := []string{}
	for _, tp := range payload.List {
		disallowTypes = append(disallowTypes, strings.ToUpper(tp))
	}

	// Create the rule
	rule := NewColumnTypeDisallowListRule(level, string(checkCtx.Rule.Type), disallowTypes)

	// Create the generic checker with the rule
	checker := NewGenericChecker([]Rule{rule})

	antlr.ParseTreeWalkerDefault.Walk(checker, tree)

	return checker.GetAdviceList(), nil
}

// ColumnTypeDisallowListRule is the rule for column disallow types.
type ColumnTypeDisallowListRule struct {
	BaseRule
	disallowTypes []string
}

// NewColumnTypeDisallowListRule creates a new ColumnTypeDisallowListRule.
func NewColumnTypeDisallowListRule(level storepb.Advice_Status, title string, disallowTypes []string) *ColumnTypeDisallowListRule {
	return &ColumnTypeDisallowListRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
		disallowTypes: disallowTypes,
	}
}

// Name returns the rule name.
func (*ColumnTypeDisallowListRule) Name() string {
	return "ColumnTypeDisallowListRule"
}

// OnEnter is called when entering a parse tree node.
func (r *ColumnTypeDisallowListRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	if nodeType == "Data_type" {
		r.enterDataType(ctx.(*parser.Data_typeContext))
	}
	return nil
}

// OnExit is called when exiting a parse tree node.
func (*ColumnTypeDisallowListRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	// This rule doesn't need exit processing
	return nil
}

func (r *ColumnTypeDisallowListRule) enterDataType(ctx *parser.Data_typeContext) {
	formatedDataType := strings.ToUpper(ctx.GetText())
	if slices.Contains(r.disallowTypes, formatedDataType) {
		r.AddAdvice(&storepb.Advice{
			Status:        r.level,
			Code:          code.DisabledColumnType.Int32(),
			Title:         r.title,
			Content:       "Column type " + formatedDataType + " is disallowed",
			StartPosition: common.ConvertANTLRLineToPosition(ctx.GetStart().GetLine()),
		})
	}
}
