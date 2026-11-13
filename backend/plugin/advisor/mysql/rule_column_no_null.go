package mysql

import (
	"cmp"
	"context"
	"fmt"
	"slices"

	"github.com/bytebase/omni/mysql/ast"
	mysqlv1review "github.com/bytebase/omni/mysql/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

var (
	_ advisor.Advisor = (*ColumnNoNullAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_COLUMN_NO_NULL, &ColumnNoNullAdvisor{})
}

// ColumnNoNullAdvisor is the advisor checking for column no NULL value.
// The judgment lives in omni (mysql/v1review.ColumnNoNull).
type ColumnNoNullAdvisor struct {
}

// Check checks for column no NULL value.
func (*ColumnNoNullAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	stmts, nodes := parsedNodes(checkCtx.ParsedStatements)
	if checkCtx.FinalMetadata == nil && slices.ContainsFunc(nodes, definesColumn) {
		// V1 read the snapshot for each column it collected and panicked on
		// a nil one; advisor.Check reports that panic as the rule's error.
		checkCtx.FinalMetadata.GetSchemaMetadata("")
	}

	type placed struct {
		line   int
		column string
		table  string
	}
	var columns []placed
	for _, f := range mysqlv1review.ColumnNoNull(nodes, v1reviewSchema(checkCtx.FinalMetadata)) {
		stmt := stmts[f.Statement]
		columns = append(columns, placed{line: stmt.BaseLine() + int(findingLine(stmt.Text, f)), column: f.Name, table: f.Table})
	}
	slices.SortStableFunc(columns, func(a, b placed) int {
		return cmp.Or(cmp.Compare(a.line, b.line), cmp.Compare(a.column, b.column))
	})

	var adviceList []*storepb.Advice
	for _, c := range columns {
		adviceList = append(adviceList, &storepb.Advice{
			Status:        level,
			Code:          code.ColumnCannotNull.Int32(),
			Title:         checkCtx.Rule.Type.String(),
			Content:       fmt.Sprintf("`%s`.`%s` cannot have NULL value", c.table, c.column),
			StartPosition: base.ConvertANTLRLineToPosition(c.line),
		})
	}
	return adviceList, nil
}

// definesColumn reports whether the statement defines a column the rule
// collects: a column of a CREATE TABLE, or one ALTER TABLE's ADD or CHANGE
// COLUMN writes, in a statement naming a table.
func definesColumn(node ast.Node) bool {
	switch n := node.(type) {
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
				return cmd.Column != nil || len(cmd.Columns) > 0
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
