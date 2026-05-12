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
	_ advisor.Advisor = (*InsertDisallowOrderByRandAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_STATEMENT_INSERT_DISALLOW_ORDER_BY_RAND, &InsertDisallowOrderByRandAdvisor{})
}

// InsertDisallowOrderByRandAdvisor is the advisor checking for to disallow order by rand in INSERT statements.
type InsertDisallowOrderByRandAdvisor struct {
}

// Check checks for to disallow order by rand in INSERT statements.
func (*InsertDisallowOrderByRandAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
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
		advice := checkStmtForOrderByRand(ostmt, level, title)
		if advice != nil {
			adviceList = append(adviceList, advice)
		}
	}

	return adviceList, nil
}

// checkStmtForOrderByRand returns at most ONE advice per top-level
// statement, mirroring pingcap-typed predecessor's break-after-first-
// RAND-match cardinality. Mysql analog emits per-item (no break) AND
// recurses into UNION set-op arms; tidb preserves the narrower
// pingcap-tidb scope (no SetOp recursion, single advice per stmt).
//
// Cumulative #1 framing: INSERT-VALUES / INSERT-SET / INSERT-TABLE
// forms have `ins.Select == nil` and skip. Only INSERT ... SELECT can
// have an ORDER BY; only that path is checked.
//
// Long-standing pre-omni tidb gap (preserved per invariant #10): the
// pingcap Enter type-asserted `insert.Select.(*ast.SelectStmt)`, so
// for `INSERT INTO t SELECT ... UNION SELECT ... ORDER BY RAND()` the
// outer pingcap shape is a SetOprStmt and the cast fails — rule did
// not fire. Omni's `InsertStmt.Select` is `*SelectStmt` with
// `SetOp != SetOpNone` carrying the union; we only check the top-
// level OrderBy, leaving UNION-arm OrderBy paths uncovered to match
// pingcap-tidb behavior. (Mysql omni recurses; that's an enhancement
// available if the tidb gap surfaces as customer signal.)
func checkStmtForOrderByRand(ostmt OmniStmt, level storepb.Advice_Status, title string) *storepb.Advice {
	ins, ok := ostmt.Node.(*ast.InsertStmt)
	if !ok {
		return nil
	}
	if ins.Select == nil {
		return nil
	}
	for _, item := range ins.Select.OrderBy {
		if item == nil {
			continue
		}
		if omniIsRandFuncCall(item.Expr) {
			return &storepb.Advice{
				Status:        level,
				Code:          advisorcode.InsertUseOrderByRand.Int32(),
				Title:         title,
				Content:       fmt.Sprintf("\"%s\" uses ORDER BY RAND in the INSERT statement", ostmt.TrimmedText()),
				StartPosition: common.ConvertANTLRLineToPosition(ostmt.FirstTokenLine()),
			}
		}
	}
	return nil
}
