package tsql

import "github.com/bytebase/omni/mssql/ast"

// collectOmniPredicateColumnRefs walks an omni expression tree rooted at a
// boolean context (WHERE, HAVING, JOIN ON) and returns every *ast.ColumnRef
// reachable through predicate paths. It mirrors the traversal performed by
// the ANTLR-based helpers in query_span_predicate.go.
//
// Scope it covers:
//   - direct column references in the expression
//   - FULLTEXT predicate column lists (CONTAINS / FREETEXT)
//   - nested subqueries: their WHERE / HAVING / ON / CTE bodies are descended
//
// Scope it does NOT cover (left to the extractor):
//   - a subquery's SELECT-list output columns that flow back into the outer
//     predicate via IN / scalar subquery / ANY|ALL — those require the
//     extractor's table-source resolution.
//   - table sources in a FROM clause other than their JOIN ON / derived-table
//     WHERE.
func collectOmniPredicateColumnRefs(expr ast.ExprNode) []*ast.ColumnRef {
	var out []*ast.ColumnRef
	walkOmniPredicateExpr(expr, &out)
	return out
}

// collectOmniSelectPredicateColumnRefs returns predicate column refs reachable
// from a SelectStmt's WHERE, HAVING, JOIN ON conditions, set-op arms, and CTE
// bodies.
func collectOmniSelectPredicateColumnRefs(sel *ast.SelectStmt) []*ast.ColumnRef {
	var out []*ast.ColumnRef
	walkOmniSelectPredicates(sel, &out)
	return out
}

func walkOmniPredicateExpr(n ast.Node, out *[]*ast.ColumnRef) {
	if n == nil {
		return
	}
	switch v := n.(type) {
	case *ast.ColumnRef:
		*out = append(*out, v)
	case *ast.FullTextPredicate:
		if v.Columns != nil {
			for _, item := range v.Columns.Items {
				if cr, ok := item.(*ast.ColumnRef); ok {
					*out = append(*out, cr)
				}
			}
		}
		walkOmniPredicateExpr(v.Value, out)
		walkOmniPredicateExpr(v.LanguageTerm, out)
	case *ast.BinaryExpr:
		walkOmniPredicateExpr(v.Left, out)
		walkOmniPredicateExpr(v.Right, out)
	case *ast.UnaryExpr:
		walkOmniPredicateExpr(v.Operand, out)
	case *ast.BetweenExpr:
		walkOmniPredicateExpr(v.Expr, out)
		walkOmniPredicateExpr(v.Low, out)
		walkOmniPredicateExpr(v.High, out)
	case *ast.LikeExpr:
		walkOmniPredicateExpr(v.Expr, out)
		walkOmniPredicateExpr(v.Pattern, out)
		walkOmniPredicateExpr(v.Escape, out)
	case *ast.IsExpr:
		walkOmniPredicateExpr(v.Expr, out)
	case *ast.InExpr:
		walkOmniPredicateExpr(v.Expr, out)
		if v.List != nil {
			for _, item := range v.List.Items {
				if e, ok := item.(ast.ExprNode); ok {
					walkOmniPredicateExpr(e, out)
				}
			}
		}
		walkOmniPredicateExpr(v.Subquery, out)
	case *ast.ExistsExpr:
		if sel, ok := v.Subquery.(*ast.SelectStmt); ok {
			walkOmniSelectPredicates(sel, out)
		}
	case *ast.SubqueryExpr:
		if sel, ok := v.Query.(*ast.SelectStmt); ok {
			walkOmniSelectPredicates(sel, out)
		}
	case *ast.SubqueryComparisonExpr:
		walkOmniPredicateExpr(v.Left, out)
		if sel, ok := v.Subquery.(*ast.SelectStmt); ok {
			walkOmniSelectPredicates(sel, out)
		}
	case *ast.CaseExpr:
		walkOmniPredicateExpr(v.Arg, out)
		if v.WhenList != nil {
			for _, item := range v.WhenList.Items {
				if cw, ok := item.(*ast.CaseWhen); ok {
					walkOmniPredicateExpr(cw.Condition, out)
					walkOmniPredicateExpr(cw.Result, out)
				}
			}
		}
		walkOmniPredicateExpr(v.ElseExpr, out)
	case *ast.IifExpr:
		walkOmniPredicateExpr(v.Condition, out)
		walkOmniPredicateExpr(v.TrueVal, out)
		walkOmniPredicateExpr(v.FalseVal, out)
	case *ast.CoalesceExpr:
		walkOmniArgList(v.Args, out)
	case *ast.NullifExpr:
		walkOmniPredicateExpr(v.Left, out)
		walkOmniPredicateExpr(v.Right, out)
	case *ast.FuncCallExpr:
		walkOmniArgList(v.Args, out)
	case *ast.MethodCallExpr:
		walkOmniArgList(v.Args, out)
	case *ast.CastExpr:
		walkOmniPredicateExpr(v.Expr, out)
	case *ast.ConvertExpr:
		walkOmniPredicateExpr(v.Expr, out)
		walkOmniPredicateExpr(v.Style, out)
	case *ast.TryCastExpr:
		walkOmniPredicateExpr(v.Expr, out)
	case *ast.TryConvertExpr:
		walkOmniPredicateExpr(v.Expr, out)
		walkOmniPredicateExpr(v.Style, out)
	case *ast.ParenExpr:
		walkOmniPredicateExpr(v.Expr, out)
	case *ast.CollateExpr:
		walkOmniPredicateExpr(v.Expr, out)
	case *ast.AtTimeZoneExpr:
		walkOmniPredicateExpr(v.Expr, out)
		walkOmniPredicateExpr(v.TimeZone, out)
	default:
	}
}

func walkOmniArgList(list *ast.List, out *[]*ast.ColumnRef) {
	if list == nil {
		return
	}
	for _, item := range list.Items {
		if e, ok := item.(ast.ExprNode); ok {
			walkOmniPredicateExpr(e, out)
		}
	}
}

func walkOmniSelectPredicates(sel *ast.SelectStmt, out *[]*ast.ColumnRef) {
	if sel == nil {
		return
	}
	if sel.Larg != nil {
		walkOmniSelectPredicates(sel.Larg, out)
	}
	if sel.Rarg != nil {
		walkOmniSelectPredicates(sel.Rarg, out)
	}
	if sel.WithClause != nil && sel.WithClause.CTEs != nil {
		for _, item := range sel.WithClause.CTEs.Items {
			if c, ok := item.(*ast.CommonTableExpr); ok {
				if q, ok := c.Query.(*ast.SelectStmt); ok {
					walkOmniSelectPredicates(q, out)
				}
			}
		}
	}
	walkOmniPredicateExpr(sel.WhereClause, out)
	walkOmniPredicateExpr(sel.HavingClause, out)
	// JOIN ON conditions are intentionally NOT walked — the legacy ANTLR extractor
	// doesn't treat them as predicate columns, and downstream fixtures assume
	// that behavior. Re-enabling them is a separate decision.
}
