// Package oracle is the advisor for oracle database.
package oracle

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/bytebase/omni/oracle/ast"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*ColumnRequireAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_ORACLE, storepb.SQLReviewRule_COLUMN_REQUIRED, &ColumnRequireAdvisor{})
}

// ColumnRequireAdvisor is the advisor checking for column requirement.
type ColumnRequireAdvisor struct {
}

// Check checks for column requirement.
func (*ColumnRequireAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	stringArrayPayload := checkCtx.Rule.GetStringArrayPayload()
	if stringArrayPayload == nil {
		return nil, errors.New("string_array_payload is required for column required rule")
	}

	rule := NewColumnRequireRule(level, checkCtx.Rule.Type.String(), checkCtx.CurrentDatabase, stringArrayPayload.List)

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule})
}

type columnSet map[string]bool

// ColumnRequireRule is the rule implementation for column requirement.
type ColumnRequireRule struct {
	BaseRule

	currentDatabase string
	requiredColumns columnSet
}

// NewColumnRequireRule creates a new ColumnRequireRule.
func NewColumnRequireRule(level storepb.Advice_Status, title string, currentDatabase string, columnList []string) *ColumnRequireRule {
	requiredColumns := make(columnSet)
	for _, column := range columnList {
		requiredColumns[column] = true
	}
	return &ColumnRequireRule{
		BaseRule:        NewBaseRule(level, title, 0),
		currentDatabase: currentDatabase,
		requiredColumns: requiredColumns,
	}
}

// Name returns the rule name.
func (*ColumnRequireRule) Name() string {
	return "column.require"
}

// OnStatement checks required columns in CREATE TABLE and ALTER TABLE.
func (r *ColumnRequireRule) OnStatement(node ast.Node) {
	switch n := node.(type) {
	case *ast.CreateTableStmt:
		missing := make(columnSet)
		for column := range r.requiredColumns {
			missing[column] = true
		}
		for _, col := range omniColumnDefs(n.Columns) {
			delete(missing, col.Name)
		}
		r.addMissingColumnsAdvice(omniLastObjectName(n.Name), missing, r.locLine(n.Loc))
	case *ast.AlterTableStmt:
		missing := make(columnSet)
		for _, cmd := range omniAlterTableCmds(n) {
			switch cmd.Action {
			case ast.AT_DROP_COLUMN:
				if _, ok := r.requiredColumns[cmd.ColumnName]; ok {
					missing[cmd.ColumnName] = true
				}
			case ast.AT_RENAME_COLUMN:
				if cmd.ColumnName != cmd.NewName {
					if _, ok := r.requiredColumns[cmd.ColumnName]; ok {
						missing[cmd.ColumnName] = true
					}
				}
			default:
			}
		}
		r.addMissingColumnsAdvice(omniLastObjectName(n.Name), missing, r.locLine(n.Loc))
	default:
	}
}

func (r *ColumnRequireRule) addMissingColumnsAdvice(tableName string, missing columnSet, line int) {
	if len(missing) == 0 {
		return
	}
	missingColumns := []string{}
	for column := range missing {
		missingColumns = append(missingColumns, fmt.Sprintf("%q", column))
	}
	slices.Sort(missingColumns)
	r.AddAdvice(
		r.level,
		code.NoRequiredColumn.Int32(),
		fmt.Sprintf("Table %q requires columns: %s", tableName, strings.Join(missingColumns, ", ")),
		common.ConvertANTLRLineToPosition(line),
	)
}

// OnEnter is called when the parser enters a rule context.

// Ignore other node types

// OnExit is called when the parser exits a rule context.

// Ignore other node types
