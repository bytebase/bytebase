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
	_ advisor.Advisor = (*StatementWhereRequiredUpdateDeleteAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, advisor.SchemaRuleStatementRequireWhereForUpdateDelete, &StatementWhereRequiredUpdateDeleteAdvisor{})
}

// StatementWhereRequiredUpdateDeleteAdvisor is the advisor checking for WHERE clause requirement in UPDATE/DELETE.
type StatementWhereRequiredUpdateDeleteAdvisor struct {
}

// Check checks for WHERE clause requirement in UPDATE/DELETE statements.
func (*StatementWhereRequiredUpdateDeleteAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	tree, err := getANTLRTree(checkCtx)
	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	checker := &statementWhereRequiredChecker{
		BasePostgreSQLParserListener: &parser.BasePostgreSQLParserListener{},
		level:                        level,
		title:                        string(checkCtx.Rule.Type),
		statementsText:               checkCtx.Statements,
	}

	antlr.ParseTreeWalkerDefault.Walk(checker, tree.Tree)

	return checker.adviceList, nil
}

type statementWhereRequiredChecker struct {
	*parser.BasePostgreSQLParserListener

	adviceList     []*storepb.Advice
	level          storepb.Advice_Status
	title          string
	statementsText string
}

// EnterUpdatestmt handles UPDATE statements
func (c *statementWhereRequiredChecker) EnterUpdatestmt(ctx *parser.UpdatestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	// Check if WHERE clause exists
	if ctx.Where_or_current_clause() == nil || ctx.Where_or_current_clause().WHERE() == nil {
		stmtText := extractStatementText(c.statementsText, ctx.GetStart().GetLine(), ctx.GetStop().GetLine())
		c.adviceList = append(c.adviceList, &storepb.Advice{
			Status:  c.level,
			Code:    advisor.StatementNoWhere.Int32(),
			Title:   c.title,
			Content: fmt.Sprintf("\"%s\" requires WHERE clause", stmtText),
			StartPosition: &storepb.Position{
				Line:   int32(ctx.GetStart().GetLine()),
				Column: 0,
			},
		})
	}
}

// EnterDeletestmt handles DELETE statements
func (c *statementWhereRequiredChecker) EnterDeletestmt(ctx *parser.DeletestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	// Check if WHERE clause exists
	if ctx.Where_or_current_clause() == nil || ctx.Where_or_current_clause().WHERE() == nil {
		stmtText := extractStatementText(c.statementsText, ctx.GetStart().GetLine(), ctx.GetStop().GetLine())
		c.adviceList = append(c.adviceList, &storepb.Advice{
			Status:  c.level,
			Code:    advisor.StatementNoWhere.Int32(),
			Title:   c.title,
			Content: fmt.Sprintf("\"%s\" requires WHERE clause", stmtText),
			StartPosition: &storepb.Position{
				Line:   int32(ctx.GetStart().GetLine()),
				Column: 0,
			},
		})
	}
}
