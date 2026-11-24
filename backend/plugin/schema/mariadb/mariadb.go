// Package mariadb registers MariaDB's DDL simulation, schema definition/metadata
// conversions, and metadata diff on omni's native MariaDB catalog.
package mariadb

import (
	"context"

	mariadbanalysis "github.com/bytebase/omni/mariadb/analysis"
	"github.com/bytebase/omni/mariadb/catalog"
	metadatapb "github.com/bytebase/omni/metadata"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	mariadbparser "github.com/bytebase/bytebase/backend/plugin/parser/mariadb"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/plugin/schema/internal/omnicatalog"
	"github.com/bytebase/bytebase/backend/store/model"
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
		WalkThrough:           WalkThroughWithContext,
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

var mariaDBWalkThroughEngine = omnicatalog.WalkThroughEngine[catalog.Catalog]{
	Engine: storepb.Engine_MARIADB,
	New:    catalog.New,
	Load: func(ctx context.Context, c *catalog.Catalog, meta *metadatapb.DatabaseSchemaMetadata) error {
		_, err := c.LoadMetadata(ctx, meta)
		return err
	},
	Exec: func(c *catalog.Catalog, sql string) ([]omnicatalog.ExecResult, error) {
		return execCatalog(c, sql, true)
	},
	DatabaseMetadata: func(c *catalog.Catalog, dbName string) *metadatapb.DatabaseSchemaMetadata {
		return c.DatabaseMetadata(dbName)
	},
	InspectAST: omnicatalog.InspectAST(mariadbparser.GetNode, mariadbanalysis.ClassifyWalkThroughStatement),
	ExtractCatalogError: omnicatalog.ExtractError(func(e *catalog.Error) (int, string) {
		return e.Code, e.Message
	}),
	SplitSQL: mariadbparser.SplitSQL,
	IsCTAS:   mariadbanalysis.IsCreateTableAs,
}

// WalkThroughWithContext performs DDL simulation using the omni MariaDB catalog.
func WalkThroughWithContext(ctx context.Context, wtCtx schema.WalkThroughContext, d *model.DatabaseMetadata, asts []base.AST) *storepb.Advice {
	return omnicatalog.RunWalkThrough(ctx, mariaDBWalkThroughEngine, wtCtx.RawSQL, d, asts, nil)
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
