// Package bigquery is the advisor for BigQuery.
package bigquery

import (
	"context"
	"fmt"

	omniast "github.com/bytebase/omni/googlesql/ast"
	googlesqlv1review "github.com/bytebase/omni/googlesql/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"

	// Registers the BigQuery parser so ParseStatements resolves the engine.
	_ "github.com/bytebase/bytebase/backend/plugin/parser/bigquery"
	"github.com/bytebase/bytebase/backend/plugin/parser/googlesql"
)

var (
	_ advisor.Advisor = (*WhereRequireForUpdateDeleteAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_BIGQUERY, storepb.SQLReviewRule_STATEMENT_WHERE_REQUIRE_UPDATE_DELETE, &WhereRequireForUpdateDeleteAdvisor{})
}

// WhereRequireForUpdateDeleteAdvisor is the advisor checking for WHERE clause
// requirement for UPDATE and DELETE statement. The judgment lives in omni
// (googlesql/v1review.StatementWhereRequireUpdateDelete); this adapter renders
// its findings as advice.
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
		ast, ok := stmt.AST.(*googlesql.AST)
		if !ok || ast.Node == nil {
			continue
		}
		for _, f := range googlesqlv1review.StatementWhereRequireUpdateDelete([]omniast.Node{ast.Node}) {
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          code.StatementNoWhere.Int32(),
				Title:         title,
				Content:       fmt.Sprintf("WHERE clause is required for %s statement.", f.Name),
				StartPosition: base.ConvertANTLRLineToPosition(stmt.BaseLine() + statementFindingLine(stmt.Text, f)),
			})
		}
	}
	return adviceList, nil
}
