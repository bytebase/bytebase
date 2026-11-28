package tidb

import (
	"strings"

	metadatapb "github.com/bytebase/omni/metadata"
	tidbanalysis "github.com/bytebase/omni/tidb/analysis"
	"github.com/bytebase/omni/tidb/catalog"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/plugin/schema/internal/omnicatalog"
)

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
		DiffMetadataMigration: omnicatalog.MetadataMigrationFunc(tidbCatalog),
		WalkThrough:           schema.OmniWalkThrough(storepb.Engine_TIDB, tidbanalysis.Engine),
	}, storepb.Engine_TIDB)
}

var tidbCatalog = omnicatalog.MySQLFamilyEngine(
	catalog.New,
	func(report *catalog.LoadMetadataReport) []map[string]error {
		return []map[string]error{report.Degraded, report.Missing, report.Incomplete}
	},
	catalog.Diff,
	catalog.GenerateMigration,
	nil,
)

// GetDatabaseDefinition renders the mysqldump-style restore script. TiDB has no
// SDL form, so ctx.SDLFormat is ignored.
func GetDatabaseDefinition(ctx schema.GetDefinitionContext, metadata *metadatapb.DatabaseSchemaMetadata) (string, error) {
	return catalog.DatabaseDefinition(metadata, catalog.DefinitionOptions{
		PrintHeader: ctx.PrintHeader,
	})
}

// catalogDBName is the throwaway scratch database used when schemaText selects
// none of its own; its name is chosen not to collide with user identifiers.
const catalogDBName = "_bytebase_schema_scratch"

// GetDatabaseMetadata parses TiDB schema DDL text and returns database metadata
// using the omni TiDB catalog.
func GetDatabaseMetadata(schemaText string) (*metadatapb.DatabaseSchemaMetadata, error) {
	return omnicatalog.ExtractDatabaseMetadata(
		schemaText,
		catalogDBName,
		catalog.New,
		execCatalog,
		(*catalog.Catalog).CurrentDatabase,
		func(c *catalog.Catalog, dbName string) *metadatapb.DatabaseSchemaMetadata {
			proto := c.DatabaseMetadata(dbName)
			if dbName == catalogDBName {
				// Return schema-relative metadata when the DDL selected no database of its own.
				// Real-database qualifiers stay intact to match information_schema.VIEWS sync.
				proto.Name = ""
				prefix := "`" + catalogDBName + "`."
				for _, s := range proto.GetSchemas() {
					for _, v := range s.GetViews() {
						v.Definition = strings.ReplaceAll(v.Definition, prefix, "")
					}
				}
			}
			return proto
		},
	)
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
