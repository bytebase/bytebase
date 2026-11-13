package oceanbase

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/oceanbase/ast"
	oceanbasev1review "github.com/bytebase/omni/oceanbase/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*ColumnDisallowDropAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_COLUMN_DISALLOW_DROP, &ColumnDisallowDropAdvisor{})
}

// ColumnDisallowDropAdvisor is the advisor checking for disallow DROP COLUMN statement.
// The judgment lives in omni (oceanbase/v1review.ColumnDisallowDrop).
type ColumnDisallowDropAdvisor struct {
}

// Check checks for disallow DROP COLUMN statement.
func (*ColumnDisallowDropAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	stmts, nodes := parsedNodes(checkCtx.ParsedStatements)
	var adviceList []*storepb.Advice
	for i, stmt := range stmts {
		for _, f := range oceanbasev1review.ColumnDisallowDrop([]ast.Node{nodes[i]}) {
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          code.DropColumn.Int32(),
				Title:         checkCtx.Rule.Type.String(),
				Content:       fmt.Sprintf("drops column \"%s\" of table \"%s\"", f.Name, f.Table),
				StartPosition: findingPosition(stmt, f),
			})
		}
	}
	return adviceList, nil
}
