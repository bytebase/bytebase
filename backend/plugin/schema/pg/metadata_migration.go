package pg

import (
	"context"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/bytebase/omni/pg/catalog"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store/model"
)

// diffMetadataMigration generates the migration between two synced snapshots
// with omni's catalog. PostgreSQL and CockroachDB share it: their metadata is
// PostgreSQL-shaped and omni has no CockroachDB catalog.
func diffMetadataMigration(oldSchema, newSchema *model.DatabaseMetadata) (string, error) {
	var oldProto, newProto *metadatapb.DatabaseSchemaMetadata
	if oldSchema != nil {
		oldProto = oldSchema.GetProto()
	}
	if newSchema != nil {
		newProto = newSchema.GetProto()
	}
	return catalog.MetadataMigration(context.Background(), oldProto, newProto, catalog.MetadataMigrationOptions{
		SourceSearchPath: getMigrationSearchPath(oldSchema),
		TargetSearchPath: getMigrationSearchPath(newSchema),
		SkipSchemas:      []string{common.BackupDatabaseNameOfEngine(storepb.Engine_POSTGRES)},
	})
}

// getMigrationSearchPath is getConfiguredSearchPath but keeps an explicitly listed
// pg_catalog, whose position lets an earlier user schema shadow a built-in.
func getMigrationSearchPath(d *model.DatabaseMetadata) []string {
	if d == nil {
		return nil
	}
	configured := model.ParsePGConfiguredSearchPathIncludingSystem(d.GetProto().GetSearchPath())
	searchPath := make([]string, 0, len(configured))
	for _, item := range configured {
		if item.CurrentUser {
			searchPath = append(searchPath, "$user")
			continue
		}
		if item.Schema != "" {
			searchPath = append(searchPath, item.Schema)
		}
	}
	return searchPath
}
