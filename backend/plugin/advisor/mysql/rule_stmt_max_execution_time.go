package mysql

import (
	"context"
	"fmt"

	mysqlv1review "github.com/bytebase/omni/mysql/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*MaxExecutionTimeAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_STATEMENT_MAX_EXECUTION_TIME, &MaxExecutionTimeAdvisor{})
}

// MaxExecutionTimeAdvisor is the advisor checking for the max execution time.
// The judgment lives in omni (mysql/v1review.StatementMaxExecutionTime); this
// adapter renders its finding as advice, which has no position.
type MaxExecutionTimeAdvisor struct {
}

func (*MaxExecutionTimeAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	nodes, _ := omniStatements(checkCtx.ParsedStatements)
	var adviceList []*storepb.Advice
	for _, f := range mysqlv1review.StatementMaxExecutionTime(nodes) {
		adviceList = append(adviceList, &storepb.Advice{
			Status:  level,
			Code:    code.StatementNoMaxExecutionTime.Int32(),
			Title:   checkCtx.Rule.Type.String(),
			Content: fmt.Sprintf("The %s is not set", f.Name),
		})
	}
	return adviceList, nil
}
