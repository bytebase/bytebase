package tidb

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/tidb/ast"
	tidbv1review "github.com/bytebase/omni/tidb/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

var (
	_ advisor.Advisor = (*ColumnDisallowSetCharsetAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_COLUMN_DISALLOW_SET_CHARSET, &ColumnDisallowSetCharsetAdvisor{})
}

// ColumnDisallowSetCharsetAdvisor is the advisor checking for disallowed column-level CHARACTER SET clauses.
// The judgment lives in omni (tidb/v1review.ColumnDisallowSetCharset).
type ColumnDisallowSetCharsetAdvisor struct {
}

// Check checks for disallowed column-level CHARACTER SET clauses.
func (*ColumnDisallowSetCharsetAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmts, err := getTiDBNodes(checkCtx)
	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	title := checkCtx.Rule.Type.String()

	var adviceList []*storepb.Advice
	for _, stmt := range stmts {
		for range tidbv1review.ColumnDisallowSetCharset([]ast.Node{stmt.Node}) {
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          code.SetColumnCharset.Int32(),
				Title:         title,
				Content:       fmt.Sprintf("Disallow set column charset but \"%s\" does", stmt.TrimmedText()),
				StartPosition: base.ConvertANTLRLineToPosition(stmt.FirstTokenLine()),
			})
		}
	}
	return adviceList, nil
}
