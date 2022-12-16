package store

import (
	"context"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/plugin/advisor/catalog"
	advisorDB "github.com/bytebase/bytebase/plugin/advisor/db"
	"github.com/bytebase/bytebase/plugin/db"
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
func (s *Store) NewCatalog(ctx context.Context, databaseID int, engineType db.Type) (catalog.Catalog, error) {
	databaseMeta, err := s.GetDBSchema(ctx, databaseID)
	if err != nil {
		return nil, err
	}
	if databaseMeta == nil {
		return nil, nil
	}

	var databaseSchema storepb.DatabaseMetadata
	if err := protojson.Unmarshal([]byte(databaseMeta.Metadata), &databaseSchema); err != nil {
		return nil, err
	}

	dbType, err := advisorDB.ConvertToAdvisorDBType(string(engineType))
	if err != nil {
		return nil, err
	}

	c := &Catalog{}
	c.Finder = catalog.NewFinder(&databaseSchema, &catalog.FinderContext{CheckIntegrity: true, EngineType: dbType})
	return c, nil
}

// GetFinder implements the catalog.Catalog interface.
func (c *Catalog) GetFinder() *catalog.Finder {
	return c.Finder
}
