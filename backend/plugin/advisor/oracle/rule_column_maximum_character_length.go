// Package oracle is the advisor for oracle database.
package oracle

import (
	"context"
	"fmt"
	"strconv"

	"github.com/pkg/errors"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/plsql"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

var (
	_ advisor.Advisor = (*ColumnMaximumCharacterLengthAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_ORACLE, storepb.SQLReviewRule_COLUMN_MAXIMUM_CHARACTER_LENGTH, &ColumnMaximumCharacterLengthAdvisor{})
}

// ColumnMaximumCharacterLengthAdvisor is the advisor checking for maximum character length.
type ColumnMaximumCharacterLengthAdvisor struct {
}

// Check checks for maximum character length.
func (*ColumnMaximumCharacterLengthAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	numberPayload := checkCtx.Rule.GetNumberPayload()
	if numberPayload == nil {
		return nil, errors.New("number_payload is required for this rule")
	}

	if int(numberPayload.Number) <= 0 {
		return nil, nil
	}

	rule := NewColumnMaximumCharacterLengthRule(level, checkCtx.Rule.Type.String(), int(numberPayload.Number))
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
		code.CharLengthExceedsLimit.Int32(),
		fmt.Sprintf("The maximum character length is %d.", r.maximum),
		common.ConvertANTLRLineToPosition(r.baseLine+ctx.GetStart().GetLine()),
	)
}
