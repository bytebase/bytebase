// Package catalog provides API definition for catalog service.
package catalog

import (
	"context"

	"github.com/bytebase/bytebase/backend/store"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// Catalog is the database catalog.
type Catalog struct {
	Finder *Finder
}

// NewCatalog creates a new database catalog.
func NewCatalog(ctx context.Context, s *store.Store, instanceID, databaseName string, engineType storepb.Engine, isCaseSensitive bool, overrideDatabaseMetadata *storepb.DatabaseSchemaMetadata) (*Catalog, error) {
	c := &Catalog{}

	dbMetadata := overrideDatabaseMetadata
	if dbMetadata == nil {
		databaseMeta, err := s.GetDBSchema(ctx, instanceID, databaseName)
		if err != nil {
			return nil, err
		}
		if databaseMeta == nil {
			return nil, nil
		}
		dbMetadata = databaseMeta.GetMetadata()
	}
	c.Finder = NewFinder(dbMetadata, &FinderContext{CheckIntegrity: true, EngineType: engineType, IgnoreCaseSensitive: !isCaseSensitive})
	return c, nil
}

// GetFinder implements the catalog.Catalog interface.
func (c *Catalog) GetFinder() *Finder {
	return c.Finder
}
