package mysql

import "sort"

type columnSet map[string]bool

// columnList returns column list in lexicographical order.
func (set columnSet) columnList() []string {
	var columnList []string
	for columnName := range set {
		columnList = append(columnList, columnName)
	}
	sort.Strings(columnList)
	return columnList
}

type tableState map[string]columnSet

func (t tableState) getTable(table string) columnSet {
	if _, ok := t[table]; !ok {
		t[table] = make(columnSet)
	}
	return t[table]
}

// tableList returns table list in lexicographical order.
func (t tableState) tableList() []string {
	var tableList []string
	for tableName := range t {
		tableList = append(tableList, tableName)
	}
	sort.Strings(tableList)
	return tableList
}
