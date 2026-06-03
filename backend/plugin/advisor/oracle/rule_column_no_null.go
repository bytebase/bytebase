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
	_ advisor.Advisor = (*ColumnNoNullAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_ORACLE, storepb.SQLReviewRule_COLUMN_NO_NULL, &ColumnNoNullAdvisor{})
}

type columnMap map[string]int

// ColumnNoNullAdvisor is the advisor checking for column no NULL value.
type ColumnNoNullAdvisor struct {
}

// Check checks for column no NULL value.
func (*ColumnNoNullAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := NewColumnNoNullRule(level, checkCtx.Rule.Type.String(), checkCtx.CurrentDatabase)

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule})
}

// ColumnNoNullRule is the rule implementation for column no NULL value.
type ColumnNoNullRule struct {
	BaseRule

	currentDatabase string
	nullableColumns columnMap
}

// NewColumnNoNullRule creates a new ColumnNoNullRule.
func NewColumnNoNullRule(level storepb.Advice_Status, title string, currentDatabase string) *ColumnNoNullRule {
	return &ColumnNoNullRule{
		BaseRule:        NewBaseRule(level, title, 0),
		currentDatabase: currentDatabase,
		nullableColumns: make(columnMap),
	}
}

// Name returns the rule name.
func (*ColumnNoNullRule) Name() string {
	return "column.no-null"
}

// OnStatement records nullable columns from omni CREATE/ALTER TABLE nodes.
func (r *ColumnNoNullRule) OnStatement(node ast.Node) {
	switch n := node.(type) {
	case *ast.CreateTableStmt:
		tableName := omniObjectName(n.Name, r.currentDatabase)
		for _, col := range omniColumnDefs(n.Columns) {
			r.recordNullableColumn(tableName, col)
		}
		for _, c := range omniTableConstraints(n.Constraints) {
			if c.Type == ast.CONSTRAINT_PRIMARY {
				for _, columnName := range omniListStrings(c.Columns) {
					delete(r.nullableColumns, fmt.Sprintf("%s.%s", tableName, columnName))
				}
			}
		}
	case *ast.AlterTableStmt:
		tableName := omniObjectName(n.Name, r.currentDatabase)
		for _, cmd := range omniAlterTableCmds(n) {
			if cmd.Action != ast.AT_MODIFY_COLUMN && cmd.Action != ast.AT_ADD_COLUMN {
				continue
			}
			for _, col := range append(omniColumnDefs(cmd.ColumnDefs), cmd.ColumnDef) {
				if col != nil {
					r.recordNullableColumn(tableName, col)
				}
			}
		}
	default:
	}
}

func (r *ColumnNoNullRule) recordNullableColumn(tableName string, col *ast.ColumnDef) {
	if col == nil {
		return
	}
	columnID := fmt.Sprintf("%s.%s", tableName, col.Name)
	if col.NotNull || omniColumnHasConstraint(col, ast.CONSTRAINT_NOT_NULL) || omniColumnHasConstraint(col, ast.CONSTRAINT_PRIMARY) {
		delete(r.nullableColumns, columnID)
		return
	}
	r.nullableColumns[columnID] = r.locLine(col.Loc)
}

// OnEnter is called when the parser enters a rule context.

// Ignore other node types

// OnExit is called when the parser exits a rule context.

// Ignore other node types

// GetAdviceList returns the advice list.
func (r *ColumnNoNullRule) GetAdviceList() ([]*storepb.Advice, error) {
	var columnIDs []string
	for columnID := range r.nullableColumns {
		columnIDs = append(columnIDs, columnID)
	}
	slices.Sort(columnIDs)
	for _, columnID := range columnIDs {
		line := r.nullableColumns[columnID]
		r.AddAdvice(
			r.level,
			code.ColumnCannotNull.Int32(),
			fmt.Sprintf("Column %q is nullable, which is not allowed.", lastIdentifier(columnID)),
			common.ConvertANTLRLineToPosition(line),
		)
	}
	return r.BaseRule.GetAdviceList()
}
