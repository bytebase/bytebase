package pg

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/pg/ast"
	pgv1review "github.com/bytebase/omni/pg/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	pgparser "github.com/bytebase/bytebase/backend/plugin/parser/pg"
)

var (
	_ advisor.Advisor = (*StatementWhereRequiredUpdateDeleteAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_STATEMENT_WHERE_REQUIRE_UPDATE_DELETE, &StatementWhereRequiredUpdateDeleteAdvisor{})
}

// StatementWhereRequiredUpdateDeleteAdvisor is the advisor checking for
// WHERE clause requirement in UPDATE/DELETE. The judgment lives in omni
// (pg/v1review.StatementWhereRequireUpdateDelete); this adapter renders its
// findings as advice.
type StatementWhereRequiredUpdateDeleteAdvisor struct {
}

// Check checks for WHERE clause requirement in UPDATE/DELETE statements.
func (*StatementWhereRequiredUpdateDeleteAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
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
		node, ok := pgparser.GetNode(stmt.AST)
		if !ok {
			continue
		}
		for range pgv1review.StatementWhereRequireUpdateDelete([]ast.Node{node}) {
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          code.StatementNoWhere.Int32(),
				Title:         title,
				Content:       fmt.Sprintf("\"%s\" requires WHERE clause", statementTrimmedText(stmt)),
				StartPosition: &storepb.Position{Line: statementFindingLine(stmt)},
			})
		}
	}
	return adviceList, nil
}
