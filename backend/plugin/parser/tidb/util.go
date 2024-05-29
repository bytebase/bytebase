package tidb

import (
	tidbast "github.com/pingcap/tidb/pkg/parser/ast"
)

func splitInitialAndRecursivePart(node *tidbast.SetOprStmt, selfName string) ([]tidbast.Node, []tidbast.Node) {
	for i, selectStmt := range node.SelectList.Selects {
		tableList := ExtractMySQLTableList(selectStmt, false /* asName */)
		for _, table := range tableList {
			if table.Schema.O == "" && table.Name.O == selfName {
				return node.SelectList.Selects[:i], node.SelectList.Selects[i:]
			}
		}
	}
	return node.SelectList.Selects, nil
}

func extractFieldName(in *tidbast.SelectField) string {
	if in.AsName.O != "" {
		return in.AsName.O
	}

	if in.Expr != nil {
		if columnName, ok := in.Expr.(*tidbast.ColumnNameExpr); ok {
			return columnName.Name.Name.O
		}
		return in.Text()
	}
	return ""
}
