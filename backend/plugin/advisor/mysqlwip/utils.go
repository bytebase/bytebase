// Package mysql implements the SQL advisor rules for MySQL.
package mysqlwip

import (
	"regexp"
	"sort"
	"strings"
)

type columnSet map[string]bool

func newColumnSet(columns []string) columnSet {
	res := make(columnSet)
	for _, col := range columns {
		res[col] = true
	}
	return res
}

type tableState map[string]columnSet

// tableList returns table list in lexicographical order.
func (t tableState) tableList() []string {
	var tableList []string
	for tableName := range t {
		tableList = append(tableList, tableName)
	}
	sort.Strings(tableList)
	return tableList
}

type tablePK map[string]columnSet

// tableList returns table list in lexicographical order.
func (t tablePK) tableList() []string {
	var tableList []string
	for tableName := range t {
		tableList = append(tableList, tableName)
	}
	sort.Strings(tableList)
	return tableList
}

// getTemplateRegexp formats the template as regex.
func getTemplateRegexp(template string, templateList []string, tokens map[string]string) (*regexp.Regexp, error) {
	for _, key := range templateList {
		if token, ok := tokens[key]; ok {
			template = strings.ReplaceAll(template, key, token)
		}
	}

	return regexp.Compile(template)
}

// tableName --> columnName --> columnType.
type tableColumnTypes map[string]map[string]string

func (t tableColumnTypes) set(tableName string, columnName string, columnType string) {
	if _, ok := t[tableName]; !ok {
		t[tableName] = make(map[string]string)
	}
	t[tableName][columnName] = columnType
}

func (t tableColumnTypes) get(tableName string, columnName string) (columnType string, ok bool) {
	if _, ok := t[tableName]; !ok {
		return "", false
	}
	col, ok := t[tableName][columnName]
	return col, ok
}

func (t tableColumnTypes) delete(tableName string, columnName string) {
	if _, ok := t[tableName]; !ok {
		return
	}
	delete(t[tableName], columnName)
}
