// Package catalog provides API definition for catalog service.
package catalog

import (
	"google.golang.org/protobuf/proto"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
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

// ToDatabaseState converts DatabaseMetadata to DatabaseState for internal use.
// DatabaseState is used internally by walk-through implementations and advisor rules.
// This is a compatibility helper that allows gradual migration of internal code.
func ToDatabaseState(d *model.DatabaseMetadata, engineType storepb.Engine) *DatabaseState {
	return NewDatabaseState(d.GetProto(), !d.GetIsObjectCaseSensitive(), engineType)
}
