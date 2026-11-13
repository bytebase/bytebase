package mysql

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/mysql/ast"
	mysqlv1review "github.com/bytebase/omni/mysql/v1review"
	"github.com/bytebase/omni/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*ColumnCommentConventionAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_COLUMN_COMMENT, &ColumnCommentConventionAdvisor{})
}

// ColumnCommentConventionAdvisor is the advisor checking for column comment convention.
// The judgment lives in omni (mysql/v1review.ColumnComment).
type ColumnCommentConventionAdvisor struct {
}

// Check checks for column comment convention.
func (*ColumnCommentConventionAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	commentPayload := checkCtx.Rule.GetCommentConventionPayload()
	// Without a payload, judge so that every column yields a finding: the
	// payload is read for each finding below, so a nil payload panics at the
	// first column, as this rule always has.
	required, maxLength := true, 0
	if commentPayload != nil {
		required, maxLength = commentPayload.Required, int(commentPayload.MaxLength)
	}

	stmts, nodes := parsedNodes(checkCtx.ParsedStatements)
	var adviceList []*storepb.Advice
	for i, stmt := range stmts {
		for _, f := range mysqlv1review.ColumnComment([]ast.Node{nodes[i]}, required, maxLength) {
			limit := commentPayload.MaxLength
			var adviceCode code.Code
			var content string
			switch f.Code {
			case v1review.ColumnCommentTooLong:
				adviceCode = code.CommentTooLong
				content = fmt.Sprintf("The length of column `%s`.`%s` comment should be within %d characters", f.Table, f.Name, limit)
			case v1review.ColumnCommentMissing:
				adviceCode = code.CommentEmpty
				content = fmt.Sprintf("Column `%s`.`%s` requires comments", f.Table, f.Name)
			default:
				continue
			}
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          adviceCode.Int32(),
				Title:         checkCtx.Rule.Type.String(),
				Content:       content,
				StartPosition: findingPosition(stmt, f),
			})
		}
	}
	return adviceList, nil
}
