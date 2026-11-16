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
	_ advisor.Advisor = (*DisallowOrderByAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_STATEMENT_DISALLOW_ORDER_BY, &DisallowOrderByAdvisor{})
}

// DisallowOrderByAdvisor is the advisor checking for no ORDER BY clause in DELETE/UPDATE statements.
// The judgment lives in omni (tidb/v1review.StatementDisallowOrderBy); this
// adapter renders its findings as advice.
type DisallowOrderByAdvisor struct {
}

// Check checks for no ORDER BY clause in DELETE/UPDATE statements.
func (*DisallowOrderByAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
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
	codes := map[string]advisorcode.Code{
		"UPDATE": advisorcode.UpdateUseOrderBy,
		"DELETE": advisorcode.DeleteUseOrderBy,
	}
	for _, ostmt := range stmts {
		for _, f := range tidbv1review.StatementDisallowOrderBy([]ast.Node{ostmt.Node}) {
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          codes[f.Name].Int32(),
				Title:         title,
				Content:       fmt.Sprintf("ORDER BY clause is forbidden in DELETE and UPDATE statements, but \"%s\" uses", ostmt.TrimmedText()),
				StartPosition: base.ConvertANTLRLineToPosition(ostmt.FirstTokenLine()),
			})
		}
	}
	return adviceList, nil
}
