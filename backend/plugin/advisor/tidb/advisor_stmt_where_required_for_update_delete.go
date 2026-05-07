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
	_ advisor.Advisor = (*WhereRequirementForUpdateDeleteAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_STATEMENT_WHERE_REQUIRE_UPDATE_DELETE, &WhereRequirementForUpdateDeleteAdvisor{})
}

// WhereRequirementForUpdateDeleteAdvisor checks the WHERE clause
// requirement for UPDATE and DELETE statements.
type WhereRequirementForUpdateDeleteAdvisor struct {
}

// Check is Recipe A (top-level type-switch) — UPDATE and DELETE cannot
// nest other UPDATE/DELETE statements in standard SQL, so a sub-walk would
// add no coverage over a top-level check. Subqueries inside UPDATE/DELETE
// (in WHERE expressions or SET values) are SELECTs, which are the
// where_required_for_select advisor's concern, not this rule's.
func (*WhereRequirementForUpdateDeleteAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmts, err := getTiDBOmniNodes(checkCtx)
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
		var noWhere bool
		switch n := ostmt.Node.(type) {
		case *ast.DeleteStmt:
			if n.Where == nil {
				noWhere = true
			}
		case *ast.UpdateStmt:
			if n.Where == nil {
				noWhere = true
			}
		default:
		}
		if !noWhere {
			continue
		}
		adviceList = append(adviceList, &storepb.Advice{
			Status:        level,
			Code:          advisorcode.StatementNoWhere.Int32(),
			Title:         title,
			Content:       fmt.Sprintf("\"%s\" requires WHERE clause", ostmt.TrimmedText()),
			StartPosition: common.ConvertANTLRLineToPosition(ostmt.FirstTokenLine()),
		})
	}
	return adviceList, nil
}
