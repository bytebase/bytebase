// Package mysql implements the SQL advisor rules for MySQL.
package mysql

import (
	"regexp"
	"slices"
	"strings"

	"github.com/bytebase/parser/mysql"
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
	slices.Sort(tableList)
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

// isKeyword checks if the keyword is a MySQL keyword.
// TODO: We should check with map instead of linear search.
func isKeyword(suspect string) bool {
	for _, item := range mysql.Keywords80 {
		if strings.EqualFold(suspect, item.Keyword) {
			return true
		}
	}
	return false
}

// isCharsetDataType checks if the data type supports charset.
func isCharsetDataType(dataType mysql.IDataTypeContext) bool {
	return dataType != nil && (dataType.CHAR_SYMBOL() != nil ||
		dataType.VARCHAR_SYMBOL() != nil ||
		dataType.VARYING_SYMBOL() != nil ||
		dataType.TINYTEXT_SYMBOL() != nil ||
		dataType.TEXT_SYMBOL() != nil ||
		dataType.MEDIUMTEXT_SYMBOL() != nil ||
		dataType.LONGTEXT_SYMBOL() != nil)
}
