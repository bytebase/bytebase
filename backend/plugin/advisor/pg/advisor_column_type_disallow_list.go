package pg

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/omni/pg/ast"
	pgv1review "github.com/bytebase/omni/pg/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*ColumnTypeDisallowListAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_COLUMN_TYPE_DISALLOW_LIST, &ColumnTypeDisallowListAdvisor{})
}

// ColumnTypeDisallowListAdvisor is the advisor checking for column type
// restriction. The judgment lives in omni (pg/v1review.ColumnTypeDisallowList);
// this adapter passes the disallowed types and renders the findings as advice.
type ColumnTypeDisallowListAdvisor struct {
}

// Check checks for column type restriction.
func (*ColumnTypeDisallowListAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	// V1 read the list without a nil check, so a missing payload panics.
	disallowed := checkCtx.Rule.GetStringArrayPayload().List

	stmts, nodes := parsedNodes(checkCtx.ParsedStatements)
	var adviceList []*storepb.Advice
	for i, stmt := range stmts {
		for _, f := range pgv1review.ColumnTypeDisallowList([]ast.Node{nodes[i]}, disallowed) {
			adviceList = append(adviceList, &storepb.Advice{
				Status:  level,
				Code:    code.DisabledColumnType.Int32(),
				Title:   checkCtx.Rule.Type.String(),
				Content: fmt.Sprintf("Disallow column type %s but column %q.%q is", strings.ToUpper(f.Value), f.Table, f.Name),
				StartPosition: &storepb.Position{
					Line:   findingLine(stmt, f),
					Column: 0,
				},
			})
		}
	}
	return adviceList, nil
}
