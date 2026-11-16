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
	_ advisor.Advisor = (*StatementCreateSpecifySchema)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_STATEMENT_CREATE_SPECIFY_SCHEMA, &StatementCreateSpecifySchema{})
}

// StatementCreateSpecifySchema is the advisor for checking that created
// tables name their schema. The judgment lives in omni
// (pg/v1review.StatementCreateSpecifySchema); this adapter renders its
// findings as advice.
type StatementCreateSpecifySchema struct {
}

func (*StatementCreateSpecifySchema) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
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
		for _, f := range pgv1review.StatementCreateSpecifySchema([]ast.Node{node}) {
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          code.StatementCreateWithoutSchemaName.Int32(),
				Title:         title,
				Content:       f.Name + " schema should be specified.",
				StartPosition: &storepb.Position{Line: statementFindingLine(stmt)},
			})
		}
	}
	return adviceList, nil
}
