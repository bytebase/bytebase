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
		DatabaseMetadata:      GetDatabaseMetadata,
		DiffMetadataMigration: omnicatalog.MetadataMigrationFunc(oceanBaseCatalog),
		WalkThrough:           schema.OmniWalkThrough(storepb.Engine_OCEANBASE, oceanbaseanalysis.Engine),
	}, storepb.Engine_OCEANBASE)
}

var oceanBaseCatalog = omnicatalog.MySQLFamilyEngine(
	catalog.New,
	func(report *catalog.LoadMetadataReport) []map[string]error {
		return []map[string]error{report.Degraded, report.Missing, report.Incomplete}
	},
	catalog.Diff,
	catalog.GenerateMigration,
	nil,
)

func GetDatabaseDefinition(ctx schema.GetDefinitionContext, metadata *metadatapb.DatabaseSchemaMetadata) (string, error) {
	return catalog.DatabaseDefinition(metadata, catalog.DefinitionOptions{
		PrintHeader: ctx.PrintHeader,
		SDLFormat:   ctx.SDLFormat,
	})
}

// GetDatabaseMetadata parses OceanBase MySQL-mode schema DDL text and returns
// database metadata using the omni catalog.
func GetDatabaseMetadata(schemaText string) (*metadatapb.DatabaseSchemaMetadata, error) {
	return omnicatalog.ExtractSimpleDatabaseMetadata(schemaText, catalog.New, execCatalog)
}

func execCatalog(c *catalog.Catalog, sql string, continueOnError bool) ([]omnicatalog.ExecResult, error) {
	var opts *catalog.ExecOptions
	if continueOnError {
		opts = &catalog.ExecOptions{ContinueOnError: true}
	}
	results, err := c.Exec(sql, opts)
	return omnicatalog.MapExecResults(results, err, func(r catalog.ExecResult) omnicatalog.ExecResult {
		return omnicatalog.ExecResult{Index: r.Index, Line: r.Line, Error: r.Error}
	})
}
