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

const (
	wildcard byte = '%'
)

var (
	_ advisor.Advisor = (*NoLeadingWildcardLikeAdvisor)(nil)
	_ ast.Visitor     = (*noLeadingWildcardLikeVisitor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_STATEMENT_WHERE_NO_LEADING_WILDCARD_LIKE, &NoLeadingWildcardLikeAdvisor{})
}

// NoLeadingWildcardLikeAdvisor checks for LIKE patterns that begin with
// `%`, e.g. `WHERE x LIKE '%foo'`. Such patterns force a full table scan;
// the rule warns even on NOT LIKE because the same scan cost applies.
type NoLeadingWildcardLikeAdvisor struct {
}

// Check is Recipe B (sub-walk via omni Visitor) — LIKE expressions can
// appear in any WHERE clause depth, including UNION arms, subqueries, and
// CTEs.
//
// Per pingcap-typed advisor's behavior (preserved here):
//   - Fires once per top-level statement, regardless of how many leading-`%`
//     LIKE expressions it contains.
//   - Doesn't distinguish LIKE from NOT LIKE — both trigger.
//   - Only `%` triggers; `_` (single-character wildcard) doesn't.
//   - ESCAPE clause doesn't affect the pattern check.
//   - Non-string-literal patterns (parameter `?`, column refs, function
//     calls) silently skip — pingcap's restoreNode produced text starting
//     with `?` / column-name / function-name, none of which start with `%`.
func (*NoLeadingWildcardLikeAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
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
		v := &noLeadingWildcardLikeVisitor{}
		ast.Walk(v, ostmt.Node)
		if !v.found {
			continue
		}
		adviceList = append(adviceList, &storepb.Advice{
			Status:        level,
			Code:          code.StatementLeadingWildcardLike.Int32(),
			Title:         title,
			Content:       fmt.Sprintf("\"%s\" uses leading wildcard LIKE", ostmt.TrimmedText()),
			StartPosition: common.ConvertANTLRLineToPosition(ostmt.FirstTokenLine()),
		})
	}
	return adviceList, nil
}

type noLeadingWildcardLikeVisitor struct {
	found bool
}

// Visit returns v to recurse into children. Once a leading-`%` is found,
// returns nil to skip the remaining subtree — the check is per-statement
// boolean, no benefit to continuing.
func (v *noLeadingWildcardLikeVisitor) Visit(node ast.Node) ast.Visitor {
	if node == nil || v.found {
		return nil
	}
	like, ok := node.(*ast.LikeExpr)
	if !ok {
		return v
	}
	// Pingcap doesn't gate on like.Not — NOT LIKE with leading wildcard
	// triggers the same as LIKE. Mirror that behavior.
	//
	// Cumulative shape divergence #11: omni wraps `'%abc' COLLATE
	// utf8mb4_bin` as *CollateExpr{Expr: *StringLit, Collation: "..."}.
	// Pingcap's *SetCollationExpr restored to text as
	// "%abc COLLATE utf8mb4_bin" (value first, COLLATE clause after), so
	// pingcap's leading-`%` check accidentally fired on the rendered text.
	// Without the unwrap, omni would silently skip and regress vs. pingcap.
	// The loop handles nested COLLATE defensively (not produced by current
	// TiDB grammar but cheap insurance).
	pattern := like.Pattern
	for {
		c, ok := pattern.(*ast.CollateExpr)
		if !ok {
			break
		}
		pattern = c.Expr
	}
	lit, ok := pattern.(*ast.StringLit)
	if !ok {
		// Non-literal patterns (parameter, column ref, expression) — skip.
		// Per pingcap parity (see Check() docstring).
		return v
	}
	if len(lit.Value) > 0 && lit.Value[0] == wildcard {
		v.found = true
		return nil
	}
	return v
}
