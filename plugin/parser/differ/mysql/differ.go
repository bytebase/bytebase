// Package mysql provides the MySQL differ plugin.
package mysql

import (
	"bytes"

	"github.com/pingcap/tidb/parser/ast"
	"github.com/pingcap/tidb/parser/format"
	"github.com/pingcap/tidb/parser/model"
)

type columnKey struct {
	tableName  string
	columnName string
}

// SchemaDiff returns the schema diff.
func SchemaDiff(old, new []ast.StmtNode) (string, error) {
	oldTableMap := make(map[string]*ast.CreateTableStmt)
	var diff []ast.Node
	for _, node := range old {
		switch stmt := node.(type) {
		case *ast.CreateTableStmt:
			tableName := stmt.Table.Name.String()
			oldTableMap[tableName] = stmt
		default:
		}
	}

	newTableMap := make(map[string]*ast.CreateTableStmt)
	for _, node := range new {
		switch stmt := node.(type) {
		case *ast.CreateTableStmt:
			tableName := stmt.Table.Name.String()
			newTableMap[tableName] = stmt
		default:
		}
	}

	for tableName, newStmt := range newTableMap {
		oldStmt, ok := oldTableMap[tableName]
		if !ok {
			newStmt.IfNotExists = true
			diff = append(diff, newStmt)
			continue
		}

		var alterTableAddColumnSpecs []*ast.AlterTableSpec
		var oldColumnMap = make(map[columnKey]*ast.ColumnDef)
		for _, oldColumnDef := range oldStmt.Cols {
			oldColumnMap[columnKey{
				tableName:  tableName,
				columnName: oldColumnDef.Name.Name.String(),
			}] = oldColumnDef
		}

		for _, columnDef := range newStmt.Cols {
			columnName := columnDef.Name.Name.String()
			if _, ok := oldColumnMap[columnKey{tableName, columnName}]; !ok {
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
