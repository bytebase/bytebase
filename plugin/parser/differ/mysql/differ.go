// Package mysql provides the MySQL differ plugin.
package mysql

import (
	"bytes"

	"github.com/pingcap/tidb/parser/ast"
	"github.com/pingcap/tidb/parser/format"
)

// SchemaDiff returns the schema diff.
func SchemaDiff(old, new []ast.StmtNode) (string, error) {
	oldTableMap := make(map[string]ast.Node)
	tableDiff := make([]ast.Node, 0)
	for _, node := range old {
		if stmt, ok := node.(*ast.CreateTableStmt); ok {
			tableName := stmt.Table.Name.String()
			oldTableMap[tableName] = node
		}
	}
	for _, node := range new {
		if stmt, ok := node.(*ast.CreateTableStmt); ok {
			if _, ok := oldTableMap[stmt.Table.Name.String()]; !ok {
				stmt.IfNotExists = true
				tableDiff = append(tableDiff, node)
			}
		}
	}

	deparse := func(nodes []ast.Node) (string, error) {
		var buf bytes.Buffer
		for _, node := range nodes {
			if err := node.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &buf)); err != nil {
				return "", err
			}
			if _, err := buf.Write([]byte("\n")); err != nil {
				return "", err
			}
		}
		return buf.String(), nil
	}
	return deparse(tableDiff)
}
