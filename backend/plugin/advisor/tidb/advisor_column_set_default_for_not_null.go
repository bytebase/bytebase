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
	_ advisor.Advisor = (*ColumnSetDefaultForNotNullAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_COLUMN_SET_DEFAULT_FOR_NOT_NULL, &ColumnSetDefaultForNotNullAdvisor{})
}

// ColumnSetDefaultForNotNullAdvisor checks for set default value for not null columns.
// The judgment lives in omni (tidb/v1review.ColumnSetDefaultForNotNull).
type ColumnSetDefaultForNotNullAdvisor struct {
}

// Check checks for set default value for not null columns. An ALTER TABLE
// finding is placed on the statement's first line, comments included.
func (*ColumnSetDefaultForNotNullAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
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
		_, isAlter := stmt.Node.(*ast.AlterTableStmt)
		for _, f := range tidbv1review.ColumnSetDefaultForNotNull([]ast.Node{stmt.Node}) {
			line := stmt.AbsoluteLine(f.Start)
			if isAlter {
				line = stmt.FirstTokenLine()
			}
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          code.NotNullColumnWithNoDefault.Int32(),
				Title:         title,
				Content:       fmt.Sprintf("Column `%s`.`%s` is NOT NULL but doesn't have DEFAULT", f.Table, f.Name),
				StartPosition: base.ConvertANTLRLineToPosition(line),
			})
		}
	}
	return adviceList, nil
}
