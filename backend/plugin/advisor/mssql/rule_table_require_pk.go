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
	_ advisor.Advisor = (*TableRequirePkAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MSSQL, storepb.SQLReviewRule_TABLE_REQUIRE_PK, &TableRequirePkAdvisor{})
}

// TableRequirePkAdvisor is the advisor checking for table require primary key..
type TableRequirePkAdvisor struct {
}

// Check checks for table require primary key..
func (*TableRequirePkAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := &tableRequirePkOmniRule{
		OmniBaseRule:  OmniBaseRule{Level: level, Title: checkCtx.Rule.Type.String()},
		tableHasPK:    make(map[string]bool),
		tableOriginal: make(map[string]string),
		tableLine:     make(map[string]int32),
		tableBaseLine: make(map[string]int),
	}

	advice := RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule})
	advice = append(advice, rule.generateFinalAdvice()...)
	return advice, nil
}

type tableRequirePkOmniRule struct {
	OmniBaseRule

	// tableHasPK tracks whether each table has a primary key.
	tableHasPK map[string]bool
	// tableOriginal maps normalized name to original text.
	tableOriginal map[string]string
	// tableLine maps normalized name to the line within the statement.
	tableLine map[string]int32
	// tableBaseLine maps normalized name to the BaseLine of the statement.
	tableBaseLine map[string]int
}

func (*tableRequirePkOmniRule) Name() string {
	return "TableRequirePkOmniRule"
}

func (r *tableRequirePkOmniRule) OnStatement(node ast.Node) {
	switch n := node.(type) {
	case *ast.CreateTableStmt:
		r.handleCreateTable(n)
	case *ast.AlterTableStmt:
		r.handleAlterTable(n)
	default:
	}
}

func (r *tableRequirePkOmniRule) handleCreateTable(n *ast.CreateTableStmt) {
	if n.Name == nil {
		return
	}
	norm := normalizeTableRef(n.Name, "", "dbo")
	original := tableRefText(n.Name)

	r.tableHasPK[norm] = false
	r.tableOriginal[norm] = original
	r.tableLine[norm] = r.LocToLine(n.Name.Loc)
	r.tableBaseLine[norm] = r.BaseLine

	// Check column-level PK constraints.
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
					if cd.Type == ast.ConstraintPrimaryKey {
						r.tableHasPK[norm] = true
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
			if cd.Type == ast.ConstraintPrimaryKey {
				r.tableHasPK[norm] = true
				return
			}
		}
	}
}

func (r *tableRequirePkOmniRule) handleAlterTable(n *ast.AlterTableStmt) {
	if n.Name == nil || n.Actions == nil {
		return
	}
	norm := normalizeTableRef(n.Name, "", "dbo")

	for _, item := range n.Actions.Items {
		action, ok := item.(*ast.AlterTableAction)
		if !ok {
			continue
		}
		switch action.Type {
		case ast.ATAddColumn:
			// Check inline PK on added column: ALTER TABLE t ADD id INT PRIMARY KEY.
			if action.Column != nil && action.Column.Constraints != nil {
				for _, cItem := range action.Column.Constraints.Items {
					if cd, ok := cItem.(*ast.ConstraintDef); ok && cd.Type == ast.ConstraintPrimaryKey {
						r.tableHasPK[norm] = true
					}
				}
			}
		case ast.ATAddConstraint:
			if action.Constraint != nil && action.Constraint.Type == ast.ConstraintPrimaryKey {
				r.tableHasPK[norm] = true
			}
		case ast.ATDropConstraint:
			// Conservatively mark as no PK when a constraint is dropped.
		default:
		}
	}
}

func (r *tableRequirePkOmniRule) generateFinalAdvice() []*storepb.Advice {
	var result []*storepb.Advice
	for norm, hasPK := range r.tableHasPK {
		if !hasPK {
			line := r.tableLine[norm] + int32(r.tableBaseLine[norm])
			result = append(result, &storepb.Advice{
				Status:        r.Level,
				Code:          code.TableNoPK.Int32(),
				Title:         r.Title,
				Content:       fmt.Sprintf("Table %s requires PRIMARY KEY.", r.tableOriginal[norm]),
				StartPosition: &storepb.Position{Line: line},
			})
		}
	}
	return result
}

// tableRefText returns a human-readable string for a TableRef.
func tableRefText(ref *ast.TableRef) string {
	if ref == nil {
		return ""
	}
	result := ref.Object
	if ref.Schema != "" {
		result = ref.Schema + "." + result
	}
	if ref.Database != "" {
		result = ref.Database + "." + result
	}
	return result
}
