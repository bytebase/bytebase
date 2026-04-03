package mysql

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/omni/mysql/ast"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*StatementMaximumLimitValueAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_STATEMENT_MAXIMUM_LIMIT_VALUE, &StatementMaximumLimitValueAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_STATEMENT_MAXIMUM_LIMIT_VALUE, &StatementMaximumLimitValueAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_STATEMENT_MAXIMUM_LIMIT_VALUE, &StatementMaximumLimitValueAdvisor{})
}

type StatementMaximumLimitValueAdvisor struct {
}

func (*StatementMaximumLimitValueAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	numberPayload := checkCtx.Rule.GetNumberPayload()
	if numberPayload == nil {
		return nil, errors.New("number_payload is required for this rule")
	}

	rule := &maxLimitValueOmniRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
		limitMaxValue: int(numberPayload.Number),
	}

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type maxLimitValueOmniRule struct {
	OmniBaseRule
	limitMaxValue int
}

func (*maxLimitValueOmniRule) Name() string {
	return "StatementMaximumLimitValueRule"
}

func (r *maxLimitValueOmniRule) OnStatement(node ast.Node) {
	sel, ok := node.(*ast.SelectStmt)
	if !ok {
		return
	}
	text := strings.TrimSpace(r.StmtText)
	r.checkSelectLimit(sel, text)
}

func (r *maxLimitValueOmniRule) checkSelectLimit(sel *ast.SelectStmt, _ string) {
	if sel == nil {
		return
	}
	if sel.SetOp != ast.SetOpNone {
		r.checkSelectLimit(sel.Left, "")
		r.checkSelectLimit(sel.Right, "")
		// Check top-level limit of set operation.
		if sel.Limit != nil {
			r.checkLimit(sel.Limit)
		}
		return
	}
	if sel.Limit != nil {
		r.checkLimit(sel.Limit)
	}
	// Check subqueries in FROM.
	for _, from := range sel.From {
		r.checkTableExpr(from)
	}
}

func (r *maxLimitValueOmniRule) checkTableExpr(te ast.TableExpr) {
	if te == nil {
		return
	}
	switch t := te.(type) {
	case *ast.SubqueryExpr:
		if t.Select != nil {
			r.checkSelectLimit(t.Select, "")
		}
	case *ast.JoinClause:
		r.checkTableExpr(t.Left)
		r.checkTableExpr(t.Right)
	default:
	}
}

func (r *maxLimitValueOmniRule) checkLimit(limit *ast.Limit) {
	if limit == nil {
		return
	}
	r.checkLimitExpr(limit.Count, limit.Loc)
	r.checkLimitExpr(limit.Offset, limit.Loc)
}

func (r *maxLimitValueOmniRule) checkLimitExpr(expr ast.ExprNode, loc ast.Loc) {
	if expr == nil {
		return
	}
	intLit, ok := expr.(*ast.IntLit)
	if !ok {
		return
	}
	if int(intLit.Value) > r.limitMaxValue {
		r.AddAdviceAbsolute(&storepb.Advice{
			Status:        r.Level,
			Code:          code.StatementExceedMaximumLimitValue.Int32(),
			Title:         r.Title,
			Content:       fmt.Sprintf("The limit value %d exceeds the maximum allowed value %d", intLit.Value, r.limitMaxValue),
			StartPosition: common.ConvertANTLRLineToPosition(r.BaseLine + int(r.LocToLine(loc))),
		})
	}
}
