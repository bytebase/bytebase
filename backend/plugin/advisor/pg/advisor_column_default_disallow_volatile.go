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
	_ advisor.Advisor = (*ColumnDefaultDisallowVolatileAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_COLUMN_DEFAULT_DISALLOW_VOLATILE, &ColumnDefaultDisallowVolatileAdvisor{})
}

// ColumnDefaultDisallowVolatileAdvisor is the advisor checking for column default volatile functions.
type ColumnDefaultDisallowVolatileAdvisor struct {
}

// Check checks for column default volatile functions.
func (*ColumnDefaultDisallowVolatileAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := &columnDefaultDisallowVolatileRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
	}

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type columnDefaultDisallowVolatileRule struct {
	OmniBaseRule
}

func (*columnDefaultDisallowVolatileRule) Name() string {
	return string(storepb.SQLReviewRule_COLUMN_DEFAULT_DISALLOW_VOLATILE)
}

func (r *columnDefaultDisallowVolatileRule) OnStatement(node ast.Node) {
	switch n := node.(type) {
	case *ast.AlterTableStmt:
		r.handleAlterTableStmt(n)
	default:
	}
}

func (r *columnDefaultDisallowVolatileRule) handleAlterTableStmt(n *ast.AlterTableStmt) {
	tableName := omniTableName(n.Relation)
	if tableName == "" {
		return
	}

	for _, cmd := range omniAlterTableCmds(n) {
		if ast.AlterTableType(cmd.Subtype) != ast.AT_AddColumn {
			continue
		}
		colDef, ok := cmd.Def.(*ast.ColumnDef)
		if !ok || colDef == nil {
			continue
		}

		if r.hasVolatileDefault(colDef) {
			r.AddAdvice(&storepb.Advice{
				Status:  r.Level,
				Code:    code.NoDefault.Int32(),
				Title:   r.Title,
				Content: fmt.Sprintf("Column %q.%q in schema %q has volatile DEFAULT", tableName, colDef.Colname, "public"),
				StartPosition: &storepb.Position{
					Line:   r.FindLineByName(colDef.Colname),
					Column: 0,
				},
			})
		}
	}
}

// hasVolatileDefault checks if a column definition has a default that contains a function call.
func (*columnDefaultDisallowVolatileRule) hasVolatileDefault(col *ast.ColumnDef) bool {
	// Check RawDefault for function calls
	if col.RawDefault != nil && containsFuncCall(col.RawDefault) {
		return true
	}

	// Check constraints for DEFAULT with function calls
	for _, c := range omniColumnConstraints(col) {
		if c.Contype == ast.CONSTR_DEFAULT && c.RawExpr != nil && containsFuncCall(c.RawExpr) {
			return true
		}
	}

	return false
}

// containsFuncCall uses ast.Inspect to walk the expression tree looking for FuncCall nodes.
func containsFuncCall(node ast.Node) bool {
	found := false
	ast.Inspect(node, func(n ast.Node) bool {
		if found {
			return false
		}
		if _, ok := n.(*ast.FuncCall); ok {
			found = true
			return false
		}
		return true
	})
	return found
}
