package catalog

import (
	"context"
)

// Catalog is the service for catalog.
type Catalog interface {
	FindIndex(ctx context.Context, find *IndexFind) (*Index, error)
}

// Index is the API message for an index.
type Index struct {
	Name              string
	TableName         string
	Type              string
	Unique            bool
	Primary           bool
	ColumnExpressions []string
}

// IndexFind is the API message for find index
type IndexFind struct {
	TableName string
	IndexName string
}
