// Package catalog provides API definition for catalog service.
package catalog

import (
	"context"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

// NewCatalog creates origin and final database catalog states.
func NewCatalog(ctx context.Context, s *store.Store, instanceID, databaseName string, engineType storepb.Engine, isCaseSensitive bool, overrideDatabaseMetadata *storepb.DatabaseSchemaMetadata) (origin *DatabaseState, final *DatabaseState, err error) {
	dbMetadata := overrideDatabaseMetadata
	if dbMetadata == nil {
		databaseMeta, err := s.GetDBSchema(ctx, &store.FindDBSchemaMessage{
			InstanceID:   instanceID,
			DatabaseName: databaseName,
		})
		if err != nil {
			return nil, nil, err
		}
		if databaseMeta == nil {
			return nil, nil, nil
		}
		dbMetadata = databaseMeta.GetMetadata()
	}
	finderCtx := &FinderContext{CheckIntegrity: true, EngineType: engineType, IgnoreCaseSensitive: !isCaseSensitive}
	origin = NewDatabaseState(dbMetadata, finderCtx)
	final = NewDatabaseState(dbMetadata, finderCtx)
	return origin, final, nil
}
