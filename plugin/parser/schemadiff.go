package parser

import (
	"bytes"

	"github.com/bytebase/bytebase/plugin/parser/ast"
)

// SchemaDiff represents the schema diff between old and new schema.
type SchemaDiff struct {
	// Diffs is an ordered list of diffs for schema changes.
	Diffs []Diff
}

// Diff represents a schema change of tables, columns and constraints.
type Diff interface {
	// String returns the DDL to create the schema change.
	String() string
}

// CreateTableDiff represents a schema change of a new table.
type CreateTableDiff struct {
	// Name is the name of the new table.
	Name string
	// Text is the original DDL of the new table.
	Text string
}

// String returns the DDL to create the new table.
func (d *CreateTableDiff) String() string {
	return d.Text
}

// String returns the migration scripts for the schema diff.
func (d *SchemaDiff) String() string {
	var buf bytes.Buffer
	for _, diff := range d.Diffs {
		_, _ = buf.WriteString(diff.String())
		_, _ = buf.WriteString("\n")
	}
	return buf.String()
}

// ComputeDiff computes the schema differences between old and new schema.
func ComputeDiff(old, new []ast.Node) (*SchemaDiff, error) {
	oldTables := make(map[string]*ast.CreateTableStmt)
	for _, node := range old {
		if n, ok := node.(*ast.CreateTableStmt); ok {
			oldTables[n.Name.Name] = n
		}
	}

	var diffs []Diff
	for _, node := range new {
		if newTable, ok := node.(*ast.CreateTableStmt); ok {
			_, hasTable := oldTables[newTable.Name.Name]
			if !hasTable {
				diffs = append(diffs,
					&CreateTableDiff{
						Name: newTable.Name.Name,
						Text: newTable.Text(),
					},
				)
				continue
			}
		}
	}

	return &SchemaDiff{
		Diffs: diffs,
	}, nil
}
