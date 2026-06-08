package redshift

import (
	redshiftast "github.com/bytebase/omni/redshift/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	// Override the existing PG-based validator with Redshift-specific one
	base.RegisterQueryValidator(storepb.Engine_REDSHIFT, ValidateSQLForEditor)
}

// ValidateSQLForEditor validates the SQL statement for SQL editor.
// We support the following SQLs:
// 1. EXPLAIN statement, except EXPLAIN ANALYZE (unless it's EXPLAIN ANALYZE SELECT)
// 2. SELECT statement
// We also support CTE with SELECT statements, but not with DML statements.
// Returns (canRunInReadOnly, returnsData, error):
// - canRunInReadOnly: whether all queries can run in read-only mode
// - returnsData: whether all queries return data
// - error: parsing error if the statement is invalid
func ValidateSQLForEditor(statement string) (bool, bool, error) {
	omniStmts, err := ParseRedshiftOmni(statement)
	if err != nil {
		return false, false, convertOmniError(err, base.Statement{
			Text:  statement,
			Start: &storepb.Position{Line: 1, Column: 1},
		})
	}

	seen := false
	returnsData := true
	for _, stmt := range omniStmts {
		if stmt.Empty() {
			continue
		}
		seen = true
		canRunInReadOnly, stmtReturnsData := validateOmniEditorStatement(stmt.AST)
		if !canRunInReadOnly {
			return false, false, nil
		}
		if !stmtReturnsData {
			returnsData = false
		}
	}
	return seen, returnsData, nil
}

func validateOmniEditorStatement(node redshiftast.Node) (bool, bool) {
	switch n := node.(type) {
	case *redshiftast.SelectStmt:
		if n.IntoClause != nil || hasOmniDMLInWithClause(n.WithClause) {
			return false, false
		}
		return true, true
	case *redshiftast.ExplainStmt:
		if hasOmniExplainAnalyze(n) {
			if selectStmt, ok := n.Query.(*redshiftast.SelectStmt); ok && selectStmt.IntoClause == nil && !hasOmniDMLInWithClause(selectStmt.WithClause) {
				return true, false
			}
			return false, false
		}
		return true, true
	case *redshiftast.RedshiftShowStmt, *redshiftast.VariableShowStmt:
		return true, true
	case *redshiftast.VariableSetStmt:
		return true, false
	default:
		return false, false
	}
}

func hasOmniExplainAnalyze(stmt *redshiftast.ExplainStmt) bool {
	if stmt == nil || stmt.Options == nil {
		return false
	}
	for _, item := range stmt.Options.Items {
		def, ok := item.(*redshiftast.DefElem)
		if !ok {
			continue
		}
		if def.Defname == "analyze" {
			return true
		}
	}
	return false
}

func hasOmniDMLInWithClause(withClause *redshiftast.WithClause) bool {
	if withClause == nil || withClause.Ctes == nil {
		return false
	}
	for _, item := range withClause.Ctes.Items {
		cte, ok := item.(*redshiftast.CommonTableExpr)
		if !ok {
			continue
		}
		if hasOmniDMLNode(cte.Ctequery) {
			return true
		}
	}
	return false
}

func hasOmniDMLNode(node redshiftast.Node) bool {
	hasDML := false
	redshiftast.Inspect(node, func(n redshiftast.Node) bool {
		switch n.(type) {
		case *redshiftast.InsertStmt, *redshiftast.UpdateStmt, *redshiftast.DeleteStmt, *redshiftast.MergeStmt:
			hasDML = true
			return false
		default:
			return !hasDML
		}
	})
	return hasDML
}
