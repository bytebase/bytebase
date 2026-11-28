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
		DatabaseMetadata:      GetDatabaseMetadata,
		DiffMetadataMigration: omnicatalog.MetadataMigrationFunc(mariaDBCatalog),
		WalkThrough:           schema.OmniWalkThrough(storepb.Engine_MARIADB, mariadbanalysis.Engine),
	}, storepb.Engine_MARIADB)
}

var mariaDBCatalog = omnicatalog.MySQLFamilyEngine(
	catalog.New,
	func(report *catalog.LoadMetadataReport) []map[string]error {
		return []map[string]error{report.Degraded, report.Missing, report.Incomplete}
	},
	catalog.Diff,
	catalog.GenerateMigration,
	clearImplicitTableCharsets,
)

func GetDatabaseDefinition(ctx schema.GetDefinitionContext, metadata *metadatapb.DatabaseSchemaMetadata) (string, error) {
	return catalog.DatabaseDefinition(metadata, catalog.DefinitionOptions{
		PrintHeader: ctx.PrintHeader,
		SDLFormat:   ctx.SDLFormat,
	})
}

// GetDatabaseMetadata parses MariaDB schema DDL text and returns database
// metadata using the omni MariaDB catalog.
func GetDatabaseMetadata(schemaText string) (*metadatapb.DatabaseSchemaMetadata, error) {
	return omnicatalog.ExtractSimpleDatabaseMetadata(schemaText, catalog.New, execCatalog)
}

// clearImplicitTableCharsets restores the legacy generator's omission of a
// charset or collation the snapshot does not specify. The catalog fills an
// implicit table with its own release defaults, which for 11.8 includes
// utf8mb4_uca1400_ai_ci, a collation MariaDB 10.7 (still supported) rejects.
func clearImplicitTableCharsets(c *catalog.Catalog, meta *metadatapb.DatabaseSchemaMetadata) {
	db := c.GetDatabase(omnicatalog.Database)
	if db == nil {
		return
	}
	for _, schema := range meta.GetSchemas() {
		for _, table := range schema.GetTables() {
			loaded := db.GetTable(table.GetName())
			if loaded == nil {
				continue
			}
			switch {
			case table.GetCharset() == "" && table.GetCollation() == "":
				loaded.Charset, loaded.Collation = "", ""
			case table.GetCollation() == "":
				loaded.Collation = ""
			default:
			}
		}
	}
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
