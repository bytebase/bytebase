// Package mysql provides the MySQL differ plugin.
package mysql

import (
	"bytes"
	"sort"

	"github.com/pingcap/tidb/parser"
	"github.com/pingcap/tidb/parser/ast"
	"github.com/pingcap/tidb/parser/format"
	"github.com/pingcap/tidb/parser/model"
	"github.com/pkg/errors"

	bbparser "github.com/bytebase/bytebase/plugin/parser"

	"github.com/bytebase/bytebase/plugin/parser/differ"

	// Register pingcap parser driver.
	_ "github.com/pingcap/tidb/types/parser_driver"
)

var (
	_ differ.SchemaDiffer = (*SchemaDiffer)(nil)
)

func init() {
	differ.Register(bbparser.MySQL, &SchemaDiffer{})
	differ.Register(bbparser.TiDB, &SchemaDiffer{})
}

// SchemaDiffer it the parser for MySQL dialect.
type SchemaDiffer struct {
}

// SchemaDiff returns the schema diff.
func (*SchemaDiffer) SchemaDiff(oldStmt, newStmt string) (string, error) {
	oldNodes, _, err := parser.New().Parse(oldStmt, "", "")
	if err != nil {
		return "", errors.Wrapf(err, "failed to parse old statement %q", oldStmt)
	}
	newNodes, _, err := parser.New().Parse(newStmt, "", "")
	if err != nil {
		return "", errors.Wrapf(err, "failed to parse old statement %q", oldStmt)
	}

	var diff []ast.Node
	oldTableMap := buildTableMap(oldNodes)

	for _, node := range newNodes {
		switch newStmt := node.(type) {
		case *ast.CreateTableStmt:
			tableName := newStmt.Table.Name.String()
			oldStmt, ok := oldTableMap[tableName]
			if !ok {
				stmt := *newStmt
				stmt.IfNotExists = true
				diff = append(diff, &stmt)
				continue
			}

			var alterTableAddColumnSpecs []*ast.AlterTableSpec
			var alterTableModifyColumnSpecs []*ast.AlterTableSpec
			var oldColumnMap = make(map[string]*ast.ColumnDef)
			for _, oldColumnDef := range oldStmt.Cols {
				oldColumnName := oldColumnDef.Name.Name.String()
				oldColumnMap[oldColumnName] = oldColumnDef
			}

			for _, columnDef := range newStmt.Cols {
				newColumnName := columnDef.Name.Name.String()
				oldColumnDef, ok := oldColumnMap[newColumnName]
				if !ok {
					alterTableAddColumnSpecs = append(alterTableAddColumnSpecs, &ast.AlterTableSpec{
						Tp:         ast.AlterTableAddColumns,
						NewColumns: []*ast.ColumnDef{columnDef},
					})
					continue
				}
				// We need to compare the two column definitions.
				if !isColumnEqual(oldColumnDef, columnDef) {
					alterTableModifyColumnSpecs = append(alterTableModifyColumnSpecs, &ast.AlterTableSpec{
						Tp:         ast.AlterTableModifyColumn,
						NewColumns: []*ast.ColumnDef{columnDef},
						Position:   &ast.ColumnPosition{Tp: ast.ColumnPositionNone},
					})
				}
			}

			if len(alterTableAddColumnSpecs) > 0 {
				diff = append(diff, &ast.AlterTableStmt{
					Table: &ast.TableName{
						Name: model.NewCIStr(tableName),
					},
					Specs: alterTableAddColumnSpecs,
				})
			}
			if len(alterTableModifyColumnSpecs) > 0 {
				diff = append(diff, &ast.AlterTableStmt{
					Table: &ast.TableName{
						Name: model.NewCIStr(tableName),
					},
					Specs: alterTableModifyColumnSpecs,
				})
			}
		default:
		}
	}

	deparse := func(nodes []ast.Node) (string, error) {
		restoreFlags := format.DefaultRestoreFlags | format.RestoreStringWithoutCharset
		var buf bytes.Buffer
		for _, node := range nodes {
			if err := node.Restore(format.NewRestoreCtx(restoreFlags, &buf)); err != nil {
				return "", err
			}
			if _, err := buf.Write([]byte(";\n")); err != nil {
				return "", err
			}
		}
		return buf.String(), nil
	}
	return deparse(diff)
}

// buildTableMap returns a map of table name to create table statements.
func buildTableMap(nodes []ast.StmtNode) map[string]*ast.CreateTableStmt {
	oldTableMap := make(map[string]*ast.CreateTableStmt)
	for _, node := range nodes {
		switch stmt := node.(type) {
		case *ast.CreateTableStmt:
			tableName := stmt.Table.Name.String()
			oldTableMap[tableName] = stmt
		default:
		}
	}
	return oldTableMap
}

// isColumnEqual returns true if definitions of two columns with the same name are the same.
func isColumnEqual(old, new *ast.ColumnDef) bool {
	if !isColumnTypesEqual(old, new) {
		return false
	}
	if !isColumnOptionsEqaul(old.Options, new.Options) {
		return false
	}
	return true
}

func isColumnTypesEqual(old, new *ast.ColumnDef) bool {
	return old.Tp.String() == new.Tp.String()
}

func isColumnOptionsEqaul(old, new []*ast.ColumnOption) bool {
	oldNormalizeOptions := normalizeColumnOptions(old)
	newNormalizeOptions := normalizeColumnOptions(new)
	if len(oldNormalizeOptions) != len(newNormalizeOptions) {
		return false
	}
	for idx, oldOption := range oldNormalizeOptions {
		newOption := newNormalizeOptions[idx]
		if oldOption.Tp != newOption.Tp {
			return false
		}
		// TODO(zp): it's not enough to compare the type for some options.
		switch oldOption.Tp {
		case ast.ColumnOptionComment:
			if oldOption.Expr.(ast.ValueExpr).GetString() != newOption.Expr.(ast.ValueExpr).GetString() {
				return false
			}
		case ast.ColumnOptionDefaultValue:
			if oldOption.Expr.(ast.ValueExpr).GetValue() != newOption.Expr.(ast.ValueExpr).GetValue() {
				return false
			}
		default:
		}
	}
	return true
}

// normalizeColumnOptions normalizes the column options.
// It skips the NULL option, NO option and then order the options by OptionType.
func normalizeColumnOptions(options []*ast.ColumnOption) []*ast.ColumnOption {
	var retOptions []*ast.ColumnOption
	for _, option := range options {
		if option.Tp == ast.ColumnOptionNull || option.Tp == ast.ColumnOptionNoOption {
			continue
		}
		retOptions = append(retOptions, option)
	}
	sort.Slice(retOptions, func(i, j int) bool {
		return retOptions[i].Tp < retOptions[j].Tp
	})
	return retOptions
}
