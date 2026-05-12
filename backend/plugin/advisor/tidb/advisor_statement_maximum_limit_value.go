package tidb

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/tidb/ast"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*StatementMaximumLimitValueAdvisor)(nil)
	_ ast.Visitor     = (*maxLimitChecker)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_STATEMENT_MAXIMUM_LIMIT_VALUE, &StatementMaximumLimitValueAdvisor{})
}

// StatementMaximumLimitValueAdvisor flags SELECT statements whose LIMIT
// count exceeds the configured maximum.
type StatementMaximumLimitValueAdvisor struct{}

// Check fires on every SelectStmt (top-level and nested) with
// Limit.Count > maximum. Recipe B (ast.Walk) preserves pingcap-tidb's
// Accept-based recursion semantics: pre-omni Enter ran via
// `stmt.Accept(checker)` which fires on every nested SelectStmt
// (subqueries in FROM, WHERE IN, UNION arms, CTEs). Omni's `ast.Walk`
// recurses into SelectStmt.{CTEs, TargetList, From, Where, GroupBy,
// Having, OrderBy, Limit, Left, Right} per walk_generated.go:720 —
// equivalent traversal coverage.
//
// Scope preservation per invariant #7:
//   - Only `Limit.Count` is checked. `Limit.Offset` is NOT (mysql
//     analog also checks Offset; tidb-omni preserves the narrower
//     pingcap-tidb scope).
//   - Non-IntLit Count values (expressions, placeholders if/when
//     omni grammar accepts them) are silently skipped — matches the
//     pre-omni `_, ok := node.Limit.Count.(*driver.ValueExpr)` cast
//     which would also have failed for non-literal counts.
//   - Strict-greater (`>`, not `>=`) — preserved.
//   - Every advice (including those fired on nested SelectStmts in
//     subqueries) uses the TOP-LEVEL statement's first-token line.
//     Pre-omni rule wrote `checker.line = stmt.OriginTextPosition()`
//     once per top-level and reused it for every advice it emitted
//     during that statement's walk.
func (*StatementMaximumLimitValueAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmts, err := getTiDBOmniNodes(checkCtx)
	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	numberPayload := checkCtx.Rule.GetNumberPayload()
	if numberPayload == nil {
		return nil, errors.New("number_payload is required for maximum limit value rule")
	}
	maximum := int64(numberPayload.Number)
	title := checkCtx.Rule.Type.String()

	var adviceList []*storepb.Advice
	for _, ostmt := range stmts {
		c := &maxLimitChecker{
			level:   level,
			title:   title,
			maximum: maximum,
			line:    ostmt.FirstTokenLine(),
		}
		ast.Walk(c, ostmt.Node)
		adviceList = append(adviceList, c.advices...)
	}
	return adviceList, nil
}

type maxLimitChecker struct {
	level   storepb.Advice_Status
	title   string
	maximum int64
	line    int
	advices []*storepb.Advice
}

// Visit implements ast.Visitor. Returns self to continue recursion;
// `Visit(nil)` is the post-order signal — we handle it with an early
// return at the top of the method.
func (c *maxLimitChecker) Visit(n ast.Node) ast.Visitor {
	if n == nil {
		return c
	}
	sel, ok := n.(*ast.SelectStmt)
	if !ok {
		return c
	}
	if sel.Limit == nil || sel.Limit.Count == nil {
		return c
	}
	lit, ok := sel.Limit.Count.(*ast.IntLit)
	if !ok {
		return c
	}
	if lit.Value > c.maximum {
		c.advices = append(c.advices, &storepb.Advice{
			Status:        c.level,
			Code:          code.StatementExceedMaximumLimitValue.Int32(),
			Title:         c.title,
			Content:       fmt.Sprintf("The limit value %d exceeds the maximum allowed value %d", lit.Value, c.maximum),
			StartPosition: common.ConvertANTLRLineToPosition(c.line),
		})
	}
	return c
}
