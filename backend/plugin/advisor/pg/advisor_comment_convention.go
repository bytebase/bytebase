package pg

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	"github.com/bytebase/omni/pg/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*CommentConventionAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_SYSTEM_COMMENT_LENGTH, &CommentConventionAdvisor{})
}

// CommentConventionAdvisor is the advisor checking for comment length.
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

	rule := &commentConventionRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
		maxLength: int(numberPayload.Number),
	}

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type commentConventionRule struct {
	OmniBaseRule

	maxLength int
}

func (*commentConventionRule) Name() string {
	return "comment_convention"
}

func (r *commentConventionRule) OnStatement(node ast.Node) {
	cs, ok := node.(*ast.CommentStmt)
	if !ok {
		return
	}

	if r.maxLength > 0 && len(cs.Comment) > r.maxLength {
		r.AddAdvice(&storepb.Advice{
			Status:  r.Level,
			Code:    code.CommentTooLong.Int32(),
			Title:   r.Title,
			Content: fmt.Sprintf("The length of comment should be within %d characters", r.maxLength),
			StartPosition: &storepb.Position{
				Line:   r.ContentStartLine(),
				Column: 0,
			},
		})
	}
}
