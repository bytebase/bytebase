package oceanbase

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/oceanbase/ast"
	oceanbasev1review "github.com/bytebase/omni/oceanbase/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*ColumnAutoIncrementMustIntegerAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_COLUMN_AUTO_INCREMENT_MUST_INTEGER, &ColumnAutoIncrementMustIntegerAdvisor{})
}

// ColumnAutoIncrementMustIntegerAdvisor is the advisor checking for auto-increment column type.
// The judgment lives in omni (oceanbase/v1review.ColumnAutoIncrementMustInteger).
type ColumnAutoIncrementMustIntegerAdvisor struct {
}

// Check checks for auto-increment column type.
func (*ColumnAutoIncrementMustIntegerAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	stmts, nodes := parsedNodes(checkCtx.ParsedStatements)
	var adviceList []*storepb.Advice
	for i, stmt := range stmts {
		for _, f := range oceanbasev1review.ColumnAutoIncrementMustInteger([]ast.Node{nodes[i]}) {
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          code.AutoIncrementColumnNotInteger.Int32(),
				Title:         checkCtx.Rule.Type.String(),
				Content:       fmt.Sprintf("Auto-increment column `%s`.`%s` requires integer type", f.Table, f.Name),
				StartPosition: columnFindingPosition(stmt, f),
			})
		}
	}
	return adviceList, nil
}
