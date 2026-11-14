package pg

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/pg/ast"
	pgv1review "github.com/bytebase/omni/pg/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*ColumnRequireDefaultAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_COLUMN_REQUIRE_DEFAULT, &ColumnRequireDefaultAdvisor{})
}

// ColumnRequireDefaultAdvisor is the advisor checking for column default
// requirement. The judgment lives in omni (pg/v1review.ColumnRequireDefault);
// this adapter renders the findings as advice.
type ColumnRequireDefaultAdvisor struct {
}

// Check checks for column default requirement.
func (*ColumnRequireDefaultAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	stmts, nodes := parsedNodes(checkCtx.ParsedStatements)
	var adviceList []*storepb.Advice
	for i, stmt := range stmts {
		for _, f := range pgv1review.ColumnRequireDefault([]ast.Node{nodes[i]}) {
			adviceList = append(adviceList, &storepb.Advice{
				Status:  level,
				Code:    code.NoDefault.Int32(),
				Title:   checkCtx.Rule.Type.String(),
				Content: fmt.Sprintf("Column %q.%q in schema %q doesn't have DEFAULT", f.Table, f.Name, "public"),
				StartPosition: &storepb.Position{
					Line:   findingLine(stmt, f),
					Column: 0,
				},
			})
		}
	}
	return adviceList, nil
}
