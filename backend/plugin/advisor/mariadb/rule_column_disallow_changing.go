package mariadb

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/mariadb/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

var (
	_ advisor.Advisor = (*ColumnDisallowChangingAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_COLUMN_DISALLOW_CHANGE, &ColumnDisallowChangingAdvisor{})
}

// ColumnDisallowChangingAdvisor is the advisor checking for disallow CHANGE COLUMN statement.
type ColumnDisallowChangingAdvisor struct {
}

// Check checks for disallow CHANGE COLUMN statement.
func (*ColumnDisallowChangingAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := &columnDisallowChangingRule{
		BaseRule: BaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
	}

	return RunRules(checkCtx.ParsedStatements, []Rule{rule}), nil
}

type columnDisallowChangingRule struct {
	BaseRule
}

func (*columnDisallowChangingRule) Name() string {
	return "ColumnDisallowChangingRule"
}

func (r *columnDisallowChangingRule) OnStatement(node ast.Node) {
	n, ok := node.(*ast.AlterTableStmt)
	if !ok {
		return
	}

	for _, cmd := range n.Commands {
		if cmd == nil {
			continue
		}
		if cmd.Type == ast.ATChangeColumn {
			r.AddAdvice(&storepb.Advice{
				Status:        r.Level,
				Code:          code.UseChangeColumnStatement.Int32(),
				Title:         r.Title,
				Content:       fmt.Sprintf("\"%s\" contains CHANGE COLUMN statement", r.QueryText()),
				StartPosition: base.ConvertANTLRLineToPosition(int(r.ContentStartLine())),
			})
		}
	}
}
