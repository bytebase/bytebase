package oceanbase

import (
	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/bytebase/omni/oceanbase/catalog"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/plugin/schema/internal/omnicatalog"
)

func init() {
	omnicatalog.RegisterMySQLFamilyDefinitions(storepb.Engine_OCEANBASE, omnicatalog.MySQLFamilyDefinitions{
		Database:  GetDatabaseDefinition,
		Table:     catalog.TableDefinition,
		View:      catalog.ViewDefinition,
		Function:  catalog.FunctionDefinition,
		Procedure: catalog.ProcedureDefinition,
	})
}

func GetDatabaseDefinition(ctx schema.GetDefinitionContext, metadata *metadatapb.DatabaseSchemaMetadata) (string, error) {
	return catalog.DatabaseDefinition(metadata, catalog.DefinitionOptions{
		PrintHeader: ctx.PrintHeader,
		SDLFormat:   ctx.SDLFormat,
	})
}
