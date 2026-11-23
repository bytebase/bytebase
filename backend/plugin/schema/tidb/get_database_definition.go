package tidb

import (
	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/bytebase/omni/tidb/catalog"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/plugin/schema/internal/omnicatalog"
)

func init() {
	omnicatalog.RegisterMySQLFamilyDefinitions(storepb.Engine_TIDB, omnicatalog.MySQLFamilyDefinitions{
		Database:  GetDatabaseDefinition,
		Table:     catalog.TableDefinition,
		View:      catalog.ViewDefinition,
		Function:  catalog.FunctionDefinition,
		Procedure: catalog.ProcedureDefinition,
	})
}

// GetDatabaseDefinition renders the mysqldump-style restore script. TiDB has no
// SDL form, so ctx.SDLFormat is ignored.
func GetDatabaseDefinition(ctx schema.GetDefinitionContext, metadata *metadatapb.DatabaseSchemaMetadata) (string, error) {
	return catalog.DatabaseDefinition(metadata, catalog.DefinitionOptions{
		PrintHeader: ctx.PrintHeader,
	})
}
