package base

import (
	"context"

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// QuerySpan is the span for a query.
type QuerySpan struct {
	// Results are the result columns of a query span.
	Results []*QuerySpanResult
	// SourceColumns are the source columns contributing to the span.
	SourceColumns map[ColumnResource]bool
}

// QuerySpanResult is the result column of a query span.
type QuerySpanResult struct {
	// Name is the result name of a query.
	Name string
	// SourceColumns are the source columns contributing to the span result.
	SourceColumns map[ColumnResource]bool
}

// ColumnResource is the resource key for a column.
type ColumnResource struct {
	Database string
	Schema   string
	Table    string
	Column   string
}

// GetDatabaseMetadataFunc is the function to get database metadata.
type GetDatabaseMetadataFunc func(context.Context, string) (*DatabaseMetadata, error)

// DatabaseMetadata is the metadata for a database.
type DatabaseMetadata struct {
	internal map[string]*SchemaMetadata
}

// NewDatabaseMetadata creates a new database metadata.
func NewDatabaseMetadata(metadata *storepb.DatabaseSchemaMetadata) *DatabaseMetadata {
	databaseMetadata := &DatabaseMetadata{
		internal: make(map[string]*SchemaMetadata),
	}
	for _, schema := range metadata.Schemas {
		schemaMetadata := &SchemaMetadata{
			internal: make(map[string]*TableMetadata),
		}
		for _, table := range schema.Tables {
			tableMetadata := &TableMetadata{
				internal: make(map[string]*storepb.ColumnMetadata),
			}
			for _, column := range table.Columns {
				tableMetadata.internal[column.Name] = column
				tableMetadata.columns = append(tableMetadata.columns, column)
			}
			schemaMetadata.internal[table.Name] = tableMetadata
		}
		databaseMetadata.internal[schema.Name] = schemaMetadata
	}
	return databaseMetadata
}

// GetSchema gets the schema by name.
func (d *DatabaseMetadata) GetSchema(name string) *SchemaMetadata {
	return d.internal[name]
}

// SchemaMetadata is the metadata for a schema.
type SchemaMetadata struct {
	internal map[string]*TableMetadata
}

// GetTable gets the schema by name.
func (s *SchemaMetadata) GetTable(name string) *TableMetadata {
	return s.internal[name]
}

// TableMetadata is the metadata for a table.
type TableMetadata struct {
	internal map[string]*storepb.ColumnMetadata
	columns  []*storepb.ColumnMetadata
}

// GetColumn gets the column by name.
func (t *TableMetadata) GetColumn(name string) *storepb.ColumnMetadata {
	return t.internal[name]
}

// GetColumns gets the columns.
func (t *TableMetadata) GetColumns() []*storepb.ColumnMetadata {
	return t.columns
}
