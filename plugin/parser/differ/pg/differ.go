// Package pg provides the PostgreSQL differ plugin.
package pg

import (
	"bytes"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/plugin/parser"
	"github.com/bytebase/bytebase/plugin/parser/ast"
	"github.com/bytebase/bytebase/plugin/parser/differ"
)

var (
	_ differ.SchemaDiffer = (*SchemaDiffer)(nil)
)

func init() {
	differ.Register(parser.Postgres, &SchemaDiffer{})
}

// SchemaDiffer it the parser for PostgreSQL dialect.
type SchemaDiffer struct {
}

// SchemaDiff computes the schema differences between old and new schema.
func (*SchemaDiffer) SchemaDiff(oldStmt, newStmt string) (string, error) {
	oldNodes, err := parser.Parse(parser.Postgres, parser.ParseContext{}, oldStmt)
	if err != nil {
		return "", errors.Wrapf(err, "failed to parse old statement %q", oldStmt)
	}
	newNodes, err := parser.Parse(parser.Postgres, parser.ParseContext{}, newStmt)
	if err != nil {
		return "", errors.Wrapf(err, "failed to parse new statement %q", newStmt)
	}

	oldTables := make(map[string]*ast.CreateTableStmt)
	for _, node := range oldNodes {
		if n, ok := node.(*ast.CreateTableStmt); ok {
			oldTables[n.Name.Name] = n
		}
	}

	var diffs []ast.Node
	for _, node := range newNodes {
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
