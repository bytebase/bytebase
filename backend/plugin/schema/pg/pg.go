// Package pg implements Bytebase's schema contracts for PostgreSQL and
// CockroachDB on top of omni's pg catalog. The catalog owns the database,
// multi-file and per-object definitions, the SDL diff and the metadata diff.
// Bytebase keeps the registry wiring, the backup-schema filter, the legacy
// index-expression repair in legacy_normalize.go, and the DDL walk-through.
//
// CockroachDB shares the catalog: its synced metadata is PostgreSQL-shaped and
// omni has no CockroachDB catalog. testdata/cutover_parity.yaml records the
// migrations the deleted Bytebase generator produced, so every text change the
// catalog introduces stays named.
package pg

import (
	"context"
	"fmt"

	"google.golang.org/protobuf/proto"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/bytebase/omni/pg/catalog"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/plugin/schema/internal/omnicatalog"
	"github.com/bytebase/bytebase/backend/store/model"
)

func init() {
	schema.Register(schema.Engine{
		Definitions: schema.Definitions{
			Database:          GetDatabaseDefinition,
			Schema:            catalog.SchemaDefinition,
			Table:             catalog.TableDefinition,
			View:              catalog.ViewDefinition,
			MaterializedView:  catalog.MaterializedViewDefinition,
			Function:          catalog.FunctionDefinition,
			Sequence:          catalog.SequenceDefinition,
			MultiFileDatabase: GetMultiFileDatabaseDefinition,
		},
		DiffSDLMigration:      pgDiffSDLMigration,
		DiffMetadataMigration: diffMetadataMigration,
		SDLDropAdvices:        pgSDLDropAdvices,
	}, storepb.Engine_POSTGRES, storepb.Engine_COCKROACHDB)
	schema.Register(schema.Engine{
		WalkThrough: WalkThroughWithContext,
	}, storepb.Engine_POSTGRES)
}

// GetDatabaseDefinition renders metadata through omni's catalog.DatabaseDefinition
// after dropping the backup schema when asked and repairing legacy metadata.
func GetDatabaseDefinition(ctx schema.GetDefinitionContext, metadata *metadatapb.DatabaseSchemaMetadata) (string, error) {
	return catalog.DatabaseDefinition(prepareDefinitionMetadata(ctx, metadata), catalog.DefinitionOptions{
		PrintHeader: ctx.PrintHeader,
		SDLFormat:   ctx.SDLFormat,
	})
}

// GetMultiFileDatabaseDefinition renders metadata through omni's
// catalog.MultiFileDefinition, prepared as GetDatabaseDefinition prepares it.
func GetMultiFileDatabaseDefinition(ctx schema.GetDefinitionContext, metadata *metadatapb.DatabaseSchemaMetadata) (*schema.MultiFileSchemaResult, error) {
	files, err := catalog.MultiFileDefinition(prepareDefinitionMetadata(ctx, metadata))
	if err != nil {
		return nil, err
	}
	result := &schema.MultiFileSchemaResult{}
	if files != nil {
		result.Files = make([]schema.File, 0, len(files))
		for _, file := range files {
			result.Files = append(result.Files, schema.File{Name: file.Name, Content: file.Content})
		}
	}
	return result, nil
}

func prepareDefinitionMetadata(ctx schema.GetDefinitionContext, metadata *metadatapb.DatabaseSchemaMetadata) *metadatapb.DatabaseSchemaMetadata {
	metadata = filterBackupSchemaIfNecessary(ctx, metadata)

	// Clone before mutating: the caller's *DatabaseSchemaMetadata is often a
	// shared pointer returned from store.dbSchemaCache (see
	// backend/store/model/database.go:GetProto), so concurrent callers could
	// race on in-place normalization writes below.
	metadata = proto.CloneOf(metadata)

	// Repair historical non-canonical metadata shapes (e.g. index key
	// expressions stored without outer parens) so emission produces valid SQL.
	// See legacy_normalize.go for removal criteria.
	normalizeLegacyMetadata(metadata)
	return metadata
}

func filterBackupSchemaIfNecessary(ctx schema.GetDefinitionContext, metadata *metadatapb.DatabaseSchemaMetadata) *metadatapb.DatabaseSchemaMetadata {
	if !ctx.SkipBackupSchema {
		return metadata
	}

	filtered := &metadatapb.DatabaseSchemaMetadata{
		Extensions:    metadata.Extensions,
		EventTriggers: metadata.EventTriggers,
	}
	for _, schema := range metadata.Schemas {
		if schema.Name == common.BackupDatabaseNameOfEngine(storepb.Engine_POSTGRES) {
			continue
		}
		filtered.Schemas = append(filtered.Schemas, schema)
	}
	return filtered
}

// diffMetadataMigration generates the migration between two synced snapshots
// with omni's catalog. PostgreSQL and CockroachDB share it: their metadata is
// PostgreSQL-shaped and omni has no CockroachDB catalog.
func diffMetadataMigration(oldSchema, newSchema *model.DatabaseMetadata) (string, error) {
	return catalog.MetadataMigration(context.Background(), omnicatalog.DatabaseProto(oldSchema), omnicatalog.DatabaseProto(newSchema), catalog.MetadataMigrationOptions{
		SourceSearchPath: getMigrationSearchPath(oldSchema),
		TargetSearchPath: getMigrationSearchPath(newSchema),
		SkipSchemas:      []string{common.BackupDatabaseNameOfEngine(storepb.Engine_POSTGRES)},
	})
}

// getMigrationSearchPath is getConfiguredSearchPath but keeps an explicitly listed
// pg_catalog, whose position lets an earlier user schema shadow a built-in.
func getMigrationSearchPath(d *model.DatabaseMetadata) []string {
	if d == nil {
		return nil
	}
	return formatSearchPath(model.ParsePGConfiguredSearchPathIncludingSystem(d.GetProto().GetSearchPath()))
}

func buildMigrationPlan(sourceText, targetText string) (*catalog.MigrationPlan, error) {
	return catalog.BuildSDLMigrationPlan(sourceText, targetText, catalog.SDLMigrationOptions{
		SkipSchemas: []string{common.BackupDatabaseNameOfEngine(storepb.Engine_POSTGRES)},
	})
}

func pgDiffSDLMigration(sourceSDL, targetSDL, _ string, _ *schema.SDLSessionContextMap) (string, error) {
	plan, err := buildMigrationPlan(sourceSDL, targetSDL)
	if err != nil || plan == nil {
		return "", err
	}
	return plan.SQL(), nil
}

func pgSDLDropAdvices(userSDLText string, currentSchema *model.DatabaseMetadata, _ string) ([]*storepb.Advice, error) {
	return omnicatalog.PlanSDLDropAdvices(
		storepb.Engine_POSTGRES,
		userSDLText,
		currentSchema,
		buildMigrationPlan,
		func(plan *catalog.MigrationPlan) []*storepb.Advice {
			var advices []*storepb.Advice
			for _, op := range plan.Ops {
				switch op.Type {
				case catalog.OpDropSchema:
					advices = append(advices, omnicatalog.SDLDropAdvice(fmt.Sprintf("Dropping schema '%s' will result in data loss.", op.ObjectName)))
				case catalog.OpDropTable:
					advices = append(advices, omnicatalog.SDLDropAdvice(fmt.Sprintf("Dropping table '%s.%s' will result in data loss.", op.SchemaName, op.ObjectName)))
				case catalog.OpDropColumn:
					advices = append(advices, omnicatalog.SDLDropAdvice(fmt.Sprintf("Dropping column from table '%s.%s' will result in data loss.", op.SchemaName, op.ParentObject)))
				case catalog.OpDropView:
					advices = append(advices, omnicatalog.SDLDropAdvice(fmt.Sprintf("Dropping view '%s.%s' will affect dependent objects.", op.SchemaName, op.ObjectName)))
				case catalog.OpDropFunction:
					advices = append(advices, omnicatalog.SDLDropAdvice(fmt.Sprintf("Dropping function '%s.%s' will affect dependent objects.", op.SchemaName, op.ObjectName)))
				case catalog.OpDropSequence:
					advices = append(advices, omnicatalog.SDLDropAdvice(fmt.Sprintf("Dropping sequence '%s.%s' may affect auto-increment columns.", op.SchemaName, op.ObjectName)))
				case catalog.OpDropType:
					advices = append(advices, omnicatalog.SDLDropAdvice(fmt.Sprintf("Dropping type '%s.%s' will affect columns using this type.", op.SchemaName, op.ObjectName)))
				case catalog.OpDropTrigger:
					advices = append(advices, omnicatalog.SDLDropAdvice(fmt.Sprintf("Dropping trigger '%s' on '%s.%s'.", op.ObjectName, op.SchemaName, op.ParentObject)))
				case catalog.OpDropConstraint:
					advices = append(advices, omnicatalog.SDLDropAdvice(fmt.Sprintf("Dropping constraint from table '%s.%s'.", op.SchemaName, op.ParentObject)))
				case catalog.OpAlterFunction:
					advices = append(advices, omnicatalog.SDLReplaceAdvice(fmt.Sprintf("Function '%s.%s' definition will be replaced.", op.SchemaName, op.ObjectName)))
				default:
				}
			}
			return advices
		},
	)
}
