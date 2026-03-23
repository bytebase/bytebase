package pg

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	"github.com/bytebase/omni/pg/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*StatementMaximumLimitValueAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_STATEMENT_MAXIMUM_LIMIT_VALUE, &StatementMaximumLimitValueAdvisor{})
}

// StatementMaximumLimitValueAdvisor is the advisor checking for maximum LIMIT value.
type StatementMaximumLimitValueAdvisor struct {
}

// Check checks for maximum LIMIT value.
func (*StatementMaximumLimitValueAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	numberPayload := checkCtx.Rule.GetNumberPayload()
	if numberPayload == nil {
		return nil, errors.New("number_payload is required for this rule")
	}

	rule := &statementMaximumLimitValueRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
		limitMaxValue: int(numberPayload.Number),
	}

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type statementMaximumLimitValueRule struct {
	OmniBaseRule
	limitMaxValue int
}

func (*statementMaximumLimitValueRule) Name() string {
	return string(storepb.SQLReviewRule_STATEMENT_MAXIMUM_LIMIT_VALUE)
}

func (r *statementMaximumLimitValueRule) OnStatement(node ast.Node) {
	sel, ok := node.(*ast.SelectStmt)
	if !ok {
		return
	}
	r.checkSelectStmt(sel)
}

func (r *statementMaximumLimitValueRule) checkSelectStmt(sel *ast.SelectStmt) {
	if sel == nil {
		return
	}

	// For set operations, recurse into children.
	if sel.Op != ast.SETOP_NONE {
		r.checkSelectStmt(sel.Larg)
		r.checkSelectStmt(sel.Rarg)
		return
	}

	limitValue := r.extractLimitValue(sel.LimitCount)
	if limitValue > 0 && limitValue > r.limitMaxValue {
		r.AddAdvice(&storepb.Advice{
			Status:  r.Level,
			Code:    code.StatementExceedMaximumLimitValue.Int32(),
			Title:   r.Title,
			Content: fmt.Sprintf("The limit value %d exceeds the maximum allowed value %d", limitValue, r.limitMaxValue),
			StartPosition: &storepb.Position{
				Line:   r.ContentStartLine(),
				Column: 0,
			},
		})
	}
}

// extractLimitValue extracts the integer limit value from a LimitCount node.
func (*statementMaximumLimitValueRule) extractLimitValue(node ast.Node) int {
	if node == nil {
		return 0
	}
	switch n := node.(type) {
	case *ast.Integer:
		return int(n.Ival)
	case *ast.A_Const:
		if iv, ok := n.Val.(*ast.Integer); ok {
			return int(iv.Ival)
		}
	default:
	}
	return 0
}
