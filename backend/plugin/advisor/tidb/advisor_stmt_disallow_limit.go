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
	_ advisor.Advisor = (*StatementDisallowLimitAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_STATEMENT_DISALLOW_LIMIT, &StatementDisallowLimitAdvisor{})
}

// StatementDisallowLimitAdvisor is the advisor checking for no LIMIT clause in INSERT/UPDATE statement.
// The judgment lives in omni (tidb/v1review.StatementDisallowLimit); this
// adapter renders its findings as advice.
type StatementDisallowLimitAdvisor struct {
}

// Check checks for no LIMIT clause in INSERT/UPDATE statement.
func (*StatementDisallowLimitAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
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
		"INSERT": advisorcode.InsertUseLimit,
		"UPDATE": advisorcode.UpdateUseLimit,
		"DELETE": advisorcode.DeleteUseLimit,
	}
	for _, ostmt := range stmts {
		for _, f := range tidbv1review.StatementDisallowLimit([]ast.Node{ostmt.Node}) {
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          codes[f.Name].Int32(),
				Title:         title,
				Content:       fmt.Sprintf("LIMIT clause is forbidden in INSERT, UPDATE and DELETE statement, but \"%s\" uses", ostmt.TrimmedText()),
				StartPosition: base.ConvertANTLRLineToPosition(ostmt.FirstTokenLine()),
			})
		}
	}
	return adviceList, nil
}
