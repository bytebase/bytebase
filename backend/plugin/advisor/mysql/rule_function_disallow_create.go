package mysql

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/mysql/ast"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	advisorcode "github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*FunctionDisallowCreateAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_SYSTEM_FUNCTION_DISALLOW_CREATE, &FunctionDisallowCreateAdvisor{})
}

// FunctionDisallowCreateAdvisor is the advisor checking for disallow creating function.
type FunctionDisallowCreateAdvisor struct {
}

// Check checks for disallow creating function.
func (*FunctionDisallowCreateAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := &functionDisallowCreateOmniRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
	}

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type functionDisallowCreateOmniRule struct {
	OmniBaseRule
}

func (*functionDisallowCreateOmniRule) Name() string {
	return "FunctionDisallowCreateRule"
}

func (r *functionDisallowCreateOmniRule) OnStatement(node ast.Node) {
	n, ok := node.(*ast.CreateFunctionStmt)
	if !ok || n.IsProcedure {
		return
	}
	if n.Name != nil {
		r.AddAdvice(&storepb.Advice{
			Status:        r.Level,
			Code:          advisorcode.DisallowCreateFunction.Int32(),
			Title:         r.Title,
			Content:       fmt.Sprintf("Function is forbidden, but \"%s\" creates", r.QueryText()),
			StartPosition: common.ConvertANTLRLineToPosition(r.BaseLine + int(r.LocToLine(n.Loc))),
		})
	}
}
