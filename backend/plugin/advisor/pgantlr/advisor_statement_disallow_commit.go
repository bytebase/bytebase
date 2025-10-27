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
	_ advisor.Advisor = (*StatementDisallowCommitAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, advisor.SchemaRuleStatementDisallowCommit, &StatementDisallowCommitAdvisor{})
}

// StatementDisallowCommitAdvisor is the advisor checking for disallowing COMMIT statements.
type StatementDisallowCommitAdvisor struct {
}

// Check checks for disallowing COMMIT statements.
func (*StatementDisallowCommitAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	tree, err := getANTLRTree(checkCtx)
	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	checker := &statementDisallowCommitChecker{
		BasePostgreSQLParserListener: &parser.BasePostgreSQLParserListener{},
		level:                        level,
		title:                        string(checkCtx.Rule.Type),
		statementsText:               checkCtx.Statements,
	}

	antlr.ParseTreeWalkerDefault.Walk(checker, tree.Tree)

	return checker.adviceList, nil
}

type statementDisallowCommitChecker struct {
	*parser.BasePostgreSQLParserListener

	adviceList     []*storepb.Advice
	level          storepb.Advice_Status
	title          string
	statementsText string
}

// EnterTransactionstmt handles COMMIT statements
func (c *statementDisallowCommitChecker) EnterTransactionstmt(ctx *parser.TransactionstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	// Check if this is a COMMIT statement
	if ctx.COMMIT() == nil {
		return
	}

	stmtText := extractStatementText(c.statementsText, ctx.GetStart().GetLine(), ctx.GetStop().GetLine())
	c.adviceList = append(c.adviceList, &storepb.Advice{
		Status:  c.level,
		Code:    advisor.StatementDisallowCommit.Int32(),
		Title:   c.title,
		Content: fmt.Sprintf("Commit is not allowed, related statement: \"%s\"", stmtText),
		StartPosition: &storepb.Position{
			Line:   int32(ctx.GetStart().GetLine()),
			Column: 0,
		},
	})
}
