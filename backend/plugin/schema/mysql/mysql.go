package mysql

import (
	"context"

	metadatapb "github.com/bytebase/omni/metadata"
	mysqlanalysis "github.com/bytebase/omni/mysql/analysis"
	"github.com/bytebase/omni/mysql/catalog"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/plugin/schema/internal/omnicatalog"
	"github.com/bytebase/bytebase/backend/store/model"
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
		DatabaseMetadata:      GetDatabaseMetadata,
		DiffSDLMigration:      mysqlDiffSDLMigrationWithContext,
		SDLDropAdvices:        mysqlSDLDropAdvices,
		DiffMetadataMigration: omniDiffMetadataMigration,
		WalkThrough:           WalkThroughWithContext,
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

// GetDatabaseMetadata parses MySQL schema DDL text and returns database metadata
// using the omni catalog.
func GetDatabaseMetadata(schemaText string) (*metadatapb.DatabaseSchemaMetadata, error) {
	return omnicatalog.ExtractSimpleDatabaseMetadata(schemaText, catalog.New, execCatalog)
}

var mysqlCatalog = omnicatalog.MySQLFamilyEngine(
	catalog.New,
	func(report *catalog.LoadMetadataReport) []map[string]error {
		return []map[string]error{report.Degraded, report.Missing, report.Incomplete}
	},
	catalog.Diff,
	catalog.GenerateMigration,
	nil,
)

func omniDiffMetadataMigration(oldSchema, newSchema *model.DatabaseMetadata) (string, error) {
	return omnicatalog.DiffMetadataMigration(mysqlCatalog, oldSchema, newSchema)
}

var mysqlWalkThroughEngine = omnicatalog.WalkThroughEngine[catalog.Catalog]{
	Engine: storepb.Engine_MYSQL,
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
	InspectAST: omnicatalog.InspectAST(mysqlparser.GetNode, mysqlanalysis.ClassifyWalkThroughStatement),
	ExtractCatalogError: omnicatalog.ExtractError(func(e *catalog.Error) (int, string) {
		return e.Code, e.Message
	}),
	SplitSQL: mysqlparser.SplitSQL,
	IsCTAS:   mysqlanalysis.IsCreateTableAs,
}

// WalkThroughWithContext performs DDL simulation using the omni MySQL catalog.
func WalkThroughWithContext(ctx context.Context, wtCtx schema.WalkThroughContext, d *model.DatabaseMetadata, asts []base.AST) *storepb.Advice {
	return omnicatalog.RunWalkThrough(ctx, mysqlWalkThroughEngine, wtCtx.RawSQL, d, asts, nil)
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
