package pg

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	"github.com/antlr4-go/antlr/v4"

	parser "github.com/bytebase/parser/postgresql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
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
		BaseRule: BaseRule{
			level: level,
			title: checkCtx.Rule.Type.String(),
		},
		maxLength: int(numberPayload.Number),
	}

	checker := NewGenericChecker([]Rule{rule})

	for _, stmt := range checkCtx.ParsedStatements {
		if stmt.AST == nil {
			continue
		}
		antlrAST, ok := base.GetANTLRAST(stmt.AST)
		if !ok {
			continue
		}
		rule.SetBaseLine(stmt.BaseLine())
		checker.SetBaseLine(stmt.BaseLine())
		antlr.ParseTreeWalkerDefault.Walk(checker, antlrAST.Tree)
	}

	return checker.GetAdviceList(), nil
}

type commentConventionRule struct {
	BaseRule

	maxLength int
}

func (*commentConventionRule) Name() string {
	return "comment_convention"
}

func (r *commentConventionRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case "Commentstmt":
		r.handleCommentstmt(ctx)
	default:
		// Do nothing for other node types
	}
	return nil
}

func (*commentConventionRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

func (r *commentConventionRule) handleCommentstmt(ctx antlr.ParserRuleContext) {
	commentstmtCtx, ok := ctx.(*parser.CommentstmtContext)
	if !ok {
		return
	}

	if !isTopLevel(commentstmtCtx.GetParent()) {
		return
	}

	// Extract comment text
	if commentstmtCtx.Comment_text() != nil && commentstmtCtx.Comment_text().Sconst() != nil {
		comment := extractStringConstant(commentstmtCtx.Comment_text().Sconst())

		// Check length
		if r.maxLength > 0 && len(comment) > r.maxLength {
			r.AddAdvice(&storepb.Advice{
				Status:  r.level,
				Code:    code.CommentTooLong.Int32(),
				Title:   r.title,
				Content: fmt.Sprintf("The length of comment should be within %d characters", r.maxLength),
				StartPosition: &storepb.Position{
					Line:   int32(commentstmtCtx.GetStart().GetLine()),
					Column: 0,
				},
			})
		}
	}
}
