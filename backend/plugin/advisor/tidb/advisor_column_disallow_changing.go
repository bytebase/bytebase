package tidb

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/tidb/ast"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*ColumnDisallowChangingAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_COLUMN_DISALLOW_CHANGE, &ColumnDisallowChangingAdvisor{})
}

// ColumnDisallowChangingAdvisor is the advisor checking for disallow CHANGE COLUMN statement.
type ColumnDisallowChangingAdvisor struct {
}

// Check checks for disallow CHANGE COLUMN statement.
func (*ColumnDisallowChangingAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmts, err := getTiDBOmniNodes(checkCtx)
	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	title := checkCtx.Rule.Type.String()
	var adviceList []*storepb.Advice
	for _, ostmt := range stmts {
		alter, ok := ostmt.Node.(*ast.AlterTableStmt)
		if !ok {
			continue
		}
		// Single-advice-per-statement contract: pingcap-tidb's Visitor
		// broke after the first match; preserve that here. (Mysql analog
		// emits per-cmd without breaking — cardinality divergence.)
		for _, cmd := range alter.Commands {
			if cmd == nil {
				continue
			}
			if cmd.Type == ast.ATChangeColumn {
				adviceList = append(adviceList, &storepb.Advice{
					Status:        level,
					Code:          code.UseChangeColumnStatement.Int32(),
					Title:         title,
					Content:       fmt.Sprintf("\"%s\" contains CHANGE COLUMN statement", ostmt.TrimmedText()),
					StartPosition: common.ConvertANTLRLineToPosition(ostmt.AbsoluteLine(alter.Loc.Start)),
				})
				break
			}
		}
	}

	return adviceList, nil
}
