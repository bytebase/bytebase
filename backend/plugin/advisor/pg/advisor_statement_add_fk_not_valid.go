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
	_ advisor.Advisor = (*StatementAddFKNotValidAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_STATEMENT_ADD_FOREIGN_KEY_NOT_VALID, &StatementAddFKNotValidAdvisor{})
}

// StatementAddFKNotValidAdvisor is the advisor checking for adding foreign
// key constraints without NOT VALID. The judgment lives in omni
// (pg/v1review.StatementAddForeignKeyNotValid); this adapter renders its
// findings as advice.
type StatementAddFKNotValidAdvisor struct {
}

// Check checks for adding foreign key constraints without NOT VALID.
func (*StatementAddFKNotValidAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
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
		for range pgv1review.StatementAddForeignKeyNotValid([]ast.Node{node}) {
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          code.StatementAddFKWithValidation.Int32(),
				Title:         title,
				Content:       "Adding foreign keys with validation will block reads and writes. You can add check foreign keys not valid and then validate separately",
				StartPosition: &storepb.Position{Line: statementFindingLine(stmt)},
			})
		}
	}
	return adviceList, nil
}
