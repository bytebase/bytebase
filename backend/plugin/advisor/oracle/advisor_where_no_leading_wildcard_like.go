// Package oracle is the advisor for oracle database.
package oracle

import (
	"strings"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/plsql-parser"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	_ advisor.Advisor = (*WhereNoLeadingWildcardLikeAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_ORACLE, advisor.OracleNoLeadingWildcardLike, &WhereNoLeadingWildcardLikeAdvisor{})
	advisor.Register(storepb.Engine_DM, advisor.OracleNoLeadingWildcardLike, &WhereNoLeadingWildcardLikeAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE_ORACLE, advisor.OracleNoLeadingWildcardLike, &WhereNoLeadingWildcardLikeAdvisor{})
}

// WhereNoLeadingWildcardLikeAdvisor is the advisor checking for no leading wildcard LIKE.
type WhereNoLeadingWildcardLikeAdvisor struct {
}

// Check checks for no leading wildcard LIKE.
func (*WhereNoLeadingWildcardLikeAdvisor) Check(ctx advisor.Context, _ string) ([]advisor.Advice, error) {
	tree, ok := ctx.AST.(antlr.Tree)
	if !ok {
		return nil, errors.Errorf("failed to convert to Tree")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}

	listener := &whereNoLeadingWildcardLikeListener{
		level:         level,
		title:         string(ctx.Rule.Type),
		currentSchema: ctx.CurrentSchema,
	}

	antlr.ParseTreeWalkerDefault.Walk(listener, tree)

	return listener.generateAdvice()
}

// whereNoLeadingWildcardLikeListener is the listener for no leading wildcard LIKE.
type whereNoLeadingWildcardLikeListener struct {
	*parser.BasePlSqlParserListener

	level         advisor.Status
	title         string
	currentSchema string
	adviceList    []advisor.Advice
}

func (l *whereNoLeadingWildcardLikeListener) generateAdvice() ([]advisor.Advice, error) {
	if len(l.adviceList) == 0 {
		l.adviceList = append(l.adviceList, advisor.Advice{
			Status:  advisor.Success,
			Code:    advisor.Ok,
			Title:   "OK",
			Content: "",
		})
	}
	return l.adviceList, nil
}

// EnterCompound_expression is called when production compound_expression is entered.
func (l *whereNoLeadingWildcardLikeListener) EnterCompound_expression(ctx *parser.Compound_expressionContext) {
	if ctx.LIKE() == nil && ctx.LIKE2() == nil && ctx.LIKE4() == nil && ctx.LIKEC() == nil {
		return
	}

	if ctx.Concatenation(1) == nil {
		return
	}

	text := ctx.Concatenation(1).GetText()
	if strings.HasPrefix(text, "'%") && strings.HasSuffix(text, "'") {
		l.adviceList = append(l.adviceList, advisor.Advice{
			Status:  l.level,
			Code:    advisor.StatementLeadingWildcardLike,
			Title:   l.title,
			Content: "Avoid using leading wildcard LIKE.",
			Line:    ctx.GetStart().GetLine(),
		})
	}
}
