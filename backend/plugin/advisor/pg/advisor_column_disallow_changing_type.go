package pg

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/omni/pg/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*ColumnDisallowChangingTypeAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_COLUMN_DISALLOW_CHANGE_TYPE, &ColumnDisallowChangingTypeAdvisor{})
}

// ColumnDisallowChangingTypeAdvisor is the advisor checking for disallow changing column type.
type ColumnDisallowChangingTypeAdvisor struct {
}

// Check checks for disallow changing column type.
func (*ColumnDisallowChangingTypeAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := &columnDisallowChangingTypeRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
	}

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type columnDisallowChangingTypeRule struct {
	OmniBaseRule
}

func (*columnDisallowChangingTypeRule) Name() string {
	return "column-disallow-changing-type"
}

func (r *columnDisallowChangingTypeRule) OnStatement(node ast.Node) {
	n, ok := node.(*ast.AlterTableStmt)
	if !ok {
		return
	}

	for _, cmd := range omniAlterTableCmds(n) {
		if ast.AlterTableType(cmd.Subtype) == ast.AT_AlterColumnType {
			stmtText := strings.TrimSpace(r.StmtText)
			// Remove trailing semicolons for cleaner display
			stmtText = strings.TrimRight(stmtText, ";")
			stmtText = strings.TrimSpace(stmtText)

			r.AddAdvice(&storepb.Advice{
				Status:  r.Level,
				Code:    code.ChangeColumnType.Int32(),
				Title:   r.Title,
				Content: fmt.Sprintf("The statement \"%s\" changes column type", stmtText),
				StartPosition: &storepb.Position{
					Line:   r.ContentStartLine(),
					Column: 0,
				},
			})
			return // Only report once per ALTER TABLE statement
		}
	}
}
