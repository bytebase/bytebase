// Package snowflake is the advisor for snowflake database.
package snowflake

import (
	"context"
	"fmt"

	snowflakeast "github.com/bytebase/omni/snowflake/ast"
	snowflakev1review "github.com/bytebase/omni/snowflake/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	snowsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/snowflake"
)

var (
	_ advisor.Advisor = (*WhereRequireForUpdateDeleteAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_SNOWFLAKE, storepb.SQLReviewRule_STATEMENT_WHERE_REQUIRE_UPDATE_DELETE, &WhereRequireForUpdateDeleteAdvisor{})
}

// WhereRequireForUpdateDeleteAdvisor is the advisor checking for WHERE clause requirement for UPDATE and DELETE statement.
// The judgment lives in omni (snowflake/v1review.StatementWhereRequireUpdateDelete);
// this adapter renders its findings as advice.
type WhereRequireForUpdateDeleteAdvisor struct {
}

// Check checks for WHERE clause requirement.
func (*WhereRequireForUpdateDeleteAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	title := checkCtx.Rule.Type.String()
	var adviceList []*storepb.Advice
	for _, stmt := range checkCtx.ParsedStatements {
		node, ok := snowsqlparser.GetNode(stmt.AST)
		if !ok {
			continue
		}
		for _, f := range snowflakev1review.StatementWhereRequireUpdateDelete([]snowflakeast.Node{node}) {
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          code.StatementNoWhere.Int32(),
				Title:         title,
				Content:       fmt.Sprintf("WHERE clause is required for %s statement.", f.Name),
				StartPosition: base.ConvertANTLRLineToPosition(stmt.BaseLine() + findingLine(stmt.Text, f)),
			})
		}
	}
	return adviceList, nil
}
