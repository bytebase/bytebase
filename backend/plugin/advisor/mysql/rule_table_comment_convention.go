package mysql

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/mysql/ast"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*TableCommentConventionAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_TABLE_COMMENT, &TableCommentConventionAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_TABLE_COMMENT, &TableCommentConventionAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_TABLE_COMMENT, &TableCommentConventionAdvisor{})
}

// TableCommentConventionAdvisor is the advisor checking for table comment convention.
type TableCommentConventionAdvisor struct {
}

// Check checks for table comment convention.
func (*TableCommentConventionAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	commentPayload := checkCtx.Rule.GetCommentConventionPayload()

	rule := &tableCommentConventionOmniRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
		payload: commentPayload,
	}

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type tableCommentConventionOmniRule struct {
	OmniBaseRule
	payload *storepb.SQLReviewRule_CommentConventionRulePayload
}

func (*tableCommentConventionOmniRule) Name() string {
	return "TableCommentConventionRule"
}

func (r *tableCommentConventionOmniRule) OnStatement(node ast.Node) {
	n, ok := node.(*ast.CreateTableStmt)
	if !ok {
		return
	}
	r.checkCreateTable(n)
}

func (r *tableCommentConventionOmniRule) checkCreateTable(n *ast.CreateTableStmt) {
	if n.Table == nil {
		return
	}
	tableName := n.Table.Name

	comment := omniTableOptionValue(n.Options, "COMMENT")
	exists := comment != ""
	// Check if any COMMENT option exists (even empty string).
	for _, opt := range n.Options {
		if opt != nil && opt.Name == "COMMENT" {
			exists = true
			break
		}
	}

	if r.payload.Required && !exists {
		r.AddAdvice(&storepb.Advice{
			Status:        r.Level,
			Code:          code.CommentEmpty.Int32(),
			Title:         r.Title,
			Content:       fmt.Sprintf("Table `%s` requires comments", tableName),
			StartPosition: common.ConvertANTLRLineToPosition(int(r.ContentStartLine())),
		})
	}
	if r.payload.MaxLength >= 0 && int32(len(comment)) > r.payload.MaxLength {
		r.AddAdvice(&storepb.Advice{
			Status:        r.Level,
			Code:          code.CommentTooLong.Int32(),
			Title:         r.Title,
			Content:       fmt.Sprintf("The length of table `%s` comment should be within %d characters", tableName, r.payload.MaxLength),
			StartPosition: common.ConvertANTLRLineToPosition(int(r.ContentStartLine())),
		})
	}
}
