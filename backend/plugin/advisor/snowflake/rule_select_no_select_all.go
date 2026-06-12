// Package snowflake is the advisor for snowflake database.
package snowflake

import (
	"context"
	"slices"
	"strings"

	omniast "github.com/bytebase/omni/snowflake/ast"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	snowsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/snowflake"
)

var (
	_ advisor.Advisor = (*SelectNoSelectAllAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_SNOWFLAKE, storepb.SQLReviewRule_STATEMENT_SELECT_NO_SELECT_ALL, &SelectNoSelectAllAdvisor{})
}

// SelectNoSelectAllAdvisor is the advisor checking for no select all.
type SelectNoSelectAllAdvisor struct {
}

// Check checks for no select all.
func (*SelectNoSelectAllAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	checker := &selectNoSelectAllChecker{
		level: level,
		title: checkCtx.Rule.Type.String(),
	}

	for _, stmt := range checkCtx.ParsedStatements {
		node, ok := snowsqlparser.GetOmniNode(stmt.AST)
		if !ok {
			continue
		}
		checker.checkStmt(node, stmt.Text, stmt.BaseLine())
	}

	return checker.adviceList, nil
}

type selectNoSelectAllChecker struct {
	adviceList []*storepb.Advice
	level      storepb.Advice_Status
	title      string
}

// checkStmt flags every `*` / `qualifier.*` SELECT-list item anywhere in the
// statement (top-level query, subqueries, CTEs, INSERT ... SELECT, ...),
// mirroring the legacy listener that fired on every Select_list_elem holding a
// column_elem_star. Stars inside expressions (e.g. COUNT(*)) are not
// SELECT-list stars and are not flagged, matching the legacy behavior.
func (c *selectNoSelectAllChecker) checkStmt(node omniast.Node, text string, baseLine int) {
	// Collect the star-token offsets first and emit advices in text order so
	// the advice order matches the legacy depth-first listener walk.
	var starOffsets []int
	omniast.Inspect(node, func(n omniast.Node) bool {
		selectStmt, ok := n.(*omniast.SelectStmt)
		if !ok {
			return true
		}
		for _, target := range selectStmt.Targets {
			if target == nil || !target.Star {
				continue
			}
			// Anchor on the `*` token itself (the legacy rule used the STAR
			// token's line). For both `*` and `qualifier.*` the star is the
			// last byte of the StarExpr.
			offset := target.Loc.Start
			if star, ok := target.Expr.(*omniast.StarExpr); ok && star.Loc.IsValid() {
				offset = star.Loc.End - 1
			}
			starOffsets = append(starOffsets, offset)
		}
		return true
	})
	slices.Sort(starOffsets)

	for _, offset := range starOffsets {
		c.adviceList = append(c.adviceList, &storepb.Advice{
			Status:        c.level,
			Code:          code.StatementSelectAll.Int32(),
			Title:         c.title,
			Content:       "Avoid using SELECT *.",
			StartPosition: common.ConvertANTLRLineToPosition(baseLine + statementLineForOffset(text, offset)),
		})
	}
}

// statementLineForOffset converts a byte offset within a statement's text into
// the 1-based line number of that offset — the same line ANTLR assigned to a
// token at that position when the legacy listeners parsed the identical
// per-statement text. Add the statement's BaseLine() to obtain the line in the
// whole script.
func statementLineForOffset(text string, offset int) int {
	if offset < 0 {
		offset = 0
	}
	if offset > len(text) {
		offset = len(text)
	}
	return strings.Count(text[:offset], "\n") + 1
}
