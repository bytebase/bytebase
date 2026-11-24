package tidb

import (
	"context"
	"strings"

	metadatapb "github.com/bytebase/omni/metadata"
	tidbanalysis "github.com/bytebase/omni/tidb/analysis"
	"github.com/bytebase/omni/tidb/catalog"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	tidbparser "github.com/bytebase/bytebase/backend/plugin/parser/tidb"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/plugin/schema/internal/omnicatalog"
	"github.com/bytebase/bytebase/backend/store/model"
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
		WalkThrough:           WalkThroughWithContext,
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

var tidbWalkThroughEngine = omnicatalog.WalkThroughEngine[catalog.Catalog]{
	Engine: storepb.Engine_TIDB,
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
	InspectAST: omnicatalog.InspectAST(tidbparser.GetNode, tidbanalysis.ClassifyWalkThroughStatement),
	ExtractCatalogError: omnicatalog.ExtractError(func(e *catalog.Error) (int, string) {
		return e.Code, e.Message
	}),
	TrackBatchTables: true,
	TiDBErrorDialect: true,
	CaseSensitiveResult: func(d *model.DatabaseMetadata) bool {
		return d.GetIsObjectCaseSensitive()
	},
}

// WalkThroughWithContext simulates the DDL in asts against the database state d using the
// omni TiDB catalog, then replaces d with the resulting schema so downstream
// advisor rules see the post-DDL metadata.
func WalkThroughWithContext(ctx context.Context, _ schema.WalkThroughContext, d *model.DatabaseMetadata, asts []base.AST) *storepb.Advice {
	rawSQL, execASTs := tidbBuildRawSQL(asts)
	return omnicatalog.RunWalkThrough(ctx, tidbWalkThroughEngine, rawSQL, d, asts, execASTs)
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

// tidbBuildRawSQL rebuilds the original SQL from the omni ASTs. Each statement
// keeps its own text; they are re-joined with a terminator so catalog.Exec
// re-parses them as a statement list. The returned slice is parallel to that
// list: execASTs[i] is the base.AST that produced the i-th executed statement,
// so a simulation error's index maps back to the submitted statement's
// position.
func tidbBuildRawSQL(asts []base.AST) (string, []base.AST) {
	var b strings.Builder
	execASTs := make([]base.AST, 0, len(asts))
	for _, a := range asts {
		wrapped, ok := a.(*tidbparser.AST)
		if !ok {
			continue
		}
		text := strings.TrimSpace(wrapped.Text)
		if text == "" {
			continue
		}
		b.WriteString(strings.TrimRight(text, ";"))
		b.WriteString(";\n")
		execASTs = append(execASTs, a)
	}
	return b.String(), execASTs
}
