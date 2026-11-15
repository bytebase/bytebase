package tidb

import (
	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/bytebase/omni/tidb/catalog"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
)

func init() {
	schema.RegisterGetDatabaseDefinition(storepb.Engine_TIDB, GetDatabaseDefinition)

	schema.RegisterGetTableDefinition(storepb.Engine_TIDB, GetTableDefinition)

	schema.RegisterGetViewDefinition(storepb.Engine_TIDB, GetViewDefinition)

	schema.RegisterGetFunctionDefinition(storepb.Engine_TIDB, GetFunctionDefinition)

	schema.RegisterGetProcedureDefinition(storepb.Engine_TIDB, GetProcedureDefinition)
}

// GetDatabaseDefinition renders the mysqldump-style restore script. TiDB has no
// SDL form, so ctx.SDLFormat is ignored.
func GetDatabaseDefinition(ctx schema.GetDefinitionContext, metadata *metadatapb.DatabaseSchemaMetadata) (string, error) {
	return catalog.DatabaseDefinition(metadata, catalog.DefinitionOptions{
		PrintHeader: ctx.PrintHeader,
	})
}

func GetTableDefinition(_ string, table *metadatapb.TableMetadata, _ []*metadatapb.SequenceMetadata) (string, error) {
	return catalog.TableDefinition(table)
}

func GetViewDefinition(_ string, view *metadatapb.ViewMetadata) (string, error) {
	return catalog.ViewDefinition(view)
}

func GetFunctionDefinition(_ string, function *metadatapb.FunctionMetadata) (string, error) {
	return catalog.FunctionDefinition(function)
}

func GetProcedureDefinition(_ string, procedure *metadatapb.ProcedureMetadata) (string, error) {
	return catalog.ProcedureDefinition(procedure)
}
