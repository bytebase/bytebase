package pg

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/pg/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*TableNoFKAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_TABLE_NO_FOREIGN_KEY, &TableNoFKAdvisor{})
}

// TableNoFKAdvisor is the advisor checking table disallow foreign key.
type TableNoFKAdvisor struct {
}

// Check checks table disallow foreign key.
func (*TableNoFKAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := &tableNoFKRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
	}

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type tableNoFKRule struct {
	OmniBaseRule
}

func (*tableNoFKRule) Name() string {
	return "table_no_fk"
}

func (r *tableNoFKRule) OnStatement(node ast.Node) {
	switch n := node.(type) {
	case *ast.CreateStmt:
		r.handleCreateStmt(n)
	case *ast.AlterTableStmt:
		r.handleAlterTableStmt(n)
	default:
	}
}

// handleCreateStmt handles CREATE TABLE with FK constraints.
func (r *tableNoFKRule) handleCreateStmt(n *ast.CreateStmt) {
	tableName := omniTableName(n.Relation)
	schema := omniSchemaName(n.Relation)

	cols, constraints := omniTableElements(n)

	// Check table-level constraints for FOREIGN KEY
	for _, c := range constraints {
		if c.Contype == ast.CONSTR_FOREIGN {
			r.addFKAdvice(schema, tableName)
			return
		}
	}

	// Check column-level constraints for REFERENCES (column-level FK)
	for _, col := range cols {
		for _, c := range omniColumnConstraints(col) {
			if c.Contype == ast.CONSTR_FOREIGN {
				r.addFKAdvice(schema, tableName)
				return
			}
		}
	}
}

// handleAlterTableStmt handles ALTER TABLE ADD CONSTRAINT with FK.
func (r *tableNoFKRule) handleAlterTableStmt(n *ast.AlterTableStmt) {
	tableName := omniTableName(n.Relation)
	schema := omniSchemaName(n.Relation)

	for _, cmd := range omniAlterTableCmds(n) {
		if ast.AlterTableType(cmd.Subtype) == ast.AT_AddConstraint {
			constraint, ok := cmd.Def.(*ast.Constraint)
			if !ok || constraint == nil {
				continue
			}
			if constraint.Contype == ast.CONSTR_FOREIGN {
				r.addFKAdvice(schema, tableName)
				return
			}
		}
	}
}

func (r *tableNoFKRule) addFKAdvice(schemaName, tableName string) {
	r.AddAdvice(&storepb.Advice{
		Status:  r.Level,
		Code:    code.TableHasFK.Int32(),
		Title:   r.Title,
		Content: fmt.Sprintf("Foreign key is not allowed in the table %q.%q, related statement: \"%s\"", schemaName, tableName, r.TrimmedStmtText()),
		StartPosition: &storepb.Position{
			Line:   r.ContentStartLine(),
			Column: 0,
		},
	})
}
