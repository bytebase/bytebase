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
	_ advisor.Advisor = (*InsertMustSpecifyColumnAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_STATEMENT_INSERT_MUST_SPECIFY_COLUMN, &InsertMustSpecifyColumnAdvisor{})
}

// InsertMustSpecifyColumnAdvisor is the advisor checking for to enforce
// column specified. The judgment lives in omni
// (pg/v1review.StatementInsertMustSpecifyColumn); this adapter renders its
// findings as advice.
type InsertMustSpecifyColumnAdvisor struct {
}

// Check checks for to enforce column specified.
func (*InsertMustSpecifyColumnAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
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
		for range pgv1review.StatementInsertMustSpecifyColumn([]ast.Node{node}) {
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          code.InsertNotSpecifyColumn.Int32(),
				Title:         title,
				Content:       fmt.Sprintf("The INSERT statement must specify columns but \"%s\" does not", statementTrimmedText(stmt)),
				StartPosition: &storepb.Position{Line: statementFindingLine(stmt)},
			})
		}
	}
	return adviceList, nil
}
