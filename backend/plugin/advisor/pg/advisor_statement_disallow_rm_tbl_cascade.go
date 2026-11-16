package pg

import (
	"context"

	"github.com/bytebase/omni/pg/ast"
	pgv1review "github.com/bytebase/omni/pg/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	pgparser "github.com/bytebase/bytebase/backend/plugin/parser/pg"
)

var (
	_ advisor.Advisor = (*StatementDisallowRemoveTblCascadeAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_STATEMENT_DISALLOW_RM_TBL_CASCADE, &StatementDisallowRemoveTblCascadeAdvisor{})
}

// StatementDisallowRemoveTblCascadeAdvisor is the advisor checking for
// disallow CASCADE option when removing tables. The judgment lives in omni
// (pg/v1review.StatementDisallowRmTblCascade); this adapter renders its
// findings as advice.
type StatementDisallowRemoveTblCascadeAdvisor struct {
}

// Check checks for CASCADE option in DROP TABLE and TRUNCATE TABLE statements.
func (*StatementDisallowRemoveTblCascadeAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
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
		for range pgv1review.StatementDisallowRmTblCascade([]ast.Node{node}) {
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          code.StatementDisallowCascade.Int32(),
				Title:         title,
				Content:       "The use of CASCADE is not permitted when removing a table",
				StartPosition: &storepb.Position{Line: statementFindingLine(stmt)},
			})
		}
	}
	return adviceList, nil
}
