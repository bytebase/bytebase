package pgantlr

import (
	"context"
	"fmt"

	"github.com/antlr4-go/antlr/v4"

	parser "github.com/bytebase/parser/postgresql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
)

var (
	_ advisor.Advisor = (*CommentConventionAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, advisor.SchemaRuleCommentLength, &CommentConventionAdvisor{})
}

// CommentConventionAdvisor is the advisor checking for comment length.
type CommentConventionAdvisor struct {
}

// Check checks for comment length.
func (*CommentConventionAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	tree, err := getANTLRTree(checkCtx)
	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	payload, err := advisor.UnmarshalNumberTypeRulePayload(checkCtx.Rule.Payload)
	if err != nil {
		return nil, err
	}

	checker := &commentConventionChecker{
		BasePostgreSQLParserListener: &parser.BasePostgreSQLParserListener{},
		level:                        level,
		title:                        string(checkCtx.Rule.Type),
		maxLength:                    payload.Number,
	}

	antlr.ParseTreeWalkerDefault.Walk(checker, tree.Tree)

	return checker.adviceList, nil
}

type commentConventionChecker struct {
	*parser.BasePostgreSQLParserListener

	adviceList []*storepb.Advice
	level      storepb.Advice_Status
	title      string
	maxLength  int
}

func (c *commentConventionChecker) EnterCommentstmt(ctx *parser.CommentstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	// Extract comment text
	if ctx.Comment_text() != nil && ctx.Comment_text().Sconst() != nil {
		comment := extractStringConstant(ctx.Comment_text().Sconst())

		// Check length
		if c.maxLength > 0 && len(comment) > c.maxLength {
			c.adviceList = append(c.adviceList, &storepb.Advice{
				Status:  c.level,
				Code:    advisor.CommentTooLong.Int32(),
				Title:   c.title,
				Content: fmt.Sprintf("The length of comment should be within %d characters", c.maxLength),
				StartPosition: &storepb.Position{
					Line:   int32(ctx.GetStart().GetLine()),
					Column: 0,
				},
			})
		}
	}
}
