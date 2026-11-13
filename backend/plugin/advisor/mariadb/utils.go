// Package mysql implements the SQL advisor rules for MySQL.
package mariadb

import (
	"regexp"
	"slices"
	"strings"

	"github.com/bytebase/omni/mariadb/ast"
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

// tableName extracts the table name from a TableRef.
func tableName(ref *ast.TableRef) string {
	if ref == nil {
		return ""
	}
	return ref.Name
}

// indexColumns extracts column names from an index column list.
// For expression-based index columns, it falls back to the column reference name.
func indexColumns(cols []*ast.IndexColumn) []string {
	if len(cols) == 0 {
		return nil
	}
	var names []string
	for _, col := range cols {
		if ref, ok := col.Expr.(*ast.ColumnRef); ok {
			names = append(names, ref.Column)
		}
	}
	return names
}

// isNullable checks if a column allows NULL.
// A column is nullable unless it has a NOT NULL constraint or is a PRIMARY KEY.
func isNullable(col *ast.ColumnDef) bool {
	if col == nil {
		return true
	}
	for _, c := range col.Constraints {
		if c.Type == ast.ColConstrNotNull || c.Type == ast.ColConstrPrimaryKey {
			return false
		}
	}
	return true
}

// hasDefault checks if a column has a DEFAULT value.
func hasDefault(col *ast.ColumnDef) bool {
	if col == nil {
		return false
	}
	if col.DefaultValue != nil {
		return true
	}
	for _, c := range col.Constraints {
		if c.Type == ast.ColConstrDefault {
			return true
		}
	}
	return false
}

// isAutoIncrement checks if a column has AUTO_INCREMENT.
func isAutoIncrement(col *ast.ColumnDef) bool {
	if col == nil {
		return false
	}
	return col.AutoIncrement
}

// columnComment extracts the COMMENT string from a column definition.
func columnComment(col *ast.ColumnDef) string {
	if col == nil {
		return ""
	}
	return col.Comment
}

// tableOptionValue extracts a table option value by name (case-insensitive).
func tableOptionValue(opts []*ast.TableOption, name string) string {
	for _, opt := range opts {
		if strings.EqualFold(opt.Name, name) {
			return opt.Value
		}
	}
	return ""
}

// constraintName returns the CONSTRAINT symbol, else the name written after
// FOREIGN KEY, which omni keeps in IndexName.
func constraintName(c *ast.Constraint) string {
	if c.Name != "" {
		return c.Name
	}
	return c.IndexName
}

// dataTypeNameCompact returns a compact, lowercase type name string.
func dataTypeNameCompact(dt *ast.DataType) string {
	if dt == nil {
		return ""
	}
	return strings.ToLower(dt.Name)
}

// isIntegerType checks if the data type is an integer type.
func isIntegerType(dt *ast.DataType) bool {
	if dt == nil {
		return false
	}
	switch strings.ToUpper(dt.Name) {
	case "INT", "TINYINT", "SMALLINT", "MEDIUMINT", "BIGINT", "INTEGER":
		return true
	default:
		return false
	}
}

// isTimeType checks if the data type is DATETIME or TIMESTAMP.
func isTimeType(dt *ast.DataType) bool {
	if dt == nil {
		return false
	}
	switch strings.ToUpper(dt.Name) {
	case "DATETIME", "TIMESTAMP":
		return true
	default:
		return false
	}
}

// isDefaultCurrentTime checks if a column has DEFAULT CURRENT_TIMESTAMP / NOW().
func isDefaultCurrentTime(col *ast.ColumnDef) bool {
	if col == nil {
		return false
	}
	return isCurrentTimeFuncCall(col.DefaultValue)
}

// isOnUpdateCurrentTime checks if a column has ON UPDATE CURRENT_TIMESTAMP / NOW().
func isOnUpdateCurrentTime(col *ast.ColumnDef) bool {
	if col == nil {
		return false
	}
	return isCurrentTimeFuncCall(col.OnUpdate)
}

// isCurrentTimeFuncCall checks if an expression is a CURRENT_TIMESTAMP/NOW() call.
func isCurrentTimeFuncCall(expr ast.ExprNode) bool {
	if expr == nil {
		return false
	}
	fc, ok := expr.(*ast.FuncCallExpr)
	if !ok {
		return false
	}
	switch strings.ToUpper(fc.Name) {
	case "NOW", "CURRENT_TIMESTAMP", "LOCALTIME", "LOCALTIMESTAMP":
		return true
	default:
		return false
	}
}

// getColumnsFromCmd extracts column definitions from an AlterTableCmd.
func getColumnsFromCmd(cmd *ast.AlterTableCmd) []*ast.ColumnDef {
	if cmd == nil {
		return nil
	}
	switch cmd.Type {
	case ast.ATAddColumn:
		if cmd.Column != nil {
			return []*ast.ColumnDef{cmd.Column}
		}
		return cmd.Columns
	case ast.ATModifyColumn, ast.ATChangeColumn:
		if cmd.Column != nil {
			return []*ast.ColumnDef{cmd.Column}
		}
	default:
	}
	return nil
}

// isPrimaryKey checks if a column has a PRIMARY KEY constraint.
func isPrimaryKey(col *ast.ColumnDef) bool {
	if col == nil {
		return false
	}
	for _, c := range col.Constraints {
		if c.Type == ast.ColConstrPrimaryKey {
			return true
		}
	}
	return false
}

// pKColumnNames returns a set of column names that are part of a primary key.
func pKColumnNames(columns []*ast.ColumnDef, constraints []*ast.Constraint) map[string]bool {
	result := make(map[string]bool)
	for _, col := range columns {
		if isPrimaryKey(col) {
			result[col.Name] = true
		}
	}
	for _, c := range constraints {
		if c.Type == ast.ConstrPrimaryKey {
			for _, name := range c.Columns {
				result[name] = true
			}
		}
	}
	return result
}

// isDefaultExemptType checks if a column type is exempt from requiring DEFAULT
// (BLOB, TEXT, JSON, GEOMETRY and spatial types).
func isDefaultExemptType(col *ast.ColumnDef) bool {
	if col == nil || col.TypeName == nil {
		return false
	}
	switch strings.ToUpper(col.TypeName.Name) {
	case "BLOB", "TINYBLOB", "MEDIUMBLOB", "LONGBLOB",
		"TEXT", "TINYTEXT", "MEDIUMTEXT", "LONGTEXT",
		"JSON", "GEOMETRY", "POINT", "LINESTRING", "POLYGON",
		"MULTIPOINT", "MULTILINESTRING", "MULTIPOLYGON", "GEOMETRYCOLLECTION",
		"GEOMCOLLECTION":
		return true
	default:
		return false
	}
}

// columnNeedDefaultNotNull checks if a NOT NULL column needs a DEFAULT value
// (excludes BLOB, TEXT, JSON, GEOMETRY, and AUTO_INCREMENT columns).
func columnNeedDefaultNotNull(col *ast.ColumnDef) bool {
	if col == nil {
		return false
	}
	if isNullable(col) || hasDefault(col) || isPrimaryKey(col) || col.AutoIncrement {
		return false
	}
	if col.TypeName != nil {
		switch strings.ToUpper(col.TypeName.Name) {
		case "BLOB", "TINYBLOB", "MEDIUMBLOB", "LONGBLOB",
			"TEXT", "TINYTEXT", "MEDIUMTEXT", "LONGTEXT",
			"JSON", "GEOMETRY", "SERIAL":
			return false
		}
	}
	return true
}
