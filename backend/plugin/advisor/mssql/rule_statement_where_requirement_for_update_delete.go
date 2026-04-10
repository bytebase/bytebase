package mssql

import (
	"context"

	"github.com/bytebase/omni/mssql/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

func init() {
	advisor.Register(storepb.Engine_MSSQL, storepb.SQLReviewRule_STATEMENT_WHERE_REQUIRE_UPDATE_DELETE, &WhereRequirementForUpdateDeleteAdvisor{})
}

type WhereRequirementForUpdateDeleteAdvisor struct{}

func (*WhereRequirementForUpdateDeleteAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	rule := &whereRequirementForUpdateDeleteRule{OmniBaseRule: OmniBaseRule{Level: level, Title: checkCtx.Rule.Type.String()}}
	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type whereRequirementForUpdateDeleteRule struct {
	OmniBaseRule
}

func (*whereRequirementForUpdateDeleteRule) Name() string {
	return "WhereRequirementForUpdateDeleteRule"
}

func (r *whereRequirementForUpdateDeleteRule) OnStatement(node ast.Node) {
	switch n := node.(type) {
	case *ast.DeleteStmt:
		if n.WhereClause == nil {
			r.AddAdvice(&storepb.Advice{
				Status:        r.Level,
				Code:          code.StatementNoWhere.Int32(),
				Title:         r.Title,
				Content:       "WHERE clause is required for DELETE statement.",
				StartPosition: &storepb.Position{Line: r.LocToLine(n.Loc)},
			})
		}
	case *ast.UpdateStmt:
		if n.WhereClause == nil {
			r.AddAdvice(&storepb.Advice{
				Status:        r.Level,
				Code:          code.StatementNoWhere.Int32(),
				Title:         r.Title,
				Content:       "WHERE clause is required for UPDATE statement.",
				StartPosition: &storepb.Position{Line: r.LocToLine(n.Loc)},
			})
		}
	default:
	}
}
