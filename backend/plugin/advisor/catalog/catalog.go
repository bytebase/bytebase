// Package catalog provides API definition for catalog service.
package catalog

import (
	"context"

	"google.golang.org/protobuf/proto"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/store/model"
)

// NewCatalogWithMetadata creates origin and final database metadata from schema metadata.
// Uses proto cloning to create independent copies.
func NewCatalogWithMetadata(metadata *storepb.DatabaseSchemaMetadata, engineType storepb.Engine, isCaseSensitive bool) (origin *model.DatabaseMetadata, final *model.DatabaseMetadata, err error) {
	// Create origin from original metadata
	originSchema := model.NewDatabaseSchema(metadata, nil, nil, engineType, isCaseSensitive)
	origin = originSchema.GetDatabaseMetadata()

	// Clone metadata for final
	clonedMetadata := proto.CloneOf(metadata)

	finalSchema := model.NewDatabaseSchema(clonedMetadata, nil, nil, engineType, isCaseSensitive)
	final = finalSchema.GetDatabaseMetadata()

	return origin, final, nil
}

// ToDatabaseState converts DatabaseMetadata to DatabaseState for use in advisor rules.
// This is a compatibility helper during the migration from DatabaseState to DatabaseMetadata.
func ToDatabaseState(d *model.DatabaseMetadata, engineType storepb.Engine) *DatabaseState {
	return NewDatabaseState(d.GetProto(), !d.GetIsObjectCaseSensitive(), engineType)
}

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
	ignoreCaseSensitive := !isCaseSensitive
	origin = NewDatabaseState(dbMetadata, ignoreCaseSensitive, engineType)
	final = NewDatabaseState(dbMetadata, ignoreCaseSensitive, engineType)
	return origin, final, nil
}
