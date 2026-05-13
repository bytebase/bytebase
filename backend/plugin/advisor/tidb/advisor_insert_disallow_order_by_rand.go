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
// RAND-match cardinality. Mysql analog emits per-item (no break);
// tidb preserves pingcap-tidb's single-advice-per-stmt contract.
//
// Cumulative #1 framing: INSERT-VALUES / INSERT-SET / INSERT-TABLE
// forms have `ins.Select == nil` and skip. Only INSERT ... SELECT can
// have an ORDER BY; only that path is checked.
//
// Cumulative #24 — silent UX improvement at the UNION boundary:
// pingcap's `InsertStmt.Select` is a `ResultSetNode` interface;
// UNION'd inserts produce `*ast.SetOprStmt`. The pre-omni rule
// type-asserted `insert.Select.(*ast.SelectStmt)` — the cast failed
// for UNION'd inputs and silently skipped the whole check. Omni's
// `InsertStmt.Select` is `*SelectStmt` (concrete) regardless of
// SetOp, so `ins.Select.OrderBy` here IS the outer-UNION ORDER BY
// list. The rule now fires on `INSERT ... SELECT ... UNION ...
// ORDER BY RAND()` (outer-UNION position) — matches the rule's
// stated intent. NOT a regression. Inner per-arm OrderBy (in
// parenthesized UNION arms, if/when omni grammar accepts that
// syntax in INSERT position — currently rejected) remains
// uncovered, matching pingcap-tidb's per-arm behavior.
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
