package mysql

import (
	"context"
	"fmt"

	"github.com/antlr4-go/antlr/v4"
	"github.com/bytebase/parser/mysql"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
)

var (
	_ advisor.Advisor = (*StatementDisallowCommitAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_STATEMENT_DISALLOW_COMMIT, &StatementDisallowCommitAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_STATEMENT_DISALLOW_COMMIT, &StatementDisallowCommitAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_STATEMENT_DISALLOW_COMMIT, &StatementDisallowCommitAdvisor{})
}

// StatementDisallowCommitAdvisor is the advisor checking for disallowing commit.
type StatementDisallowCommitAdvisor struct {
}

// Check checks for disallowing commit.
func (*StatementDisallowCommitAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	// Create the rule
	rule := NewStatementDisallowCommitRule(level, checkCtx.Rule.Type.String())

	// Create the generic checker with the rule
	checker := NewGenericChecker([]Rule{rule})

	for _, stmt := range checkCtx.ParsedStatements {
		rule.SetBaseLine(stmt.BaseLine())
		checker.SetBaseLine(stmt.BaseLine())
		if stmt.AST == nil {
			continue
		}
		antlrAST, ok := base.GetANTLRAST(stmt.AST)
		if !ok {
			continue
		}
		antlr.ParseTreeWalkerDefault.Walk(checker, antlrAST.Tree)
	}

	return checker.GetAdviceList(), nil
}

// StatementDisallowCommitRule checks for disallowing commit.
type StatementDisallowCommitRule struct {
	BaseRule
	text string
}

// NewStatementDisallowCommitRule creates a new StatementDisallowCommitRule.
func NewStatementDisallowCommitRule(level storepb.Advice_Status, title string) *StatementDisallowCommitRule {
	return &StatementDisallowCommitRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
	}
}

// Name returns the rule name.
func (*StatementDisallowCommitRule) Name() string {
	return "StatementDisallowCommitRule"
}

// OnEnter is called when entering a parse tree node.
func (r *StatementDisallowCommitRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case NodeTypeQuery:
		queryCtx, ok := ctx.(*mysql.QueryContext)
		if !ok {
			return nil
		}
		r.text = queryCtx.GetParser().GetTokenStream().GetTextFromRuleContext(queryCtx)
	case NodeTypeTransactionStatement:
		r.checkTransactionStatement(ctx.(*mysql.TransactionStatementContext))
	default:
	}
	return nil
}

// OnExit is called when exiting a parse tree node.
func (*StatementDisallowCommitRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

func (r *StatementDisallowCommitRule) checkTransactionStatement(ctx *mysql.TransactionStatementContext) {
	if !mysqlparser.IsTopMySQLRule(&ctx.BaseParserRuleContext) {
		return
	}
	if ctx.COMMIT_SYMBOL() == nil {
		return
	}

	r.AddAdvice(&storepb.Advice{
		Status:        r.level,
		Code:          code.StatementDisallowCommit.Int32(),
		Title:         r.title,
		Content:       fmt.Sprintf("Commit is not allowed, related statement: \"%s\"", r.text),
		StartPosition: common.ConvertANTLRLineToPosition(r.baseLine + ctx.GetStart().GetLine()),
	})
}
