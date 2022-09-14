package parser

import (
	"bytes"

	"github.com/bytebase/bytebase/plugin/parser/ast"
)

// SchemaDiff computes the schema differences between old and new schema.
func SchemaDiff(old, new []ast.Node) (string, error) {
	oldTables := make(map[string]*ast.CreateTableStmt)
	for _, node := range old {
		if n, ok := node.(*ast.CreateTableStmt); ok {
			oldTables[n.Name.Name] = n
		}
	}

	var diffs []ast.Node
	for _, node := range new {
		if newTable, ok := node.(*ast.CreateTableStmt); ok {
			_, hasTable := oldTables[newTable.Name.Name]
			if !hasTable {
				diffs = append(diffs, node)
				continue
			}
		}
	}

	// NOTE: Due to limitation of current deparse implementation, we directly
	// generate the final DDLs here instead of returning []ast.Node to the caller.
	deparse := func(nodes []ast.Node) string {
		var buf bytes.Buffer
		for _, node := range nodes {
			if n, ok := node.(*ast.CreateTableStmt); ok {
				_, _ = buf.WriteString(n.Text())
				_, _ = buf.WriteString("\n")
			}
		}
		return buf.String()
	}

	return deparse(diffs), nil
}
