package tidb

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/tidb/ast"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	advisorcode "github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*DisallowOrderByAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_STATEMENT_DISALLOW_ORDER_BY, &DisallowOrderByAdvisor{})
}

// DisallowOrderByAdvisor is the advisor checking for no ORDER BY clause in DELETE/UPDATE statements.
type DisallowOrderByAdvisor struct {
}

// Check checks for no ORDER BY clause in DELETE/UPDATE statements.
func (*DisallowOrderByAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmts, err := getTiDBOmniNodes(checkCtx)
	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	var adviceList []*storepb.Advice
	for _, ostmt := range stmts {
		code := advisorcode.Ok
		switch n := ostmt.Node.(type) {
		case *ast.UpdateStmt:
			if len(n.OrderBy) > 0 {
				code = advisorcode.UpdateUseOrderBy
			}
		case *ast.DeleteStmt:
			if len(n.OrderBy) > 0 {
				code = advisorcode.DeleteUseOrderBy
			}
		default:
		}

		if code != advisorcode.Ok {
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          code.Int32(),
				Title:         checkCtx.Rule.Type.String(),
				Content:       fmt.Sprintf("ORDER BY clause is forbidden in DELETE and UPDATE statements, but \"%s\" uses", ostmt.TrimmedText()),
				StartPosition: common.ConvertANTLRLineToPosition(ostmt.FirstTokenLine()),
			})
		}
	}

	return adviceList, nil
}
