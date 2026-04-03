package mysql

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/omni/mysql/ast"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*InsertDisallowOrderByRandAdvisor)(nil)
)

const RandFn = "rand()"

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_STATEMENT_INSERT_DISALLOW_ORDER_BY_RAND, &InsertDisallowOrderByRandAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_STATEMENT_INSERT_DISALLOW_ORDER_BY_RAND, &InsertDisallowOrderByRandAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_STATEMENT_INSERT_DISALLOW_ORDER_BY_RAND, &InsertDisallowOrderByRandAdvisor{})
}

// InsertDisallowOrderByRandAdvisor is the advisor checking for to disallow order by rand in INSERT statements.
type InsertDisallowOrderByRandAdvisor struct {
}

// Check checks for to disallow order by rand in INSERT statements.
func (*InsertDisallowOrderByRandAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := &insertDisallowOrderByRandOmniRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
	}

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type insertDisallowOrderByRandOmniRule struct {
	OmniBaseRule
}

func (*insertDisallowOrderByRandOmniRule) Name() string {
	return "InsertDisallowOrderByRandRule"
}

func (r *insertDisallowOrderByRandOmniRule) OnStatement(node ast.Node) {
	ins, ok := node.(*ast.InsertStmt)
	if !ok {
		return
	}

	// Only check INSERT ... SELECT statements.
	if ins.Select == nil {
		return
	}

	text := r.TrimmedStmtText() + ";"

	r.checkSelectForRandOrderBy(ins.Select, text)
}

func (r *insertDisallowOrderByRandOmniRule) checkSelectForRandOrderBy(sel *ast.SelectStmt, text string) {
	if sel == nil {
		return
	}

	for _, item := range sel.OrderBy {
		if r.isRandFunc(item.Expr) {
			r.AddAdviceAbsolute(&storepb.Advice{
				Status:        r.Level,
				Code:          code.InsertUseOrderByRand.Int32(),
				Title:         r.Title,
				Content:       fmt.Sprintf("\"%s\" uses ORDER BY RAND in the INSERT statement", text),
				StartPosition: common.ConvertANTLRLineToPosition(r.BaseLine + int(r.ContentStartLine())),
			})
		}
	}

	// Check set operations.
	if sel.SetOp != ast.SetOpNone {
		r.checkSelectForRandOrderBy(sel.Left, text)
		r.checkSelectForRandOrderBy(sel.Right, text)
	}
}

func (*insertDisallowOrderByRandOmniRule) isRandFunc(expr ast.ExprNode) bool {
	if expr == nil {
		return false
	}
	fc, ok := expr.(*ast.FuncCallExpr)
	if !ok {
		return false
	}
	return strings.EqualFold(fc.Name, "rand")
}
