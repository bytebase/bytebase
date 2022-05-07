package mysql

import "sort"

type columnSet map[string]bool

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

func (t tableState) tableList() []string {
	var tableList []string
	for tableName := range t {
		tableList = append(tableList, tableName)
	}
	sort.Strings(tableList)
	return tableList
}
