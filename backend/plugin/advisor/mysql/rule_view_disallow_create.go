package mysql

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/mysql/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	advisorcode "github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

var (
	_ advisor.Advisor = (*ViewDisallowCreateAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_SYSTEM_VIEW_DISALLOW_CREATE, &ViewDisallowCreateAdvisor{})
}

// ViewDisallowCreateAdvisor is the advisor checking for disallow creating view.
type ViewDisallowCreateAdvisor struct {
}

// Check checks for disallow creating view.
func (*ViewDisallowCreateAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := &viewDisallowCreateRule{
		BaseRule: BaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
	}

	return RunRules(checkCtx.ParsedStatements, []Rule{rule}), nil
}

type viewDisallowCreateRule struct {
	BaseRule
}

func (*viewDisallowCreateRule) Name() string {
	return "ViewDisallowCreateRule"
}

func (r *viewDisallowCreateRule) OnStatement(node ast.Node) {
	n, ok := node.(*ast.CreateViewStmt)
	if !ok {
		return
	}
	if n.Name != nil {
		r.AddAdvice(&storepb.Advice{
			Status:        r.Level,
			Code:          advisorcode.DisallowCreateView.Int32(),
			Title:         r.Title,
			Content:       fmt.Sprintf("View is forbidden, but \"%s\" creates", r.QueryText()),
			StartPosition: base.ConvertANTLRLineToPosition(r.BaseLine + int(r.LocToLine(n.Loc))),
		})
	}
}
