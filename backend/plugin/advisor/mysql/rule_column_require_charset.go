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
	_ advisor.Advisor = (*ColumnRequireCharsetAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_COLUMN_REQUIRE_CHARSET, &ColumnRequireCharsetAdvisor{})
}

// ColumnRequireCharsetAdvisor is the advisor checking for require charset.
// The judgment lives in omni (mysql/v1review.ColumnRequireCharset).
type ColumnRequireCharsetAdvisor struct {
}

// Check checks for require charset.
func (*ColumnRequireCharsetAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	stmts, nodes := parsedNodes(checkCtx.ParsedStatements)
	var adviceList []*storepb.Advice
	for i, stmt := range stmts {
		for _, f := range mysqlv1review.ColumnRequireCharset([]ast.Node{nodes[i]}) {
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          code.NoCharset.Int32(),
				Title:         checkCtx.Rule.Type.String(),
				Content:       fmt.Sprintf("Column %s does not have a character set specified", f.Name),
				StartPosition: columnFindingPosition(stmt, f),
			})
		}
	}
	return adviceList, nil
}
