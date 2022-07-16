package store

import (
	"context"
	"sort"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/advisor/catalog"
)

var (
	_ catalog.Catalog = (*Catalog)(nil)
)

// Catalog is the database catalog.
type Catalog struct {
	databaseID *int
	store      *Store
	mode       common.ReleaseMode
}

// NewCatalog creates a new database catalog.
func NewCatalog(databaseID *int, store *Store, mode common.ReleaseMode) *Catalog {
	return &Catalog{
		databaseID: databaseID,
		store:      store,
		mode:       mode,
	}
}

// FindIndex finds the index by IndexFind. Implement the catalog.Catalog interface.
func (c *Catalog) FindIndex(ctx context.Context, find *catalog.IndexFind) (*catalog.Index, error) {
	table, err := c.store.GetTable(ctx, &api.TableFind{
		DatabaseID: c.databaseID,
		Name:       &find.TableName,
	})
	if err != nil {
		return nil, err
	}
	if table == nil {
		return nil, nil
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

	return &catalog.Index{
		Name:              indexList[0].Name,
		TableName:         table.Name,
		Type:              indexList[0].Type,
		Unique:            indexList[0].Unique,
		Primary:           indexList[0].Primary,
		ColumnExpressions: columnExpressions,
	}, nil
}
