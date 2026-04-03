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
	_ advisor.Advisor = (*TableDisallowTriggerAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_TABLE_DISALLOW_TRIGGER, &TableDisallowTriggerAdvisor{})
}

// TableDisallowTriggerAdvisor is the advisor checking for disallow table trigger.
type TableDisallowTriggerAdvisor struct {
}

// Check checks for disallow table trigger.
func (*TableDisallowTriggerAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := &tableDisallowTriggerOmniRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
	}

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type tableDisallowTriggerOmniRule struct {
	OmniBaseRule
}

func (*tableDisallowTriggerOmniRule) Name() string {
	return "TableDisallowTriggerRule"
}

func (r *tableDisallowTriggerOmniRule) OnStatement(node ast.Node) {
	n, ok := node.(*ast.CreateTriggerStmt)
	if !ok {
		return
	}
	if n.Name != "" {
		r.AddAdvice(&storepb.Advice{
			Status:        r.Level,
			Code:          advisorcode.CreateTableTrigger.Int32(),
			Title:         r.Title,
			Content:       fmt.Sprintf("Trigger is forbidden, but \"%s\" creates", r.QueryText()),
			StartPosition: common.ConvertANTLRLineToPosition(r.BaseLine + int(r.LocToLine(n.Loc))),
		})
	}
}
