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
	_ advisor.Advisor = (*NoLeadingWildcardLikeAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_STATEMENT_WHERE_NO_LEADING_WILDCARD_LIKE, &NoLeadingWildcardLikeAdvisor{})
}

// NoLeadingWildcardLikeAdvisor checks for LIKE patterns that begin with
// `%`, e.g. `WHERE x LIKE '%foo'`. Such patterns force a full table scan;
// the rule warns even on NOT LIKE because the same scan cost applies.
// The judgment lives in omni (tidb/v1review.StatementWhereNoLeadingWildcardLike); this
// adapter renders its findings as advice.
type NoLeadingWildcardLikeAdvisor struct {
}

// Check checks for LIKE patterns that begin with `%`.
func (*NoLeadingWildcardLikeAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
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
		for range tidbv1review.StatementWhereNoLeadingWildcardLike([]ast.Node{ostmt.Node}) {
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          code.StatementLeadingWildcardLike.Int32(),
				Title:         title,
				Content:       fmt.Sprintf("\"%s\" uses leading wildcard LIKE", ostmt.TrimmedText()),
				StartPosition: base.ConvertANTLRLineToPosition(ostmt.FirstTokenLine()),
			})
		}
	}
	return adviceList, nil
}
