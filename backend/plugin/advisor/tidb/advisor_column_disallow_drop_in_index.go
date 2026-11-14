package tidb

import (
	"context"
	"fmt"
	"slices"

	"github.com/bytebase/omni/tidb/ast"
	tidbv1review "github.com/bytebase/omni/tidb/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

var (
	_ advisor.Advisor = (*ColumnDisallowDropInIndexAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_COLUMN_DISALLOW_DROP_IN_INDEX, &ColumnDisallowDropInIndexAdvisor{})
}

// ColumnDisallowDropInIndexAdvisor is the advisor checking for disallow DROP COLUMN in index.
// The judgment lives in omni (tidb/v1review.ColumnDisallowDropInIndex).
type ColumnDisallowDropInIndexAdvisor struct{}

// Check checks for disallow DROP COLUMN in index.
func (*ColumnDisallowDropInIndexAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmts, err := getTiDBNodes(checkCtx)
	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	title := checkCtx.Rule.Type.String()
	if checkCtx.OriginalMetadata == nil && slices.ContainsFunc(stmts, dropsColumn) {
		// V1 read the snapshot at the first DROP COLUMN and panicked on a
		// nil one; advisor.Check reports that panic as the rule's error.
		checkCtx.OriginalMetadata.GetSchemaMetadata("")
	}

	var adviceList []*storepb.Advice
	for _, f := range tidbv1review.ColumnDisallowDropInIndex(stmtNodes(stmts), v1reviewSchema(checkCtx.OriginalMetadata)) {
		adviceList = append(adviceList, &storepb.Advice{
			Status:        level,
			Code:          code.DropIndexColumn.Int32(),
			Title:         title,
			Content:       fmt.Sprintf("`%s`.`%s` cannot drop index column", f.Table, f.Name),
			StartPosition: base.ConvertANTLRLineToPosition(stmts[f.Statement].FirstTokenLine()),
		})
	}
	return adviceList, nil
}

// dropsColumn reports whether the statement is an ALTER TABLE naming a table
// with a DROP COLUMN action.
func dropsColumn(stmt Stmt) bool {
	alter, ok := stmt.Node.(*ast.AlterTableStmt)
	if !ok || alter.Table == nil {
		return false
	}
	return slices.ContainsFunc(alter.Commands, func(cmd *ast.AlterTableCmd) bool {
		return cmd != nil && cmd.Type == ast.ATDropColumn
	})
}
