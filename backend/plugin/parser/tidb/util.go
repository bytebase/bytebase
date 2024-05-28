package tidb

import (
	"strings"

	tidbast "github.com/pingcap/tidb/pkg/parser/ast"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
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

func mergeJoinField(node *tidbast.Join, leftField []base.FieldInfo, rightField []base.FieldInfo) ([]base.FieldInfo, error) {
	leftFieldMap := make(map[string]base.FieldInfo)
	rightFieldMap := make(map[string]base.FieldInfo)
	var result []base.FieldInfo
	for _, field := range leftField {
		// Column name in MySQL is NOT case-sensitive.
		leftFieldMap[strings.ToLower(field.Name)] = field
	}
	for _, field := range rightField {
		// Column name in MySQL is NOT case-sensitive.
		rightFieldMap[strings.ToLower(field.Name)] = field
	}
	if node.NaturalJoin {
		// Natural Join will merge the same column name field.
		for _, field := range leftField {
			// Merge the sensitive attribute for the same column name field.
			if rField, exists := rightFieldMap[strings.ToLower(field.Name)]; exists {
				field.MaskingAttributes.TransmittedBy(rField.MaskingAttributes)
			}
			result = append(result, field)
		}

		for _, field := range rightField {
			if _, exists := leftFieldMap[strings.ToLower(field.Name)]; !exists {
				result = append(result, field)
			}
		}
	} else {
		if len(node.Using) != 0 {
			// ... JOIN ... USING (...) will merge the column in USING.
			usingMap := make(map[string]bool)
			for _, column := range node.Using {
				// Column name in MySQL is NOT case-sensitive.
				usingMap[column.Name.L] = true
			}

			for _, field := range leftField {
				_, existsInUsingMap := usingMap[strings.ToLower(field.Name)]
				rField, existsInRightField := rightFieldMap[strings.ToLower(field.Name)]
				// Merge the sensitive attribute for the column name field in USING.
				if existsInUsingMap && existsInRightField {
					field.MaskingAttributes.TransmittedBy(rField.MaskingAttributes)
				}
				result = append(result, field)
			}

			for _, field := range rightField {
				_, existsInUsingMap := usingMap[strings.ToLower(field.Name)]
				_, existsInLeftField := leftFieldMap[strings.ToLower(field.Name)]
				if existsInUsingMap && existsInLeftField {
					continue
				}
				result = append(result, field)
			}
		} else {
			result = append(result, leftField...)
			result = append(result, rightField...)
		}
	}

	return result, nil
}
