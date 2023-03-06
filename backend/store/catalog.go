package store

import (
	"context"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/catalog"
	advisorDB "github.com/bytebase/bytebase/backend/plugin/advisor/db"
	"github.com/bytebase/bytebase/backend/plugin/db"
)

var (
	_ catalog.Catalog = (*Catalog)(nil)
)

// Catalog is the database catalog.
type Catalog struct {
	Finder *catalog.Finder
}

// NewCatalog creates a new database catalog.
func (s *Store) NewCatalog(ctx context.Context, databaseID int, engineType db.Type, syntaxMode advisor.SyntaxMode) (catalog.Catalog, error) {
	c := &Catalog{}

	dbType, err := advisorDB.ConvertToAdvisorDBType(string(engineType))
	if err != nil {
		return nil, err
	}

	if syntaxMode == advisor.SyntaxModeSDL {
		c.Finder = catalog.NewEmptyFinder(&catalog.FinderContext{CheckIntegrity: true, EngineType: dbType})
		return c, nil
	}

	databaseMeta, err := s.GetDBSchema(ctx, databaseID)
	if err != nil {
		return nil, err
	}
	if databaseMeta == nil {
		return nil, nil
	}

	c.Finder = catalog.NewFinder(databaseMeta.Metadata, &catalog.FinderContext{CheckIntegrity: true, EngineType: dbType})
	return c, nil
}

// GetFinder implements the catalog.Catalog interface.
func (c *Catalog) GetFinder() *catalog.Finder {
	return c.Finder
}
