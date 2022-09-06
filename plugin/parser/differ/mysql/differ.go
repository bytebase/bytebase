// Package mysql provides the MySQL differ plugin.
package mysql

import (
	"bytes"

	"github.com/pingcap/tidb/parser/ast"
	"github.com/pingcap/tidb/parser/format"
)

// Differ is the differ plugin for MySQL.
type Differ struct {
	oldTables map[string]*ast.CreateTableStmt
	tableDiff []ast.Node
}

// NewDiffer returns a new differ.
func NewDiffer() *Differ {
	return &Differ{
		oldTables: make(map[string]*ast.CreateTableStmt),
	}
}

// SchemaDiff returns the schema diff.
func (d *Differ) SchemaDiff(old, new []ast.StmtNode) (string, error) {
	for _, node := range old {
		if stmt, ok := node.(*ast.CreateTableStmt); ok {
			d.oldTables[stmt.Table.Name.String()] = stmt
		}
	}
	for _, node := range new {
		if stmt, ok := node.(*ast.CreateTableStmt); ok {
			if _, ok := d.oldTables[stmt.Table.Name.String()]; !ok {
				d.tableDiff = append(d.tableDiff, node)
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
	return deparse(d.tableDiff)
}
