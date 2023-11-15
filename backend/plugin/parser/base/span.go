package base

import (
	"context"
	"strings"

	"github.com/bytebase/bytebase/backend/store/model"
)

type SourceColumnSet map[ColumnResource]bool

// MergeSourceColumnSet merges two source column maps, returns true if there is difference.
func MergeSourceColumnSet(m, n SourceColumnSet) (SourceColumnSet, bool) {
	r := make(SourceColumnSet)
	for k := range m {
		r[k] = true
	}
	for k := range n {
		if _, ok := r[k]; !ok {
			r[k] = true
		}
	}

	return r, len(r) != len(m)
}

// QuerySpan is the span for a query.
type QuerySpan struct {
	// Results are the result columns of a query span.
	Results []QuerySpanResult
	// SourceColumns are the source columns contributing to the span.
	SourceColumns SourceColumnSet
}

// QuerySpanResult is the result column of a query span.
type QuerySpanResult struct {
	// Name is the result name of a query.
	Name string
	// SourceColumns are the source columns contributing to the span result.
	SourceColumns SourceColumnSet
}

// ColumnResource is the resource key for a column.
type ColumnResource struct {
	// Server is the normalized server name, it's empty if the column comes from the connected server.
	Server string
	// Database is the normalized database name, it should not be empty.
	Database string
	// Schema is the normalized schema name, it should not be empty for the engines that support schema, and should be empty for the engines that don't support schema.
	Schema string
	// Table is the normalized table name, it should not be empty.
	Table string
	// Column is the normalized column name, it should not be empty.
	Column string
}

type TableSource interface {
	// Interface guard to forbid other types outside this package to implement this interface.
	isTableSource()

	GetTableName() string
	GetSchemaName() string
	GetDatabaseName() string
	GetServerName() string
}

// baseTableSource is the base implementation table source.
type baseTableSource struct {
}

// isTableSource implements the TableSource interface.
func (baseTableSource) isTableSource() {}

// PseudoTable is the resource of table, it's useful for some pseudo/temporary tables likes CTE, AS.
type PseudoTable struct {
	baseTableSource

	// Name is the normalized table name.
	Name string

	// Columns are the columns of the table.
	Columns []QuerySpanResult
}

func (p *PseudoTable) GetTableName() string {
	return p.Name
}

func (*PseudoTable) GetSchemaName() string {
	return ""
}

func (*PseudoTable) GetDatabaseName() string {
	return ""
}

func (*PseudoTable) GetServerName() string {
	return ""
}

func (p *PseudoTable) SetColumnName(i int, name string) {
	p.Columns[i].Name = name
}

// PhysicalTable is the resource of a physical table.
type PhysicalTable struct {
	baseTableSource

	// Server is the normalized server name, it's empty if the column comes from the connected server.
	Server string
	// Database is the normalized database name, it should not be empty.
	Database string
	// Schema is the normalized schema name, it should not be empty for the engines that support schema, and should be empty for the engines that don't support schema.
	Schema string
	// Name is the normalized table name, it should not be empty.
	Name string
	// Columns are the columns of the table.
	Columns []string
}

func (p *PhysicalTable) GetTableName() string {
	return p.Name
}

func (p *PhysicalTable) GetSchemaName() string {
	return p.Schema
}

func (p *PhysicalTable) GetDatabaseName() string {
	return p.Database
}

func (p *PhysicalTable) GetServerName() string {
	return p.Server
}

// String returns the string format of the column resource.
func (c ColumnResource) String() string {
	var list []string
	if c.Server != "" {
		list = append(list, c.Server)
	}
	if c.Database != "" {
		list = append(list, c.Database)
	}
	if c.Schema != "" {
		list = append(list, c.Schema)
	}
	if c.Table != "" {
		list = append(list, c.Table)
	}
	if c.Column != "" {
		list = append(list, c.Column)
	}
	return strings.Join(list, ".")
}

// GetDatabaseMetadataFunc is the function to get database metadata.
type GetDatabaseMetadataFunc func(context.Context, string) (*model.DatabaseMetadata, error)
