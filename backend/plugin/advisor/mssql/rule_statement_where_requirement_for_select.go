package mssql

import (
	"context"

	"github.com/bytebase/omni/mssql/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

func init() {
	advisor.Register(storepb.Engine_MSSQL, storepb.SQLReviewRule_STATEMENT_WHERE_REQUIRE_SELECT, &WhereRequirementForSelectAdvisor{})
}

type WhereRequirementForSelectAdvisor struct{}

func (*WhereRequirementForSelectAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	rule := &whereRequirementForSelectRule{OmniBaseRule: OmniBaseRule{Level: level, Title: checkCtx.Rule.Type.String()}}
	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type whereRequirementForSelectRule struct {
	OmniBaseRule
}

func (*whereRequirementForSelectRule) Name() string {
	return "WhereRequirementForSelectRule"
}

func (r *whereRequirementForSelectRule) OnStatement(node ast.Node) {
	ast.Inspect(node, func(n ast.Node) bool {
		sel, ok := n.(*ast.SelectStmt)
		if !ok {
			return true
		}
		// Allow SELECT queries without a FROM clause, e.g. SELECT 1.
		if sel.FromClause != nil && sel.WhereClause == nil {
			r.AddAdvice(&storepb.Advice{
				Status:        r.Level,
				Code:          code.StatementNoWhere.Int32(),
				Title:         r.Title,
				Content:       "WHERE clause is required for SELETE statement.",
				StartPosition: &storepb.Position{Line: r.LocToLine(sel.Loc)},
			})
		}
		return true
	})
}
