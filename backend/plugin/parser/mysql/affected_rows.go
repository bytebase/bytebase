package mysql

import (
	"fmt"
	"strings"

	"github.com/bytebase/omni/mysql/ast"
	"github.com/bytebase/omni/mysql/deparse"
)

// AffectedRowsCountSQL returns a SELECT COUNT(*) query for single-table UPDATE/DELETE
// statements whose subquery predicates make raw EXPLAIN rows an unsafe affected-row estimate.
func AffectedRowsCountSQL(statement string) (string, bool) {
	list, err := ParseMySQLOmni(statement)
	if err != nil || list == nil || len(list.Items) != 1 {
		return "", false
	}

	var table *ast.TableRef
	var where ast.ExprNode
	switch stmt := list.Items[0].(type) {
	case *ast.UpdateStmt:
		if len(stmt.Tables) != 1 {
			return "", false
		}
		var ok bool
		table, ok = stmt.Tables[0].(*ast.TableRef)
		if !ok {
			return "", false
		}
		where = stmt.Where
	case *ast.DeleteStmt:
		if len(stmt.Tables) != 1 || len(stmt.Using) > 0 {
			return "", false
		}
		var ok bool
		table, ok = stmt.Tables[0].(*ast.TableRef)
		if !ok {
			return "", false
		}
		where = stmt.Where
	default:
		return "", false
	}
	if where == nil || !containsSubquery(where) {
		return "", false
	}

	tableSQL := tableRefSQL(statement, table)
	whereSQL := deparse.Deparse(where)
	if tableSQL == "" || whereSQL == "" {
		return "", false
	}
	return fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s", tableSQL, whereSQL), true
}

func tableRefSQL(statement string, table *ast.TableRef) string {
	if table == nil {
		return ""
	}
	if table.Loc.Start >= 0 && table.Loc.Start < table.Loc.End && table.Loc.End <= len(statement) {
		return strings.TrimSpace(statement[table.Loc.Start:table.Loc.End])
	}

	var sb strings.Builder
	if table.Schema != "" {
		sb.WriteString(table.Schema)
		sb.WriteString(".")
	}
	sb.WriteString(table.Name)
	if table.Alias != "" {
		sb.WriteString(" ")
		sb.WriteString(table.Alias)
	}
	return sb.String()
}

func containsSubquery(node ast.Node) bool {
	found := false
	ast.Inspect(node, func(n ast.Node) bool {
		if found {
			return false
		}
		switch expr := n.(type) {
		case *ast.ExistsExpr, *ast.SubqueryExpr:
			found = true
			return false
		case *ast.InExpr:
			if expr.Select != nil {
				found = true
				return false
			}
		default:
		}
		return true
	})
	return found
}
