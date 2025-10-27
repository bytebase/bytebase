package pgantlr

import (
	"context"

	"github.com/antlr4-go/antlr/v4"

	parser "github.com/bytebase/parser/postgresql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/db/pg"
)

var (
	_ advisor.Advisor = (*NonTransactionalAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, advisor.SchemaRuleStatementNonTransactional, &NonTransactionalAdvisor{})
}

// NonTransactionalAdvisor is the advisor checking for non-transactional statements.
type NonTransactionalAdvisor struct {
}

// Check checks for non-transactional statements.
func (*NonTransactionalAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	tree, err := getANTLRTree(checkCtx)
	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	checker := &nonTransactionalChecker{
		BasePostgreSQLParserListener: &parser.BasePostgreSQLParserListener{},
		level:                        level,
		title:                        string(checkCtx.Rule.Type),
		statementsText:               checkCtx.Statements,
	}

	antlr.ParseTreeWalkerDefault.Walk(checker, tree.Tree)

	return checker.adviceList, nil
}

type nonTransactionalChecker struct {
	*parser.BasePostgreSQLParserListener

	adviceList     []*storepb.Advice
	level          storepb.Advice_Status
	title          string
	statementsText string
}

// checkStatement checks if a statement is non-transactional
func (c *nonTransactionalChecker) checkStatement(ctx antlr.ParserRuleContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	stmtText := extractStatementText(c.statementsText, ctx.GetStart().GetLine(), ctx.GetStop().GetLine())
	if pg.IsNonTransactionStatement(stmtText) {
		c.adviceList = append(c.adviceList, &storepb.Advice{
			Status:  c.level,
			Code:    advisor.StatementNonTransactional.Int32(),
			Title:   c.title,
			Content: "This statement is non-transactional",
			StartPosition: &storepb.Position{
				Line:   int32(ctx.GetStart().GetLine()),
				Column: 0,
			},
		})
	}
}

// EnterDropdbstmt handles DROP DATABASE
func (c *nonTransactionalChecker) EnterDropdbstmt(ctx *parser.DropdbstmtContext) {
	c.checkStatement(ctx)
}

// EnterIndexstmt handles CREATE INDEX
func (c *nonTransactionalChecker) EnterIndexstmt(ctx *parser.IndexstmtContext) {
	c.checkStatement(ctx)
}

// EnterDropstmt handles DROP INDEX (and other DROP statements)
func (c *nonTransactionalChecker) EnterDropstmt(ctx *parser.DropstmtContext) {
	c.checkStatement(ctx)
}

// EnterVacuumstmt handles VACUUM
func (c *nonTransactionalChecker) EnterVacuumstmt(ctx *parser.VacuumstmtContext) {
	c.checkStatement(ctx)
}
