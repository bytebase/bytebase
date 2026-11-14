// Package mysql implements the SQL advisor rules for MySQL.
package oceanbase

import (
	"strings"

	"github.com/bytebase/omni/oceanbase/ast"
	"github.com/bytebase/omni/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

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

// contentStartIndex returns the offset of the first non-whitespace byte of text,
// which is where a statement's own text begins after any leading blank lines.
func contentStartIndex(text string) int {
	for i, c := range text {
		if c != ' ' && c != '\t' && c != '\n' && c != '\r' {
			return i
		}
	}
	return 0
}

// columnFindingPosition is where a finding's advice starts: the statement's first
// line plus the line of the finding's anchor within it.
func columnFindingPosition(stmt base.ParsedStatement, f v1review.Finding) *storepb.Position {
	return base.ConvertANTLRLineToPosition(stmt.BaseLine() + int(findingLine(stmt.Text, f)))
}

// statementPosition is where advice about a whole statement starts: the line
// of the statement's first non-blank character.
func statementPosition(stmt base.ParsedStatement) *storepb.Position {
	return columnFindingPosition(stmt, v1review.Finding{})
}

// v1reviewSchema returns the snapshot md holds for an omni rule, an empty
// one when md is nil.
func v1reviewSchema(md *model.DatabaseMetadata) v1review.Schema {
	if md == nil {
		return v1review.Schema{}
	}
	return v1review.Schema{Metadata: md.GetProto(), ObjectCaseSensitive: md.GetIsObjectCaseSensitive()}
}
