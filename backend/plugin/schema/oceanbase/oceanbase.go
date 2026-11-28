package oceanbase

import (
	metadatapb "github.com/bytebase/omni/metadata"
	oceanbaseanalysis "github.com/bytebase/omni/oceanbase/analysis"
	"github.com/bytebase/omni/oceanbase/catalog"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/plugin/schema/internal/omnicatalog"
)

// Runs on OceanBase's own catalog fork so OceanBase-specific index options,
// TTL, and column groups load and render accurately.
func init() {
	schema.Register(schema.Engine{
		Definitions: omnicatalog.MySQLFamilyDefinitions{
			Database:  GetDatabaseDefinition,
			Table:     catalog.TableDefinition,
			View:      catalog.ViewDefinition,
			Function:  catalog.FunctionDefinition,
			Procedure: catalog.ProcedureDefinition,
		}.SchemaDefinitions(),
		DatabaseMetadata:      schema.OmniDatabaseMetadata(oceanbaseanalysis.Engine),
		DiffMetadataMigration: schema.OmniMetadataMigration(oceanbaseanalysis.Engine),
		WalkThrough:           schema.OmniWalkThrough(storepb.Engine_OCEANBASE, oceanbaseanalysis.Engine),
	}, storepb.Engine_OCEANBASE)
}

func GetDatabaseDefinition(ctx schema.GetDefinitionContext, metadata *metadatapb.DatabaseSchemaMetadata) (string, error) {
	return catalog.DatabaseDefinition(metadata, catalog.DefinitionOptions{
		PrintHeader: ctx.PrintHeader,
		SDLFormat:   ctx.SDLFormat,
	})
}
