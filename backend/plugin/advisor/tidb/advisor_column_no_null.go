package tidb

import (
	"cmp"
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
	_ advisor.Advisor = (*ColumnNoNullAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_COLUMN_NO_NULL, &ColumnNoNullAdvisor{})
}

// ColumnNoNullAdvisor is the advisor checking for column no NULL value.
// The judgment lives in omni (tidb/v1review.ColumnNoNull).
type ColumnNoNullAdvisor struct {
}

// Check checks for column no NULL value. Advice is ordered by line, then
// column name, the two statements of one line included.
func (*ColumnNoNullAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmts, err := getTiDBNodes(checkCtx)
	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	title := checkCtx.Rule.Type.String()
	if checkCtx.FinalMetadata == nil && slices.ContainsFunc(stmts, definesColumn) {
		// V1 read the snapshot once it had a column to judge and panicked
		// on a nil one; advisor.Check reports that panic as the rule's error.
		checkCtx.FinalMetadata.GetSchemaMetadata("")
	}

	type located struct {
		line   int
		column string
		advice *storepb.Advice
	}
	var found []located
	for _, f := range tidbv1review.ColumnNoNull(stmtNodes(stmts), v1reviewSchema(checkCtx.FinalMetadata)) {
		line := stmts[f.Statement].AbsoluteLine(f.Start)
		found = append(found, located{line, f.Name, &storepb.Advice{
			Status:        level,
			Code:          code.ColumnCannotNull.Int32(),
			Title:         title,
			Content:       fmt.Sprintf("`%s`.`%s` cannot have NULL value", f.Table, f.Name),
			StartPosition: base.ConvertANTLRLineToPosition(line),
		}})
	}
	slices.SortStableFunc(found, func(a, b located) int {
		return cmp.Or(cmp.Compare(a.line, b.line), cmp.Compare(a.column, b.column))
	})
	var adviceList []*storepb.Advice
	for _, l := range found {
		adviceList = append(adviceList, l.advice)
	}
	return adviceList, nil
}

// definesColumn reports whether the statement gives the rule a column to
// judge: a CREATE TABLE naming a table with a column, or an ALTER TABLE
// naming a table that adds a column or changes one.
func definesColumn(stmt Stmt) bool {
	switch n := stmt.Node.(type) {
	case *ast.CreateTableStmt:
		return n.Table != nil && slices.ContainsFunc(n.Columns, func(col *ast.ColumnDef) bool { return col != nil })
	case *ast.AlterTableStmt:
		if n.Table == nil {
			return false
		}
		return slices.ContainsFunc(n.Commands, func(cmd *ast.AlterTableCmd) bool {
			if cmd == nil {
				return false
			}
			switch cmd.Type {
			case ast.ATAddColumn:
				return slices.ContainsFunc(addColumnTargets(cmd), func(col *ast.ColumnDef) bool { return col != nil })
			case ast.ATChangeColumn:
				return cmd.Column != nil
			default:
				return false
			}
		})
	default:
		return false
	}
}
