package mssql

import (
	"context"

	"github.com/bytebase/omni/mssql/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

func init() {
	advisor.Register(storepb.Engine_MSSQL, storepb.SQLReviewRule_STATEMENT_SELECT_NO_SELECT_ALL, &SelectNoSelectAllAdvisor{})
}

type SelectNoSelectAllAdvisor struct{}

func (*SelectNoSelectAllAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	rule := &selectNoSelectAllRule{OmniBaseRule: OmniBaseRule{Level: level, Title: checkCtx.Rule.Type.String()}}
	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type selectNoSelectAllRule struct {
	OmniBaseRule
}

func (*selectNoSelectAllRule) Name() string {
	return "SelectNoSelectAllRule"
}

func (r *selectNoSelectAllRule) OnStatement(node ast.Node) {
	ast.Inspect(node, func(n ast.Node) bool {
		if star, ok := n.(*ast.StarExpr); ok {
			r.AddAdvice(&storepb.Advice{
				Status:        r.Level,
				Code:          code.StatementSelectAll.Int32(),
				Title:         r.Title,
				Content:       "Avoid using SELECT *.",
				StartPosition: &storepb.Position{Line: r.LocToLine(star.Loc)},
			})
		}
		return true
	})
}
