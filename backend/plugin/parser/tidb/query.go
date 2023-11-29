package tidb

import (
	"sort"

	tidbast "github.com/pingcap/tidb/pkg/parser/ast"
	"github.com/pingcap/tidb/pkg/parser/model"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func init() {
	base.RegisterQueryValidator(storepb.Engine_TIDB, validateQuery)
	base.RegisterExtractResourceListFunc(storepb.Engine_TIDB, ExtractResourceList)
}

// validateQuery validates the SQL statement for SQL editor.
// We validate the statement by following steps:
// 1. Remove all quoted text(quoted identifier, string literal) and comments from the statement.
// 2. Use regexp to check if the statement is a normal SELECT statement and EXPLAIN statement.
// 3. For CTE, use regexp to check if the statement has UPDATE, DELETE and INSERT statements.
func validateQuery(statement string) (bool, error) {
	stmtList, err := ParseTiDB(statement, "", "")
	if err != nil {
		return false, err
	}
	for _, stmt := range stmtList {
		switch stmt := stmt.(type) {
		case *tidbast.SelectStmt:
		case *tidbast.SetOprStmt:
		case *tidbast.ExplainStmt:
			// Disable DESC command.
			if _, ok := stmt.Stmt.(*tidbast.ShowStmt); ok {
				return false, nil
			}
		default:
			return false, nil
		}
	}
	return true, nil
}

func ExtractResourceList(currentDatabase string, _, sql string) ([]base.SchemaResource, error) {
	nodes, err := ParseTiDB(sql, "", "")
	if err != nil {
		return nil, err
	}

	resourceMap := make(map[string]base.SchemaResource)

	for _, node := range nodes {
		tableList := ExtractMySQLTableList(node, false /* asName */)
		for _, table := range tableList {
			resource := base.SchemaResource{
				Database: table.Schema.O,
				Schema:   "",
				Table:    table.Name.O,
			}
			if resource.Database == "" {
				resource.Database = currentDatabase
			}
			if _, ok := resourceMap[resource.String()]; !ok {
				resourceMap[resource.String()] = resource
			}
		}
	}

	resourceList := make([]base.SchemaResource, 0, len(resourceMap))
	for _, resource := range resourceMap {
		resourceList = append(resourceList, resource)
	}
	sort.Slice(resourceList, func(i, j int) bool {
		return resourceList[i].String() < resourceList[j].String()
	})

	return resourceList, nil
}

// ExtractMySQLTableList extracts all the TableNames from node.
// If asName is true, extract AsName prior to OrigName.
func ExtractMySQLTableList(in tidbast.Node, asName bool) []*tidbast.TableName {
	input := []*tidbast.TableName{}
	return extractTableList(in, input, asName)
}

// -------------------------------------------- DO NOT TOUCH --------------------------------------------

// extractTableList extracts all the TableNames from node.
// If asName is true, extract AsName prior to OrigName.
// Privilege check should use OrigName, while expression may use AsName.
// WARNING: copy from TiDB core code, do NOT touch!
func extractTableList(node tidbast.Node, input []*tidbast.TableName, asName bool) []*tidbast.TableName {
	switch x := node.(type) {
	case *tidbast.SelectStmt:
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
			if s, ok := f.Expr.(*tidbast.SubqueryExpr); ok {
				input = extractTableList(s, input, asName)
			}
		}
	case *tidbast.DeleteStmt:
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
	case *tidbast.UpdateStmt:
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
	case *tidbast.InsertStmt:
		input = extractTableList(x.Table.TableRefs, input, asName)
		input = extractTableList(x.Select, input, asName)
	case *tidbast.SetOprStmt:
		l := &tidbast.SetOprSelectList{}
		unfoldSelectList(x.SelectList, l)
		for _, s := range l.Selects {
			input = extractTableList(s.(tidbast.ResultSetNode), input, asName)
		}
	case *tidbast.PatternInExpr:
		if s, ok := x.Sel.(*tidbast.SubqueryExpr); ok {
			input = extractTableList(s, input, asName)
		}
	case *tidbast.ExistsSubqueryExpr:
		if s, ok := x.Sel.(*tidbast.SubqueryExpr); ok {
			input = extractTableList(s, input, asName)
		}
	case *tidbast.BinaryOperationExpr:
		if s, ok := x.R.(*tidbast.SubqueryExpr); ok {
			input = extractTableList(s, input, asName)
		}
	case *tidbast.SubqueryExpr:
		input = extractTableList(x.Query, input, asName)
	case *tidbast.Join:
		input = extractTableList(x.Left, input, asName)
		input = extractTableList(x.Right, input, asName)
	case *tidbast.TableSource:
		if s, ok := x.Source.(*tidbast.TableName); ok {
			if x.AsName.L != "" && asName {
				newTableName := *s
				newTableName.Name = x.AsName
				newTableName.Schema = model.NewCIStr("")
				input = append(input, &newTableName)
			} else {
				input = append(input, s)
			}
		} else if s, ok := x.Source.(*tidbast.SelectStmt); ok {
			if s.From != nil {
				var innerList []*tidbast.TableName
				innerList = extractTableList(s.From.TableRefs, innerList, asName)
				if len(innerList) > 0 {
					innerTableName := innerList[0]
					if x.AsName.L != "" && asName {
						newTableName := *innerList[0]
						newTableName.Name = x.AsName
						newTableName.Schema = model.NewCIStr("")
						innerTableName = &newTableName
					}
					input = append(input, innerTableName)
				}
			}
		}
	}
	return input
}

// WARNING: copy from TiDB core code, do NOT touch!
func unfoldSelectList(list *tidbast.SetOprSelectList, unfoldList *tidbast.SetOprSelectList) {
	for _, sel := range list.Selects {
		switch s := sel.(type) {
		case *tidbast.SelectStmt:
			unfoldList.Selects = append(unfoldList.Selects, s)
		case *tidbast.SetOprSelectList:
			unfoldSelectList(s, unfoldList)
		}
	}
}
