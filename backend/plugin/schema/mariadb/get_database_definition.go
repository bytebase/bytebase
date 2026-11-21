package mariadb

import (
	"github.com/bytebase/omni/mariadb/catalog"
	metadatapb "github.com/bytebase/omni/metadata"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
)

func init() {
	schema.RegisterGetDatabaseDefinition(storepb.Engine_MARIADB, GetDatabaseDefinition)
	schema.RegisterGetTableDefinition(storepb.Engine_MARIADB, GetTableDefinition)
	schema.RegisterGetViewDefinition(storepb.Engine_MARIADB, GetViewDefinition)
	schema.RegisterGetFunctionDefinition(storepb.Engine_MARIADB, GetFunctionDefinition)
	schema.RegisterGetProcedureDefinition(storepb.Engine_MARIADB, GetProcedureDefinition)
	schema.RegisterGetSequenceDefinition(storepb.Engine_MARIADB, GetSequenceDefinition)
}

func GetDatabaseDefinition(ctx schema.GetDefinitionContext, metadata *metadatapb.DatabaseSchemaMetadata) (string, error) {
	return catalog.DatabaseDefinition(metadata, catalog.DefinitionOptions{
		PrintHeader: ctx.PrintHeader,
		SDLFormat:   ctx.SDLFormat,
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

func GetSequenceDefinition(_ string, sequence *metadatapb.SequenceMetadata) (string, error) {
	return catalog.SequenceDefinition(sequence)
}
