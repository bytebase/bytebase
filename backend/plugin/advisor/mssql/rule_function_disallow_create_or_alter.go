package mssql

import (
	"context"

	"github.com/bytebase/omni/mssql/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

func init() {
	advisor.Register(storepb.Engine_MSSQL, storepb.SQLReviewRule_SYSTEM_FUNCTION_DISALLOW_CREATE, &FunctionDisallowCreateOrAlterAdvisor{})
}

type FunctionDisallowCreateOrAlterAdvisor struct{}

func (*FunctionDisallowCreateOrAlterAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	rule := &functionDisallowCreateOrAlterRule{OmniBaseRule: OmniBaseRule{Level: level, Title: checkCtx.Rule.Type.String()}}
	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type functionDisallowCreateOrAlterRule struct {
	OmniBaseRule
}

func (*functionDisallowCreateOrAlterRule) Name() string {
	return "FunctionDisallowCreateOrAlterRule"
}

func (r *functionDisallowCreateOrAlterRule) OnStatement(node ast.Node) {
	if n, ok := node.(*ast.CreateFunctionStmt); ok {
		r.AddAdvice(&storepb.Advice{
			Status:        r.Level,
			Code:          code.DisallowCreateFunction.Int32(),
			Title:         r.Title,
			Content:       "Creating or altering functions is prohibited",
			StartPosition: &storepb.Position{Line: r.LocToLine(n.Loc)},
		})
	}
}
