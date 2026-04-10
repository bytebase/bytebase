package mssql

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/omni/mssql/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*ColumnRequireAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MSSQL, storepb.SQLReviewRule_COLUMN_REQUIRED, &ColumnRequireAdvisor{})
}

// ColumnRequireAdvisor is the advisor checking for column requirement..
type ColumnRequireAdvisor struct {
}

// Check checks for column requirement..
func (*ColumnRequireAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	stringArrayPayload := checkCtx.Rule.GetStringArrayPayload()
	if stringArrayPayload == nil {
		return nil, errors.New("string_array_payload is required for column required rule")
	}

	rule := &columnRequireOmniRule{
		OmniBaseRule:   OmniBaseRule{Level: level, Title: checkCtx.Rule.Type.String()},
		requireColumns: make(map[string]any),
	}
	for _, col := range stringArrayPayload.List {
		rule.requireColumns[col] = true
	}

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type columnRequireOmniRule struct {
	OmniBaseRule

	requireColumns map[string]any
}

func (*columnRequireOmniRule) Name() string {
	return "ColumnRequireOmniRule"
}

func (r *columnRequireOmniRule) OnStatement(node ast.Node) {
	switch n := node.(type) {
	case *ast.CreateTableStmt:
		r.handleCreateTable(n)
	case *ast.AlterTableStmt:
		r.handleAlterTable(n)
	default:
	}
}

func (r *columnRequireOmniRule) handleCreateTable(n *ast.CreateTableStmt) {
	if n.Name == nil {
		return
	}

	// Build set of columns present.
	present := make(map[string]bool)
	if n.Columns != nil {
		for _, item := range n.Columns.Items {
			col, ok := item.(*ast.ColumnDef)
			if !ok {
				continue
			}
			present[strings.ToLower(col.Name)] = true
		}
	}

	// Find missing required columns.
	var missing []string
	for col := range r.requireColumns {
		if !present[strings.ToLower(col)] {
			missing = append(missing, col)
		}
	}
	slices.Sort(missing)

	tableName := tableRefText(n.Name)
	for _, col := range missing {
		r.AddAdvice(&storepb.Advice{
			Status:        r.Level,
			Code:          code.NoRequiredColumn.Int32(),
			Title:         r.Title,
			Content:       fmt.Sprintf("Table %s missing required column \"%s\"", tableName, col),
			StartPosition: &storepb.Position{Line: r.LocToLine(n.Loc)},
		})
	}
}

func (r *columnRequireOmniRule) handleAlterTable(n *ast.AlterTableStmt) {
	if n.Name == nil || n.Actions == nil {
		return
	}

	tableName := tableRefText(n.Name)
	for _, item := range n.Actions.Items {
		action, ok := item.(*ast.AlterTableAction)
		if !ok {
			continue
		}
		if action.Type == ast.ATDropColumn {
			colName := strings.ToLower(action.ColName)
			if _, ok := r.requireColumns[colName]; ok {
				r.AddAdvice(&storepb.Advice{
					Status:        r.Level,
					Code:          code.NoRequiredColumn.Int32(),
					Title:         r.Title,
					Content:       fmt.Sprintf("Table %s missing required column \"%s\"", tableName, colName),
					StartPosition: &storepb.Position{Line: r.LocToLine(n.Loc)},
				})
			}
		}
	}
}
