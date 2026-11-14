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
	_ advisor.Advisor = (*IndexNoDuplicateColumnAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_INDEX_NO_DUPLICATE_COLUMN, &IndexNoDuplicateColumnAdvisor{})
}

// IndexNoDuplicateColumnAdvisor is the advisor checking for no duplicate
// columns in index. The judgment lives in omni
// (pg/v1review.IndexNoDuplicateColumn); this adapter renders the findings as
// advice.
type IndexNoDuplicateColumnAdvisor struct {
}

// Check checks for no duplicate columns in index.
func (*IndexNoDuplicateColumnAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	stmts, nodes := scriptNodes(checkCtx.ParsedStatements)
	title := checkCtx.Rule.Type.String()
	var adviceList []*storepb.Advice
	for i, node := range nodes {
		for _, f := range pgv1review.IndexNoDuplicateColumn([]ast.Node{node}) {
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          code.DuplicateColumnInIndex.Int32(),
				Title:         title,
				Content:       fmt.Sprintf("%s %q has duplicate column %q.%q", f.Value, f.Name, f.Table, f.Other),
				StartPosition: findingPosition(stmts[i], f),
			})
		}
	}
	return adviceList, nil
}
