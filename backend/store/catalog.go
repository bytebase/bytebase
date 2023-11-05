package store

import (
	"context"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/catalog"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	_ catalog.Catalog = (*Catalog)(nil)
)

// Catalog is the database catalog.
type Catalog struct {
	Finder *catalog.Finder
}

// NewCatalog creates a new database catalog.
func (s *Store) NewCatalog(ctx context.Context, databaseID int, engineType storepb.Engine, ignoreCaseSensitive bool, syntaxMode advisor.SyntaxMode) (catalog.Catalog, error) {
	c := &Catalog{}

	if syntaxMode == advisor.SyntaxModeSDL {
		return NewEmptyCatalog(engineType)
	}

	databaseMeta, err := s.GetDBSchema(ctx, databaseID)
	if err != nil {
		return nil, err
	}
	if databaseMeta == nil {
		return nil, nil
	}

	c.Finder = catalog.NewFinder(databaseMeta.GetMetadata(), &catalog.FinderContext{CheckIntegrity: true, EngineType: engineType, IgnoreCaseSensitive: ignoreCaseSensitive})
	return c, nil
}

// GetFinder implements the catalog.Catalog interface.
func (c *Catalog) GetFinder() *catalog.Finder {
	return c.Finder
}

// NewEmptyCatalog creates a new empty database catalog.
func NewEmptyCatalog(engineType storepb.Engine) (catalog.Catalog, error) {
	return &Catalog{
		catalog.NewEmptyFinder(&catalog.FinderContext{CheckIntegrity: false, EngineType: engineType, IgnoreCaseSensitive: false}),
	}, nil
}
