// Package catalog provides API definition for catalog service.
package catalog

import (
	"errors"

	"google.golang.org/protobuf/proto"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store/model"
)

// NewCatalogWithMetadata creates original and final database catalogs from schema metadata.
// Both are DatabaseMetadata - original is the starting state, final is mutable for walk-through.
func NewCatalogWithMetadata(metadata *storepb.DatabaseSchemaMetadata, engineType storepb.Engine, isObjectCaseSensitive bool) (originalMetadata *model.DatabaseMetadata, final *model.DatabaseMetadata, err error) {
	// Determine detail case sensitivity based on engine type
	isDetailCaseSensitive := getIsDetailCaseSensitive(engineType)

	// Create original metadata as read-only
	originalMetadata = model.NewDatabaseMetadata(metadata, isObjectCaseSensitive, isDetailCaseSensitive)

	// Clone metadata for final to avoid modifying the original
	cloned := proto.Clone(metadata)
	clonedMetadata, ok := cloned.(*storepb.DatabaseSchemaMetadata)
	if !ok {
		return nil, nil, errors.New("failed to clone database schema metadata")
	}

	// Create final as mutable for walk-through
	final = model.NewDatabaseMetadata(clonedMetadata, isObjectCaseSensitive, isDetailCaseSensitive)

	return originalMetadata, final, nil
}

// getIsDetailCaseSensitive determines if detail names (columns, indexes) are case-sensitive
// based on the database engine type.
func getIsDetailCaseSensitive(engine storepb.Engine) bool {
	switch engine {
	case storepb.Engine_MYSQL, storepb.Engine_MARIADB, storepb.Engine_TIDB, storepb.Engine_MSSQL, storepb.Engine_OCEANBASE:
		return false
	default:
		return true
	}
}
