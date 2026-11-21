package mariadb

import (
	"github.com/bytebase/omni/mariadb/catalog"
	metadatapb "github.com/bytebase/omni/metadata"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/plugin/schema/internal/omnicatalog"
)

func init() {
	omnicatalog.RegisterMySQLFamilyDefinitions(storepb.Engine_MARIADB, omnicatalog.MySQLFamilyDefinitions{
		Database:  GetDatabaseDefinition,
		Table:     catalog.TableDefinition,
		View:      catalog.ViewDefinition,
		Function:  catalog.FunctionDefinition,
		Procedure: catalog.ProcedureDefinition,
		Sequence:  catalog.SequenceDefinition,
	})
}

func GetDatabaseDefinition(ctx schema.GetDefinitionContext, metadata *metadatapb.DatabaseSchemaMetadata) (string, error) {
	return catalog.DatabaseDefinition(metadata, catalog.DefinitionOptions{
		PrintHeader: ctx.PrintHeader,
		SDLFormat:   ctx.SDLFormat,
	})
}
