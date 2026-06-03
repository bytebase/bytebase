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
	_ advisor.Advisor = (*TableRequirePKAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_ORACLE, storepb.SQLReviewRule_TABLE_REQUIRE_PK, &TableRequirePKAdvisor{})
}

// TableRequirePKAdvisor is the advisor checking table requires PK.
type TableRequirePKAdvisor struct {
}

// Check checks table requires PK.
func (*TableRequirePKAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := NewTableRequirePKRule(level, checkCtx.Rule.Type.String(), checkCtx.CurrentDatabase)

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule})
}

// TableRequirePKRule is the rule implementation for table requires PK.
type TableRequirePKRule struct {
	BaseRule

	currentDatabase string
	tableWitPK      map[string]bool
	tableLine       map[string]int
}

// NewTableRequirePKRule creates a new TableRequirePKRule.
func NewTableRequirePKRule(level storepb.Advice_Status, title string, currentDatabase string) *TableRequirePKRule {
	return &TableRequirePKRule{
		BaseRule:        NewBaseRule(level, title, 0),
		currentDatabase: currentDatabase,
		tableWitPK:      make(map[string]bool),
		tableLine:       make(map[string]int),
	}
}

// Name returns the rule name.
func (*TableRequirePKRule) Name() string {
	return "table.require-pk"
}

// OnStatement checks primary-key state from omni DDL nodes.
func (r *TableRequirePKRule) OnStatement(node ast.Node) {
	switch n := node.(type) {
	case *ast.CreateTableStmt:
		tableName := omniObjectName(n.Name, r.currentDatabase)
		r.tableWitPK[tableName] = r.createTableHasPK(n)
		r.tableLine[tableName] = r.locLine(n.Loc)
	case *ast.AlterTableStmt:
		tableName := omniObjectName(n.Name, r.currentDatabase)
		for _, cmd := range omniAlterTableCmds(n) {
			switch cmd.Action {
			case ast.AT_ADD_CONSTRAINT:
				if cmd.Constraint != nil && cmd.Constraint.Type == ast.CONSTRAINT_PRIMARY {
					r.tableWitPK[tableName] = true
				}
			case ast.AT_DROP_CONSTRAINT:
				if strings.Contains(strings.ToUpper(cmd.Subtype), "PRIMARY") || strings.EqualFold(cmd.ColumnName, "PRIMARY") {
					r.tableWitPK[tableName] = false
					r.tableLine[tableName] = r.locLine(cmd.Loc)
				}
			default:
			}
		}
	case *ast.DropStmt:
		for _, item := range listItems(n.Names) {
			name, ok := item.(*ast.ObjectName)
			if ok {
				delete(r.tableWitPK, omniObjectName(name, r.currentDatabase))
			}
		}
	default:
	}
}

func (*TableRequirePKRule) createTableHasPK(stmt *ast.CreateTableStmt) bool {
	for _, col := range omniColumnDefs(stmt.Columns) {
		if omniColumnHasConstraint(col, ast.CONSTRAINT_PRIMARY) {
			return true
		}
	}
	for _, c := range omniTableConstraints(stmt.Constraints) {
		if c.Type == ast.CONSTRAINT_PRIMARY {
			return true
		}
	}
	return false
}

// OnEnter is called when the parser enters a rule context.

// OnExit is called when the parser exits a rule context.

// GetAdviceList returns the advice list.
func (r *TableRequirePKRule) GetAdviceList() ([]*storepb.Advice, error) {
	for tableName, hasPK := range r.tableWitPK {
		if !hasPK {
			r.AddAdvice(
				r.level,
				code.TableNoPK.Int32(),
				fmt.Sprintf("Table %s requires PRIMARY KEY.", normalizeIdentifierName(tableName)),
				common.ConvertANTLRLineToPosition(r.tableLine[tableName]),
			)
		}
	}
	return r.BaseRule.GetAdviceList()
}
