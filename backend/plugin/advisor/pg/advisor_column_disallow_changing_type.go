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
	_ advisor.Advisor = (*ColumnDisallowChangingTypeAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_COLUMN_DISALLOW_CHANGE_TYPE, &ColumnDisallowChangingTypeAdvisor{})
}

// ColumnDisallowChangingTypeAdvisor is the advisor checking for disallow
// changing column type. The judgment lives in omni
// (pg/v1review.ColumnDisallowChangeType); this adapter renders the findings as
// advice that prints the statement.
type ColumnDisallowChangingTypeAdvisor struct {
}

// Check checks for disallow changing column type.
func (*ColumnDisallowChangingTypeAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	stmts, nodes := parsedNodes(checkCtx.ParsedStatements)
	var adviceList []*storepb.Advice
	for i, stmt := range stmts {
		for _, f := range pgv1review.ColumnDisallowChangeType([]ast.Node{nodes[i]}) {
			stmtText := strings.TrimSpace(strings.TrimRight(strings.TrimSpace(stmt.Text), ";"))
			adviceList = append(adviceList, &storepb.Advice{
				Status:  level,
				Code:    code.ChangeColumnType.Int32(),
				Title:   checkCtx.Rule.Type.String(),
				Content: fmt.Sprintf("The statement \"%s\" changes column type", stmtText),
				StartPosition: &storepb.Position{
					Line:   findingLine(stmt, f),
					Column: 0,
				},
			})
		}
	}
	return adviceList, nil
}
