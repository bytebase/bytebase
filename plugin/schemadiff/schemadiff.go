package schemadiff

import (
	"bytes"
	"sync"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/plugin/parser"
	"github.com/bytebase/bytebase/plugin/parser/ast"
)

// SchemaDiff represents the schema diff between old and new schema.
type SchemaDiff struct {
	// Diffs is an ordered list of diffs for schema changes.
	Diffs []Diff
}

// String returns the migration scripts for the schema diff.
func (d *SchemaDiff) String() string {
	var buf bytes.Buffer
	for _, diff := range d.Diffs {
		buf.WriteString(diff.String())
		buf.WriteString("\n")
	}
	return buf.String()
}

// todo docstring
type Diff interface {
	// todo docstring
	String() string
}

// todo docstring
type CreateTableDiff struct {
	// todo docstring
	Name string
	// todo docstring
	Text string
}

// todo docstring
func (d *CreateTableDiff) String() string {
	return d.Text
}

// todo docstring
type AddColumnDiff struct {
	differ Differ
	// todo docstring
	TableName string
	// todo docstring
	ColumnName string
	// todo docstring
	ColumnType string
}

// todo docstring
func (d *AddColumnDiff) String() string {
	return d.differ.AddColumn(d.TableName, d.ColumnName, d.ColumnType)
}

// todo docstring
type DropColumnDiff struct {
	differ Differ
	// todo docstring
	TableName string
	// todo docstring
	ColumnName string
}

// todo docstring
func (d *DropColumnDiff) String() string {
	return d.differ.DropColumn(d.TableName, d.ColumnName)
}

// todo docstring
type Differ interface {
	// todo docstring
	AddColumn(tableName, columnName, columnType string) string
	// todo docstring
	DropColumn(tableName, columnName string) string
}

var (
	differMu sync.RWMutex
	differs  = make(map[parser.EngineType]Differ)
)

// Register makes a Differ available by the provided engine type. It panics when
// the Register is called twice for the same name or the Differ is nil.
func Register(engineType parser.EngineType, d Differ) {
	if d == nil {
		panic("schemadiff: register differ is nil")
	}
	differMu.Lock()
	defer differMu.Unlock()
	if _, dup := differs[engineType]; dup {
		panic("schemadiff: register called twice for differ " + engineType)
	}
	differs[engineType] = d
}

// Compute computes the schema diff between old and new schema.
func Compute(engineType parser.EngineType, old, new []ast.Node) (*SchemaDiff, error) {
	differMu.RLock()
	differ, ok := differs[engineType]
	differMu.RUnlock()
	if !ok {
		return nil, errors.Errorf("unknown engine type %v", engineType)
	}

	oldTables := make(map[string]*ast.CreateTableStmt)
	for _, node := range old {
		if n, ok := node.(*ast.CreateTableStmt); ok {
			oldTables[n.Name.Name] = n
		}
	}

	var diffs []Diff
	for _, node := range new {
		if newTable, ok := node.(*ast.CreateTableStmt); ok {
			oldTable, hasTable := oldTables[newTable.Name.Name]
			if !hasTable {
				diffs = append(diffs,
					&CreateTableDiff{
						Name: newTable.Name.Name,
						Text: newTable.Text(),
					},
				)
				continue
			}

			diffs = append(diffs, diffColumns(differ, newTable.Name.Name, oldTable.ColumnList, newTable.ColumnList)...)
		}
	}

	return &SchemaDiff{
		Diffs: diffs,
	}, nil
}

// diffColumns returns a list of diffs between old and new columns of the table.
func diffColumns(differ Differ, tableName string, old, new []*ast.ColumnDef) []Diff {
	oldColumns := make(map[string]struct{})
	for _, c := range old {
		oldColumns[c.ColumnName] = struct{}{}
	}

	newColumns := make(map[string]struct{})
	var columnDiffs []Diff
	for _, newColumn := range new {
		newColumns[newColumn.ColumnName] = struct{}{}
		if _, hasColumn := oldColumns[newColumn.ColumnName]; !hasColumn {
			columnDiffs = append(columnDiffs,
				&AddColumnDiff{
					differ:     differ,
					TableName:  tableName,
					ColumnName: newColumn.ColumnName,
					ColumnType: newColumn.ColumnType,
					// TODO: Support column constraints
				},
			)
		}
	}

	for _, oldColumn := range old {
		if _, hasColumn := newColumns[oldColumn.ColumnName]; !hasColumn {
			columnDiffs = append(columnDiffs,
				&DropColumnDiff{
					differ:     differ,
					TableName:  tableName,
					ColumnName: oldColumn.ColumnName,
				},
			)
		}
	}
	return columnDiffs
}
