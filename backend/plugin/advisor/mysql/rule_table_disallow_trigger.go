package mysql

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/omni/mysql/ast"
	mysqlv1review "github.com/bytebase/omni/mysql/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
)

var (
	_ advisor.Advisor = (*TableDisallowTriggerAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_TABLE_DISALLOW_TRIGGER, &TableDisallowTriggerAdvisor{})
}

// TableDisallowTriggerAdvisor is the advisor checking for disallow table trigger.
// The judgment lives in omni (mysql/v1review.TableDisallowTrigger).
type TableDisallowTriggerAdvisor struct {
}

// Check checks for disallow table trigger.
func (*TableDisallowTriggerAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
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
		for _, f := range mysqlv1review.TableDisallowTrigger([]ast.Node{node}) {
			// V1 added the statement's base line twice: once itself and
			// once more in BaseRule.AddAdvice.
			position := base.ConvertANTLRLineToPosition(stmt.BaseLine() + int(findingLine(stmt.Text, f)))
			position.Line += int32(stmt.BaseLine())
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          code.CreateTableTrigger.Int32(),
				Title:         title,
				Content:       fmt.Sprintf("Trigger is forbidden, but \"%s\" creates", strings.TrimSpace(stmt.Text)),
				StartPosition: position,
			})
		}
	}
	return adviceList, nil
}
