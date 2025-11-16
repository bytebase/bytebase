package snowflake

import (
	"context"
	"fmt"
	"strconv"

	"github.com/bytebase/bytebase/backend/plugin/advisor/code"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/snowflake"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
)

const (
	// varcharDefaultLength is the default length of varchar in Snowflake.
	// https://docs.snowflake.com/en/sql-reference/data-types-text
	varcharDefaultLength = 16_777_216
)

var (
	_ advisor.Advisor = (*ColumnMaximumVarcharLengthAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_SNOWFLAKE, advisor.SchemaRuleColumnMaximumVarcharLength, &ColumnMaximumVarcharLengthAdvisor{})
}

// ColumnMaximumVarcharLengthAdvisor is the advisor checking for maximum varchar length.
type ColumnMaximumVarcharLengthAdvisor struct {
}

// Check checks for maximum varchar length.
func (*ColumnMaximumVarcharLengthAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	tree, ok := checkCtx.AST.(antlr.Tree)
	if !ok {
		return nil, errors.Errorf("failed to convert to Tree")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	payload, err := advisor.UnmarshalNumberTypeRulePayload(checkCtx.Rule.Payload)
	if err != nil {
		return nil, err
	}

	if payload.Number <= 0 {
		return nil, nil
	}

	// Create the rule
	rule := NewColumnMaximumVarcharLengthRule(level, string(checkCtx.Rule.Type), payload.Number)

	// Create the generic checker with the rule
	checker := NewGenericChecker([]Rule{rule})

	antlr.ParseTreeWalkerDefault.Walk(checker, tree)

	return checker.GetAdviceList(), nil
}

// ColumnMaximumVarcharLengthRule checks for maximum varchar length.
type ColumnMaximumVarcharLengthRule struct {
	BaseRule
	maximum int
}

// NewColumnMaximumVarcharLengthRule creates a new ColumnMaximumVarcharLengthRule.
func NewColumnMaximumVarcharLengthRule(level storepb.Advice_Status, title string, maximum int) *ColumnMaximumVarcharLengthRule {
	return &ColumnMaximumVarcharLengthRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
		maximum: maximum,
	}
}

// Name returns the rule name.
func (*ColumnMaximumVarcharLengthRule) Name() string {
	return "ColumnMaximumVarcharLengthRule"
}

// OnEnter is called when entering a parse tree node.
func (r *ColumnMaximumVarcharLengthRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	if nodeType == NodeTypeDataType {
		r.checkDataType(ctx.(*parser.Data_typeContext))
	}
	return nil
}

// OnExit is called when exiting a parse tree node.
func (*ColumnMaximumVarcharLengthRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	// This rule doesn't need exit processing
	return nil
}

func (r *ColumnMaximumVarcharLengthRule) checkDataType(ctx *parser.Data_typeContext) {
	if ctx.VARCHAR() == nil {
		return
	}

	length := varcharDefaultLength
	if v := ctx.Num(0); v != nil {
		var err error
		length, err = strconv.Atoi(v.GetText())
		if err != nil {
			return
		}
	}

	if length > r.maximum {
		r.AddAdvice(&storepb.Advice{
			Status:        r.level,
			Code:          code.VarcharLengthExceedsLimit.Int32(),
			Title:         r.title,
			Content:       fmt.Sprintf("The maximum varchar length is %d.", r.maximum),
			StartPosition: common.ConvertANTLRLineToPosition(r.baseLine + ctx.GetStart().GetLine()),
		})
	}
}
