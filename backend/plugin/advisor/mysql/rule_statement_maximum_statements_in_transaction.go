package mysql

import (
	"context"

	"github.com/antlr4-go/antlr/v4"
	"github.com/bytebase/parser/mysql"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

var (
	_ advisor.Advisor = (*StatementMaximumStatementsInTransactionAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_STATEMENT_MAXIMUM_STATEMENTS_IN_TRANSACTION, &StatementMaximumStatementsInTransactionAdvisor{})
}

type StatementMaximumStatementsInTransactionAdvisor struct {
}

func (*StatementMaximumStatementsInTransactionAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	numberPayload := checkCtx.Rule.GetNumberPayload()
	if numberPayload == nil {
		return nil, errors.New("number_payload is required for this rule")
	}

	// Create the rule
	rule := NewStatementMaximumStatementsInTransactionRule(level, checkCtx.Rule.Type.String(), int(numberPayload.Number))

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
