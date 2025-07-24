// Package oracle is the advisor for oracle database.
package oracle

import (
	"context"
	"fmt"
	"strconv"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/plsql-parser"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
)

var (
	_ advisor.Advisor = (*ColumnMaximumCharacterLengthAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_ORACLE, advisor.OracleColumnMaximumCharacterLength, &ColumnMaximumCharacterLengthAdvisor{})
	advisor.Register(storepb.Engine_DM, advisor.OracleColumnMaximumCharacterLength, &ColumnMaximumCharacterLengthAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE_ORACLE, advisor.OracleColumnMaximumCharacterLength, &ColumnMaximumCharacterLengthAdvisor{})
}

// ColumnMaximumCharacterLengthAdvisor is the advisor checking for maximum character length.
type ColumnMaximumCharacterLengthAdvisor struct {
}

// Check checks for maximum character length.
func (*ColumnMaximumCharacterLengthAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
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

	rule := NewColumnMaximumCharacterLengthRule(level, string(checkCtx.Rule.Type), payload.Number)
	checker := NewGenericChecker([]Rule{rule})

	antlr.ParseTreeWalkerDefault.Walk(checker, tree)

	return checker.GetAdviceList()
}

// ColumnMaximumCharacterLengthRule is the rule implementation for maximum character length.
type ColumnMaximumCharacterLengthRule struct {
	BaseRule

	maximum int
}

// NewColumnMaximumCharacterLengthRule creates a new ColumnMaximumCharacterLengthRule.
func NewColumnMaximumCharacterLengthRule(level storepb.Advice_Status, title string, maximum int) *ColumnMaximumCharacterLengthRule {
	return &ColumnMaximumCharacterLengthRule{
		BaseRule: NewBaseRule(level, title, 0),
		maximum:  maximum,
	}
}

// Name returns the rule name.
func (*ColumnMaximumCharacterLengthRule) Name() string {
	return "column.maximum-character-length"
}

// OnEnter is called when the parser enters a rule context.
func (r *ColumnMaximumCharacterLengthRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	if nodeType == "Datatype" {
		r.handleDatatype(ctx.(*parser.DatatypeContext))
	}
	return nil
}

// OnExit is called when the parser exits a rule context.
func (*ColumnMaximumCharacterLengthRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

func (r *ColumnMaximumCharacterLengthRule) handleDatatype(ctx *parser.DatatypeContext) {
	if ctx.Native_datatype_element() == nil {
		return
	}

	if ctx.Native_datatype_element().CHAR() == nil && ctx.Native_datatype_element().CHARACTER() == nil {
		return
	}

	if ctx.Precision_part() == nil {
		return
	}

	if ctx.Precision_part().Numeric(0) != nil {
		lengthText := ctx.Precision_part().Numeric(0).GetText()
		length, err := strconv.Atoi(lengthText)
		if err != nil || length <= r.maximum {
			return
		}
	}

	r.AddAdvice(
		r.level,
		advisor.CharLengthExceedsLimit.Int32(),
		fmt.Sprintf("The maximum character length is %d.", r.maximum),
		common.ConvertANTLRLineToPosition(ctx.GetStart().GetLine()),
	)
}
