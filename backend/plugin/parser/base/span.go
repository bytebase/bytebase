package base

import (
	"context"
	"strings"

	"github.com/bytebase/bytebase/backend/store/model"
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

// Pretty returns the pretty string of the resource.
func (c ColumnResource) String() string {
	var list []string
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
