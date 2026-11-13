package pg

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/omni/pg/ast"
	pgv1review "github.com/bytebase/omni/pg/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	pgparser "github.com/bytebase/bytebase/backend/plugin/parser/pg"
)

var (
	_ advisor.Advisor = (*StatementWhereRequiredSelectAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_STATEMENT_WHERE_REQUIRE_SELECT, &StatementWhereRequiredSelectAdvisor{})
}

// StatementWhereRequiredSelectAdvisor is the advisor checking for WHERE
// clause requirement in SELECT statements. The judgment lives in omni
// (pg/v1review.StatementWhereRequireSelect); this adapter renders its
// findings as advice.
type StatementWhereRequiredSelectAdvisor struct {
}

// Check checks for WHERE clause requirement in SELECT statements.
func (*StatementWhereRequiredSelectAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
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
		for _, f := range pgv1review.StatementWhereRequireSelect([]ast.Node{node}) {
			// A finding on a nested SELECT prints that SELECT's text, with
			// the parentheses around it.
			text := strings.TrimSpace(stmt.Text)
			if start, end := f.Start, f.End; end > start && end <= len(stmt.Text) {
				if start > 0 && stmt.Text[start-1] == '(' && end < len(stmt.Text) && stmt.Text[end] == ')' {
					start--
					end++
				}
				text = stmt.Text[start:end]
			}
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          code.StatementNoWhere.Int32(),
				Title:         title,
				Content:       fmt.Sprintf("\"%s\" requires WHERE clause", text),
				StartPosition: &storepb.Position{Line: statementFindingLine(stmt)},
			})
		}
	}
	return adviceList, nil
}
