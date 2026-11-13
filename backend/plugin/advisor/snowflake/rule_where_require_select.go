// Package snowflake is the advisor for snowflake database.
package snowflake

import (
	"context"
	"strings"

	snowflakeast "github.com/bytebase/omni/snowflake/ast"
	snowflakev1review "github.com/bytebase/omni/snowflake/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	snowsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/snowflake"
)

var (
	_ advisor.Advisor = (*WhereRequireForSelectAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_SNOWFLAKE, storepb.SQLReviewRule_STATEMENT_WHERE_REQUIRE_SELECT, &WhereRequireForSelectAdvisor{})
}

// WhereRequireForSelectAdvisor is the advisor checking for WHERE clause requirement for SELECT statement.
// The judgment lives in omni (snowflake/v1review.StatementWhereRequireSelect);
// this adapter renders its findings as advice.
type WhereRequireForSelectAdvisor struct {
}

// Check checks for WHERE clause requirement.
func (*WhereRequireForSelectAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	title := checkCtx.Rule.Type.String()
	var adviceList []*storepb.Advice
	for _, stmt := range checkCtx.ParsedStatements {
		node, ok := snowsqlparser.GetNode(stmt.AST)
		if !ok {
			continue
		}
		for _, f := range snowflakev1review.StatementWhereRequireSelect([]snowflakeast.Node{node}) {
			// A WITH ... SELECT finding anchors the end of its last CTE; the
			// line is that of the next token, the body's SELECT keyword.
			offset := f.Start
			if offset > 0 {
				if next, ok := skipSpaceAndComments(stmt.Text, offset); ok {
					offset = next
				}
			}
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          code.StatementNoWhere.Int32(),
				Title:         title,
				Content:       "WHERE clause is required for SELECT statement.",
				StartPosition: base.ConvertANTLRLineToPosition(stmt.BaseLine() + statementLineForOffset(stmt.Text, offset)),
			})
		}
	}
	return adviceList, nil
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
