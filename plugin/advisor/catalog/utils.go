package catalog

import (
	"strings"
)

// JoinColumnListForIndex joins the column name with sep.
func JoinColumnListForIndex(indexList []*Index, sep string) string {
	return strings.Join(GetColumnListForIndex(indexList), sep)
}

// GetColumnListForIndex returns the column name list.
func GetColumnListForIndex(indexList []*Index) []string {
	var columnList []string
	for _, column := range indexList {
		columnList = append(columnList, column.Expression)
	}
	return columnList
}
