package mysql

import (
	"context"

	"github.com/bytebase/omni/mysql/ast"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
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
	_, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	numberPayload := checkCtx.Rule.GetNumberPayload()
	if numberPayload == nil {
		return nil, errors.New("number_payload is required for this rule")
	}

	rule := &maxStmtsInTxnRule{
		BaseRule: BaseRule{},
	}

	// This rule was a no-op in ANTLR (OnEnter only stored text, no advice generated).
	return RunRules(checkCtx.ParsedStatements, []Rule{rule}), nil
}

type maxStmtsInTxnRule struct {
	BaseRule
}

func (*maxStmtsInTxnRule) Name() string {
	return "StatementMaximumStatementsInTransactionRule"
}

func (*maxStmtsInTxnRule) OnStatement(_ ast.Node) {
	// The original ANTLR rule only stored query text but never generated advice.
}
