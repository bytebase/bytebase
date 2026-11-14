// Package oracle is the advisor for oracle database.
package oracle

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/oracle/ast"
	oraclev1review "github.com/bytebase/omni/oracle/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	plsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/plsql"
)

var (
	_ advisor.Advisor = (*ColumnTypeDisallowListAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_ORACLE, storepb.SQLReviewRule_COLUMN_TYPE_DISALLOW_LIST, &ColumnTypeDisallowListAdvisor{})
}

// ColumnTypeDisallowListAdvisor is the advisor checking for column type disallow list.
// The judgment lives in omni (oracle/v1review.ColumnTypeDisallowList); this
// adapter passes the list and renders the findings as advice.
type ColumnTypeDisallowListAdvisor struct {
}

// Check checks for column type disallow list.
func (*ColumnTypeDisallowListAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	// A missing payload panics here, which advisor.Check reports as an error.
	disallowList := checkCtx.Rule.GetStringArrayPayload().List

	title := checkCtx.Rule.Type.String()
	var adviceList []*storepb.Advice
	for _, stmt := range checkCtx.ParsedStatements {
		if stmt.AST == nil {
			continue
		}
		node, ok := plsqlparser.GetNode(stmt.AST)
		if !ok {
			continue
		}
		for _, f := range oraclev1review.ColumnTypeDisallowList([]ast.Node{node}, disallowList) {
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          code.DisabledColumnType.Int32(),
				Title:         title,
				Content:       fmt.Sprintf("Disallow column type %s but column \"%s\" is", f.Value, f.Name),
				StartPosition: base.ConvertANTLRLineToPosition(stmt.BaseLine() + findingLine(stmt.Text, f)),
			})
		}
	}
	return adviceList, nil
}
