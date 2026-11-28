package schema

import (
	"context"

	omnianalysis "github.com/bytebase/omni/analysis"
	metadatapb "github.com/bytebase/omni/metadata"

	"github.com/bytebase/bytebase/backend/store/model"
)

// OmniMetadataMigration returns a DiffMetadataMigration capability backed by
// an omni engine's metadata migrator. skipSchemas are left out of the
// migration.
func OmniMetadataMigration(migrator omnianalysis.MetadataMigrator, skipSchemas ...string) func(oldSchema, newSchema *model.DatabaseMetadata) (string, error) {
	return func(oldSchema, newSchema *model.DatabaseMetadata) (string, error) {
		return migrator.MetadataMigration(context.Background(), omnianalysis.MetadataMigrationRequest{
			From:        omniDatabase(oldSchema),
			To:          omniDatabase(newSchema),
			SkipSchemas: skipSchemas,
		})
	}
}

// OmniDatabaseMetadata returns a DatabaseMetadata capability backed by an omni
// engine's metadata extractor.
func OmniDatabaseMetadata(extractor omnianalysis.MetadataExtractor) func(string) (*metadatapb.DatabaseSchemaMetadata, error) {
	return func(schemaText string) (*metadatapb.DatabaseSchemaMetadata, error) {
		return extractor.ExtractMetadata(context.Background(), schemaText)
	}
}

func omniDatabase(d *model.DatabaseMetadata) *omnianalysis.Database {
	if d == nil {
		return nil
	}
	return &omnianalysis.Database{
		Name:                d.DatabaseName(),
		Metadata:            d.GetProto(),
		ObjectCaseSensitive: d.GetIsObjectCaseSensitive(),
	}
}
