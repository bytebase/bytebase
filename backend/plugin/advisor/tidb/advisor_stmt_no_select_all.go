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
	_ advisor.Advisor = (*NoSelectAllAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_STATEMENT_SELECT_NO_SELECT_ALL, &NoSelectAllAdvisor{})
}

// NoSelectAllAdvisor is the advisor checking for no "select *".
// The judgment lives in omni (tidb/v1review.StatementSelectNoSelectAll); this
// adapter renders its findings as advice.
type NoSelectAllAdvisor struct {
}

// Check checks for no "select *".
func (*NoSelectAllAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
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
		for range tidbv1review.StatementSelectNoSelectAll([]ast.Node{ostmt.Node}) {
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          code.StatementSelectAll.Int32(),
				Title:         title,
				Content:       fmt.Sprintf("\"%s\" uses SELECT all", ostmt.TrimmedText()),
				StartPosition: base.ConvertANTLRLineToPosition(ostmt.FirstTokenLine()),
			})
		}
	}
	return adviceList, nil
}
