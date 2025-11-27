package pg

import (
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"

	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/store/model"
)

// processSchemaChanges processes explicit CREATE SCHEMA statements in the SDL
func processSchemaChanges(currentChunks, previousChunks *schema.SDLChunks, currentMeta *model.DatabaseMetadata, diff *schema.MetadataDiff) {
	// Build set of existing schemas in current database
	existingSchemas := make(map[string]bool)
	if currentMeta != nil {
		for _, schemaName := range currentMeta.ListSchemaNames() {
			existingSchemas[schemaName] = true
		}
	}

	// Process current schemas to find created schemas
	for schemaName := range currentChunks.Schemas {
		if _, existsInPrevious := previousChunks.Schemas[schemaName]; !existsInPrevious {
			// Schema is new in current SDL
			// Only add if it doesn't exist in current database
			if !existingSchemas[schemaName] {
				diff.SchemaChanges = append(diff.SchemaChanges, &schema.SchemaDiff{
					Action:     schema.MetadataDiffActionCreate,
					SchemaName: schemaName,
					NewSchema: &storepb.SchemaMetadata{
						Name: schemaName,
					},
				})
			}
		}
		// Note: We don't handle ALTER SCHEMA or schema modification here as it's rarely used
		// If needed in the future, we can add that logic
	}

	// Process previous schemas to find dropped schemas
	for schemaName := range previousChunks.Schemas {
		if _, existsInCurrent := currentChunks.Schemas[schemaName]; !existsInCurrent {
			// Schema was removed from SDL
			diff.SchemaChanges = append(diff.SchemaChanges, &schema.SchemaDiff{
				Action:     schema.MetadataDiffActionDrop,
				SchemaName: schemaName,
				OldSchema: &storepb.SchemaMetadata{
					Name: schemaName,
				},
			})
		}
	}
}

// addImplicitSchemaCreation adds schema creation diffs for schemas that are referenced
// by new objects but don't exist in the current database and aren't explicitly created in the SDL.
// This handles the case where users write "CREATE TABLE new_schema.t(...)" without "CREATE SCHEMA new_schema".
func addImplicitSchemaCreation(diff *schema.MetadataDiff, currentMeta *model.DatabaseMetadata) {
	if currentMeta == nil {
		return
	}

	// Collect all schemas that are referenced by objects being created
	referencedSchemas := make(map[string]bool)

	// Check tables
	for _, tableDiff := range diff.TableChanges {
		if tableDiff.Action == schema.MetadataDiffActionCreate {
			referencedSchemas[tableDiff.SchemaName] = true
		}
	}

	// Check views
	for _, viewDiff := range diff.ViewChanges {
		if viewDiff.Action == schema.MetadataDiffActionCreate {
			referencedSchemas[viewDiff.SchemaName] = true
		}
	}

	// Check materialized views
	for _, mvDiff := range diff.MaterializedViewChanges {
		if mvDiff.Action == schema.MetadataDiffActionCreate {
			referencedSchemas[mvDiff.SchemaName] = true
		}
	}

	// Check functions
	for _, funcDiff := range diff.FunctionChanges {
		if funcDiff.Action == schema.MetadataDiffActionCreate {
			referencedSchemas[funcDiff.SchemaName] = true
		}
	}

	// Check procedures
	for _, procDiff := range diff.ProcedureChanges {
		if procDiff.Action == schema.MetadataDiffActionCreate {
			referencedSchemas[procDiff.SchemaName] = true
		}
	}

	// Check sequences
	for _, seqDiff := range diff.SequenceChanges {
		if seqDiff.Action == schema.MetadataDiffActionCreate {
			referencedSchemas[seqDiff.SchemaName] = true
		}
	}

	// Check enum types
	for _, enumDiff := range diff.EnumTypeChanges {
		if enumDiff.Action == schema.MetadataDiffActionCreate {
			referencedSchemas[enumDiff.SchemaName] = true
		}
	}

	// Get existing schemas from current database
	existingSchemas := make(map[string]bool)
	for _, schemaName := range currentMeta.ListSchemaNames() {
		existingSchemas[schemaName] = true
	}

	// Check which schemas are already in SchemaChanges with CREATE action
	schemasBeingCreated := make(map[string]bool)
	for _, schemaDiff := range diff.SchemaChanges {
		if schemaDiff.Action == schema.MetadataDiffActionCreate {
			schemasBeingCreated[schemaDiff.SchemaName] = true
		}
	}

	// Add schema creation for schemas that:
	// 1. Are referenced by new objects
	// 2. Don't exist in current database
	// 3. Aren't already being created
	for schemaName := range referencedSchemas {
		if !existingSchemas[schemaName] && !schemasBeingCreated[schemaName] {
			// Add schema creation diff
			diff.SchemaChanges = append(diff.SchemaChanges, &schema.SchemaDiff{
				Action:     schema.MetadataDiffActionCreate,
				SchemaName: schemaName,
				NewSchema: &storepb.SchemaMetadata{
					Name: schemaName,
				},
			})
		}
	}
}
