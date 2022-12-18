package store

import (
	"context"

	"github.com/bytebase/bytebase/plugin/advisor/catalog"
	advisorDB "github.com/bytebase/bytebase/plugin/advisor/db"
	"github.com/bytebase/bytebase/plugin/db"
)

var (
	_ catalog.Catalog = (*Catalog)(nil)
)

// Catalog is the database catalog.
type Catalog struct {
	Finder *catalog.Finder
}

// NewCatalog creates a new database catalog.
func (s *Store) NewCatalog(ctx context.Context, databaseID int, engineType db.Type) (catalog.Catalog, error) {
	databaseMeta, err := s.GetDBSchema(ctx, databaseID)
	if err != nil {
		return nil, err
	}
	if databaseMeta == nil {
		return nil, nil
	}

	dbType, err := advisorDB.ConvertToAdvisorDBType(string(engineType))
	if err != nil {
		return nil, err
	}

	c := &Catalog{}
	c.Finder = catalog.NewFinder(databaseMeta.Metadata, &catalog.FinderContext{CheckIntegrity: true, EngineType: dbType})
	return c, nil
}

// GetFinder implements the catalog.Catalog interface.
func (c *Catalog) GetFinder() *catalog.Finder {
	return c.Finder
}
