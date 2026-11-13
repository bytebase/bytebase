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
	_ advisor.Advisor = (*ColumnAutoIncrementMustIntegerAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_COLUMN_AUTO_INCREMENT_MUST_INTEGER, &ColumnAutoIncrementMustIntegerAdvisor{})
}

// ColumnAutoIncrementMustIntegerAdvisor is the advisor checking for auto-increment column type.
// The judgment lives in omni (tidb/v1review.ColumnAutoIncrementMustInteger).
type ColumnAutoIncrementMustIntegerAdvisor struct {
}

// Check checks for auto-increment column type.
func (*ColumnAutoIncrementMustIntegerAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
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
		for _, f := range tidbv1review.ColumnAutoIncrementMustInteger([]ast.Node{stmt.Node}) {
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          code.AutoIncrementColumnNotInteger.Int32(),
				Title:         title,
				Content:       fmt.Sprintf("Auto-increment column `%s`.`%s` requires integer type", f.Table, f.Name),
				StartPosition: base.ConvertANTLRLineToPosition(stmt.AbsoluteLine(f.Start)),
			})
		}
	}
	return adviceList, nil
}
