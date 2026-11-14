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
	_ advisor.Advisor = (*ColumnDefaultDisallowVolatileAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_COLUMN_DEFAULT_DISALLOW_VOLATILE, &ColumnDefaultDisallowVolatileAdvisor{})
}

// ColumnDefaultDisallowVolatileAdvisor is the advisor checking for column
// default volatile functions. The judgment lives in omni
// (pg/v1review.ColumnDefaultDisallowVolatile); this adapter renders the
// findings as advice.
type ColumnDefaultDisallowVolatileAdvisor struct {
}

// Check checks for column default volatile functions.
func (*ColumnDefaultDisallowVolatileAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	stmts, nodes := parsedNodes(checkCtx.ParsedStatements)
	var adviceList []*storepb.Advice
	for i, stmt := range stmts {
		for _, f := range pgv1review.ColumnDefaultDisallowVolatile([]ast.Node{nodes[i]}) {
			adviceList = append(adviceList, &storepb.Advice{
				Status:  level,
				Code:    code.NoDefault.Int32(),
				Title:   checkCtx.Rule.Type.String(),
				Content: fmt.Sprintf("Column %q.%q in schema %q has volatile DEFAULT", f.Table, f.Name, "public"),
				StartPosition: &storepb.Position{
					Line:   findingLine(stmt, f),
					Column: 0,
				},
			})
		}
	}
	return adviceList, nil
}
