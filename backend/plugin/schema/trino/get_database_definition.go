// Package trino implements the bytebase schema.GetDatabaseDefinition contract
// for the Trino engine, backed by the omni Trino deparser
// (github.com/bytebase/omni/trino/deparse).
package trino

import (
	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/bytebase/omni/trino/deparse"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
)

func init() {
	schema.Register(schema.Engine{
		Definitions: schema.Definitions{
			Database: GetDatabaseDefinition,
		},
	}, storepb.Engine_TRINO)
}

// GetDatabaseDefinition renders the SDL for a Trino database schema by
// delegating to omni's deparse.DatabaseDefinitionFromProto.
func GetDatabaseDefinition(_ schema.GetDefinitionContext, metadata *metadatapb.DatabaseSchemaMetadata) (string, error) {
	return deparse.DatabaseDefinitionFromProto(metadata)
}
