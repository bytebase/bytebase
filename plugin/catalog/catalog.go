package catalog

import (
	"context"
	"sort"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/store"
)

// Catalog is the database catalog.
type Catalog struct {
	// catalog special fields.
	databaseID *int
	store      *store.Store
}

// Service is the service for catalog.
type Service interface {
	FindIndex(ctx context.Context, find *IndexFind) (*Index, error)
}

// Index is the API message for an index.
type Index struct {
	Name              string
	TableName         string
	Type              string
	Unique            bool
	ColumnExpressions []string
}

// IndexFind is the API message for find index
type IndexFind struct {
	TableName string
	IndexName string
}

// NewService creates a new instance of Catalog
func NewService(databaseID *int, store *store.Store) *Catalog {
	return &Catalog{
		databaseID: databaseID,
		store:      store,
	}
}

// FindIndex finds the index by IndexFind
func (c *Catalog) FindIndex(ctx context.Context, find *IndexFind) (*Index, error) {
	table, err := c.store.GetTable(ctx, &api.TableFind{
		DatabaseID: c.databaseID,
		Name:       &find.TableName,
	})
	if err != nil {
		return nil, err
	}

	indexList, err := c.store.FindIndex(ctx, &api.IndexFind{
		DatabaseID: c.databaseID,
		TableID:    &table.ID,
		Name:       &find.IndexName,
	})
	if err != nil {
		return nil, err
	}
	if len(indexList) == 0 {
		return nil, nil
	}

	sort.Slice(indexList, func(i, j int) bool {
		return indexList[i].Position < indexList[j].Position
	})

	var columnExpressions []string
	for _, index := range indexList {
		columnExpressions = append(columnExpressions, index.Expression)
	}

	return &Index{
		Name:              indexList[0].Name,
		TableName:         table.Name,
		Type:              indexList[0].Type,
		Unique:            indexList[0].Unique,
		ColumnExpressions: columnExpressions,
	}, nil
}
