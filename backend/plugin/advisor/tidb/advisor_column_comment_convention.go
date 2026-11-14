package tidb

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/tidb/ast"
	tidbv1review "github.com/bytebase/omni/tidb/v1review"
	"github.com/bytebase/omni/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

var (
	_ advisor.Advisor = (*ColumnCommentConventionAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_COLUMN_COMMENT, &ColumnCommentConventionAdvisor{})
}

// ColumnCommentConventionAdvisor checks the comment convention for columns.
// The judgment lives in omni (tidb/v1review.ColumnComment).
type ColumnCommentConventionAdvisor struct {
}

// Check checks the comment convention for columns.
func (*ColumnCommentConventionAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmts, err := getTiDBNodes(checkCtx)
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
	for _, stmt := range stmts {
		_, isAlter := stmt.Node.(*ast.AlterTableStmt)
		for _, f := range tidbv1review.ColumnComment([]ast.Node{stmt.Node}, payload.GetRequired(), commentMaxLength(payload)) {
			line := stmt.AbsoluteLine(f.Start)
			if isAlter {
				line = stmt.FirstTokenLine()
			}
			advice := &storepb.Advice{
				Status:        level,
				Title:         title,
				StartPosition: base.ConvertANTLRLineToPosition(line),
			}
			switch f.Code {
			case v1review.ColumnCommentMissing:
				advice.Code = code.CommentEmpty.Int32()
				advice.Content = fmt.Sprintf("Column `%s`.`%s` requires comments", f.Table, f.Name)
			case v1review.ColumnCommentTooLong:
				if payload.MaxLength < 0 {
					continue
				}
				advice.Code = code.CommentTooLong.Int32()
				advice.Content = fmt.Sprintf("The length of column `%s`.`%s` comment should be within %d characters", f.Table, f.Name, payload.MaxLength)
			default:
				continue
			}
			adviceList = append(adviceList, advice)
		}
	}
	return adviceList, nil
}

// commentMaxLength is the payload's MaxLength, and -1 for a nil payload so
// that every column comes back and reading the payload for it fails, as V1
// did for the first column it judged.
func commentMaxLength(payload *storepb.SQLReviewRule_CommentConventionRulePayload) int {
	if payload == nil {
		return -1
	}
	return int(payload.MaxLength)
}
