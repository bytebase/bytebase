// Package oracle implements Bytebase's schema contracts for Oracle on top of
// omni's Oracle catalog. The catalog owns metadata extraction, deparse, diff,
// and migration; this package keeps the registry wiring and the unrendered-fact
// strip until those clauses are enabled.
package oracle

import (
	"context"
	"strings"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/bytebase/omni/oracle/catalog"
	"google.golang.org/protobuf/proto"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/plugin/schema/internal/omnicatalog"
)

// SCHEMA and SEQUENCE are deliberately left unregistered: Oracle schemas map
// 1:1 to database users (covered by GetDatabaseDefinition), and sequence
// rendering is not yet wired through GetSchemaString.
func init() {
	schema.Register(schema.Engine{
		Definitions: schema.Definitions{
			Database:         GetDatabaseDefinition,
			Table:            GetTableDefinition,
			View:             GetViewDefinition,
			MaterializedView: GetMaterializedViewDefinition,
			Function:         GetFunctionDefinition,
			Procedure:        GetProcedureDefinition,
		},
		DatabaseMetadata:      GetDatabaseMetadata,
		DiffMetadataMigration: omnicatalog.MetadataMigrationFunc(oracleCatalog),
	}, storepb.Engine_ORACLE)
}

var oracleCatalog = omnicatalog.Engine[catalog.Catalog, catalog.LoadMetadataReport]{
	New: catalog.New,
	Load: func(c *catalog.Catalog, meta *metadatapb.DatabaseSchemaMetadata) (*catalog.LoadMetadataReport, error) {
		return c.LoadMetadata(context.Background(), stripUnrenderedFacts(meta))
	},
	Diff:                     omnicatalog.DiffSQL(catalog.Diff, catalog.GenerateMigration),
	SkipStripDatabaseContext: true,
	// The legacy generator ended every non-empty script with a newline; omni's plan does not.
	PostProcess: func(migration string) string {
		if migration != "" && !strings.HasSuffix(migration, "\n") {
			return migration + "\n"
		}
		return migration
	},
}

// GetDatabaseMetadata parses Oracle DDL text and returns database metadata
// using omni's Oracle catalog.
func GetDatabaseMetadata(schemaText string) (*metadatapb.DatabaseSchemaMetadata, error) {
	return catalog.GetDatabaseMetadata(schemaText)
}

// GetDatabaseDefinition renders a synced Oracle snapshot as the DDL that
// recreates it.
func GetDatabaseDefinition(_ schema.GetDefinitionContext, metadata *metadatapb.DatabaseSchemaMetadata) (string, error) {
	return catalog.DatabaseDefinition(stripUnrenderedFacts(metadata))
}

func GetTableDefinition(_ string, table *metadatapb.TableMetadata, _ []*metadatapb.SequenceMetadata) (string, error) {
	if table.GetSkipDump() {
		return "", nil
	}
	return catalog.GetTableDefinition("", stripTableUnrenderedFacts(table), nil)
}

func GetViewDefinition(_ string, view *metadatapb.ViewMetadata) (string, error) {
	return catalog.GetViewDefinition("", view)
}

func GetMaterializedViewDefinition(_ string, view *metadatapb.MaterializedViewMetadata) (string, error) {
	return catalog.GetMaterializedViewDefinition("", stripMaterializedViewUnrenderedFacts(view))
}

func GetFunctionDefinition(_ string, function *metadatapb.FunctionMetadata) (string, error) {
	return catalog.GetFunctionDefinition("", function)
}

func GetProcedureDefinition(_ string, procedure *metadatapb.ProcedureMetadata) (string, error) {
	return catalog.GetProcedureDefinition("", procedure)
}

// stripUnrenderedFacts keeps identity clauses and index visibility out of both
// the definition and the migration until #22119 fixes them.
func stripUnrenderedFacts(metadata *metadatapb.DatabaseSchemaMetadata) *metadatapb.DatabaseSchemaMetadata {
	if metadata == nil {
		return nil
	}
	out := proto.CloneOf(metadata)
	for _, schema := range out.GetSchemas() {
		for _, table := range schema.GetTables() {
			stripTableFacts(table)
		}
		for _, view := range schema.GetMaterializedViews() {
			resetIndexVisibility(view.GetIndexes())
		}
	}
	return out
}

func stripMaterializedViewUnrenderedFacts(view *metadatapb.MaterializedViewMetadata) *metadatapb.MaterializedViewMetadata {
	out := proto.CloneOf(view)
	resetIndexVisibility(out.GetIndexes())
	return out
}

func stripTableUnrenderedFacts(table *metadatapb.TableMetadata) *metadatapb.TableMetadata {
	out := proto.CloneOf(table)
	stripTableFacts(out)
	return out
}

func stripTableFacts(table *metadatapb.TableMetadata) {
	for _, column := range table.GetColumns() {
		column.IsIdentity = false
		column.IdentityGeneration = metadatapb.ColumnMetadata_BY_DEFAULT
		column.IdentitySeed = 0
		column.IdentityIncrement = 0
	}
	resetIndexVisibility(table.GetIndexes())
}

func resetIndexVisibility(indexes []*metadatapb.IndexMetadata) {
	for _, index := range indexes {
		index.Visible = true
	}
}
