package mysql

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/mysql/ast"
	mysqlv1review "github.com/bytebase/omni/mysql/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*ColumnRequireCollationAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_COLUMN_REQUIRE_COLLATION, &ColumnRequireCollationAdvisor{})
}

// ColumnRequireCollationAdvisor is the advisor checking for require collation.
// The judgment lives in omni (mysql/v1review.ColumnRequireCollation).
type ColumnRequireCollationAdvisor struct {
}

// Check checks for require collation.
func (*ColumnRequireCollationAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	stmts, nodes := parsedNodes(checkCtx.ParsedStatements)
	var adviceList []*storepb.Advice
	for i, stmt := range stmts {
		for _, f := range mysqlv1review.ColumnRequireCollation([]ast.Node{nodes[i]}) {
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          code.NoCollation.Int32(),
				Title:         checkCtx.Rule.Type.String(),
				Content:       fmt.Sprintf("Column %s does not have a collation specified", f.Name),
				StartPosition: columnFindingPosition(stmt, f),
			})
		}
	}
	return adviceList, nil
}
