// Package mariadb registers MariaDB's DDL simulation, schema definition/metadata
// conversions, and metadata diff on omni's native MariaDB catalog.
package mariadb

import (
	mariadbanalysis "github.com/bytebase/omni/mariadb/analysis"
	"github.com/bytebase/omni/mariadb/catalog"
	metadatapb "github.com/bytebase/omni/metadata"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/plugin/schema/internal/omnicatalog"
)

// Runs on MariaDB's own catalog fork rather than MySQL's: the MySQL loader
// drops WITH SYSTEM VERSIONING and installs ROW START/END as plain NOT NULL columns.
func init() {
	schema.Register(schema.Engine{
		Definitions: omnicatalog.MySQLFamilyDefinitions{
			Database:  GetDatabaseDefinition,
			Table:     catalog.TableDefinition,
			View:      catalog.ViewDefinition,
			Function:  catalog.FunctionDefinition,
			Procedure: catalog.ProcedureDefinition,
			Sequence:  catalog.SequenceDefinition,
		}.SchemaDefinitions(),
		DatabaseMetadata:      schema.OmniDatabaseMetadata(mariadbanalysis.Engine),
		DiffMetadataMigration: schema.OmniMetadataMigration(mariadbanalysis.Engine),
		WalkThrough:           schema.OmniWalkThrough(storepb.Engine_MARIADB, mariadbanalysis.Engine),
	}, storepb.Engine_MARIADB)
}

func GetDatabaseDefinition(ctx schema.GetDefinitionContext, metadata *metadatapb.DatabaseSchemaMetadata) (string, error) {
	return catalog.DatabaseDefinition(metadata, catalog.DefinitionOptions{
		PrintHeader: ctx.PrintHeader,
		SDLFormat:   ctx.SDLFormat,
	})
}
