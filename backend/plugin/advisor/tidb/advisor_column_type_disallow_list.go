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
	_ advisor.Advisor = (*ColumnTypeDisallowListAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_COLUMN_TYPE_DISALLOW_LIST, &ColumnTypeDisallowListAdvisor{})
}

// ColumnTypeDisallowListAdvisor is the advisor checking for column type restriction.
// The judgment lives in omni (tidb/v1review.ColumnTypeDisallowList).
type ColumnTypeDisallowListAdvisor struct {
}

// Check checks for column type restriction. A nil string array payload
// panics, which advisor.Check reports as an error.
func (*ColumnTypeDisallowListAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmts, err := getTiDBNodes(checkCtx)
	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	title := checkCtx.Rule.Type.String()
	disallowed := checkCtx.Rule.GetStringArrayPayload().List

	var adviceList []*storepb.Advice
	for _, stmt := range stmts {
		for _, f := range tidbv1review.ColumnTypeDisallowList([]ast.Node{stmt.Node}, disallowed) {
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          code.DisabledColumnType.Int32(),
				Title:         title,
				Content:       fmt.Sprintf("Disallow column type %s but column `%s`.`%s` is", f.Value, f.Table, f.Name),
				StartPosition: base.ConvertANTLRLineToPosition(stmt.AbsoluteLine(f.Start)),
			})
		}
	}
	return adviceList, nil
}
