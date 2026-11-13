package pg

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/pg/ast"
	pgv1review "github.com/bytebase/omni/pg/v1review"
	"github.com/bytebase/omni/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*ColumnCommentConventionAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_COLUMN_COMMENT, &ColumnCommentConventionAdvisor{})
}

// ColumnCommentConventionAdvisor is the advisor checking for column comment
// convention. The judgment lives in omni (pg/v1review.ColumnComment); this
// adapter passes the payload and renders the findings as advice.
type ColumnCommentConventionAdvisor struct {
}

// Check checks for column comment convention.
func (*ColumnCommentConventionAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	payload := checkCtx.Rule.GetCommentConventionPayload()
	required, maxLength := commentLimits(payload)

	stmts, nodes := parsedNodes(checkCtx.ParsedStatements)
	var adviceList []*storepb.Advice
	for _, f := range pgv1review.ColumnComment(nodes, required, maxLength) {
		// V1 read the payload for each column it judged, so a nil payload
		// panics here as it did there.
		limit := payload.MaxLength
		var adviceCode code.Code
		var content string
		switch f.Code {
		case v1review.ColumnCommentMissing:
			adviceCode = code.CommentEmpty
			content = fmt.Sprintf("Comment is required for column `%s.%s`", f.Table, f.Name)
		case v1review.ColumnCommentTooLong:
			adviceCode = code.CommentTooLong
			content = fmt.Sprintf("Column `%s.%s` comment is too long. The length of comment should be within %d characters", f.Table, f.Name, limit)
		default:
			continue
		}
		adviceList = append(adviceList, &storepb.Advice{
			Status:  level,
			Code:    adviceCode.Int32(),
			Title:   checkCtx.Rule.Type.String(),
			Content: content,
			StartPosition: &storepb.Position{
				Line:   findingLine(stmts[f.Statement], f),
				Column: 0,
			},
		})
	}
	return adviceList, nil
}

// commentLimits returns ColumnComment's arguments for a payload. A nil
// payload is judged as required with every comment too long, so each column
// gives a finding.
func commentLimits(payload *storepb.SQLReviewRule_CommentConventionRulePayload) (bool, int) {
	if payload == nil {
		return true, 0
	}
	if payload.MaxLength <= 0 {
		return payload.Required, -1
	}
	return payload.Required, int(payload.MaxLength)
}

// stringVal extracts a string value from a Node.
func stringVal(n ast.Node) string {
	if s, ok := n.(*ast.String); ok {
		return s.Str
	}
	return ""
}
