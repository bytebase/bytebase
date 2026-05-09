package tidb

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/tidb/ast"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	advisorcode "github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*WhereRequirementForUpdateDeleteAdvisor)(nil)
	_ ast.Visitor     = (*whereRequireUpdateDeleteVisitor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_STATEMENT_WHERE_REQUIRE_UPDATE_DELETE, &WhereRequirementForUpdateDeleteAdvisor{})
}

// WhereRequirementForUpdateDeleteAdvisor checks the WHERE clause
// requirement for UPDATE and DELETE statements.
type WhereRequirementForUpdateDeleteAdvisor struct {
}

// Check is Recipe B (sub-walk via omni Visitor). The initial batch-5
// implementation was Recipe A on the assumption that UPDATE and DELETE
// can't nest in standard SQL — but that missed statement-wrapper forms
// like `EXPLAIN DELETE FROM t` and `EXPLAIN UPDATE t SET x = 1`. Both
// pingcap and omni produce *ExplainStmt{Stmt: *DeleteStmt}, and pingcap's
// Visitor walked into the inner DML to fire the rule. A Recipe A
// migration silently regresses on the EXPLAIN-wrapped form.
//
// Per Phase 1.5 cumulative shape divergence #13 (Codex P2 round-1 catch
// on PR #20211): wrapper statements (EXPLAIN, TRACE, DESCRIBE, ...) embed
// DML at omni's nodes that mechanical top-level type-switch ignores.
// Recipe B with omniast.Walk handles every wrapper kind automatically.
func (*WhereRequirementForUpdateDeleteAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmts, err := getTiDBOmniNodes(checkCtx)
	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	title := checkCtx.Rule.Type.String()
	var adviceList []*storepb.Advice
	for _, ostmt := range stmts {
		// Pingcap parity: every match reports the OUTER statement's
		// trimmed text and first-token line. For EXPLAIN DELETE FROM t,
		// the advice content is the EXPLAIN statement, not the inner
		// DeleteStmt — same as pingcap's `checker.text/line` set once
		// before Accept().
		v := &whereRequireUpdateDeleteVisitor{
			level:   level,
			title:   title,
			text:    ostmt.TrimmedText(),
			line:    ostmt.FirstTokenLine(),
			advices: &adviceList,
		}
		ast.Walk(v, ostmt.Node)
	}
	return adviceList, nil
}

type whereRequireUpdateDeleteVisitor struct {
	level   storepb.Advice_Status
	title   string
	text    string
	line    int
	advices *[]*storepb.Advice
}

// Visit returns v to recurse into children, including ExplainStmt.Stmt
// and other statement-wrapper forms.
func (v *whereRequireUpdateDeleteVisitor) Visit(node ast.Node) ast.Visitor {
	if node == nil {
		return nil
	}
	var noWhere bool
	switch n := node.(type) {
	case *ast.DeleteStmt:
		if n.Where == nil {
			noWhere = true
		}
	case *ast.UpdateStmt:
		if n.Where == nil {
			noWhere = true
		}
	default:
	}
	if noWhere {
		*v.advices = append(*v.advices, &storepb.Advice{
			Status:        v.level,
			Code:          advisorcode.StatementNoWhere.Int32(),
			Title:         v.title,
			Content:       fmt.Sprintf("\"%s\" requires WHERE clause", v.text),
			StartPosition: common.ConvertANTLRLineToPosition(v.line),
		})
	}
	return v
}
