// Package catalog provides API definition for catalog service.
package catalog

import (
	"google.golang.org/protobuf/proto"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store/model"
)

// NewCatalogWithMetadata creates original and final database catalogs from schema metadata.
// OriginalMetadata is DatabaseMetadata (read-only), FinalCatalog is DatabaseState (mutable for walk-through).
func NewCatalogWithMetadata(metadata *storepb.DatabaseSchemaMetadata, engineType storepb.Engine, isCaseSensitive bool) (originalMetadata *model.DatabaseMetadata, final *DatabaseState, err error) {
	// Create original metadata from original metadata as DatabaseMetadata (read-only)
	originalSchema := model.NewDatabaseSchema(metadata, nil, nil, engineType, isCaseSensitive)
	originalMetadata = originalSchema.GetDatabaseMetadata()

	// Clone metadata for final to avoid modifying the original
	clonedMetadata := proto.CloneOf(metadata)

	// Create final as DatabaseState (mutable for walk-through)
	final = NewDatabaseState(clonedMetadata, !isCaseSensitive, engineType)

	return originalMetadata, final, nil
}

// ToDatabaseState converts DatabaseMetadata to DatabaseState for internal use.
// DatabaseState is used internally by walk-through implementations and advisor rules.
// This is a compatibility helper that allows gradual migration of internal code.
func ToDatabaseState(d *model.DatabaseMetadata, engineType storepb.Engine) *DatabaseState {
	return NewDatabaseState(d.GetProto(), !d.GetIsObjectCaseSensitive(), engineType)
}
