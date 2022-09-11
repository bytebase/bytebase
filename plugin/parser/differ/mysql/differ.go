// Package mysql provides the MySQL differ plugin.
package mysql

import (
	"bytes"

	"github.com/pingcap/tidb/parser/ast"
	"github.com/pingcap/tidb/parser/format"
	"github.com/pingcap/tidb/parser/model"
)

// columnMap maps column name to columnDef.
type columnMap map[string]*ast.ColumnDef

// tableColumnMap maps table name to columnMap.
type tableColumnMap map[string]columnMap

// set sets the columnDef of the `tableName`.`columnName`.
func (m tableColumnMap) set(tableName, columnName string, columnDef *ast.ColumnDef) {
	if _, ok := m[tableName]; !ok {
		m[tableName] = make(columnMap)
	}
	m[tableName][columnName] = columnDef
}

// get gets the columnDef of the `tableName`.`columnName`.
func (m tableColumnMap) get(tableName, columnName string) (*ast.ColumnDef, bool) {
	if _, ok := m[tableName]; !ok {
		return nil, false
	}
	columnDef, ok := m[tableName][columnName]
	return columnDef, ok
}

// SchemaDiff returns the schema diff.
func SchemaDiff(old, new []ast.StmtNode) (string, error) {
	oldTableMap := make(map[string]*ast.CreateTableStmt)
	oldColumnMap := make(tableColumnMap)
	diff := make([]ast.Node, 0)
	for _, node := range old {
		switch stmt := node.(type) {
		case *ast.CreateTableStmt:
			tableName := stmt.Table.Name.String()
			oldTableMap[tableName] = stmt
			for _, column := range stmt.Cols {
				columnName := column.Name.Name.String()
				oldColumnMap.set(tableName, columnName, column)
			}
		default:
		}
	}

	newColumnMap := make(tableColumnMap)
	newTableMap := make(map[string]*ast.CreateTableStmt)
	for _, node := range new {
		switch stmt := node.(type) {
		case *ast.CreateTableStmt:
			tableName := stmt.Table.Name.String()
			newTableMap[tableName] = stmt
			for _, column := range stmt.Cols {
				columnName := column.Name.Name.String()
				newColumnMap.set(tableName, columnName, column)
			}
		default:
		}
	}

	for tableName, stmt := range newTableMap {
		if _, ok := oldTableMap[tableName]; !ok {
			stmt.IfNotExists = true
			diff = append(diff, stmt)
			continue
		}
		var alterTableAddColumnSpecs []*ast.AlterTableSpec
		for columnName, columnDef := range newColumnMap[tableName] {
			if _, ok := oldColumnMap.get(tableName, columnName); !ok {
				alterTableAddColumnSpecs = append(alterTableAddColumnSpecs, &ast.AlterTableSpec{
					Tp:         ast.AlterTableAddColumns,
					NewColumns: []*ast.ColumnDef{columnDef},
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
	}

	deparse := func(nodes []ast.Node) (string, error) {
		var buf bytes.Buffer
		for _, node := range nodes {
			if err := node.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &buf)); err != nil {
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
