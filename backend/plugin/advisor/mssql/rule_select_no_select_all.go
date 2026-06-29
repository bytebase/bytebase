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
	// Find all SelectStmt nodes, then check their TargetList for StarExpr.
	// This avoids false positives from COUNT(*) and similar expressions.
	ast.Inspect(node, func(n ast.Node) bool {
		sel, ok := n.(*ast.SelectStmt)
		if !ok || sel.TargetList == nil {
			return true
		}
		for _, item := range sel.TargetList.Items {
			switch v := item.(type) {
			case *ast.StarExpr:
				r.AddAdvice(&storepb.Advice{
					Status:        r.Level,
					Code:          code.StatementSelectAll.Int32(),
					Title:         r.Title,
					Content:       "Avoid using SELECT *.",
					StartPosition: &storepb.Position{Line: r.LocToLine(v.Loc)},
				})
			case *ast.ResTarget:
				if star, ok := v.Val.(*ast.StarExpr); ok {
					r.AddAdvice(&storepb.Advice{
						Status:        r.Level,
						Code:          code.StatementSelectAll.Int32(),
						Title:         r.Title,
						Content:       "Avoid using SELECT *.",
						StartPosition: &storepb.Position{Line: r.LocToLine(star.Loc)},
					})
				}
			default:
			}
		}
		return true
	})
}
