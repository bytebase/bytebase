package tidb

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/tidb/ast"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*ColumnCommentConventionAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_COLUMN_COMMENT, &ColumnCommentConventionAdvisor{})
}

// ColumnCommentConventionAdvisor checks the comment convention for columns.
type ColumnCommentConventionAdvisor struct {
}

// Check is Recipe A. Matches *ast.CreateTableStmt and *ast.AlterTableStmt
// at top level; neither nests. EXPLAIN-DDL is invalid grammar in both
// pingcap and omni.
//
// Cumulative shape divergence #14: column comment in omni lives in the
// direct field ColumnDef.Comment (a normalized string), not in
// column.Options[].Expr as in pingcap. The pingcap-typed advisor needed a
// fallible restoreNode call (and an internal-error advice path on
// failure); omni's direct field eliminates both. The internal-error
// advice path is intentionally removed in this migration — there's no
// fallible op left to wrap.
func (*ColumnCommentConventionAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmts, err := getTiDBOmniNodes(checkCtx)
	if err != nil {
		return nil, err
	}
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	payload := checkCtx.Rule.GetCommentConventionPayload()
	title := checkCtx.Rule.Type.String()

	var adviceList []*storepb.Advice
	for _, ostmt := range stmts {
		var rows []columnCommentRow
		switch n := ostmt.Node.(type) {
		case *ast.CreateTableStmt:
			if n.Table == nil {
				continue
			}
			tableName := n.Table.Name
			for _, col := range n.Columns {
				if col == nil {
					continue
				}
				rows = append(rows, columnCommentRow{
					table:   tableName,
					column:  col.Name,
					exists:  omniColumnHasComment(col),
					comment: col.Comment,
					line:    ostmt.AbsoluteLine(col.Loc.Start),
				})
			}
		case *ast.AlterTableStmt:
			if n.Table == nil {
				continue
			}
			tableName := n.Table.Name
			stmtLine := ostmt.FirstTokenLine()
			for _, cmd := range n.Commands {
				if cmd == nil {
					continue
				}
				switch cmd.Type {
				case ast.ATAddColumn:
					for _, col := range addColumnTargets(cmd) {
						if col == nil {
							continue
						}
						rows = append(rows, columnCommentRow{
							table:   tableName,
							column:  col.Name,
							exists:  omniColumnHasComment(col),
							comment: col.Comment,
							line:    stmtLine,
						})
					}
				case ast.ATChangeColumn, ast.ATModifyColumn:
					if cmd.Column == nil {
						continue
					}
					rows = append(rows, columnCommentRow{
						table:   tableName,
						column:  cmd.Column.Name,
						exists:  omniColumnHasComment(cmd.Column),
						comment: cmd.Column.Comment,
						line:    stmtLine,
					})
				default:
				}
			}
		default:
			continue
		}

		for _, r := range rows {
			if payload.Required && !r.exists {
				adviceList = append(adviceList, &storepb.Advice{
					Status:        level,
					Code:          code.CommentEmpty.Int32(),
					Title:         title,
					Content:       fmt.Sprintf("Column `%s`.`%s` requires comments", r.table, r.column),
					StartPosition: common.ConvertANTLRLineToPosition(r.line),
				})
			}
			if payload.MaxLength >= 0 && int32(len(r.comment)) > payload.MaxLength {
				adviceList = append(adviceList, &storepb.Advice{
					Status:        level,
					Code:          code.CommentTooLong.Int32(),
					Title:         title,
					Content:       fmt.Sprintf("The length of column `%s`.`%s` comment should be within %d characters", r.table, r.column, payload.MaxLength),
					StartPosition: common.ConvertANTLRLineToPosition(r.line),
				})
			}
		}
	}
	return adviceList, nil
}

type columnCommentRow struct {
	table   string
	column  string
	exists  bool
	comment string
	line    int
}
