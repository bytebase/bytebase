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
	_ advisor.Advisor = (*ColumnDisallowDropAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_COLUMN_DISALLOW_DROP, &ColumnDisallowDropAdvisor{})
}

// ColumnDisallowDropAdvisor is the advisor checking for disallow DROP COLUMN statement.
type ColumnDisallowDropAdvisor struct {
}

// Check checks for disallow DROP COLUMN statement.
func (*ColumnDisallowDropAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := &columnDisallowDropOmniRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
	}

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type columnDisallowDropOmniRule struct {
	OmniBaseRule
}

func (*columnDisallowDropOmniRule) Name() string {
	return "ColumnDisallowDropRule"
}

func (r *columnDisallowDropOmniRule) OnStatement(node ast.Node) {
	n, ok := node.(*ast.AlterTableStmt)
	if !ok || n.Table == nil {
		return
	}

	tableName := n.Table.Name
	for _, cmd := range n.Commands {
		if cmd == nil || cmd.Type != ast.ATDropColumn {
			continue
		}
		r.AddAdvice(&storepb.Advice{
			Status:        r.Level,
			Code:          code.DropColumn.Int32(),
			Title:         r.Title,
			Content:       fmt.Sprintf("drops column \"%s\" of table \"%s\"", cmd.Name, tableName),
			StartPosition: common.ConvertANTLRLineToPosition(int(r.LocToLine(cmd.Loc))),
		})
	}
}
