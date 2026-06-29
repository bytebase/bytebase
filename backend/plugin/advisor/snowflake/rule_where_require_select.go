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
	_ advisor.Advisor = (*WhereRequireForSelectAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_SNOWFLAKE, storepb.SQLReviewRule_STATEMENT_WHERE_REQUIRE_SELECT, &WhereRequireForSelectAdvisor{})
}

// WhereRequireForSelectAdvisor is the advisor checking for WHERE clause requirement for SELECT statement.
type WhereRequireForSelectAdvisor struct {
}

// Check checks for WHERE clause requirement.
func (*WhereRequireForSelectAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	checker := &whereRequireForSelectChecker{
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

type whereRequireForSelectChecker struct {
	adviceList []*storepb.Advice
	level      storepb.Advice_Status
	title      string
}

// checkStmt flags every SELECT with a FROM clause but no WHERE clause anywhere
// in the statement — including CTE bodies, derived tables, and subqueries —
// mirroring the legacy listener that fired on every Select_statement context.
// SELECTs without a FROM clause (e.g. SELECT 1) are allowed.
func (c *whereRequireForSelectChecker) checkStmt(node omniast.Node, text string, baseLine int) {
	// Collect anchor offsets first and emit advices in text order so the
	// advice order matches the legacy depth-first listener walk (a CTE body
	// is reported before the main SELECT that follows it).
	var offsets []int
	omniast.Inspect(node, func(n omniast.Node) bool {
		selectStmt, ok := n.(*omniast.SelectStmt)
		if !ok {
			return true
		}
		// Allow SELECT queries without a FROM clause to proceed, e.g. SELECT 1.
		if selectStmt.Where == nil && selectStmt.From != nil {
			offsets = append(offsets, selectBodyStart(selectStmt, text))
		}
		return true
	})
	slices.Sort(offsets)

	for _, offset := range offsets {
		c.adviceList = append(c.adviceList, &storepb.Advice{
			Status:        c.level,
			Code:          code.StatementNoWhere.Int32(),
			Title:         c.title,
			Content:       "WHERE clause is required for SELECT statement.",
			StartPosition: common.ConvertANTLRLineToPosition(baseLine + statementLineForOffset(text, offset)),
		})
	}
}

// selectBodyStart returns the byte offset of the SELECT keyword that opens the
// body of selectStmt within text. For a plain SELECT that is Loc.Start; for a
// WITH ... SELECT the omni parser extends Loc.Start back over the CTE list,
// while the legacy ANTLR Select_statement context started at the body SELECT
// itself — so skip past the last CTE's closing paren and any trivia to land on
// the SELECT keyword, reproducing the legacy line exactly.
func selectBodyStart(selectStmt *omniast.SelectStmt, text string) int {
	start := selectStmt.Loc.Start
	if n := len(selectStmt.With); n > 0 {
		last := selectStmt.With[n-1]
		if last != nil && last.Loc.End >= 0 && last.Loc.End <= len(text) {
			if offset, ok := skipSpaceAndComments(text, last.Loc.End); ok {
				start = offset
			}
		}
	}
	return start
}

// skipSpaceAndComments advances offset past whitespace and SQL comments
// (`--` and `//` line comments, `/* */` block comments) and returns the offset
// of the next token, or false when only trivia remains.
func skipSpaceAndComments(text string, offset int) (int, bool) {
	for offset < len(text) {
		switch c := text[offset]; {
		case c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\v' || c == '\f':
			offset++
		case strings.HasPrefix(text[offset:], "--") || strings.HasPrefix(text[offset:], "//"):
			next := strings.IndexByte(text[offset:], '\n')
			if next < 0 {
				return 0, false
			}
			offset += next + 1
		case strings.HasPrefix(text[offset:], "/*"):
			end := strings.Index(text[offset+2:], "*/")
			if end < 0 {
				return 0, false
			}
			offset += 2 + end + 2
		default:
			return offset, true
		}
	}
	return 0, false
}
