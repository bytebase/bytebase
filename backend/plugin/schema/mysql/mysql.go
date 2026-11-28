package mysql

import (
	metadatapb "github.com/bytebase/omni/metadata"
	mysqlanalysis "github.com/bytebase/omni/mysql/analysis"
	"github.com/bytebase/omni/mysql/catalog"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/plugin/schema/internal/omnicatalog"
)

func init() {
	// MultiFileDatabase, DiffSDLMigration, and SDLDropAdvices are MySQL-only:
	// OceanBase is intentionally unregistered pending live oracle validation.
	schema.Register(schema.Engine{
		Definitions: omnicatalog.MySQLFamilyDefinitions{
			Database:          GetDatabaseDefinition,
			MultiFileDatabase: GetMultiFileDatabaseDefinition,
			Table:             catalog.TableDefinition,
			View:              catalog.ViewDefinition,
			Function:          catalog.FunctionDefinition,
			Procedure:         catalog.ProcedureDefinition,
		}.SchemaDefinitions(),
		DatabaseMetadata:      schema.OmniDatabaseMetadata(mysqlanalysis.Engine),
		DiffSDLMigration:      mysqlDiffSDLMigrationWithContext,
		SDLDropAdvices:        mysqlSDLDropAdvices,
		DiffMetadataMigration: schema.OmniMetadataMigration(mysqlanalysis.Engine),
		WalkThrough:           schema.OmniWalkThrough(storepb.Engine_MYSQL, mysqlanalysis.Engine),
	}, storepb.Engine_MYSQL)
}

func GetDatabaseDefinition(ctx schema.GetDefinitionContext, metadata *metadatapb.DatabaseSchemaMetadata) (string, error) {
	return catalog.DatabaseDefinition(metadata, catalog.DefinitionOptions{
		PrintHeader: ctx.PrintHeader,
		SDLFormat:   ctx.SDLFormat,
	})
}

// GetMultiFileDatabaseDefinition emits the database schema as one SDL file per object.
// Each declarative rollout re-concatenates the files (action/command/file.go) into the
// statement set the single-file SDL dump carries.
func GetMultiFileDatabaseDefinition(_ schema.GetDefinitionContext, metadata *metadatapb.DatabaseSchemaMetadata) (*schema.MultiFileSchemaResult, error) {
	files, err := catalog.MultiFileDefinition(metadata)
	if err != nil {
		return nil, err
	}
	var result []schema.File
	if files != nil {
		result = make([]schema.File, 0, len(files))
	}
	for _, file := range files {
		result = append(result, schema.File{Name: file.Name, Content: file.Content})
	}
	return &schema.MultiFileSchemaResult{Files: result}, nil
}
