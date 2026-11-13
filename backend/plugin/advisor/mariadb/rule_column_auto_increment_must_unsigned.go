package mariadb

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/mariadb/ast"
	mariadbv1review "github.com/bytebase/omni/mariadb/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*ColumnAutoIncrementMustUnsignedAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_COLUMN_AUTO_INCREMENT_MUST_UNSIGNED, &ColumnAutoIncrementMustUnsignedAdvisor{})
}

// ColumnAutoIncrementMustUnsignedAdvisor is the advisor checking for unsigned auto-increment column.
// The judgment lives in omni (mariadb/v1review.ColumnAutoIncrementMustUnsigned).
type ColumnAutoIncrementMustUnsignedAdvisor struct {
}

// Check checks for unsigned auto-increment column.
func (*ColumnAutoIncrementMustUnsignedAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	stmts, nodes := parsedNodes(checkCtx.ParsedStatements)
	var adviceList []*storepb.Advice
	for i, stmt := range stmts {
		for _, f := range mariadbv1review.ColumnAutoIncrementMustUnsigned([]ast.Node{nodes[i]}) {
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          code.AutoIncrementColumnSigned.Int32(),
				Title:         checkCtx.Rule.Type.String(),
				Content:       fmt.Sprintf("Auto-increment column `%s`.`%s` is not UNSIGNED type", f.Table, f.Name),
				StartPosition: findingPosition(stmt, f),
			})
		}
	}
	return adviceList, nil
}
