// Package snowflake is the advisor for snowflake database.
package snowflake

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/advisor/code"

	omniast "github.com/bytebase/omni/snowflake/ast"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	snowsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/snowflake"
)

var (
	_ advisor.Advisor = (*ColumnRequireAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_SNOWFLAKE, storepb.SQLReviewRule_COLUMN_REQUIRED, &ColumnRequireAdvisor{})
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

	requireColumns := make(map[string]any)
	for _, column := range stringArrayPayload.List {
		requireColumns[column] = true
	}

	rule := NewColumnRequireRule(level, checkCtx.Rule.Type.String(), requireColumns)

	for _, stmt := range checkCtx.ParsedStatements {
		if stmt.AST == nil {
			continue
		}
		node, ok := snowsqlparser.GetOmniNode(stmt.AST)
		if !ok {
			continue
		}
		rule.SetBaseLine(stmt.BaseLine())
		rule.checkStatement(node, stmt.Text)
	}

	return rule.GetAdviceList(), nil
}

// ColumnRequireRule checks for required columns.
type ColumnRequireRule struct {
	BaseRule
	// requireColumns is the required columns, the key is the normalized column name.
	requireColumns map[string]any
	// stmtText is the SQL text of the statement currently being checked; node
	// Loc offsets are relative to it.
	stmtText string
}

// NewColumnRequireRule creates a new ColumnRequireRule.
func NewColumnRequireRule(level storepb.Advice_Status, title string, requireColumns map[string]any) *ColumnRequireRule {
	return &ColumnRequireRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
		requireColumns: requireColumns,
	}
}

// Name returns the rule name.
func (*ColumnRequireRule) Name() string {
	return "ColumnRequireRule"
}

// checkStatement checks one statement's omni AST node.
func (r *ColumnRequireRule) checkStatement(node omniast.Node, stmtText string) {
	r.stmtText = stmtText
	switch n := node.(type) {
	case *omniast.CreateTableStmt:
		r.checkCreateTable(n)
	case *omniast.AlterTableStmt:
		r.checkAlterTable(n)
	default:
	}
}

func (r *ColumnRequireRule) checkCreateTable(stmt *omniast.CreateTableStmt) {
	// The legacy advisor only fired on plain CREATE TABLE (the Create_table
	// context); CREATE TABLE ... AS SELECT / LIKE / CLONE parsed as different
	// contexts and were never checked.
	if stmt.AsSelect != nil || stmt.Like != nil || stmt.Clone != nil {
		return
	}

	missingColumns := make(map[string]any)
	for column := range r.requireColumns {
		missingColumns[column] = true
	}
	for _, column := range stmt.Columns {
		delete(missingColumns, column.Name.Normalize())
	}
	if len(missingColumns) == 0 {
		return
	}

	// The legacy advisor reported the stop line of the column declaration list,
	// i.e. the line where its last item (column or out-of-line constraint) ends.
	line := r.line(stmt.Loc.Start)
	lastItemEnd := -1
	for _, column := range stmt.Columns {
		if column.Loc.End > lastItemEnd {
			lastItemEnd = column.Loc.End
		}
	}
	for _, constraint := range stmt.Constraints {
		if constraint.Loc.End > lastItemEnd {
			lastItemEnd = constraint.Loc.End
		}
	}
	if lastItemEnd > 0 {
		line = r.line(lastItemEnd - 1)
	}

	r.addMissingColumnAdvice(missingColumns, stmt.Name.String(), line)
}

func (r *ColumnRequireRule) checkAlterTable(stmt *omniast.AlterTableStmt) {
	missingColumns := make(map[string]any)
	actionLine := 0
	for _, action := range stmt.Actions {
		if action.Kind != omniast.AlterTableDropColumn {
			continue
		}
		if actionLine == 0 {
			actionLine = r.line(action.Loc.Start)
		}
		for _, column := range action.DropColumnNames {
			normalizedColumnName := column.Normalize()
			if _, ok := r.requireColumns[normalizedColumnName]; ok {
				missingColumns[normalizedColumnName] = true
			}
		}
	}
	if len(missingColumns) == 0 {
		return
	}

	r.addMissingColumnAdvice(missingColumns, stmt.Name.String(), actionLine)
}

func (r *ColumnRequireRule) addMissingColumnAdvice(missingColumns map[string]any, originalTableName string, line int) {
	columnNames := make([]string, 0, len(missingColumns))
	for column := range missingColumns {
		columnNames = append(columnNames, column)
	}
	slices.Sort(columnNames)

	for _, column := range columnNames {
		r.AddAdvice(&storepb.Advice{
			Status:        r.level,
			Code:          code.NoRequiredColumn.Int32(),
			Title:         r.title,
			Content:       fmt.Sprintf("Table %s missing required column %q", originalTableName, column),
			StartPosition: common.ConvertANTLRLineToPosition(r.baseLine + line),
		})
	}
}

// line converts a byte offset within the current statement text to a 1-based
// line number, mirroring the ANTLR token line the legacy advisor reported.
func (r *ColumnRequireRule) line(offset int) int {
	if offset < 0 {
		return 1
	}
	if offset > len(r.stmtText) {
		offset = len(r.stmtText)
	}
	return 1 + strings.Count(r.stmtText[:offset], "\n")
}
