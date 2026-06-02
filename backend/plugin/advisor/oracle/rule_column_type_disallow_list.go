// Package oracle is the advisor for oracle database.
package oracle

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/omni/oracle/ast"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*ColumnTypeDisallowListAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_ORACLE, storepb.SQLReviewRule_COLUMN_TYPE_DISALLOW_LIST, &ColumnTypeDisallowListAdvisor{})
}

// ColumnTypeDisallowListAdvisor is the advisor checking for column type disallow list.
type ColumnTypeDisallowListAdvisor struct {
}

// Check checks for column type disallow list.
func (*ColumnTypeDisallowListAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	stringArrayPayload := checkCtx.Rule.GetStringArrayPayload()

	rule := NewColumnTypeDisallowListRule(level, checkCtx.Rule.Type.String(), checkCtx.CurrentDatabase, stringArrayPayload.List)

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule})
}

// ColumnTypeDisallowListRule is the rule implementation for column type disallow list.
type ColumnTypeDisallowListRule struct {
	BaseRule

	currentDatabase string
	disallowList    []string
}

// NewColumnTypeDisallowListRule creates a new ColumnTypeDisallowListRule.
func NewColumnTypeDisallowListRule(level storepb.Advice_Status, title string, currentDatabase string, disallowList []string) *ColumnTypeDisallowListRule {
	return &ColumnTypeDisallowListRule{
		BaseRule:        NewBaseRule(level, title, 0),
		currentDatabase: currentDatabase,
		disallowList:    disallowList,
	}
}

// Name returns the rule name.
func (*ColumnTypeDisallowListRule) Name() string {
	return "column.type-disallow-list"
}

// OnStatement checks column data types in the omni AST.
func (r *ColumnTypeDisallowListRule) OnStatement(node ast.Node) {
	omniWalk(node, func(n ast.Node) {
		col, ok := n.(*ast.ColumnDef)
		if !ok || col.TypeName == nil {
			return
		}
		typeName := omniTypeName(col.TypeName)
		for _, disallowType := range r.disallowList {
			if strings.EqualFold(typeName, disallowType) {
				r.AddAdvice(
					r.level,
					code.DisabledColumnType.Int32(),
					fmt.Sprintf("Disallow column type %s but column \"%s\" is", typeName, col.Name),
					common.ConvertANTLRLineToPosition(r.locLine(col.TypeName.Loc)),
				)
				return
			}
		}
	})
}

// OnEnter is called when the parser enters a rule context.

// Ignore other node types

// OnExit is called when the parser exits a rule context.
