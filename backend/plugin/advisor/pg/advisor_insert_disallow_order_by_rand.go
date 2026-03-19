package pg

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/omni/pg/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*InsertDisallowOrderByRandAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_STATEMENT_INSERT_DISALLOW_ORDER_BY_RAND, &InsertDisallowOrderByRandAdvisor{})
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

	rule := &insertDisallowOrderByRandRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
	}

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type insertDisallowOrderByRandRule struct {
	OmniBaseRule
}

func (*insertDisallowOrderByRandRule) Name() string {
	return string(storepb.SQLReviewRule_STATEMENT_INSERT_DISALLOW_ORDER_BY_RAND)
}

func (r *insertDisallowOrderByRandRule) OnStatement(node ast.Node) {
	ins, ok := node.(*ast.InsertStmt)
	if !ok {
		return
	}

	sel, ok := ins.SelectStmt.(*ast.SelectStmt)
	if !ok || sel == nil {
		return
	}

	if r.hasOrderByRandom(sel) {
		r.AddAdvice(&storepb.Advice{
			Status:  r.Level,
			Code:    code.InsertUseOrderByRand.Int32(),
			Title:   r.Title,
			Content: fmt.Sprintf("The INSERT statement uses ORDER BY random() or random_between(), related statement \"%s\"", r.TrimmedStmtText()),
			StartPosition: &storepb.Position{
				Line:   r.ContentStartLine(),
				Column: 0,
			},
		})
	}
}

func (r *insertDisallowOrderByRandRule) hasOrderByRandom(sel *ast.SelectStmt) bool {
	if sel == nil {
		return false
	}

	// For set operations, recurse into children.
	if sel.Op != ast.SETOP_NONE {
		return r.hasOrderByRandom(sel.Larg) || r.hasOrderByRandom(sel.Rarg)
	}

	if sel.SortClause == nil {
		return false
	}

	for _, item := range sel.SortClause.Items {
		sb, ok := item.(*ast.SortBy)
		if !ok {
			continue
		}
		fc, ok := sb.Node.(*ast.FuncCall)
		if !ok {
			continue
		}
		funcName := omniListStrings(fc.Funcname)
		for _, name := range funcName {
			lower := strings.ToLower(name)
			if lower == "random" || lower == "random_between" {
				return true
			}
		}
	}

	return false
}
