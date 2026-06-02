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
	_ advisor.Advisor = (*ColumnMaximumVarcharLengthAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_ORACLE, storepb.SQLReviewRule_COLUMN_MAXIMUM_VARCHAR_LENGTH, &ColumnMaximumVarcharLengthAdvisor{})
}

// ColumnMaximumVarcharLengthAdvisor is the advisor checking for maximum varchar length.
type ColumnMaximumVarcharLengthAdvisor struct {
}

// Check checks for maximum varchar length.
func (*ColumnMaximumVarcharLengthAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
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

	rule := NewColumnMaximumVarcharLengthRule(level, checkCtx.Rule.Type.String(), int(numberPayload.Number))

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule})
}

// ColumnMaximumVarcharLengthRule is the rule implementation for maximum varchar length.
type ColumnMaximumVarcharLengthRule struct {
	BaseRule

	maximum int
}

// NewColumnMaximumVarcharLengthRule creates a new ColumnMaximumVarcharLengthRule.
func NewColumnMaximumVarcharLengthRule(level storepb.Advice_Status, title string, maximum int) *ColumnMaximumVarcharLengthRule {
	return &ColumnMaximumVarcharLengthRule{
		BaseRule: NewBaseRule(level, title, 0),
		maximum:  maximum,
	}
}

// Name returns the rule name.
func (*ColumnMaximumVarcharLengthRule) Name() string {
	return "column.maximum-varchar-length"
}

// OnStatement checks VARCHAR/VARCHAR2 type modifiers in the omni AST.
func (r *ColumnMaximumVarcharLengthRule) OnStatement(node ast.Node) {
	omniWalk(node, func(n ast.Node) {
		col, ok := n.(*ast.ColumnDef)
		if !ok || col.TypeName == nil {
			return
		}
		typeName := omniTypeName(col.TypeName)
		if typeName != "VARCHAR" && typeName != "VARCHAR2" {
			return
		}
		length, ok := omniFirstTypeModInt(col.TypeName)
		if !ok || length <= r.maximum {
			return
		}
		r.AddAdvice(
			r.level,
			code.VarcharLengthExceedsLimit.Int32(),
			fmt.Sprintf("The maximum varchar length is %d.", r.maximum),
			common.ConvertANTLRLineToPosition(r.locLine(col.TypeName.Loc)),
		)
	})
}

// OnEnter is called when the parser enters a rule context.

// OnExit is called when the parser exits a rule context.
