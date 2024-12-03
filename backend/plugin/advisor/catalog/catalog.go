// Package catalog provides API definition for catalog service.
package catalog

import (
	"context"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/component/sheet"
	"github.com/bytebase/bytebase/backend/store"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// Catalog is the database catalog.
type Catalog struct {
	Finder *Finder
}

// NewCatalog creates a new database catalog.
func NewCatalog(
	ctx context.Context,
	s *store.Store,
	sheetManager *sheet.Manager,
	databaseID int,
	engineType storepb.Engine,
	ignoreCaseSensitive bool,
	// overrideDatabaseMetadata is used to override the database metadata instead of fetching from the store.
	overrideDatabaseMetadata *storepb.DatabaseSchemaMetadata,
	// baseStatement is the base statement to used to establish additional database state by walking through its AST.
	baseStatement *string,
) (*Catalog, error) {
	c := &Catalog{}

	dbMetadata := overrideDatabaseMetadata
	if dbMetadata == nil {
		databaseMeta, err := s.GetDBSchema(ctx, databaseID)
		if err != nil {
			return nil, err
		}
		if databaseMeta == nil {
			return nil, nil
		}
		dbMetadata = databaseMeta.GetMetadata()
	}
	c.Finder = NewFinder(dbMetadata, &FinderContext{CheckIntegrity: true, EngineType: engineType, IgnoreCaseSensitive: ignoreCaseSensitive})
	if baseStatement != nil {
		asts, _ := sheetManager.GetASTsForChecks(engineType, *baseStatement)
		// Walk through the base statement to establish additional database state.
		if err := c.Finder.WalkThrough(asts); err != nil {
			return nil, errors.Wrap(err, "failed to walk through base statement")
		}
	}
	return c, nil
}

// GetFinder implements the catalog.Catalog interface.
func (c *Catalog) GetFinder() *Finder {
	return c.Finder
}

// NewEmptyCatalog creates a new empty database catalog.
func NewEmptyCatalog(engineType storepb.Engine) (*Catalog, error) {
	return &Catalog{
		NewEmptyFinder(&FinderContext{CheckIntegrity: false, EngineType: engineType, IgnoreCaseSensitive: false}),
	}, nil
}
