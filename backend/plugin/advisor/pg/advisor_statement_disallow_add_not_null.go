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
	_ advisor.Advisor = (*StatementDisallowAddNotNullAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_STATEMENT_DISALLOW_ADD_NOT_NULL, &StatementDisallowAddNotNullAdvisor{})
}

// StatementDisallowAddNotNullAdvisor is the advisor checking for to disallow
// add not null. The judgment lives in omni
// (pg/v1review.StatementDisallowAddNotNull); this adapter renders its
// findings as advice.
type StatementDisallowAddNotNullAdvisor struct {
}

// Check checks for to disallow add not null.
func (*StatementDisallowAddNotNullAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
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
		for _, f := range pgv1review.StatementDisallowAddNotNull([]ast.Node{node}) {
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          code.StatementAddNotNull.Int32(),
				Title:         title,
				Content:       fmt.Sprintf("Setting NOT NULL will block reads and writes. You can use CHECK (%q IS NOT NULL) instead", f.Name),
				StartPosition: &storepb.Position{Line: statementFindingLine(stmt)},
			})
		}
	}
	return adviceList, nil
}
