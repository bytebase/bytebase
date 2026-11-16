package oceanbase

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/omni/oceanbase/ast"
	oceanbasev1review "github.com/bytebase/omni/oceanbase/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*ColumnDisallowSetCharsetAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_COLUMN_DISALLOW_SET_CHARSET, &ColumnDisallowSetCharsetAdvisor{})
}

// ColumnDisallowSetCharsetAdvisor is the advisor checking for disallow set column charset.
// The judgment lives in omni (oceanbase/v1review.ColumnDisallowSetCharset); this adapter prints each
// finding's statement and places the advice at its first line.
type ColumnDisallowSetCharsetAdvisor struct {
}

// Check checks for disallow set column charset.
func (*ColumnDisallowSetCharsetAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	stmts, nodes := parsedNodes(checkCtx.ParsedStatements)
	var adviceList []*storepb.Advice
	for i, stmt := range stmts {
		for range oceanbasev1review.ColumnDisallowSetCharset([]ast.Node{nodes[i]}) {
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          code.SetColumnCharset.Int32(),
				Title:         checkCtx.Rule.Type.String(),
				Content:       fmt.Sprintf("Disallow set column charset but \"%s\" does", strings.TrimSpace(stmt.Text)),
				StartPosition: statementPosition(stmt),
			})
		}
	}
	return adviceList, nil
}
