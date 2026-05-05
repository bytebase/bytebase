package tidb

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/tidb/ast"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*NoSelectAllAdvisor)(nil)
	_ ast.Visitor     = (*noSelectAllVisitor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_STATEMENT_SELECT_NO_SELECT_ALL, &NoSelectAllAdvisor{})
}

// NoSelectAllAdvisor is the advisor checking for no "select *".
type NoSelectAllAdvisor struct {
}

// Check checks for no "select *". Walks the full statement tree because
// the wildcard can appear in nested subqueries (e.g. SELECT a FROM
// (SELECT * FROM t) t).
func (*NoSelectAllAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmts, err := getTiDBOmniNodes(checkCtx)
	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	var adviceList []*storepb.Advice
	for _, ostmt := range stmts {
		// Pingcap parity: every wildcard hit reports the OUTER statement's
		// first-token line, not the inner SelectStmt's line.
		v := &noSelectAllVisitor{
			level:   level,
			title:   checkCtx.Rule.Type.String(),
			text:    ostmt.TrimmedText(),
			line:    ostmt.FirstTokenLine(),
			advices: &adviceList,
		}
		ast.Walk(v, ostmt.Node)
	}

	return adviceList, nil
}

type noSelectAllVisitor struct {
	level   storepb.Advice_Status
	title   string
	text    string
	line    int
	advices *[]*storepb.Advice
}

// Visit returns v to recurse into children, including SelectStmt.Left/Right
// (UNION/INTERSECT/EXCEPT arms) per omni's Walk contract.
func (v *noSelectAllVisitor) Visit(node ast.Node) ast.Visitor {
	if node == nil {
		return nil
	}
	sel, ok := node.(*ast.SelectStmt)
	if !ok {
		return v
	}
	if !targetListHasStar(sel.TargetList) {
		return v
	}
	*v.advices = append(*v.advices, &storepb.Advice{
		Status:        v.level,
		Code:          code.StatementSelectAll.Int32(),
		Title:         v.title,
		Content:       fmt.Sprintf("\"%s\" uses SELECT all", v.text),
		StartPosition: common.ConvertANTLRLineToPosition(v.line),
	})
	return v
}

// targetListHasStar reports whether any item in a SELECT's target list is
// a wildcard. omni distinguishes:
//   - bare "*"        → *ast.StarExpr
//   - qualified "t.*" → *ast.ColumnRef with Star: true
func targetListHasStar(targets []ast.ExprNode) bool {
	for _, expr := range targets {
		switch e := expr.(type) {
		case *ast.StarExpr:
			return true
		case *ast.ColumnRef:
			if e.Star {
				return true
			}
		default:
		}
	}
	return false
}
