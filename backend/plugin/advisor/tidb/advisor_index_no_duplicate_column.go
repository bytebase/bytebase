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
	_ advisor.Advisor = (*IndexNoDuplicateColumnAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_INDEX_NO_DUPLICATE_COLUMN, &IndexNoDuplicateColumnAdvisor{})
}

// IndexNoDuplicateColumnAdvisor flags index/constraint declarations
// where the same column appears twice in the key list. The judgment lives in
// omni (tidb/v1review.IndexNoDuplicateColumn); this adapter renders the
// findings as advice.
type IndexNoDuplicateColumnAdvisor struct{}

// Check checks for no duplicate columns in index.
func (*IndexNoDuplicateColumnAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmts, err := getTiDBNodes(checkCtx)
	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	title := checkCtx.Rule.Type.String()

	adviceList := []*storepb.Advice{}
	for _, stmt := range stmts {
		for _, f := range tidbv1review.IndexNoDuplicateColumn([]ast.Node{stmt.Node}) {
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          code.DuplicateColumnInIndex.Int32(),
				Title:         title,
				Content:       fmt.Sprintf("%s `%s` has duplicate column `%s`.`%s`", f.Value, f.Name, f.Table, f.Other),
				StartPosition: base.ConvertANTLRLineToPosition(findingLine(stmt, f)),
			})
		}
	}
	return adviceList, nil
}
