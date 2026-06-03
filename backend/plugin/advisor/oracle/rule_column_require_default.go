// Package oracle is the advisor for oracle database.
package oracle

import (
	"context"
	"fmt"
	"slices"

	"github.com/bytebase/omni/oracle/ast"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*ColumnRequireDefaultAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_ORACLE, storepb.SQLReviewRule_COLUMN_REQUIRE_DEFAULT, &ColumnRequireDefaultAdvisor{})
}

// ColumnRequireDefaultAdvisor is the advisor checking for column default requirement.
type ColumnRequireDefaultAdvisor struct {
}

// Check checks for column default requirement.
func (*ColumnRequireDefaultAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := NewColumnRequireDefaultRule(level, checkCtx.Rule.Type.String(), checkCtx.CurrentDatabase)

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule})
}

// ColumnRequireDefaultRule is the rule implementation for column default requirement.
type ColumnRequireDefaultRule struct {
	BaseRule

	currentDatabase  string
	noDefaultColumns columnMap
}

// NewColumnRequireDefaultRule creates a new ColumnRequireDefaultRule.
func NewColumnRequireDefaultRule(level storepb.Advice_Status, title string, currentDatabase string) *ColumnRequireDefaultRule {
	return &ColumnRequireDefaultRule{
		BaseRule:         NewBaseRule(level, title, 0),
		currentDatabase:  currentDatabase,
		noDefaultColumns: make(columnMap),
	}
}

// Name returns the rule name.
func (*ColumnRequireDefaultRule) Name() string {
	return "column.require-default"
}

// OnStatement records columns without defaults from omni CREATE/ALTER TABLE nodes.
func (r *ColumnRequireDefaultRule) OnStatement(node ast.Node) {
	switch n := node.(type) {
	case *ast.CreateTableStmt:
		tableName := omniObjectName(n.Name, r.currentDatabase)
		for _, col := range omniColumnDefs(n.Columns) {
			r.recordNoDefaultColumn(tableName, col)
		}
	case *ast.AlterTableStmt:
		tableName := omniObjectName(n.Name, r.currentDatabase)
		for _, cmd := range omniAlterTableCmds(n) {
			for _, col := range append(omniColumnDefs(cmd.ColumnDefs), cmd.ColumnDef) {
				if col == nil {
					continue
				}
				columnID := fmt.Sprintf("%s.%s", tableName, col.Name)
				if col.Default != nil {
					delete(r.noDefaultColumns, columnID)
				} else if cmd.Action == ast.AT_ADD_COLUMN {
					r.noDefaultColumns[columnID] = r.locLine(col.Loc)
				}
			}
		}
	default:
	}
}

func (r *ColumnRequireDefaultRule) recordNoDefaultColumn(tableName string, col *ast.ColumnDef) {
	if col == nil {
		return
	}
	columnID := fmt.Sprintf("%s.%s", tableName, col.Name)
	if col.Default == nil {
		r.noDefaultColumns[columnID] = r.locLine(col.Loc)
	} else {
		delete(r.noDefaultColumns, columnID)
	}
}

// OnEnter is called when the parser enters a rule context.

// Ignore other node types

// OnExit is called when the parser exits a rule context.

// Ignore other node types

// GetAdviceList returns the advice list.
func (r *ColumnRequireDefaultRule) GetAdviceList() ([]*storepb.Advice, error) {
	var columnIDs []string
	for columnID := range r.noDefaultColumns {
		columnIDs = append(columnIDs, columnID)
	}
	slices.Sort(columnIDs)
	for _, columnID := range columnIDs {
		line := r.noDefaultColumns[columnID]
		r.AddAdvice(
			r.level,
			code.NoDefault.Int32(),
			fmt.Sprintf("Column %q doesn't have default value", lastIdentifier(columnID)),
			common.ConvertANTLRLineToPosition(line),
		)
	}
	return r.BaseRule.GetAdviceList()
}
