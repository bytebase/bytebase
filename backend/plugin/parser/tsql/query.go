package tsql

import (
	"github.com/bytebase/omni/mssql/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterQueryValidator(storepb.Engine_MSSQL, ValidateSQLForEditor)
}

// ValidateSQLForEditor validates that every statement in the SQL is a read-only
// SELECT (no SELECT INTO). Returns (valid, allAlike, err). For MSSQL the two
// booleans move together — we reject on the first non-SELECT.
func ValidateSQLForEditor(statement string) (bool, bool, error) {
	stmts, err := ParseTSQLOmni(statement)
	if err != nil {
		return false, false, err
	}
	if len(stmts) == 0 {
		return false, false, nil
	}

	for _, s := range stmts {
		if s.Empty() {
			continue
		}
		if !isReadOnlySelect(s.AST) {
			return false, false, nil
		}
	}
	return true, true, nil
}

// isReadOnlySelect returns true for SELECT statements without an INTO target.
// Non-SELECT nodes and SELECT ... INTO are rejected — INTO materialises a new
// table which is a DDL-like side effect.
func isReadOnlySelect(node ast.Node) bool {
	sel, ok := node.(*ast.SelectStmt)
	if !ok {
		return false
	}
	return !HasSelectInto(sel)
}

// HasSelectInto reports whether sel (or any branch of its set operations)
// carries an INTO clause.
func HasSelectInto(sel *ast.SelectStmt) bool {
	if sel == nil {
		return false
	}
	if sel.IntoTable != nil {
		return true
	}
	if sel.Larg != nil && HasSelectInto(sel.Larg) {
		return true
	}
	if sel.Rarg != nil && HasSelectInto(sel.Rarg) {
		return true
	}
	return false
}
