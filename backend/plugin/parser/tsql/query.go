package tsql

import (
	"github.com/bytebase/omni/mssql/analysis"
	"github.com/bytebase/omni/mssql/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterQueryValidator(storepb.Engine_MSSQL, ValidateSQLForEditor)
}

// ValidateSQLForEditor validates that every statement in the SQL is a read-only
// SELECT (no SELECT INTO) or sets read-only session options. Returns (valid,
// allQuery, err); a session option returns no data, so it clears allQuery.
func ValidateSQLForEditor(statement string) (bool, bool, error) {
	stmts, err := ParseTSQL(statement)
	if err != nil {
		return false, false, err
	}
	valid, allQuery := analysis.ValidateQuery(stmts)
	return valid, allQuery, nil
}

// HasSelectInto reports whether sel (or any branch of its set operations)
// carries an INTO clause.
func HasSelectInto(sel *ast.SelectStmt) bool {
	return analysis.HasSelectInto(sel)
}

// selectIntoTarget returns the INTO target of sel, searching set-operation
// arms — T-SQL attaches INTO to the first arm of a UNION, not the root.
func selectIntoTarget(sel *ast.SelectStmt) *ast.TableRef {
	if sel == nil {
		return nil
	}
	if sel.IntoTable != nil {
		return sel.IntoTable
	}
	if target := selectIntoTarget(sel.Larg); target != nil {
		return target
	}
	return selectIntoTarget(sel.Rarg)
}
