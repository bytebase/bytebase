package mysql

import (
	"context"

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

// StatementMaximumStatementsInTransactionAdvisor validates the rule's level
// and number payload and reports nothing: the MySQL rule has never judged
// statements.
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
	return nil, nil
}
