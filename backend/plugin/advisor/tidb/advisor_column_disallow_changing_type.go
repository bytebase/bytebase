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
	_ advisor.Advisor = (*ColumnDisallowChangingTypeAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_COLUMN_DISALLOW_CHANGE_TYPE, &ColumnDisallowChangingTypeAdvisor{})
}

// ColumnDisallowChangingTypeAdvisor is the advisor checking for disallow changing column type.
// The judgment lives in omni (tidb/v1review.ColumnDisallowChangeType).
type ColumnDisallowChangingTypeAdvisor struct {
}

// Check checks for disallow changing column type.
func (*ColumnDisallowChangingTypeAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmts, err := getTiDBNodes(checkCtx)
	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	title := checkCtx.Rule.Type.String()
	if checkCtx.OriginalMetadata == nil && slices.ContainsFunc(stmts, readsColumnType) {
		// V1 read the snapshot at the first such action and panicked on a
		// nil one; advisor.Check reports that panic as the rule's error.
		checkCtx.OriginalMetadata.GetSchemaMetadata("")
	}
	schema := v1reviewSchema(checkCtx.OriginalMetadata)

	var adviceList []*storepb.Advice
	for _, stmt := range stmts {
		for _, f := range tidbv1review.ColumnDisallowChangeType([]ast.Node{stmt.Node}, schema) {
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          code.ChangeColumnType.Int32(),
				Title:         title,
				Content:       fmt.Sprintf("\"%s\" changes column type", stmt.TrimmedText()),
				StartPosition: base.ConvertANTLRLineToPosition(stmt.AbsoluteLine(f.Start)),
			})
		}
	}
	return adviceList, nil
}

// readsColumnType reports whether the statement is an ALTER TABLE naming a
// table with a CHANGE or MODIFY COLUMN action whose definition has a type,
// for which the rule looks the column up in the snapshot.
func readsColumnType(stmt Stmt) bool {
	alter, ok := stmt.Node.(*ast.AlterTableStmt)
	if !ok || alter.Table == nil {
		return false
	}
	return slices.ContainsFunc(alter.Commands, func(cmd *ast.AlterTableCmd) bool {
		if cmd == nil || cmd.Column == nil || cmd.Column.TypeName == nil {
			return false
		}
		switch cmd.Type {
		case ast.ATChangeColumn:
			return cmd.Name != ""
		case ast.ATModifyColumn:
			return cmd.Column.Name != ""
		default:
			return false
		}
	})
}
