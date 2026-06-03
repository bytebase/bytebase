// Package oracle is the advisor for oracle database.
package oracle

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/oracle/ast"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*ColumnAddNotNullColumnRequireDefaultAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_ORACLE, storepb.SQLReviewRule_COLUMN_ADD_NOT_NULL_REQUIRE_DEFAULT, &ColumnAddNotNullColumnRequireDefaultAdvisor{})
}

// ColumnAddNotNullColumnRequireDefaultAdvisor is the advisor checking for adding not null column requires default.
type ColumnAddNotNullColumnRequireDefaultAdvisor struct {
}

// Check checks for adding not null column requires default.
func (*ColumnAddNotNullColumnRequireDefaultAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := NewColumnAddNotNullColumnRequireDefaultRule(level, checkCtx.Rule.Type.String(), checkCtx.CurrentDatabase)

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule})
}

// ColumnAddNotNullColumnRequireDefaultRule is the rule implementation for adding not null column requires default.
type ColumnAddNotNullColumnRequireDefaultRule struct {
	BaseRule

	currentDatabase string
}

// NewColumnAddNotNullColumnRequireDefaultRule creates a new ColumnAddNotNullColumnRequireDefaultRule.
func NewColumnAddNotNullColumnRequireDefaultRule(level storepb.Advice_Status, title string, currentDatabase string) *ColumnAddNotNullColumnRequireDefaultRule {
	return &ColumnAddNotNullColumnRequireDefaultRule{
		BaseRule:        NewBaseRule(level, title, 0),
		currentDatabase: currentDatabase,
	}
}

// Name returns the rule name.
func (*ColumnAddNotNullColumnRequireDefaultRule) Name() string {
	return "column.add-not-null-column-require-default"
}

// OnStatement checks ADD COLUMN actions for NOT NULL columns without DEFAULT.
func (r *ColumnAddNotNullColumnRequireDefaultRule) OnStatement(node ast.Node) {
	stmt, ok := node.(*ast.AlterTableStmt)
	if !ok {
		return
	}
	for _, cmd := range omniAlterTableCmds(stmt) {
		if cmd.Action != ast.AT_ADD_COLUMN {
			continue
		}
		for _, col := range append(omniColumnDefs(cmd.ColumnDefs), cmd.ColumnDef) {
			if col == nil || col.Default != nil {
				continue
			}
			if col.NotNull || omniColumnHasConstraint(col, ast.CONSTRAINT_NOT_NULL) {
				r.AddAdvice(
					r.level,
					code.NotNullColumnWithNoDefault.Int32(),
					fmt.Sprintf("Adding not null column %q requires default.", col.Name),
					common.ConvertANTLRLineToPosition(r.locLine(col.Loc)),
				)
			}
		}
	}
}
