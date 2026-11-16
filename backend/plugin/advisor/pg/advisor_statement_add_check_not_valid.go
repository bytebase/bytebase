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
	_ advisor.Advisor = (*StatementAddCheckNotValidAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_STATEMENT_ADD_CHECK_NOT_VALID, &StatementAddCheckNotValidAdvisor{})
}

// StatementAddCheckNotValidAdvisor is the advisor checking for to add check
// not valid. The judgment lives in omni
// (pg/v1review.StatementAddCheckNotValid); this adapter renders its findings
// as advice.
type StatementAddCheckNotValidAdvisor struct {
}

// Check checks for to add check not valid.
func (*StatementAddCheckNotValidAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
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
		for range pgv1review.StatementAddCheckNotValid([]ast.Node{node}) {
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          code.StatementAddCheckWithValidation.Int32(),
				Title:         title,
				Content:       "Adding check constraints with validation will block reads and writes. You can add check constraints not valid and then validate separately",
				StartPosition: &storepb.Position{Line: statementFindingLine(stmt)},
			})
		}
	}
	return adviceList, nil
}
