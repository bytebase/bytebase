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
	_ advisor.Advisor = (*StatementCheckSetRoleVariable)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_STATEMENT_CHECK_SET_ROLE_VARIABLE, &StatementCheckSetRoleVariable{})
}

// StatementCheckSetRoleVariable is the advisor for checking that the role is
// set first. The judgment lives in omni
// (pg/v1review.StatementCheckSetRoleVariable); this adapter renders its
// findings as advice.
type StatementCheckSetRoleVariable struct {
}

func (*StatementCheckSetRoleVariable) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	title := checkCtx.Rule.Type.String()
	var nodes []ast.Node
	for _, stmt := range checkCtx.ParsedStatements {
		if stmt.AST == nil {
			continue
		}
		if node, ok := pgparser.GetNode(stmt.AST); ok {
			nodes = append(nodes, node)
		}
	}
	var adviceList []*storepb.Advice
	for range pgv1review.StatementCheckSetRoleVariable(nodes) {
		adviceList = append(adviceList, &storepb.Advice{
			Status:        level,
			Code:          code.StatementCheckSetRoleVariable.Int32(),
			Title:         title,
			Content:       "No SET ROLE statement found.",
			StartPosition: nil,
		})
	}
	return adviceList, nil
}
