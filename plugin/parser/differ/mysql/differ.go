// Package mysql provides the MySQL differ plugin.
package mysql

import (
	"bytes"

	"github.com/pingcap/tidb/parser/ast"
	"github.com/pingcap/tidb/parser/format"
	"github.com/pingcap/tidb/parser/model"
)

// SchemaDiff returns the schema diff.
func SchemaDiff(old, new []ast.StmtNode) (string, error) {
	var diff []ast.Node
	oldTableMap := buildTableMap(old)

	for _, node := range new {
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
				if !isTwoColumnsSame(oldColumnDef, columnDef) {
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

// isTwoColumnsSame returns true if the two columns are the same.
func isTwoColumnsSame(old, new *ast.ColumnDef) bool {
	return old.Tp.String() == new.Tp.String()
}
