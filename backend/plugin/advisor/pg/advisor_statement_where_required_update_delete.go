package pg

import (
	"context"
	"fmt"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/postgresql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

var (
	_ advisor.Advisor = (*StatementWhereRequiredUpdateDeleteAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_STATEMENT_WHERE_REQUIRE_UPDATE_DELETE, &StatementWhereRequiredUpdateDeleteAdvisor{})
}

// StatementWhereRequiredUpdateDeleteAdvisor is the advisor checking for WHERE clause requirement in UPDATE/DELETE.
type StatementWhereRequiredUpdateDeleteAdvisor struct {
}

// Check checks for WHERE clause requirement in UPDATE/DELETE statements.
func (*StatementWhereRequiredUpdateDeleteAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	var adviceList []*storepb.Advice
	for _, stmtInfo := range checkCtx.ParsedStatements {
		if stmtInfo.AST == nil {
			continue
		}
		antlrAST, ok := base.GetANTLRAST(stmtInfo.AST)
		if !ok {
			continue
		}
		rule := &statementWhereRequiredRule{
			BaseRule: BaseRule{
				level: level,
				title: checkCtx.Rule.Type.String(),
			},
			tokens: antlrAST.Tokens,
		}
		rule.SetBaseLine(stmtInfo.BaseLine())

		checker := NewGenericChecker([]Rule{rule})
		antlr.ParseTreeWalkerDefault.Walk(checker, antlrAST.Tree)
		adviceList = append(adviceList, checker.GetAdviceList()...)
	}

	return adviceList, nil
}

type statementWhereRequiredRule struct {
	BaseRule
	tokens *antlr.CommonTokenStream
}

// Name returns the rule name.
func (*statementWhereRequiredRule) Name() string {
	return "statement.where-required-update-delete"
}

// OnEnter is called when the parser enters a rule context.
func (r *statementWhereRequiredRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case "Updatestmt":
		r.handleUpdatestmt(ctx.(*parser.UpdatestmtContext))
	case "Deletestmt":
		r.handleDeletestmt(ctx.(*parser.DeletestmtContext))
	default:
		// Do nothing for other node types
	}
	return nil
}

// OnExit is called when the parser exits a rule context.
func (*statementWhereRequiredRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

func (r *statementWhereRequiredRule) handleUpdatestmt(ctx *parser.UpdatestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	// Check if WHERE clause exists
	if ctx.Where_or_current_clause() == nil || ctx.Where_or_current_clause().WHERE() == nil {
		stmtText := getTextFromTokens(r.tokens, ctx)
		if stmtText == "" {
			stmtText = "<unknown statement>"
		}
		r.AddAdvice(&storepb.Advice{
			Status:  r.level,
			Code:    code.StatementNoWhere.Int32(),
			Title:   r.title,
			Content: fmt.Sprintf("\"%s\" requires WHERE clause", stmtText),
			StartPosition: &storepb.Position{
				Line:   int32(ctx.GetStart().GetLine()),
				Column: 0,
			},
		})
	}
}

func (r *statementWhereRequiredRule) handleDeletestmt(ctx *parser.DeletestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	// Check if WHERE clause exists
	if ctx.Where_or_current_clause() == nil || ctx.Where_or_current_clause().WHERE() == nil {
		stmtText := getTextFromTokens(r.tokens, ctx)
		if stmtText == "" {
			stmtText = "<unknown statement>"
		}
		r.AddAdvice(&storepb.Advice{
			Status:  r.level,
			Code:    code.StatementNoWhere.Int32(),
			Title:   r.title,
			Content: fmt.Sprintf("\"%s\" requires WHERE clause", stmtText),
			StartPosition: &storepb.Position{
				Line:   int32(ctx.GetStart().GetLine()),
				Column: 0,
			},
		})
	}
}
