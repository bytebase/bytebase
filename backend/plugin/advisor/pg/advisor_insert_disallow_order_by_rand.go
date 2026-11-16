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
	_ advisor.Advisor = (*InsertDisallowOrderByRandAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_STATEMENT_INSERT_DISALLOW_ORDER_BY_RAND, &InsertDisallowOrderByRandAdvisor{})
}

// InsertDisallowOrderByRandAdvisor is the advisor checking for to disallow
// order by rand in INSERT statements. The judgment lives in omni
// (pg/v1review.StatementInsertDisallowOrderByRand); this adapter renders its
// findings as advice.
type InsertDisallowOrderByRandAdvisor struct {
}

// Check checks for to disallow order by rand in INSERT statements.
func (*InsertDisallowOrderByRandAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
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
		for range pgv1review.StatementInsertDisallowOrderByRand([]ast.Node{node}) {
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          code.InsertUseOrderByRand.Int32(),
				Title:         title,
				Content:       fmt.Sprintf("The INSERT statement uses ORDER BY random() or random_between(), related statement \"%s\"", statementTrimmedText(stmt)),
				StartPosition: &storepb.Position{Line: statementFindingLine(stmt)},
			})
		}
	}
	return adviceList, nil
}
