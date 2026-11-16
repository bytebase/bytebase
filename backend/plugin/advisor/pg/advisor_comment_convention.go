package pg

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	"github.com/bytebase/omni/pg/ast"
	pgv1review "github.com/bytebase/omni/pg/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	pgparser "github.com/bytebase/bytebase/backend/plugin/parser/pg"
)

var (
	_ advisor.Advisor = (*CommentConventionAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_SYSTEM_COMMENT_LENGTH, &CommentConventionAdvisor{})
}

// CommentConventionAdvisor is the advisor checking for comment length. The
// judgment lives in omni (pg/v1review.SystemCommentLength); this adapter
// passes the rule's payload and renders the findings as advice.
type CommentConventionAdvisor struct {
}

// Check checks for comment length.
func (*CommentConventionAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	numberPayload := checkCtx.Rule.GetNumberPayload()
	if numberPayload == nil {
		return nil, errors.New("number_payload is required for this rule")
	}
	maxLength := int(numberPayload.Number)

	title := checkCtx.Rule.Type.String()
	var adviceList []*storepb.Advice
	for _, stmt := range checkCtx.ParsedStatements {
		if stmt.AST == nil {
			continue
		}
		node, ok := pgparser.GetNode(stmt.AST)
		if !ok {
			continue
		}
		line := stmt.BaseLine() + int((&BaseRule{StmtText: stmt.Text}).ContentStartLine())
		for range pgv1review.SystemCommentLength([]ast.Node{node}, maxLength) {
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          code.CommentTooLong.Int32(),
				Title:         title,
				Content:       fmt.Sprintf("The length of comment should be within %d characters", maxLength),
				StartPosition: base.ConvertANTLRLineToPosition(line),
			})
		}
	}
	return adviceList, nil
}
