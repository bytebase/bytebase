package mysql

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/mysql/ast"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*ColumnDisallowChangingOrderAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_COLUMN_DISALLOW_CHANGING_ORDER, &ColumnDisallowChangingOrderAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_COLUMN_DISALLOW_CHANGING_ORDER, &ColumnDisallowChangingOrderAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_COLUMN_DISALLOW_CHANGING_ORDER, &ColumnDisallowChangingOrderAdvisor{})
}

// ColumnDisallowChangingOrderAdvisor is the advisor checking for disallow changing column order.
type ColumnDisallowChangingOrderAdvisor struct {
}

// Check checks for disallow changing column order.
func (*ColumnDisallowChangingOrderAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := &columnDisallowChangingOrderOmniRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
	}

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type columnDisallowChangingOrderOmniRule struct {
	OmniBaseRule
}

func (*columnDisallowChangingOrderOmniRule) Name() string {
	return "ColumnDisallowChangingOrderRule"
}

func (r *columnDisallowChangingOrderOmniRule) OnStatement(node ast.Node) {
	n, ok := node.(*ast.AlterTableStmt)
	if !ok {
		return
	}

	for _, cmd := range n.Commands {
		if cmd == nil {
			continue
		}
		switch cmd.Type {
		case ast.ATModifyColumn, ast.ATChangeColumn:
			// Check if FIRST or AFTER is specified (column reordering).
			if cmd.First || cmd.After != "" {
				r.AddAdvice(&storepb.Advice{
					Status:        r.Level,
					Code:          code.ChangeColumnOrder.Int32(),
					Title:         r.Title,
					Content:       fmt.Sprintf("\"%s\" changes column order", r.QueryText()),
					StartPosition: common.ConvertANTLRLineToPosition(int(r.ContentStartLine())),
				})
			}
		default:
		}
	}
}
