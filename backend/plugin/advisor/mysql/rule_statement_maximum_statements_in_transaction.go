package mysql

import (
	"context"

	"github.com/antlr4-go/antlr/v4"
	mysql "github.com/bytebase/mysql-parser"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
)

var (
	_ advisor.Advisor = (*StatementMaximumStatementsInTransactionAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.SchemaRuleStatementMaximumStatementsInTransaction, &StatementMaximumStatementsInTransactionAdvisor{})
}

type StatementMaximumStatementsInTransactionAdvisor struct {
}

func (*StatementMaximumStatementsInTransactionAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmtList, ok := checkCtx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to mysql parse result")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	payload, err := advisor.UnmarshalNumberTypeRulePayload(checkCtx.Rule.Payload)
	if err != nil {
		return nil, err
	}

	// Create the rule
	rule := NewStatementMaximumStatementsInTransactionRule(level, string(checkCtx.Rule.Type), payload.Number)

	// Create the generic checker with the rule
	checker := NewGenericChecker([]Rule{rule})

	for _, stmt := range stmtList {
		rule.SetBaseLine(stmt.BaseLine)
		checker.SetBaseLine(stmt.BaseLine)
		antlr.ParseTreeWalkerDefault.Walk(checker, stmt.Tree)
	}

	return checker.GetAdviceList(), nil
}

// StatementMaximumStatementsInTransactionRule checks for maximum statements in transaction.
type StatementMaximumStatementsInTransactionRule struct {
	BaseRule
	text          string
	limitMaxValue int
}

// NewStatementMaximumStatementsInTransactionRule creates a new StatementMaximumStatementsInTransactionRule.
func NewStatementMaximumStatementsInTransactionRule(level storepb.Advice_Status, title string, limitMaxValue int) *StatementMaximumStatementsInTransactionRule {
	return &StatementMaximumStatementsInTransactionRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
		limitMaxValue: limitMaxValue,
	}
}

// Name returns the rule name.
func (*StatementMaximumStatementsInTransactionRule) Name() string {
	return "StatementMaximumStatementsInTransactionRule"
}

// OnEnter is called when entering a parse tree node.
func (r *StatementMaximumStatementsInTransactionRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	if nodeType == NodeTypeQuery {
		queryCtx, ok := ctx.(*mysql.QueryContext)
		if !ok {
			return nil
		}
		r.text = queryCtx.GetParser().GetTokenStream().GetTextFromRuleContext(queryCtx)
	}
	return nil
}

// OnExit is called when exiting a parse tree node.
func (*StatementMaximumStatementsInTransactionRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}
