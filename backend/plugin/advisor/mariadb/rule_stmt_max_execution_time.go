package mariadb

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/mariadb/ast"
	mariadbv1review "github.com/bytebase/omni/mariadb/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	mariadbparser "github.com/bytebase/bytebase/backend/plugin/parser/mariadb"
)

var (
	_ advisor.Advisor = (*MaxExecutionTimeAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_STATEMENT_MAX_EXECUTION_TIME, &MaxExecutionTimeAdvisor{})
}

// MaxExecutionTimeAdvisor is the advisor checking for the max execution time.
// The judgment lives in omni (mariadb/v1review.StatementMaxExecutionTime);
// this adapter renders its findings as advice.
type MaxExecutionTimeAdvisor struct {
}

func (*MaxExecutionTimeAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	var nodes []ast.Node
	for _, stmt := range checkCtx.ParsedStatements {
		if stmt.AST == nil {
			continue
		}
		node, ok := mariadbparser.GetNode(stmt.AST)
		if !ok {
			continue
		}
		nodes = append(nodes, node)
	}
	var allAdvice []*storepb.Advice
	for _, f := range mariadbv1review.StatementMaxExecutionTime(nodes) {
		allAdvice = append(allAdvice, &storepb.Advice{
			Status:  level,
			Code:    code.StatementNoMaxExecutionTime.Int32(),
			Title:   checkCtx.Rule.Type.String(),
			Content: fmt.Sprintf("The %s is not set", f.Name),
		})
	}
	return allAdvice, nil
}
