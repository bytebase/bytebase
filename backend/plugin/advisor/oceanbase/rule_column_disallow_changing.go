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
	_ advisor.Advisor = (*ColumnDisallowChangingAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_COLUMN_DISALLOW_CHANGE, &ColumnDisallowChangingAdvisor{})
}

// ColumnDisallowChangingAdvisor is the advisor checking for disallow CHANGE COLUMN statement.
// The judgment lives in omni (oceanbase/v1review.ColumnDisallowChange); this adapter prints each
// finding's statement and places the advice at its first line.
type ColumnDisallowChangingAdvisor struct {
}

// Check checks for disallow CHANGE COLUMN statement.
func (*ColumnDisallowChangingAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	stmts, nodes := parsedNodes(checkCtx.ParsedStatements)
	var adviceList []*storepb.Advice
	for i, stmt := range stmts {
		for range oceanbasev1review.ColumnDisallowChange([]ast.Node{nodes[i]}) {
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          code.UseChangeColumnStatement.Int32(),
				Title:         checkCtx.Rule.Type.String(),
				Content:       fmt.Sprintf("\"%s\" contains CHANGE COLUMN statement", strings.TrimSpace(stmt.Text)),
				StartPosition: statementPosition(stmt),
			})
		}
	}
	return adviceList, nil
}
