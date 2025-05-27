// Package mysql implements the SQL advisor rules for MySQL.
package mysql

import (
	"regexp"
	"sort"
	"strings"

	parser "github.com/bytebase/mysql-parser"
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

type tableData struct {
	tableName                string
	defaultCurrentTimeCount  int
	onUpdateCurrentTimeCount int
	line                     int
}

// isKeyword checks if the keyword is a MySQL keyword.
// TODO: We should check with map instead of linear search.
func isKeyword(suspect string) bool {
	for _, item := range parser.Keywords80 {
		if strings.EqualFold(suspect, item.Keyword) {
			return true
		}
	}
	return false
}

func columnNeedDefault(ctx parser.IFieldDefinitionContext) bool {
	if ctx.GENERATED_SYMBOL() != nil {
		return false
	}
	for _, attr := range ctx.AllColumnAttribute() {
		if attr.AUTO_INCREMENT_SYMBOL() != nil || attr.PRIMARY_SYMBOL() != nil {
			return false
		}
	}

	if ctx.DataType() == nil {
		return false
	}

	switch ctx.DataType().GetType_().GetTokenType() {
	case parser.MySQLParserBLOB_SYMBOL,
		parser.MySQLParserTINYBLOB_SYMBOL,
		parser.MySQLParserMEDIUMBLOB_SYMBOL,
		parser.MySQLParserLONGBLOB_SYMBOL,
		parser.MySQLParserJSON_SYMBOL,
		parser.MySQLParserTINYTEXT_SYMBOL,
		parser.MySQLParserTEXT_SYMBOL,
		parser.MySQLParserMEDIUMTEXT_SYMBOL,
		parser.MySQLParserLONGTEXT_SYMBOL,
		// LONG VARBINARY and LONG VARCHAR.
		parser.MySQLParserLONG_SYMBOL,
		parser.MySQLParserSERIAL_SYMBOL,
		parser.MySQLParserGEOMETRY_SYMBOL,
		parser.MySQLParserGEOMETRYCOLLECTION_SYMBOL,
		parser.MySQLParserPOINT_SYMBOL,
		parser.MySQLParserMULTIPOINT_SYMBOL,
		parser.MySQLParserLINESTRING_SYMBOL,
		parser.MySQLParserMULTILINESTRING_SYMBOL,
		parser.MySQLParserPOLYGON_SYMBOL,
		parser.MySQLParserMULTIPOLYGON_SYMBOL:
		return false
	}
	return true
}
