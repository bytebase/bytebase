package mysql

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/omni/mysql/ast"
	mysqlv1review "github.com/bytebase/omni/mysql/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	advisorcode "github.com/bytebase/bytebase/backend/plugin/advisor/code"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
)

var (
	_ advisor.Advisor = (*EventDisallowCreateAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_SYSTEM_EVENT_DISALLOW_CREATE, &EventDisallowCreateAdvisor{})
}

// EventDisallowCreateAdvisor is the advisor checking for disallow creating event. The judgment
// lives in omni (mysql/v1review.SystemEventDisallowCreate); this adapter renders the findings as
// advice.
type EventDisallowCreateAdvisor struct {
}

// Check checks for disallow creating event.
func (*EventDisallowCreateAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	title := checkCtx.Rule.Type.String()
	var adviceList []*storepb.Advice
	for _, stmt := range checkCtx.ParsedStatements {
		if stmt.AST == nil {
			continue
		}
		node, ok := mysqlparser.GetNode(stmt.AST)
		if !ok {
			continue
		}
		for _, f := range mysqlv1review.SystemEventDisallowCreate([]ast.Node{node}) {
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          advisorcode.DisallowCreateEvent.Int32(),
				Title:         title,
				Content:       fmt.Sprintf("Event is forbidden, but \"%s\" creates", strings.TrimSpace(stmt.Text)),
				StartPosition: findingPositionBaseLineTwice(stmt, f),
			})
		}
	}
	return adviceList, nil
}
