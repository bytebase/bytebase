package tidb

import (
	"github.com/pingcap/tidb/pkg/parser/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterQueryValidator(storepb.Engine_TIDB, validateQuery)
}

// validateQuery validates the SQL statement for SQL editor.
// We validate the statement by following steps:
// 1. Remove all quoted text(quoted identifier, string literal) and comments from the statement.
// 2. Use regexp to check if the statement is a normal SELECT statement and EXPLAIN statement.
// 3. For CTE, use regexp to check if the statement has UPDATE, DELETE and INSERT statements.
func validateQuery(statement string) (bool, bool, error) {
	stmtList, err := ParseTiDB(statement, "", "")
	if err != nil {
		return false, false, err
	}
	hasExecute := false
	readOnly := true
	for _, stmt := range stmtList {
		switch stmt := stmt.(type) {
		case *ast.SelectStmt:
		case *ast.SetOprStmt:
		case *ast.SetStmt:
			hasExecute = true
		case *ast.ShowStmt:
		case *ast.ExplainStmt:
			if stmt.Analyze {
				readOnly = false
			}
		default:
			return false, false, nil
		}
	}
	return readOnly, !hasExecute, nil
}

// ExtractMySQLTableList extracts all the TableNames from node.
// If asName is true, extract AsName prior to OrigName.
func ExtractMySQLTableList(in ast.Node, asName bool) []*ast.TableName {
	input := []*ast.TableName{}
	return extractTableList(in, input, asName)
}

// -------------------------------------------- DO NOT TOUCH --------------------------------------------

// extractTableList extracts all the TableNames from node.
// If asName is true, extract AsName prior to OrigName.
// Privilege check should use OrigName, while expression may use AsName.
// WARNING: copy from TiDB core code, do NOT touch!
func extractTableList(node ast.Node, input []*ast.TableName, asName bool) []*ast.TableName {
	switch x := node.(type) {
	case *ast.SelectStmt:
		if x.From != nil {
			input = extractTableList(x.From.TableRefs, input, asName)
		}
		if x.Where != nil {
			input = extractTableList(x.Where, input, asName)
		}
		if x.With != nil {
			for _, cte := range x.With.CTEs {
				input = extractTableList(cte.Query, input, asName)
			}
		}
		for _, f := range x.Fields.Fields {
			if s, ok := f.Expr.(*ast.SubqueryExpr); ok {
				input = extractTableList(s, input, asName)
			}
		}
	case *ast.DeleteStmt:
		input = extractTableList(x.TableRefs.TableRefs, input, asName)
		if x.IsMultiTable {
			for _, t := range x.Tables.Tables {
				input = extractTableList(t, input, asName)
			}
		}
		if x.Where != nil {
			input = extractTableList(x.Where, input, asName)
		}
		if x.With != nil {
			for _, cte := range x.With.CTEs {
				input = extractTableList(cte.Query, input, asName)
			}
		}
	case *ast.UpdateStmt:
		input = extractTableList(x.TableRefs.TableRefs, input, asName)
		for _, e := range x.List {
			input = extractTableList(e.Expr, input, asName)
		}
		if x.Where != nil {
			input = extractTableList(x.Where, input, asName)
		}
		if x.With != nil {
			for _, cte := range x.With.CTEs {
				input = extractTableList(cte.Query, input, asName)
			}
		}
	case *ast.InsertStmt:
		input = extractTableList(x.Table.TableRefs, input, asName)
		input = extractTableList(x.Select, input, asName)
	case *ast.SetOprStmt:
		l := &ast.SetOprSelectList{}
		unfoldSelectList(x.SelectList, l)
		for _, s := range l.Selects {
			input = extractTableList(s.(ast.ResultSetNode), input, asName)
		}
	case *ast.PatternInExpr:
		if s, ok := x.Sel.(*ast.SubqueryExpr); ok {
			input = extractTableList(s, input, asName)
		}
	case *ast.ExistsSubqueryExpr:
		if s, ok := x.Sel.(*ast.SubqueryExpr); ok {
			input = extractTableList(s, input, asName)
		}
	case *ast.BinaryOperationExpr:
		if s, ok := x.R.(*ast.SubqueryExpr); ok {
			input = extractTableList(s, input, asName)
		}
	case *ast.SubqueryExpr:
		input = extractTableList(x.Query, input, asName)
	case *ast.Join:
		input = extractTableList(x.Left, input, asName)
		input = extractTableList(x.Right, input, asName)
	case *ast.TableSource:
		if s, ok := x.Source.(*ast.TableName); ok {
			if x.AsName.L != "" && asName {
				newTableName := *s
				newTableName.Name = x.AsName
				newTableName.Schema = ast.NewCIStr("")
				input = append(input, &newTableName)
			} else {
				input = append(input, s)
			}
		} else if s, ok := x.Source.(*ast.SelectStmt); ok {
			if s.From != nil {
				var innerList []*ast.TableName
				innerList = extractTableList(s.From.TableRefs, innerList, asName)
				if len(innerList) > 0 {
					innerTableName := innerList[0]
					if x.AsName.L != "" && asName {
						newTableName := *innerList[0]
						newTableName.Name = x.AsName
						newTableName.Schema = ast.NewCIStr("")
						innerTableName = &newTableName
					}
					input = append(input, innerTableName)
				}
			}
		} else if s, ok := x.Source.(*ast.SetOprStmt); ok {
			// BYT-8415: Handle UNION ALL and other set operations in FROM subqueries
			input = extractTableList(s, input, asName)
		}
	}
	return input
}

// WARNING: copy from TiDB core code, do NOT touch!
func unfoldSelectList(list *ast.SetOprSelectList, unfoldList *ast.SetOprSelectList) {
	for _, sel := range list.Selects {
		switch s := sel.(type) {
		case *ast.SelectStmt:
			unfoldList.Selects = append(unfoldList.Selects, s)
		case *ast.SetOprSelectList:
			unfoldSelectList(s, unfoldList)
		}
	}
}
