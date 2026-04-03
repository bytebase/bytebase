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
	_ advisor.Advisor = (*EventDisallowCreateAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_SYSTEM_EVENT_DISALLOW_CREATE, &EventDisallowCreateAdvisor{})
}

// EventDisallowCreateAdvisor is the advisor checking for disallow creating event.
type EventDisallowCreateAdvisor struct {
}

// Check checks for disallow creating event.
func (*EventDisallowCreateAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := &eventDisallowCreateOmniRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
	}

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type eventDisallowCreateOmniRule struct {
	OmniBaseRule
}

func (*eventDisallowCreateOmniRule) Name() string {
	return "EventDisallowCreateRule"
}

func (r *eventDisallowCreateOmniRule) OnStatement(node ast.Node) {
	n, ok := node.(*ast.CreateEventStmt)
	if !ok {
		return
	}
	if n.Name != "" {
		r.AddAdvice(&storepb.Advice{
			Status:        r.Level,
			Code:          advisorcode.DisallowCreateEvent.Int32(),
			Title:         r.Title,
			Content:       fmt.Sprintf("Event is forbidden, but \"%s\" creates", r.QueryText()),
			StartPosition: common.ConvertANTLRLineToPosition(r.BaseLine + int(r.LocToLine(n.Loc))),
		})
	}
}
