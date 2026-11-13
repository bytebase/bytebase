// Package mysql implements the SQL advisor rules for MySQL.
package mysql

import (
	"regexp"
	"slices"
	"strings"

	"github.com/bytebase/omni/mysql/ast"
	"github.com/bytebase/omni/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
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

// findingPosition is where a finding's advice starts: the statement's first
// line plus the line of the finding's anchor within it.
func findingPosition(stmt base.ParsedStatement, f v1review.Finding) *storepb.Position {
	return base.ConvertANTLRLineToPosition(stmt.BaseLine() + int(findingLine(stmt.Text, f)))
}

// statementPosition is where advice about a whole statement starts: the line
// of the statement's first non-blank character.
func statementPosition(stmt base.ParsedStatement) *storepb.Position {
	return findingPosition(stmt, v1review.Finding{})
}
