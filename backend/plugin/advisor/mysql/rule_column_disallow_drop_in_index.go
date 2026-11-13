package mysql

import (
	"context"
	"fmt"
	"slices"

	"github.com/bytebase/omni/mysql/ast"
	mysqlv1review "github.com/bytebase/omni/mysql/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*ColumnDisallowDropInIndexAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_COLUMN_DISALLOW_DROP_IN_INDEX, &ColumnDisallowDropInIndexAdvisor{})
}

// ColumnDisallowDropInIndexAdvisor is the advisor checking for disallow DROP COLUMN in index.
// The judgment lives in omni (mysql/v1review.ColumnDisallowDropInIndex).
type ColumnDisallowDropInIndexAdvisor struct {
}

// Check checks for disallow Drop COLUMN in index statement.
func (*ColumnDisallowDropInIndexAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	stmts, nodes := parsedNodes(checkCtx.ParsedStatements)
	if checkCtx.OriginalMetadata == nil && slices.ContainsFunc(nodes, dropsColumn) {
		// V1 read the snapshot at the first DROP COLUMN and panicked on a nil
		// one; advisor.Check reports that panic as the rule's error.
		checkCtx.OriginalMetadata.GetSchemaMetadata("")
	}

	var adviceList []*storepb.Advice
	for _, f := range mysqlv1review.ColumnDisallowDropInIndex(nodes, v1reviewSchema(checkCtx.OriginalMetadata)) {
		adviceList = append(adviceList, &storepb.Advice{
			Status:        level,
			Code:          code.DropIndexColumn.Int32(),
			Title:         checkCtx.Rule.Type.String(),
			Content:       fmt.Sprintf("`%s`.`%s` cannot drop index column", f.Table, f.Name),
			StartPosition: findingPosition(stmts[f.Statement], f),
		})
	}
	return adviceList, nil
}

// dropsColumn reports whether the statement is an ALTER TABLE naming a table
// with a DROP COLUMN action, at which the rule reads the snapshot.
func dropsColumn(node ast.Node) bool {
	alter, ok := node.(*ast.AlterTableStmt)
	if !ok || alter.Table == nil {
		return false
	}
	return slices.ContainsFunc(alter.Commands, func(cmd *ast.AlterTableCmd) bool {
		return cmd != nil && cmd.Type == ast.ATDropColumn
	})
}
