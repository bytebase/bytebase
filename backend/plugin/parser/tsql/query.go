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
// SELECT (no SELECT INTO) or sets read-only session options. Returns (valid,
// allQuery, err); a session option returns no data, so it clears allQuery.
func ValidateSQLForEditor(statement string) (bool, bool, error) {
	stmts, err := ParseTSQL(statement)
	if err != nil {
		return false, false, err
	}
	if len(stmts) == 0 {
		return false, false, nil
	}

	allQuery := true
	for _, s := range stmts {
		if s.Empty() {
			continue
		}
		if isReadOnlySelect(s.AST) {
			continue
		}
		if !isReadOnlySessionOption(s.AST) {
			return false, false, nil
		}
		allQuery = false
	}
	return true, allQuery, nil
}

// readOnlySessionOptions are the SET options a read-only request may carry: they
// write nothing and add no result set, unlike SET STATISTICS XML. DECLARE and SET
// @variable stay refused (BOT-166).
var readOnlySessionOptions = map[string]bool{
	"TRANSACTION ISOLATION LEVEL": true,
	"NOCOUNT":                     true,
}

// isReadOnlySessionOption reports whether node is a SET whose options are all
// read-only session options. One SET can name several, as in SET NOCOUNT, FMTONLY ON.
func isReadOnlySessionOption(node ast.Node) bool {
	set, ok := node.(*ast.SetOptionStmt)
	if !ok || len(set.Options) == 0 {
		return false
	}
	for _, option := range set.Options {
		if !readOnlySessionOptions[option] {
			return false
		}
	}
	return true
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
	return selectIntoTarget(sel) != nil
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
