package tidb

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/tidb/ast"
	tidbv1review "github.com/bytebase/omni/tidb/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	advisorcode "github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

var (
	_ advisor.Advisor = (*InsertDisallowOrderByRandAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_STATEMENT_INSERT_DISALLOW_ORDER_BY_RAND, &InsertDisallowOrderByRandAdvisor{})
}

// InsertDisallowOrderByRandAdvisor is the advisor checking for to disallow order by rand in INSERT statements.
// The judgment lives in omni (tidb/v1review.StatementInsertDisallowOrderByRand); this
// adapter renders its findings as advice.
type InsertDisallowOrderByRandAdvisor struct {
}

// Check checks for to disallow order by rand in INSERT statements.
func (*InsertDisallowOrderByRandAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
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
	for _, ostmt := range stmts {
		for range tidbv1review.StatementInsertDisallowOrderByRand([]ast.Node{ostmt.Node}) {
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          advisorcode.InsertUseOrderByRand.Int32(),
				Title:         title,
				Content:       fmt.Sprintf("\"%s\" uses ORDER BY RAND in the INSERT statement", ostmt.TrimmedText()),
				StartPosition: base.ConvertANTLRLineToPosition(ostmt.FirstTokenLine()),
			})
		}
	}
	return adviceList, nil
}
