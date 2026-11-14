package tidb

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/tidb/ast"
	tidbv1review "github.com/bytebase/omni/tidb/v1review"
	"github.com/bytebase/omni/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

var (
	_ advisor.Advisor = (*TableNoFKAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_TABLE_NO_FOREIGN_KEY, &TableNoFKAdvisor{})
}

// TableNoFKAdvisor is the advisor checking table disallow foreign key. The
// judgment lives in omni (tidb/v1review.TableNoForeignKey); this adapter
// renders the findings as advice.
type TableNoFKAdvisor struct {
}

// Check checks table disallow foreign key.
func (*TableNoFKAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmts, err := getTiDBNodes(checkCtx)
	if err != nil {
		return nil, err
	}
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	title := checkCtx.Rule.Type.String()
	var adviceList []*storepb.Advice
	for _, stmt := range stmts {
		for _, f := range tidbv1review.TableNoForeignKey([]ast.Node{stmt.Node}) {
			if f.Code != v1review.TableForeignKey {
				continue
			}
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          code.TableHasFK.Int32(),
				Title:         title,
				Content:       fmt.Sprintf("Foreign key is not allowed in the table `%s`", f.Table),
				StartPosition: base.ConvertANTLRLineToPosition(stmt.AbsoluteLine(f.Start)),
			})
		}
	}
	return adviceList, nil
}
