// Package oracle implements Bytebase's schema contracts for Oracle on top of
// omni's Oracle catalog. The catalog owns deparse, diff, and migration; this
// package keeps the registry wiring, plus the metadata extractor that
// get_database_metadata.go still holds. Moving extraction needs a parity
// decision first: omni normalizes check expressions, adds view dependency
// columns, and leaves the inferred type off an untyped virtual column.
package oracle

import (
	"context"
	"strings"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/bytebase/omni/oracle/catalog"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/store/model"
)

// Two of GetSchemaString's object types are deliberately left unregistered.
//
// SCHEMA: an Oracle schema is a user, one per database, so its definition is the
// database definition GetDatabaseDefinition already renders. Only pg registers a
// schema definition; MSSQL, MySQL and TiDB have real schemas and do not.
//
// SEQUENCE: no Oracle sequence renderer is registered, so GetSchemaString
// answers nothing for one. The driver's dump does not render sequences here
// either: it goes through omni's catalog, which writes the server's definition.
// A sequence's Start, Increment, MaxValue, Cache and Cycle do reach this package
// through the sync, so registering the object type is a change of its own.
func init() {
	schema.RegisterGetDatabaseDefinition(storepb.Engine_ORACLE, GetDatabaseDefinition)
	schema.RegisterGetTableDefinition(storepb.Engine_ORACLE, GetTableDefinition)
	schema.RegisterGetViewDefinition(storepb.Engine_ORACLE, GetViewDefinition)
	schema.RegisterGetMaterializedViewDefinition(storepb.Engine_ORACLE, GetMaterializedViewDefinition)
	schema.RegisterGetFunctionDefinition(storepb.Engine_ORACLE, GetFunctionDefinition)
	schema.RegisterGetProcedureDefinition(storepb.Engine_ORACLE, GetProcedureDefinition)
	schema.RegisterDiffMetadataMigration(storepb.Engine_ORACLE, diffMetadataMigration)
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
	return catalog.GetMaterializedViewDefinition("", view)
}

func GetFunctionDefinition(_ string, function *metadatapb.FunctionMetadata) (string, error) {
	return catalog.GetFunctionDefinition("", function)
}

func GetProcedureDefinition(_ string, procedure *metadatapb.ProcedureMetadata) (string, error) {
	return catalog.GetProcedureDefinition("", procedure)
}

// stripIdentity preserves the pre-cutover renderer's output: it never emitted
// an identity clause, and it never wrote an index's invisible state. Keep both
// defects until a separate change can ship the fixes with their own release note.
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
			for _, index := range view.GetIndexes() {
				index.Visible = true
			}
		}
	}
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
	for _, index := range table.GetIndexes() {
		index.Visible = true
	}
}

// diffMetadataMigration diffs two synced snapshots with omni's catalog. The
// catalog stores each snapshot as the sync reported it, so nothing can fail to
// load; the diff and generator are the decisions.
func diffMetadataMigration(oldSchema, newSchema *model.DatabaseMetadata) (string, error) {
	from, err := loadCatalog(oldSchema)
	if err != nil {
		return "", errors.Wrap(err, "failed to load source schema")
	}
	to, err := loadCatalog(newSchema)
	if err != nil {
		return "", errors.Wrap(err, "failed to load target schema")
	}
	diff := catalog.Diff(from, to)
	if diff.IsEmpty() {
		return "", nil
	}
	migration := catalog.GenerateMigration(from, to, diff).SQL()
	// The legacy generator ended every non-empty script with at least one
	// newline; the catalog's plan does not, so add the one it omits. Six scripts
	// also carried a trailing blank line, recorded in cutover_parity.yaml.
	if migration != "" && !strings.HasSuffix(migration, "\n") {
		migration += "\n"
	}
	return migration, nil
}

func loadCatalog(d *model.DatabaseMetadata) (*catalog.Catalog, error) {
	c := catalog.New()
	var metadata *metadatapb.DatabaseSchemaMetadata
	if d != nil {
		metadata = d.GetProto()
	}
	if _, err := c.LoadMetadata(context.Background(), metadata); err != nil {
		return nil, err
	}
	return c, nil
}
