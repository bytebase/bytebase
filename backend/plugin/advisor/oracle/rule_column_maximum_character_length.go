// Package oracle is the advisor for oracle database.
package oracle

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/oracle/ast"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
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

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule})
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

// OnStatement checks CHAR/CHARACTER type modifiers in the omni AST.
func (r *ColumnMaximumCharacterLengthRule) OnStatement(node ast.Node) {
	omniWalk(node, func(n ast.Node) {
		col, ok := n.(*ast.ColumnDef)
		if !ok || col.TypeName == nil {
			return
		}
		typeName := omniTypeName(col.TypeName)
		if typeName != "CHAR" && typeName != "CHARACTER" {
			return
		}
		length, ok := omniFirstTypeModInt(col.TypeName)
		if !ok || length <= r.maximum {
			return
		}
		r.AddAdvice(
			r.level,
			code.CharLengthExceedsLimit.Int32(),
			fmt.Sprintf("The maximum character length is %d.", r.maximum),
			common.ConvertANTLRLineToPosition(r.locLine(col.TypeName.Loc)),
		)
	})
}

// OnEnter is called when the parser enters a rule context.

// OnExit is called when the parser exits a rule context.
