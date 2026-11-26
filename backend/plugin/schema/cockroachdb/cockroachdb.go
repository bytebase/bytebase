// Package cockroachdb implements Bytebase's schema contracts for CockroachDB on
// top of omni's CockroachDB catalog, which renders the database, multi-file
// and per-object definitions and generates the metadata and SDL migrations.
// Bytebase keeps the registry wiring and the SDL drop advices.
package cockroachdb

import (
	"fmt"

	"github.com/bytebase/omni/cockroachdb/catalog"
	metadatapb "github.com/bytebase/omni/metadata"

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
		DiffSDLMigration:      diffSDLMigration,
		DiffMetadataMigration: diffMetadataMigration,
		SDLDropAdvices:        sdlDropAdvices,
	}, storepb.Engine_COCKROACHDB)
}

// GetDatabaseDefinition renders metadata through omni's
// catalog.DatabaseDefinition. CockroachDB has no backup schema, so
// SkipBackupSchema leaves nothing out.
func GetDatabaseDefinition(ctx schema.GetDefinitionContext, metadata *metadatapb.DatabaseSchemaMetadata) (string, error) {
	return catalog.DatabaseDefinition(metadata, catalog.DefinitionOptions{
		PrintHeader: ctx.PrintHeader,
		SDLFormat:   ctx.SDLFormat,
	})
}

// GetMultiFileDatabaseDefinition renders metadata through omni's
// catalog.MultiFileDefinition.
func GetMultiFileDatabaseDefinition(_ schema.GetDefinitionContext, metadata *metadatapb.DatabaseSchemaMetadata) (*schema.MultiFileSchemaResult, error) {
	files, err := catalog.MultiFileDefinition(metadata)
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

// diffMetadataMigration generates the migration between two synced snapshots
// with omni's CockroachDB catalog.
func diffMetadataMigration(oldSchema, newSchema *model.DatabaseMetadata) (string, error) {
	return catalog.MetadataMigration(omnicatalog.DatabaseProto(oldSchema), omnicatalog.DatabaseProto(newSchema), catalog.MetadataMigrationOptions{})
}

func buildMigrationPlan(sourceText, targetText string) (*catalog.MigrationPlan, error) {
	return catalog.BuildSDLMigrationPlan(sourceText, targetText, catalog.SDLMigrationOptions{})
}

func diffSDLMigration(sourceSDL, targetSDL, _ string, _ *schema.SDLSessionContextMap) (string, error) {
	plan, err := buildMigrationPlan(sourceSDL, targetSDL)
	if err != nil || plan == nil {
		return "", err
	}
	return plan.SQL(), nil
}

func sdlDropAdvices(userSDLText string, currentSchema *model.DatabaseMetadata, _ string) ([]*storepb.Advice, error) {
	return omnicatalog.PlanSDLDropAdvices(
		storepb.Engine_COCKROACHDB,
		userSDLText,
		currentSchema,
		buildMigrationPlan,
		func(plan *catalog.MigrationPlan) []*storepb.Advice {
			var advices []*storepb.Advice
			// A routine dropped and created again holds two ops; it is one
			// replacement.
			replaced := make(map[string]bool)
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
				case catalog.OpDropConstraint:
					advices = append(advices, omnicatalog.SDLDropAdvice(fmt.Sprintf("Dropping constraint from table '%s.%s'.", op.SchemaName, op.ParentObject)))
				case catalog.OpAlterFunction:
					if replaced[op.SchemaName+"."+op.ObjectName] {
						continue
					}
					replaced[op.SchemaName+"."+op.ObjectName] = true
					advices = append(advices, omnicatalog.SDLReplaceAdvice(fmt.Sprintf("Function '%s.%s' definition will be replaced.", op.SchemaName, op.ObjectName)))
				default:
				}
			}
			return advices
		},
	)
}
