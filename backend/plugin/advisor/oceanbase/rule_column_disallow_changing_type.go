package oceanbase

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/bytebase/omni/oceanbase/ast"
	oceanbasev1review "github.com/bytebase/omni/oceanbase/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*ColumnDisallowChangingTypeAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_COLUMN_DISALLOW_CHANGE_TYPE, &ColumnDisallowChangingTypeAdvisor{})
}

// ColumnDisallowChangingTypeAdvisor is the advisor checking for disallow changing column type.
// The judgment lives in omni (oceanbase/v1review.ColumnDisallowChangeType).
type ColumnDisallowChangingTypeAdvisor struct {
}

// Check checks for disallow changing column type.
func (*ColumnDisallowChangingTypeAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	stmts, nodes := parsedNodes(checkCtx.ParsedStatements)
	if checkCtx.OriginalMetadata == nil && slices.ContainsFunc(nodes, readsColumnType) {
		// V1 read the snapshot at the first such action and panicked on a
		// nil one; advisor.Check reports that panic as the rule's error.
		checkCtx.OriginalMetadata.GetSchemaMetadata("")
	}
	schema := v1reviewSchema(checkCtx.OriginalMetadata)

	var adviceList []*storepb.Advice
	for i, stmt := range stmts {
		for _, f := range oceanbasev1review.ColumnDisallowChangeType([]ast.Node{nodes[i]}, schema) {
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          code.ChangeColumnType.Int32(),
				Title:         checkCtx.Rule.Type.String(),
				Content:       fmt.Sprintf("\"%s\" changes column type", strings.TrimSpace(stmt.Text)),
				StartPosition: columnFindingPosition(stmt, f),
			})
		}
	}
	return adviceList, nil
}

// readsColumnType reports whether the statement is an ALTER TABLE naming a
// table with a CHANGE or MODIFY COLUMN action whose definition has a type,
// for which the rule looks the column up in the snapshot.
func readsColumnType(node ast.Node) bool {
	alter, ok := node.(*ast.AlterTableStmt)
	if !ok || alter.Table == nil {
		return false
	}
	return slices.ContainsFunc(alter.Commands, func(cmd *ast.AlterTableCmd) bool {
		if cmd == nil || cmd.Column == nil || cmd.Column.TypeName == nil {
			return false
		}
		return cmd.Type == ast.ATChangeColumn || cmd.Type == ast.ATModifyColumn
	})
}
