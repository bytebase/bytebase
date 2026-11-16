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
	_ advisor.Advisor = (*StatementDisallowOnDelCascadeAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_STATEMENT_DISALLOW_ON_DEL_CASCADE, &StatementDisallowOnDelCascadeAdvisor{})
}

// StatementDisallowOnDelCascadeAdvisor is the advisor checking for ON DELETE
// CASCADE. The judgment lives in omni
// (pg/v1review.StatementDisallowOnDelCascade); this adapter renders its
// findings as advice.
type StatementDisallowOnDelCascadeAdvisor struct {
}

// Check checks for ON DELETE CASCADE.
func (*StatementDisallowOnDelCascadeAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
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
		for range pgv1review.StatementDisallowOnDelCascade([]ast.Node{node}) {
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          code.StatementDisallowCascade.Int32(),
				Title:         title,
				Content:       "The CASCADE option is not permitted for ON DELETE clauses",
				StartPosition: &storepb.Position{Line: statementFindingLine(stmt)},
			})
		}
	}
	return adviceList, nil
}
