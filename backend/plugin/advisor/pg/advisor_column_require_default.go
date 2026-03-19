package pg

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/omni/pg/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*ColumnRequireDefaultAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_COLUMN_REQUIRE_DEFAULT, &ColumnRequireDefaultAdvisor{})
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

	rule := &columnRequireDefaultRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
	}

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type columnRequireDefaultRule struct {
	OmniBaseRule
}

func (*columnRequireDefaultRule) Name() string {
	return "column_require_default"
}

func (r *columnRequireDefaultRule) OnStatement(node ast.Node) {
	switch n := node.(type) {
	case *ast.CreateStmt:
		r.handleCreateStmt(n)
	case *ast.AlterTableStmt:
		r.handleAlterTableStmt(n)
	default:
	}
}

func (r *columnRequireDefaultRule) handleCreateStmt(n *ast.CreateStmt) {
	tableName := omniTableName(n.Relation)
	if tableName == "" {
		return
	}

	cols, _ := omniTableElements(n)
	for _, col := range cols {
		if !r.hasDefault(col) {
			r.AddAdvice(&storepb.Advice{
				Status:  r.Level,
				Code:    code.NoDefault.Int32(),
				Title:   r.Title,
				Content: fmt.Sprintf("Column %q.%q in schema %q doesn't have DEFAULT", tableName, col.Colname, "public"),
				StartPosition: &storepb.Position{
					Line:   r.FindLineByName(col.Colname),
					Column: 0,
				},
			})
		}
	}
}

func (r *columnRequireDefaultRule) handleAlterTableStmt(n *ast.AlterTableStmt) {
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
		if !r.hasDefault(colDef) {
			r.AddAdvice(&storepb.Advice{
				Status:  r.Level,
				Code:    code.NoDefault.Int32(),
				Title:   r.Title,
				Content: fmt.Sprintf("Column %q.%q in schema %q doesn't have DEFAULT", tableName, colDef.Colname, "public"),
				StartPosition: &storepb.Position{
					Line:   r.FindLineByName(colDef.Colname),
					Column: 0,
				},
			})
		}
	}
}

// hasDefault checks if a column definition has a DEFAULT clause
// or uses a serial type (which has an implicit default).
func (*columnRequireDefaultRule) hasDefault(col *ast.ColumnDef) bool {
	// Check if ColumnDef has RawDefault set directly
	if col.RawDefault != nil {
		return true
	}

	// Check if the type is serial/bigserial/smallserial (which have implicit defaults)
	typeName := omniTypeName(col.TypeName)
	lower := strings.ToLower(typeName)
	if lower == "serial" || lower == "bigserial" || lower == "smallserial" ||
		lower == "serial4" || lower == "serial8" || lower == "serial2" {
		return true
	}

	// Check for explicit DEFAULT constraint in the constraints list
	for _, c := range omniColumnConstraints(col) {
		if c.Contype == ast.CONSTR_DEFAULT {
			return true
		}
	}

	return false
}
