// Package snowflake is the advisor for snowflake database.
package snowflake

import (
	"context"

	snowflakeast "github.com/bytebase/omni/snowflake/ast"
	snowflakev1review "github.com/bytebase/omni/snowflake/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	snowsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/snowflake"
)

var (
	_ advisor.Advisor = (*SelectNoSelectAllAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_SNOWFLAKE, storepb.SQLReviewRule_STATEMENT_SELECT_NO_SELECT_ALL, &SelectNoSelectAllAdvisor{})
}

// SelectNoSelectAllAdvisor is the advisor checking for no select all.
// The judgment lives in omni (snowflake/v1review.StatementSelectNoSelectAll);
// this adapter renders its findings as advice.
type SelectNoSelectAllAdvisor struct {
}

// Check checks for no select all.
func (*SelectNoSelectAllAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
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
		for _, f := range snowflakev1review.StatementSelectNoSelectAll([]snowflakeast.Node{node}) {
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          code.StatementSelectAll.Int32(),
				Title:         title,
				Content:       "Avoid using SELECT *.",
				StartPosition: base.ConvertANTLRLineToPosition(stmt.BaseLine() + findingLine(stmt.Text, f)),
			})
		}
	}
	return adviceList, nil
}
