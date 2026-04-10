package mssql

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/omni/mssql/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*ColumnNoNullAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MSSQL, storepb.SQLReviewRule_COLUMN_NO_NULL, &ColumnNoNullAdvisor{})
}

// ColumnNoNullAdvisor is the advisor checking for column no NULL value..
type ColumnNoNullAdvisor struct {
}

// Check checks for column no NULL value..
func (*ColumnNoNullAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := &columnNoNullOmniRule{
		OmniBaseRule: OmniBaseRule{Level: level, Title: checkCtx.Rule.Type.String()},
	}
	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type columnNoNullOmniRule struct {
	OmniBaseRule
}

func (*columnNoNullOmniRule) Name() string {
	return "ColumnNoNullOmniRule"
}

func (r *columnNoNullOmniRule) OnStatement(node ast.Node) {
	switch n := node.(type) {
	case *ast.CreateTableStmt:
		r.handleCreateTable(n)
	case *ast.AlterTableStmt:
		r.handleAlterTable(n)
	default:
	}
}

func (r *columnNoNullOmniRule) handleCreateTable(n *ast.CreateTableStmt) {
	if n.Name == nil {
		return
	}

	// Collect columns that are part of table-level PK constraints.
	pkColumns := make(map[string]bool)
	if n.Constraints != nil {
		for _, item := range n.Constraints.Items {
			cd, ok := item.(*ast.ConstraintDef)
			if !ok {
				continue
			}
			if cd.Type == ast.ConstraintPrimaryKey && cd.Columns != nil {
				for _, colItem := range cd.Columns.Items {
					if ic, ok := colItem.(*ast.IndexColumn); ok {
						pkColumns[strings.ToLower(ic.Name)] = true
					}
				}
			}
		}
	}

	if n.Columns == nil {
		return
	}
	for _, item := range n.Columns.Items {
		col, ok := item.(*ast.ColumnDef)
		if !ok {
			continue
		}
		r.checkColumnDef(col, pkColumns)
	}
}

func (r *columnNoNullOmniRule) handleAlterTable(n *ast.AlterTableStmt) {
	if n.Actions == nil {
		return
	}
	for _, item := range n.Actions.Items {
		action, ok := item.(*ast.AlterTableAction)
		if !ok {
			continue
		}
		if action.Type != ast.ATAddColumn && action.Type != ast.ATAlterColumn {
			continue
		}
		if action.Column != nil {
			r.checkColumnDef(action.Column, nil)
		}
	}
}

func (r *columnNoNullOmniRule) checkColumnDef(col *ast.ColumnDef, pkColumns map[string]bool) {
	// If column is in table-level PK, it's implicitly NOT NULL.
	if pkColumns != nil && pkColumns[strings.ToLower(col.Name)] {
		return
	}

	// Check column-level constraints for PK or NOT NULL.
	if col.Constraints != nil {
		for _, cItem := range col.Constraints.Items {
			cd, ok := cItem.(*ast.ConstraintDef)
			if !ok {
				continue
			}
			if cd.Type == ast.ConstraintPrimaryKey || cd.Type == ast.ConstraintNotNull {
				return
			}
		}
	}

	// Check the Nullable spec.
	if col.Nullable != nil && col.Nullable.NotNull {
		return
	}

	// Column is nullable.
	r.AddAdvice(&storepb.Advice{
		Status:        r.Level,
		Code:          code.ColumnCannotNull.Int32(),
		Title:         r.Title,
		Content:       fmt.Sprintf("Column [%s] is nullable, which is not allowed.", strings.ToLower(col.Name)),
		StartPosition: &storepb.Position{Line: r.LocToLine(col.Loc)},
	})
}
