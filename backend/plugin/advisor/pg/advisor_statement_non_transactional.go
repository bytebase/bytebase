package pg

import (
	"context"

	"github.com/antlr4-go/antlr/v4"

	parser "github.com/bytebase/parser/postgresql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/db/pg"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

var (
	_ advisor.Advisor = (*NonTransactionalAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_STATEMENT_NON_TRANSACTIONAL, &NonTransactionalAdvisor{})
}

// NonTransactionalAdvisor is the advisor checking for non-transactional statements.
type NonTransactionalAdvisor struct {
}

// Check checks for non-transactional statements.
func (*NonTransactionalAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
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
		rule := &nonTransactionalRule{
			BaseRule: BaseRule{
				level: level,
				title: checkCtx.Rule.Type.String(),
			},
			tokens: antlrAST.Tokens,
		}
		checker := NewGenericChecker([]Rule{rule})
		rule.SetBaseLine(stmtInfo.BaseLine())
		checker.SetBaseLine(stmtInfo.BaseLine())
		antlr.ParseTreeWalkerDefault.Walk(checker, antlrAST.Tree)
		adviceList = append(adviceList, checker.GetAdviceList()...)
	}

	return adviceList, nil
}

type nonTransactionalRule struct {
	BaseRule
	tokens *antlr.CommonTokenStream
}

// Name returns the rule name.
func (*nonTransactionalRule) Name() string {
	return "statement.non-transactional"
}

// OnEnter is called when the parser enters a rule context.
func (r *nonTransactionalRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case "Dropdbstmt":
		r.handleDropdbstmt(ctx.(*parser.DropdbstmtContext))
	case "Indexstmt":
		r.handleIndexstmt(ctx.(*parser.IndexstmtContext))
	case "Dropstmt":
		r.handleDropstmt(ctx.(*parser.DropstmtContext))
	case "Vacuumstmt":
		r.handleVacuumstmt(ctx.(*parser.VacuumstmtContext))
	default:
		// Do nothing for other node types
	}
	return nil
}

// OnExit is called when the parser exits a rule context.
func (*nonTransactionalRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

func (r *nonTransactionalRule) handleDropdbstmt(ctx *parser.DropdbstmtContext) {
	r.checkStatement(ctx)
}

func (r *nonTransactionalRule) handleIndexstmt(ctx *parser.IndexstmtContext) {
	r.checkStatement(ctx)
}

func (r *nonTransactionalRule) handleDropstmt(ctx *parser.DropstmtContext) {
	r.checkStatement(ctx)
}

func (r *nonTransactionalRule) handleVacuumstmt(ctx *parser.VacuumstmtContext) {
	r.checkStatement(ctx)
}

// checkStatement checks if a statement is non-transactional
func (r *nonTransactionalRule) checkStatement(ctx antlr.ParserRuleContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	statementText := getTextFromTokens(r.tokens, ctx)
	if pg.IsNonTransactionStatement(statementText) {
		r.AddAdvice(&storepb.Advice{
			Status:  r.level,
			Code:    code.StatementNonTransactional.Int32(),
			Title:   r.title,
			Content: "This statement is non-transactional",
			StartPosition: &storepb.Position{
				Line:   int32(ctx.GetStart().GetLine()),
				Column: 0,
			},
		})
	}
}
