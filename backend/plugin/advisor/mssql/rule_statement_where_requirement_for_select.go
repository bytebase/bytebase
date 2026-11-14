package mssql

import (
	"context"

	"github.com/bytebase/omni/mssql/ast"
	mssqlv1review "github.com/bytebase/omni/mssql/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	tsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/tsql"
)

func init() {
	advisor.Register(storepb.Engine_MSSQL, storepb.SQLReviewRule_STATEMENT_WHERE_REQUIRE_SELECT, &WhereRequirementForSelectAdvisor{})
}

// WhereRequirementForSelectAdvisor is the advisor requiring a WHERE clause
// for SELECT. The judgment lives in omni
// (mssql/v1review.StatementWhereRequireSelect); this adapter renders its
// findings as advice.
type WhereRequirementForSelectAdvisor struct{}

func (*WhereRequirementForSelectAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	title := checkCtx.Rule.Type.String()
	var adviceList []*storepb.Advice
	for _, stmt := range checkCtx.ParsedStatements {
		if stmt.AST == nil {
			continue
		}
		node, ok := tsqlparser.GetNode(stmt.AST)
		if !ok {
			continue
		}
		for _, f := range mssqlv1review.StatementWhereRequireSelect([]ast.Node{node}) {
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          code.StatementNoWhere.Int32(),
				Title:         title,
				Content:       "WHERE clause is required for SELETE statement.",
				StartPosition: &storepb.Position{Line: int32(stmt.BaseLine()) + findingLine(stmt.Text, f)},
			})
		}
	}
	return adviceList, nil
}
