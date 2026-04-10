package mssql

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/mssql/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*TableNoForeignKeyAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MSSQL, storepb.SQLReviewRule_TABLE_NO_FOREIGN_KEY, &TableNoForeignKeyAdvisor{})
}

// TableNoForeignKeyAdvisor is the advisor checking for table disallow foreign key..
type TableNoForeignKeyAdvisor struct {
}

// Check checks for table disallow foreign key..
func (*TableNoForeignKeyAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := &tableNoForeignKeyOmniRule{
		OmniBaseRule:  OmniBaseRule{Level: level, Title: checkCtx.Rule.Type.String()},
		tableHasFK:    make(map[string]bool),
		tableOriginal: make(map[string]string),
		tableLine:     make(map[string]int32),
		tableBaseLine: make(map[string]int),
	}

	advice := RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule})
	advice = append(advice, rule.generateFinalAdvice()...)
	return advice, nil
}

type tableNoForeignKeyOmniRule struct {
	OmniBaseRule

	tableHasFK    map[string]bool
	tableOriginal map[string]string
	tableLine     map[string]int32
	tableBaseLine map[string]int
}

func (*tableNoForeignKeyOmniRule) Name() string {
	return "TableNoForeignKeyOmniRule"
}

func (r *tableNoForeignKeyOmniRule) OnStatement(node ast.Node) {
	switch n := node.(type) {
	case *ast.CreateTableStmt:
		r.handleCreateTable(n)
	case *ast.AlterTableStmt:
		r.handleAlterTable(n)
	default:
	}
}

func (r *tableNoForeignKeyOmniRule) handleCreateTable(n *ast.CreateTableStmt) {
	if n.Name == nil {
		return
	}
	norm := normalizeTableRef(n.Name, "", "dbo")
	original := tableRefText(n.Name)

	r.tableHasFK[norm] = false
	r.tableOriginal[norm] = original
	r.tableLine[norm] = r.LocToLine(n.Name.Loc)
	r.tableBaseLine[norm] = r.BaseLine

	// Check column-level FK constraints.
	if n.Columns != nil {
		for _, item := range n.Columns.Items {
			col, ok := item.(*ast.ColumnDef)
			if !ok {
				continue
			}
			if col.Constraints != nil {
				for _, cItem := range col.Constraints.Items {
					cd, ok := cItem.(*ast.ConstraintDef)
					if !ok {
						continue
					}
					if cd.Type == ast.ConstraintForeignKey {
						r.tableHasFK[norm] = true
						r.tableLine[norm] = r.LocToLine(cd.Loc)
						return
					}
				}
			}
		}
	}

	// Check table-level constraints.
	if n.Constraints != nil {
		for _, item := range n.Constraints.Items {
			cd, ok := item.(*ast.ConstraintDef)
			if !ok {
				continue
			}
			if cd.Type == ast.ConstraintForeignKey {
				r.tableHasFK[norm] = true
				r.tableLine[norm] = r.LocToLine(cd.Loc)
				return
			}
		}
	}
}

func (r *tableNoForeignKeyOmniRule) handleAlterTable(n *ast.AlterTableStmt) {
	if n.Name == nil || n.Actions == nil {
		return
	}
	norm := normalizeTableRef(n.Name, "", "dbo")

	for _, item := range n.Actions.Items {
		action, ok := item.(*ast.AlterTableAction)
		if !ok {
			continue
		}
		if action.Type == ast.ATAddConstraint && action.Constraint != nil && action.Constraint.Type == ast.ConstraintForeignKey {
			r.tableHasFK[norm] = true
			if _, exists := r.tableOriginal[norm]; !exists {
				r.tableOriginal[norm] = tableRefText(n.Name)
			}
			r.tableLine[norm] = r.LocToLine(action.Constraint.Loc)
			r.tableBaseLine[norm] = r.BaseLine
		}
	}
}

func (r *tableNoForeignKeyOmniRule) generateFinalAdvice() []*storepb.Advice {
	var result []*storepb.Advice
	for norm, hasFK := range r.tableHasFK {
		if hasFK {
			line := r.tableLine[norm] + int32(r.tableBaseLine[norm])
			result = append(result, &storepb.Advice{
				Status:        r.Level,
				Code:          code.TableHasFK.Int32(),
				Title:         r.Title,
				Content:       fmt.Sprintf("FOREIGN KEY is not allowed in the table %s.", r.tableOriginal[norm]),
				StartPosition: &storepb.Position{Line: line},
			})
		}
	}
	return result
}
