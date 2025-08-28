package pg

import (
	"fmt"
	"slices"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	pgparser "github.com/bytebase/parser/postgresql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	parserbase "github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/plugin/schema"
)

func init() {
	schema.RegisterGenerateMigration(storepb.Engine_POSTGRES, generateMigration)
}

func generateMigration(diff *schema.MetadataDiff) (string, error) {
	var buf strings.Builder

	// Safe order for migrations:
	// 1. Drop dependent objects first (in reverse dependency order)
	//    - Use topological sort to drop in safe order
	// 2. Create/Alter objects (in dependency order)
	//    - Use topological sort to create in safe order

	// Phase 1: Drop dependent objects using topological sort
	dropObjectsInOrder(diff, &buf)

	// Only add blank line if we have drops AND we're about to create something
	dropPhaseHasContent := buf.Len() > 0
	createPhaseWillHaveContent := hasCreateOrAlterObjects(diff)

	if dropPhaseHasContent && createPhaseWillHaveContent {
		_, _ = buf.WriteString("\n")
	}

	// Phase 2: Create/Alter objects using topological sort
	if err := createObjectsInOrder(diff, &buf); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// dropObjectsInOrder drops all objects in reverse topological order (most dependent first)
func dropObjectsInOrder(diff *schema.MetadataDiff, buf *strings.Builder) {
	// First, drop all triggers that might depend on functions we're about to drop
	// This is necessary because PostgreSQL doesn't allow dropping functions that are used by triggers
	functionsBeingDropped := make(map[string]bool)
	for _, funcDiff := range diff.FunctionChanges {
		if funcDiff.Action == schema.MetadataDiffActionDrop {
			funcName := strings.ToLower(funcDiff.FunctionName)
			functionsBeingDropped[funcName] = true
		}
	}

	// Drop triggers that might depend on functions being dropped
	// This includes triggers from tables being dropped AND from tables being altered
	for _, tableDiff := range diff.TableChanges {
		if tableDiff.OldTable != nil {
			for _, trigger := range tableDiff.OldTable.Triggers {
				// Check if trigger body references any function being dropped
				triggerBody := strings.ToLower(trigger.Body)
				for funcName := range functionsBeingDropped {
					if strings.Contains(triggerBody, funcName) {
						writeDropTrigger(buf, tableDiff.SchemaName, tableDiff.TableName, trigger.Name)
						break
					}
				}
			}
		}
	}

	// Build dependency graph for all objects being dropped or altered
	graph := parserbase.NewGraph()

	// Maps to store different object types
	viewMap := make(map[string]*schema.ViewDiff)
	materializedViewMap := make(map[string]*schema.MaterializedViewDiff)
	tableMap := make(map[string]*schema.TableDiff)
	functionMap := make(map[string]*schema.FunctionDiff)

	// Track all object IDs for dependency resolution
	allObjects := make(map[string]bool)

	// Collect schemas being dropped to ensure their objects are dropped first
	schemasBeingDropped := make(map[string]bool)
	for _, schemaDiff := range diff.SchemaChanges {
		if schemaDiff.Action == schema.MetadataDiffActionDrop {
			schemasBeingDropped[schemaDiff.SchemaName] = true
		}
	}

	// Add views to graph
	for _, viewDiff := range diff.ViewChanges {
		if viewDiff.Action == schema.MetadataDiffActionDrop || viewDiff.Action == schema.MetadataDiffActionAlter {
			viewID := getMigrationObjectID(viewDiff.SchemaName, viewDiff.ViewName)
			graph.AddNode(viewID)
			viewMap[viewID] = viewDiff
			allObjects[viewID] = true
		}
	}

	// Add materialized views to graph
	for _, mvDiff := range diff.MaterializedViewChanges {
		if mvDiff.Action == schema.MetadataDiffActionDrop || mvDiff.Action == schema.MetadataDiffActionAlter {
			mvID := getMigrationObjectID(mvDiff.SchemaName, mvDiff.MaterializedViewName)
			graph.AddNode(mvID)
			materializedViewMap[mvID] = mvDiff
			allObjects[mvID] = true
		}
	}

	// Add functions to graph
	for _, funcDiff := range diff.FunctionChanges {
		if funcDiff.Action == schema.MetadataDiffActionDrop {
			funcID := getMigrationObjectID(funcDiff.SchemaName, funcDiff.FunctionName)
			graph.AddNode(funcID)
			functionMap[funcID] = funcDiff
			allObjects[funcID] = true
		}
	}

	// Add tables to graph
	for _, tableDiff := range diff.TableChanges {
		if tableDiff.Action == schema.MetadataDiffActionDrop {
			tableID := getMigrationObjectID(tableDiff.SchemaName, tableDiff.TableName)
			graph.AddNode(tableID)
			tableMap[tableID] = tableDiff
			allObjects[tableID] = true
		}
	}

	// Add dependency edges
	// For views depending on tables/views
	for viewID, viewDiff := range viewMap {
		if viewDiff.OldView != nil {
			for _, dep := range viewDiff.OldView.DependencyColumns {
				depID := getMigrationObjectID(dep.Schema, dep.Table)
				if allObjects[depID] {
					// Edge from dependent to dependency (view depends on table/view)
					graph.AddEdge(viewID, depID)
				}
			}
		}
	}

	// For materialized views depending on tables/views
	for mvID, mvDiff := range materializedViewMap {
		if mvDiff.OldMaterializedView != nil {
			for _, dep := range mvDiff.OldMaterializedView.DependencyColumns {
				depID := getMigrationObjectID(dep.Schema, dep.Table)
				if allObjects[depID] {
					// Edge from dependent to dependency
					graph.AddEdge(mvID, depID)
				}
			}
		}
	}

	// For functions depending on tables
	for funcID, funcDiff := range functionMap {
		if funcDiff.OldFunction != nil {
			for _, dep := range funcDiff.OldFunction.DependencyTables {
				depID := getMigrationObjectID(dep.Schema, dep.Table)
				if allObjects[depID] {
					// Edge from function to table
					graph.AddEdge(funcID, depID)
				}
			}
		}
	}

	// For tables with foreign keys depending on other tables
	for tableID, tableDiff := range tableMap {
		if tableDiff.OldTable != nil {
			for _, fk := range tableDiff.OldTable.ForeignKeys {
				depID := getMigrationObjectID(fk.ReferencedSchema, fk.ReferencedTable)
				if allObjects[depID] && depID != tableID {
					// Edge from table with FK to referenced table
					graph.AddEdge(tableID, depID)
				}
			}
		}
	}

	// Get topological order
	orderedList, err := graph.TopologicalSort()
	if err != nil {
		// If there's a cycle, fall back to a safe order
		// First drop foreign keys from ALTER operations that might reference tables being dropped
		for _, tableDiff := range diff.TableChanges {
			if tableDiff.Action == schema.MetadataDiffActionAlter {
				for _, fkDiff := range tableDiff.ForeignKeyChanges {
					if fkDiff.Action == schema.MetadataDiffActionDrop {
						writeDropForeignKey(buf, tableDiff.SchemaName, tableDiff.TableName, fkDiff.OldForeignKey.Name)
					}
				}
			}
		}

		// Drop triggers
		for _, tableDiff := range diff.TableChanges {
			if tableDiff.Action == schema.MetadataDiffActionDrop && tableDiff.OldTable != nil {
				for _, trigger := range tableDiff.OldTable.Triggers {
					writeDropTrigger(buf, tableDiff.SchemaName, tableDiff.TableName, trigger.Name)
				}
			}
		}

		// Drop foreign keys from tables being dropped
		for _, tableDiff := range diff.TableChanges {
			if tableDiff.Action == schema.MetadataDiffActionDrop && tableDiff.OldTable != nil {
				for _, fk := range tableDiff.OldTable.ForeignKeys {
					writeDropForeignKey(buf, tableDiff.SchemaName, tableDiff.TableName, fk.Name)
				}
			}
		}

		// Drop views
		for _, viewDiff := range viewMap {
			writeDropView(buf, viewDiff.SchemaName, viewDiff.ViewName)
		}

		// Drop materialized views
		for _, mvDiff := range materializedViewMap {
			writeDropMaterializedView(buf, mvDiff.SchemaName, mvDiff.MaterializedViewName)
		}

		// Drop functions
		for _, funcDiff := range functionMap {
			writeDropFunction(buf, funcDiff.SchemaName, funcDiff.FunctionName)
		}

		// Drop tables
		for _, tableDiff := range tableMap {
			writeDropTable(buf, tableDiff.SchemaName, tableDiff.TableName)
		}

		// Handle remaining ALTER table operations (constraints, indexes, columns)
		for _, tableDiff := range diff.TableChanges {
			if tableDiff.Action == schema.MetadataDiffActionAlter {
				// Drop check constraints
				for _, checkDiff := range tableDiff.CheckConstraintChanges {
					if checkDiff.Action == schema.MetadataDiffActionDrop {
						writeDropCheckConstraint(buf, tableDiff.SchemaName, tableDiff.TableName, checkDiff.OldCheckConstraint.Name)
					}
				}

				// Drop indexes
				for _, indexDiff := range tableDiff.IndexChanges {
					if indexDiff.Action == schema.MetadataDiffActionDrop {
						if indexDiff.OldIndex.IsConstraint {
							writeDropConstraint(buf, tableDiff.SchemaName, tableDiff.TableName, indexDiff.OldIndex.Name)
						} else {
							writeDropIndex(buf, tableDiff.SchemaName, indexDiff.OldIndex.Name)
						}
					}
				}

				// Drop columns
				for _, colDiff := range tableDiff.ColumnChanges {
					if colDiff.Action == schema.MetadataDiffActionDrop {
						writeDropColumn(buf, tableDiff.SchemaName, tableDiff.TableName, colDiff.OldColumn.Name)
					}
				}
			}
		}
	} else {
		// First, handle ALTER table drops (foreign keys that might reference tables being dropped)
		for _, tableDiff := range diff.TableChanges {
			if tableDiff.Action == schema.MetadataDiffActionAlter {
				// Drop foreign keys first
				for _, fkDiff := range tableDiff.ForeignKeyChanges {
					if fkDiff.Action == schema.MetadataDiffActionDrop {
						writeDropForeignKey(buf, tableDiff.SchemaName, tableDiff.TableName, fkDiff.OldForeignKey.Name)
					}
				}
			}
		}

		// Drop in topological order (most dependent first)
		for _, objID := range orderedList {
			// Drop triggers for tables being dropped
			if tableDiff, ok := tableMap[objID]; ok && tableDiff.OldTable != nil {
				for _, trigger := range tableDiff.OldTable.Triggers {
					writeDropTrigger(buf, tableDiff.SchemaName, tableDiff.TableName, trigger.Name)
				}
			}

			// Drop the object itself
			if viewDiff, ok := viewMap[objID]; ok {
				writeDropView(buf, viewDiff.SchemaName, viewDiff.ViewName)
			} else if mvDiff, ok := materializedViewMap[objID]; ok {
				writeDropMaterializedView(buf, mvDiff.SchemaName, mvDiff.MaterializedViewName)
			} else if funcDiff, ok := functionMap[objID]; ok {
				writeDropFunction(buf, funcDiff.SchemaName, funcDiff.FunctionName)
			} else if tableDiff, ok := tableMap[objID]; ok {
				// Drop foreign keys before table
				if tableDiff.OldTable != nil {
					for _, fk := range tableDiff.OldTable.ForeignKeys {
						writeDropForeignKey(buf, tableDiff.SchemaName, tableDiff.TableName, fk.Name)
					}
				}
				writeDropTable(buf, tableDiff.SchemaName, tableDiff.TableName)
			}
		}

		// Handle remaining ALTER table drops (constraints, indexes, columns)
		for _, tableDiff := range diff.TableChanges {
			if tableDiff.Action == schema.MetadataDiffActionAlter {
				// Drop check constraints
				for _, checkDiff := range tableDiff.CheckConstraintChanges {
					if checkDiff.Action == schema.MetadataDiffActionDrop {
						writeDropCheckConstraint(buf, tableDiff.SchemaName, tableDiff.TableName, checkDiff.OldCheckConstraint.Name)
					}
				}

				// Drop indexes
				for _, indexDiff := range tableDiff.IndexChanges {
					if indexDiff.Action == schema.MetadataDiffActionDrop {
						if indexDiff.OldIndex.IsConstraint {
							writeDropConstraint(buf, tableDiff.SchemaName, tableDiff.TableName, indexDiff.OldIndex.Name)
						} else {
							writeDropIndex(buf, tableDiff.SchemaName, indexDiff.OldIndex.Name)
						}
					}
				}

				// Drop columns
				for _, colDiff := range tableDiff.ColumnChanges {
					if colDiff.Action == schema.MetadataDiffActionDrop {
						writeDropColumn(buf, tableDiff.SchemaName, tableDiff.TableName, colDiff.OldColumn.Name)
					}
				}
			}
		}
	}

	// Drop enum types
	for _, enumDiff := range diff.EnumTypeChanges {
		if enumDiff.Action == schema.MetadataDiffActionDrop {
			writeDropType(buf, enumDiff.SchemaName, enumDiff.EnumTypeName)
		}
	}

	// Drop sequences - prioritize sequences in schemas being dropped
	var sequencesInDroppedSchemas []string
	var otherSequences []string

	for _, seqDiff := range diff.SequenceChanges {
		if seqDiff.Action == schema.MetadataDiffActionDrop {
			seqInfo := fmt.Sprintf("%s.%s", seqDiff.SchemaName, seqDiff.SequenceName)
			if schemasBeingDropped[seqDiff.SchemaName] {
				sequencesInDroppedSchemas = append(sequencesInDroppedSchemas, seqInfo)
			} else {
				otherSequences = append(otherSequences, seqInfo)
			}
		}
	}

	// Drop sequences in schemas being dropped first
	for _, seqInfo := range sequencesInDroppedSchemas {
		parts := strings.Split(seqInfo, ".")
		writeDropSequence(buf, parts[0], parts[1])
	}

	// Then drop other sequences
	for _, seqInfo := range otherSequences {
		parts := strings.Split(seqInfo, ".")
		writeDropSequence(buf, parts[0], parts[1])
	}

	// Drop schemas (must be empty)
	for _, schemaDiff := range diff.SchemaChanges {
		if schemaDiff.Action == schema.MetadataDiffActionDrop {
			// Skip dropping pg_catalog and public schemas as they are system schemas
			if schemaDiff.SchemaName == "pg_catalog" || schemaDiff.SchemaName == "public" {
				continue
			}

			// Before dropping a schema, we need to drop all objects inside it
			// This handles cases where the diff doesn't include objects within a schema being dropped
			if schemaDiff.OldSchema != nil {
				// Drop all objects in the schema in the correct order

				// Drop views first (they might depend on tables)
				for _, view := range schemaDiff.OldSchema.Views {
					writeDropView(buf, schemaDiff.SchemaName, view.Name)
				}

				// Drop materialized views
				for _, mv := range schemaDiff.OldSchema.MaterializedViews {
					writeDropMaterializedView(buf, schemaDiff.SchemaName, mv.Name)
				}

				// Drop functions
				for _, fn := range schemaDiff.OldSchema.Functions {
					writeDropFunction(buf, schemaDiff.SchemaName, fn.Name)
				}

				// Drop tables (this will handle foreign keys internally)
				for _, table := range schemaDiff.OldSchema.Tables {
					// Drop triggers first
					for _, trigger := range table.Triggers {
						writeDropTrigger(buf, schemaDiff.SchemaName, table.Name, trigger.Name)
					}

					// Drop the table
					writeDropTable(buf, schemaDiff.SchemaName, table.Name)
				}

				// Drop sequences
				for _, seq := range schemaDiff.OldSchema.Sequences {
					writeDropSequence(buf, schemaDiff.SchemaName, seq.Name)
				}

				// Drop types (enums)
				for _, enum := range schemaDiff.OldSchema.EnumTypes {
					writeDropType(buf, schemaDiff.SchemaName, enum.Name)
				}
			}

			writeDropSchema(buf, schemaDiff.SchemaName)
		} else if schemaDiff.Action == schema.MetadataDiffActionAlter {
			// Handle ALTER schema - check for enum types that need to be dropped
			// This handles cases where enum types are added to an existing schema
			if schemaDiff.OldSchema != nil && schemaDiff.NewSchema != nil {
				// Build a map of new enum types
				newEnumMap := make(map[string]bool)
				for _, enum := range schemaDiff.NewSchema.EnumTypes {
					newEnumMap[enum.Name] = true
				}

				// Drop enum types that exist in old but not in new
				for _, enum := range schemaDiff.OldSchema.EnumTypes {
					if !newEnumMap[enum.Name] {
						writeDropType(buf, schemaDiff.SchemaName, enum.Name)
					}
				}
			}
		}
	}
}

// createObjectsInOrder creates all objects in topological order (dependencies first)
func createObjectsInOrder(diff *schema.MetadataDiff, buf *strings.Builder) error {
	// First create schemas (they don't have dependencies)
	var schemasToCreate []string
	for _, schemaDiff := range diff.SchemaChanges {
		if schemaDiff.Action == schema.MetadataDiffActionCreate {
			// Skip creating pg_catalog and public schemas as they already exist by default
			if schemaDiff.SchemaName != "pg_catalog" && schemaDiff.SchemaName != "public" {
				schemasToCreate = append(schemasToCreate, schemaDiff.SchemaName)
			}
		}
	}
	slices.Sort(schemasToCreate)
	for _, schemaName := range schemasToCreate {
		if err := writeCreateSchema(buf, schemaName); err != nil {
			return err
		}
	}

	// Add blank line after schema creation only if we have schemas and more content follows
	if len(schemasToCreate) > 0 && (hasCreateOrAlterTables(diff) || hasCreateViewsOrFunctions(diff)) {
		_, _ = buf.WriteString("\n")
	}

	// Create enum types (before tables as they might be used in column definitions)
	for _, enumDiff := range diff.EnumTypeChanges {
		if enumDiff.Action == schema.MetadataDiffActionCreate {
			if err := writeCreateEnumType(buf, enumDiff.SchemaName, enumDiff.NewEnumType); err != nil {
				return err
			}
		}
	}

	// Create sequences (before tables as they might be used in column defaults)
	for _, seqDiff := range diff.SequenceChanges {
		if seqDiff.Action == schema.MetadataDiffActionCreate {
			if err := writeMigrationCreateSequence(buf, seqDiff.SchemaName, seqDiff.NewSequence); err != nil {
				return err
			}
		}
	}

	// Build dependency graph for all objects being created or altered
	graph := parserbase.NewGraph()

	// Maps to store different object types
	viewMap := make(map[string]*schema.ViewDiff)
	materializedViewMap := make(map[string]*schema.MaterializedViewDiff)
	tableMap := make(map[string]*schema.TableDiff)
	functionMap := make(map[string]*schema.FunctionDiff)

	// Track all object IDs for dependency resolution
	allObjects := make(map[string]bool)

	// Add tables to graph (both CREATE and ALTER for column additions)
	for _, tableDiff := range diff.TableChanges {
		if tableDiff.Action == schema.MetadataDiffActionCreate || tableDiff.Action == schema.MetadataDiffActionAlter {
			tableID := getMigrationObjectID(tableDiff.SchemaName, tableDiff.TableName)
			graph.AddNode(tableID)
			tableMap[tableID] = tableDiff
			allObjects[tableID] = true
		}
	}

	// Add views to graph
	for _, viewDiff := range diff.ViewChanges {
		if viewDiff.Action == schema.MetadataDiffActionCreate || viewDiff.Action == schema.MetadataDiffActionAlter {
			viewID := getMigrationObjectID(viewDiff.SchemaName, viewDiff.ViewName)
			graph.AddNode(viewID)
			viewMap[viewID] = viewDiff
			allObjects[viewID] = true
		}
	}

	// Add materialized views to graph
	for _, mvDiff := range diff.MaterializedViewChanges {
		if mvDiff.Action == schema.MetadataDiffActionCreate || mvDiff.Action == schema.MetadataDiffActionAlter {
			mvID := getMigrationObjectID(mvDiff.SchemaName, mvDiff.MaterializedViewName)
			graph.AddNode(mvID)
			materializedViewMap[mvID] = mvDiff
			allObjects[mvID] = true
		}
	}

	// Add functions to graph
	for _, funcDiff := range diff.FunctionChanges {
		if funcDiff.Action == schema.MetadataDiffActionCreate || funcDiff.Action == schema.MetadataDiffActionAlter {
			funcID := getMigrationObjectID(funcDiff.SchemaName, funcDiff.FunctionName)
			graph.AddNode(funcID)
			functionMap[funcID] = funcDiff
			allObjects[funcID] = true
		}
	}

	// Add dependency edges
	// For tables with foreign keys depending on other tables (only for CREATE operations)
	for tableID, tableDiff := range tableMap {
		if tableDiff.Action == schema.MetadataDiffActionCreate && tableDiff.NewTable != nil {
			for _, fk := range tableDiff.NewTable.ForeignKeys {
				depID := getMigrationObjectID(fk.ReferencedSchema, fk.ReferencedTable)
				if depID != tableID {
					// Edge from dependency to dependent (referenced table to table with FK)
					graph.AddEdge(depID, tableID)
				}
			}
		}
	}

	// For views depending on tables/views
	for viewID, viewDiff := range viewMap {
		if viewDiff.NewView != nil {
			for _, dep := range viewDiff.NewView.DependencyColumns {
				depID := getMigrationObjectID(dep.Schema, dep.Table)
				// Edge from dependency to dependent (table/view to view)
				graph.AddEdge(depID, viewID)
			}
		}
	}

	// For materialized views depending on tables/views
	for mvID, mvDiff := range materializedViewMap {
		if mvDiff.NewMaterializedView != nil {
			for _, dep := range mvDiff.NewMaterializedView.DependencyColumns {
				depID := getMigrationObjectID(dep.Schema, dep.Table)
				// Edge from dependency to dependent
				graph.AddEdge(depID, mvID)
			}
		}
	}

	// For functions depending on tables
	for funcID, funcDiff := range functionMap {
		if funcDiff.NewFunction != nil {
			for _, dep := range funcDiff.NewFunction.DependencyTables {
				depID := getMigrationObjectID(dep.Schema, dep.Table)
				// Edge from table to function
				graph.AddEdge(depID, funcID)
			}
		}
	}

	// Get topological order
	orderedList, err := graph.TopologicalSort()
	if err != nil {
		// If there's a cycle, fall back to a safe order
		// Create tables first (without foreign keys)
		for _, tableDiff := range tableMap {
			if tableDiff.Action == schema.MetadataDiffActionCreate {
				createTableSQL, err := generateCreateTable(tableDiff.SchemaName, tableDiff.TableName, tableDiff.NewTable, false)
				if err != nil {
					return err
				}
				_, _ = buf.WriteString(createTableSQL)
				if createTableSQL != "" {
					_, _ = buf.WriteString("\n")
				}

				// Add table and column comments for newly created tables
				if tableDiff.NewTable != nil && tableDiff.NewTable.Comment != "" {
					writeCommentOnTable(buf, tableDiff.SchemaName, tableDiff.TableName, tableDiff.NewTable.Comment)
				}
				if tableDiff.NewTable != nil {
					for _, col := range tableDiff.NewTable.Columns {
						if col.Comment != "" {
							writeCommentOnColumn(buf, tableDiff.SchemaName, tableDiff.TableName, col.Name, col.Comment)
						}
					}
				}
			}
		}

		// Handle column additions for ALTER operations (only columns in topological order)
		for _, tableDiff := range tableMap {
			if tableDiff.Action == schema.MetadataDiffActionAlter {
				for _, colDiff := range tableDiff.ColumnChanges {
					if colDiff.Action == schema.MetadataDiffActionCreate {
						writeAddColumn(buf, tableDiff.SchemaName, tableDiff.TableName, colDiff.NewColumn)
					}
				}
			}
		}

		// Create views
		for _, viewDiff := range viewMap {
			switch viewDiff.Action {
			case schema.MetadataDiffActionCreate, schema.MetadataDiffActionAlter:
				if err := writeMigrationView(buf, viewDiff.SchemaName, viewDiff.NewView); err != nil {
					return err
				}
			default:
				// No action needed
			}
			// Add view comment for newly created views
			if viewDiff.NewView != nil && viewDiff.NewView.Comment != "" {
				writeCommentOnView(buf, viewDiff.SchemaName, viewDiff.ViewName, viewDiff.NewView.Comment)
			}
		}

		// Create materialized views
		for _, mvDiff := range materializedViewMap {
			switch mvDiff.Action {
			case schema.MetadataDiffActionCreate, schema.MetadataDiffActionAlter:
				if err := writeMigrationMaterializedView(buf, mvDiff.SchemaName, mvDiff.NewMaterializedView); err != nil {
					return err
				}
			default:
				// No action needed
			}
			// Add materialized view comment for newly created materialized views
			if mvDiff.NewMaterializedView != nil && mvDiff.NewMaterializedView.Comment != "" {
				writeCommentOnMaterializedView(buf, mvDiff.SchemaName, mvDiff.MaterializedViewName, mvDiff.NewMaterializedView.Comment)
			}
		}

		// Create functions
		for _, funcDiff := range functionMap {
			if err := writeFunctionDiff(buf, funcDiff); err != nil {
				return err
			}
			// Add function comment for newly created functions
			if funcDiff.Action == schema.MetadataDiffActionCreate && funcDiff.NewFunction != nil && funcDiff.NewFunction.Comment != "" {
				writeCommentOnFunction(buf, funcDiff.SchemaName, funcDiff.NewFunction.Signature, funcDiff.NewFunction.Comment)
			}
		}

		// Set sequence ownership after all tables are created
		for _, seqDiff := range diff.SequenceChanges {
			if seqDiff.Action == schema.MetadataDiffActionCreate && seqDiff.NewSequence.OwnerTable != "" && seqDiff.NewSequence.OwnerColumn != "" {
				if err := writeMigrationSequenceOwnership(buf, seqDiff.SchemaName, seqDiff.NewSequence); err != nil {
					return err
				}
			}
		}

		// Add foreign keys (only for CREATE table operations)
		for _, tableDiff := range tableMap {
			if tableDiff.Action == schema.MetadataDiffActionCreate && tableDiff.NewTable != nil {
				for _, fk := range tableDiff.NewTable.ForeignKeys {
					if err := writeMigrationForeignKey(buf, tableDiff.SchemaName, tableDiff.TableName, fk); err != nil {
						return err
					}
				}
			}
		}
	} else {
		// Follow topological order completely - create objects in dependency order
		for _, objID := range orderedList {
			if tableDiff, ok := tableMap[objID]; ok {
				switch tableDiff.Action {
				case schema.MetadataDiffActionCreate:
					// Create table without foreign keys
					createTableSQL, err := generateCreateTable(tableDiff.SchemaName, tableDiff.TableName, tableDiff.NewTable, false)
					if err != nil {
						return err
					}
					_, _ = buf.WriteString(createTableSQL)
					if createTableSQL != "" {
						_, _ = buf.WriteString("\n")
					}

					// Add table and column comments for newly created tables
					if tableDiff.NewTable != nil && tableDiff.NewTable.Comment != "" {
						writeCommentOnTable(buf, tableDiff.SchemaName, tableDiff.TableName, tableDiff.NewTable.Comment)
					}
					if tableDiff.NewTable != nil {
						for _, col := range tableDiff.NewTable.Columns {
							if col.Comment != "" {
								writeCommentOnColumn(buf, tableDiff.SchemaName, tableDiff.TableName, col.Name, col.Comment)
							}
						}
					}
				case schema.MetadataDiffActionAlter:
					// Handle column additions for ALTER operations
					for _, colDiff := range tableDiff.ColumnChanges {
						if colDiff.Action == schema.MetadataDiffActionCreate {
							writeAddColumn(buf, tableDiff.SchemaName, tableDiff.TableName, colDiff.NewColumn)
						}
					}
				default:
					// No action needed for other operations
				}
			} else if viewDiff, ok := viewMap[objID]; ok {
				switch viewDiff.Action {
				case schema.MetadataDiffActionCreate, schema.MetadataDiffActionAlter:
					if err := writeMigrationView(buf, viewDiff.SchemaName, viewDiff.NewView); err != nil {
						return err
					}
				default:
					// No action needed for other operations
				}
				// Add view comment for newly created or altered views
				if viewDiff.NewView != nil && viewDiff.NewView.Comment != "" {
					writeCommentOnView(buf, viewDiff.SchemaName, viewDiff.ViewName, viewDiff.NewView.Comment)
				}
			} else if mvDiff, ok := materializedViewMap[objID]; ok {
				switch mvDiff.Action {
				case schema.MetadataDiffActionCreate:
					if err := writeMigrationMaterializedView(buf, mvDiff.SchemaName, mvDiff.NewMaterializedView); err != nil {
						return err
					}
				case schema.MetadataDiffActionAlter:
					// For PostgreSQL materialized views, we need to drop and recreate
					// since ALTER MATERIALIZED VIEW doesn't support changing the definition
					writeDropMaterializedView(buf, mvDiff.SchemaName, mvDiff.MaterializedViewName)
					if err := writeMigrationMaterializedView(buf, mvDiff.SchemaName, mvDiff.NewMaterializedView); err != nil {
						return err
					}
				default:
					// No action needed for other operations
				}
				// Add materialized view comment for newly created or altered materialized views
				if mvDiff.NewMaterializedView != nil && mvDiff.NewMaterializedView.Comment != "" {
					writeCommentOnMaterializedView(buf, mvDiff.SchemaName, mvDiff.MaterializedViewName, mvDiff.NewMaterializedView.Comment)
				}
			} else if funcDiff, ok := functionMap[objID]; ok {
				if err := writeFunctionDiff(buf, funcDiff); err != nil {
					return err
				}
				// Add function comment for newly created or altered functions
				if (funcDiff.Action == schema.MetadataDiffActionCreate || funcDiff.Action == schema.MetadataDiffActionAlter) && funcDiff.NewFunction != nil && funcDiff.NewFunction.Comment != "" {
					writeCommentOnFunction(buf, funcDiff.SchemaName, funcDiff.NewFunction.Signature, funcDiff.NewFunction.Comment)
				}
			}
		}

		// Set sequence ownership after all tables are created
		for _, seqDiff := range diff.SequenceChanges {
			if seqDiff.Action == schema.MetadataDiffActionCreate && seqDiff.NewSequence.OwnerTable != "" && seqDiff.NewSequence.OwnerColumn != "" {
				if err := writeMigrationSequenceOwnership(buf, seqDiff.SchemaName, seqDiff.NewSequence); err != nil {
					return err
				}
			}
		}

		// Add foreign keys after all tables are created (only for CREATE table operations)
		for _, tableDiff := range tableMap {
			if tableDiff.Action == schema.MetadataDiffActionCreate && tableDiff.NewTable != nil {
				for _, fk := range tableDiff.NewTable.ForeignKeys {
					if err := writeMigrationForeignKey(buf, tableDiff.SchemaName, tableDiff.TableName, fk); err != nil {
						return err
					}
				}
			}
		}

		// Create triggers after foreign keys (only for CREATE table operations)
		for _, tableDiff := range tableMap {
			if tableDiff.Action == schema.MetadataDiffActionCreate && tableDiff.NewTable != nil {
				for _, trigger := range tableDiff.NewTable.Triggers {
					writeMigrationTrigger(buf, trigger)
				}
			}
		}

		// Create indexes for materialized views after they are created
		for _, mvDiff := range diff.MaterializedViewChanges {
			if mvDiff.Action == schema.MetadataDiffActionCreate && mvDiff.NewMaterializedView != nil {
				for _, index := range mvDiff.NewMaterializedView.Indexes {
					writeMigrationMaterializedViewIndex(buf, mvDiff.SchemaName, mvDiff.NewMaterializedView.Name, index)
				}
			}
		}
	}

	// Handle ALTER table operations
	for _, tableDiff := range diff.TableChanges {
		if tableDiff.Action == schema.MetadataDiffActionAlter {
			// Skip column additions as they were handled earlier
			alterTableSQL, err := generateAlterTableWithOptions(tableDiff, false)
			if err != nil {
				return err
			}
			_, _ = buf.WriteString(alterTableSQL)
			if alterTableSQL != "" {
				_, _ = buf.WriteString("\n")
			}

			// Handle table comment changes
			if err := generateTableCommentChanges(buf, tableDiff); err != nil {
				return err
			}

			// Handle column comment changes
			if err := generateColumnCommentChanges(buf, tableDiff); err != nil {
				return err
			}

			// Handle index comment changes
			if err := generateIndexCommentChanges(buf, tableDiff); err != nil {
				return err
			}
		}
	}

	// Handle schema comment changes
	if err := generateSchemaCommentChanges(buf, diff); err != nil {
		return err
	}

	// Handle view comment changes
	if err := generateViewCommentChanges(buf, diff); err != nil {
		return err
	}

	// Handle materialized view comment changes
	if err := generateMaterializedViewCommentChanges(buf, diff); err != nil {
		return err
	}

	// Handle function comment changes
	if err := generateFunctionCommentChanges(buf, diff); err != nil {
		return err
	}

	// Handle sequence comment changes
	if err := generateSequenceCommentChanges(buf, diff); err != nil {
		return err
	}

	// Handle enum type comment changes
	return generateEnumTypeCommentChanges(buf, diff)
}

func generateCreateTable(schemaName, tableName string, table *storepb.TableMetadata, includeForeignKeys bool) (string, error) {
	var buf strings.Builder

	writeMigrationCreateTable(&buf, schemaName, tableName, table.Columns, table.CheckConstraints)

	// Add partitioning clause if needed
	if len(table.Partitions) > 0 {
		writeMigrationPartitionClause(&buf, table.Partitions[0])
	}

	if _, err := buf.WriteString(";\n"); err != nil {
		return "", err
	}

	// Add constraints (primary key, unique)
	for _, index := range table.Indexes {
		if index.Primary {
			writeMigrationPrimaryKey(&buf, schemaName, tableName, index)
		} else if index.Unique && index.IsConstraint {
			writeMigrationUniqueKey(&buf, schemaName, tableName, index)
		}
	}

	// Add non-constraint indexes
	for _, index := range table.Indexes {
		if !index.IsConstraint {
			writeMigrationIndex(&buf, schemaName, tableName, index)
		}
	}

	// Optionally add foreign keys
	if includeForeignKeys && table != nil {
		for _, fk := range table.ForeignKeys {
			if err := writeMigrationForeignKey(&buf, schemaName, tableName, fk); err != nil {
				return "", err
			}
		}
	}

	return buf.String(), nil
}

func generateAlterTableWithOptions(tableDiff *schema.TableDiff, includeColumnAdditions bool) (string, error) {
	var buf strings.Builder

	// Add columns first (other operations might depend on them)
	if includeColumnAdditions {
		for _, colDiff := range tableDiff.ColumnChanges {
			if colDiff.Action == schema.MetadataDiffActionCreate {
				writeAddColumn(&buf, tableDiff.SchemaName, tableDiff.TableName, colDiff.NewColumn)
			}
		}
	}

	// Alter columns
	for _, colDiff := range tableDiff.ColumnChanges {
		if colDiff.Action == schema.MetadataDiffActionAlter {
			alterColSQL := generateAlterColumn(tableDiff.SchemaName, tableDiff.TableName, colDiff)
			_, _ = buf.WriteString(alterColSQL)
		}
	}

	// Add indexes
	for _, indexDiff := range tableDiff.IndexChanges {
		if indexDiff.Action == schema.MetadataDiffActionCreate {
			if indexDiff.NewIndex.IsConstraint {
				// Add constraint (primary key or unique constraint)
				if indexDiff.NewIndex.Primary {
					writeMigrationPrimaryKey(&buf, tableDiff.SchemaName, tableDiff.TableName, indexDiff.NewIndex)
				} else if indexDiff.NewIndex.Unique {
					writeMigrationUniqueKey(&buf, tableDiff.SchemaName, tableDiff.TableName, indexDiff.NewIndex)
				}
			} else {
				writeMigrationIndex(&buf, tableDiff.SchemaName, tableDiff.TableName, indexDiff.NewIndex)
			}
		}
	}

	// Add check constraints
	for _, checkDiff := range tableDiff.CheckConstraintChanges {
		if checkDiff.Action == schema.MetadataDiffActionCreate {
			writeAddCheckConstraint(&buf, tableDiff.SchemaName, tableDiff.TableName, checkDiff.NewCheckConstraint)
		}
	}

	// Add foreign keys last (they depend on other tables/columns)
	for _, fkDiff := range tableDiff.ForeignKeyChanges {
		if fkDiff.Action == schema.MetadataDiffActionCreate {
			if err := writeMigrationForeignKey(&buf, tableDiff.SchemaName, tableDiff.TableName, fkDiff.NewForeignKey); err != nil {
				return "", err
			}
		}
	}

	// Add triggers after foreign keys
	for _, triggerDiff := range tableDiff.TriggerChanges {
		if triggerDiff.Action == schema.MetadataDiffActionCreate {
			writeMigrationTrigger(&buf, triggerDiff.NewTrigger)
		}
	}

	return buf.String(), nil
}

func generateAlterColumn(schemaName, tableName string, colDiff *schema.ColumnDiff) string {
	var buf strings.Builder

	// In PostgreSQL, we need to handle different aspects of column changes separately

	// If type changed, alter the column type
	if colDiff.OldColumn.Type != colDiff.NewColumn.Type {
		writeAlterColumnType(&buf, schemaName, tableName, colDiff.NewColumn.Name, colDiff.NewColumn.Type)
	}

	// If nullability changed
	if colDiff.OldColumn.Nullable != colDiff.NewColumn.Nullable {
		if colDiff.NewColumn.Nullable {
			writeAlterColumnDropNotNull(&buf, schemaName, tableName, colDiff.NewColumn.Name)
		} else {
			writeAlterColumnSetNotNull(&buf, schemaName, tableName, colDiff.NewColumn.Name)
		}
	}

	// Handle default value changes
	oldHasDefault := hasDefaultValue(colDiff.OldColumn)
	newHasDefault := hasDefaultValue(colDiff.NewColumn)
	if oldHasDefault || newHasDefault {
		if !defaultValuesEqual(colDiff.OldColumn, colDiff.NewColumn) {
			// First drop the old default if it exists
			if oldHasDefault {
				writeAlterColumnDropDefault(&buf, schemaName, tableName, colDiff.OldColumn.Name)
			}

			// Add new default if needed
			if newHasDefault {
				defaultExpr := getDefaultExpression(colDiff.NewColumn)
				writeAlterColumnSetDefault(&buf, schemaName, tableName, colDiff.NewColumn.Name, defaultExpr)
			}
		}
	}

	return buf.String()
}

// hasDefaultValue checks if a column has any default value
func hasDefaultValue(column *storepb.ColumnMetadata) bool {
	if column == nil {
		return false
	}
	return column.GetDefault() != ""
}

// defaultValuesEqual checks if two columns have the same default value
func defaultValuesEqual(col1, col2 *storepb.ColumnMetadata) bool {
	if col1 == nil || col2 == nil {
		return col1 == col2
	}

	// Check default value
	def1 := col1.GetDefault()
	def2 := col2.GetDefault()
	return def1 == def2
}

// getDefaultExpression returns the SQL expression for a column's default value
func getDefaultExpression(column *storepb.ColumnMetadata) string {
	if column == nil {
		return ""
	}

	if column.Default != "" {
		return column.Default
	}

	return ""
}

// Write functions for various DDL statements

func writeDropTrigger(out *strings.Builder, schema, table, trigger string) {
	_, _ = out.WriteString(`DROP TRIGGER IF EXISTS "`)
	_, _ = out.WriteString(trigger)
	_, _ = out.WriteString(`" ON "`)
	_, _ = out.WriteString(schema)
	_, _ = out.WriteString(`"."`)
	_, _ = out.WriteString(table)
	_, _ = out.WriteString(`";`)
	_, _ = out.WriteString("\n")
}

func writeDropForeignKey(out *strings.Builder, schema, table, constraint string) {
	_, _ = out.WriteString(`ALTER TABLE "`)
	_, _ = out.WriteString(schema)
	_, _ = out.WriteString(`"."`)
	_, _ = out.WriteString(table)
	_, _ = out.WriteString(`" DROP CONSTRAINT IF EXISTS "`)
	_, _ = out.WriteString(constraint)
	_, _ = out.WriteString(`";`)
	_, _ = out.WriteString("\n")
}

func writeDropFunction(out *strings.Builder, schema, function string) {
	_, _ = out.WriteString(`DROP FUNCTION IF EXISTS "`)
	_, _ = out.WriteString(schema)
	_, _ = out.WriteString(`"."`)
	_, _ = out.WriteString(function)
	_, _ = out.WriteString(`";`)
	_, _ = out.WriteString("\n")
}

func writeDropView(out *strings.Builder, schema, view string) {
	_, _ = out.WriteString(`DROP VIEW IF EXISTS "`)
	_, _ = out.WriteString(schema)
	_, _ = out.WriteString(`"."`)
	_, _ = out.WriteString(view)
	_, _ = out.WriteString(`";`)
	_, _ = out.WriteString("\n")
}

func writeDropMaterializedView(out *strings.Builder, schema, view string) {
	_, _ = out.WriteString(`DROP MATERIALIZED VIEW IF EXISTS "`)
	_, _ = out.WriteString(schema)
	_, _ = out.WriteString(`"."`)
	_, _ = out.WriteString(view)
	_, _ = out.WriteString(`";`)
	_, _ = out.WriteString("\n")
}

func writeDropCheckConstraint(out *strings.Builder, schema, table, constraint string) {
	_, _ = out.WriteString(`ALTER TABLE "`)
	_, _ = out.WriteString(schema)
	_, _ = out.WriteString(`"."`)
	_, _ = out.WriteString(table)
	_, _ = out.WriteString(`" DROP CONSTRAINT IF EXISTS "`)
	_, _ = out.WriteString(constraint)
	_, _ = out.WriteString(`";`)
	_, _ = out.WriteString("\n")
}

func writeDropConstraint(out *strings.Builder, schema, table, constraint string) {
	_, _ = out.WriteString(`ALTER TABLE "`)
	_, _ = out.WriteString(schema)
	_, _ = out.WriteString(`"."`)
	_, _ = out.WriteString(table)
	_, _ = out.WriteString(`" DROP CONSTRAINT IF EXISTS "`)
	_, _ = out.WriteString(constraint)
	_, _ = out.WriteString(`";`)
	_, _ = out.WriteString("\n")
}

func writeDropIndex(out *strings.Builder, schema, index string) {
	_, _ = out.WriteString(`DROP INDEX IF EXISTS "`)
	_, _ = out.WriteString(schema)
	_, _ = out.WriteString(`"."`)
	_, _ = out.WriteString(index)
	_, _ = out.WriteString(`";`)
	_, _ = out.WriteString("\n")
}

func writeDropColumn(out *strings.Builder, schema, table, column string) {
	_, _ = out.WriteString(`ALTER TABLE "`)
	_, _ = out.WriteString(schema)
	_, _ = out.WriteString(`"."`)
	_, _ = out.WriteString(table)
	_, _ = out.WriteString(`" DROP COLUMN IF EXISTS "`)
	_, _ = out.WriteString(column)
	_, _ = out.WriteString(`";`)
	_, _ = out.WriteString("\n")
}

func writeDropTable(out *strings.Builder, schema, table string) {
	_, _ = out.WriteString(`DROP TABLE IF EXISTS "`)
	_, _ = out.WriteString(schema)
	_, _ = out.WriteString(`"."`)
	_, _ = out.WriteString(table)
	_, _ = out.WriteString(`";`)
	_, _ = out.WriteString("\n")
}

func writeDropSequence(out *strings.Builder, schema, sequence string) {
	_, _ = out.WriteString(`DROP SEQUENCE IF EXISTS "`)
	_, _ = out.WriteString(schema)
	_, _ = out.WriteString(`"."`)
	_, _ = out.WriteString(sequence)
	_, _ = out.WriteString(`";`)
	_, _ = out.WriteString("\n")
}

func writeDropSchema(out *strings.Builder, schema string) {
	_, _ = out.WriteString(`DROP SCHEMA IF EXISTS "`)
	_, _ = out.WriteString(schema)
	_, _ = out.WriteString(`";`)
	_, _ = out.WriteString("\n")
}

func writeDropType(out *strings.Builder, schema, typeName string) {
	_, _ = out.WriteString(`DROP TYPE IF EXISTS "`)
	_, _ = out.WriteString(schema)
	_, _ = out.WriteString(`"."`)
	_, _ = out.WriteString(typeName)
	_, _ = out.WriteString(`";`)
	_, _ = out.WriteString("\n")
}

func writeCreateEnumType(out *strings.Builder, schema string, enum *storepb.EnumTypeMetadata) error {
	_, _ = out.WriteString(`CREATE TYPE "`)
	_, _ = out.WriteString(schema)
	_, _ = out.WriteString(`"."`)
	_, _ = out.WriteString(enum.Name)
	_, _ = out.WriteString(`" AS ENUM (`)

	for i, value := range enum.Values {
		if i > 0 {
			_, _ = out.WriteString(", ")
		}
		_, _ = out.WriteString("'")
		_, _ = out.WriteString(value)
		_, _ = out.WriteString("'")
	}

	_, _ = out.WriteString(");")
	_, _ = out.WriteString("\n")
	return nil
}

func writeCreateSchema(out *strings.Builder, schema string) error {
	_, _ = out.WriteString(`CREATE SCHEMA IF NOT EXISTS "`)
	_, _ = out.WriteString(schema)
	_, _ = out.WriteString(`";`)
	_, _ = out.WriteString("\n")
	return nil
}

func writeAddColumn(out *strings.Builder, schema, table string, column *storepb.ColumnMetadata) {
	_, _ = out.WriteString(`ALTER TABLE "`)
	_, _ = out.WriteString(schema)
	_, _ = out.WriteString(`"."`)
	_, _ = out.WriteString(table)
	_, _ = out.WriteString(`" ADD COLUMN "`)
	_, _ = out.WriteString(column.Name)
	_, _ = out.WriteString(`" `)
	_, _ = out.WriteString(column.Type)

	// Handle default values
	defaultExpr := getDefaultExpression(column)
	if defaultExpr != "" {
		_, _ = out.WriteString(` DEFAULT `)
		_, _ = out.WriteString(defaultExpr)
	}

	if !column.Nullable {
		_, _ = out.WriteString(` NOT NULL`)
	}

	_, _ = out.WriteString(`;`)
	_, _ = out.WriteString("\n")
}

func writeAlterColumnType(out *strings.Builder, schema, table, column, newType string) {
	_, _ = out.WriteString(`ALTER TABLE "`)
	_, _ = out.WriteString(schema)
	_, _ = out.WriteString(`"."`)
	_, _ = out.WriteString(table)
	_, _ = out.WriteString(`" ALTER COLUMN "`)
	_, _ = out.WriteString(column)
	_, _ = out.WriteString(`" TYPE `)
	_, _ = out.WriteString(newType)
	_, _ = out.WriteString(`;`)
	_, _ = out.WriteString("\n")
}

func writeAlterColumnSetNotNull(out *strings.Builder, schema, table, column string) {
	_, _ = out.WriteString(`ALTER TABLE "`)
	_, _ = out.WriteString(schema)
	_, _ = out.WriteString(`"."`)
	_, _ = out.WriteString(table)
	_, _ = out.WriteString(`" ALTER COLUMN "`)
	_, _ = out.WriteString(column)
	_, _ = out.WriteString(`" SET NOT NULL;`)
	_, _ = out.WriteString("\n")
}

func writeAlterColumnDropNotNull(out *strings.Builder, schema, table, column string) {
	_, _ = out.WriteString(`ALTER TABLE "`)
	_, _ = out.WriteString(schema)
	_, _ = out.WriteString(`"."`)
	_, _ = out.WriteString(table)
	_, _ = out.WriteString(`" ALTER COLUMN "`)
	_, _ = out.WriteString(column)
	_, _ = out.WriteString(`" DROP NOT NULL;`)
	_, _ = out.WriteString("\n")
}

func writeAlterColumnSetDefault(out *strings.Builder, schema, table, column, defaultExpr string) {
	_, _ = out.WriteString(`ALTER TABLE "`)
	_, _ = out.WriteString(schema)
	_, _ = out.WriteString(`"."`)
	_, _ = out.WriteString(table)
	_, _ = out.WriteString(`" ALTER COLUMN "`)
	_, _ = out.WriteString(column)
	_, _ = out.WriteString(`" SET DEFAULT `)
	_, _ = out.WriteString(defaultExpr)
	_, _ = out.WriteString(`;`)
	_, _ = out.WriteString("\n")
}

func writeAlterColumnDropDefault(out *strings.Builder, schema, table, column string) {
	_, _ = out.WriteString(`ALTER TABLE "`)
	_, _ = out.WriteString(schema)
	_, _ = out.WriteString(`"."`)
	_, _ = out.WriteString(table)
	_, _ = out.WriteString(`" ALTER COLUMN "`)
	_, _ = out.WriteString(column)
	_, _ = out.WriteString(`" DROP DEFAULT;`)
	_, _ = out.WriteString("\n")
}

func writeAddCheckConstraint(out *strings.Builder, schema, table string, check *storepb.CheckConstraintMetadata) {
	_, _ = out.WriteString(`ALTER TABLE "`)
	_, _ = out.WriteString(schema)
	_, _ = out.WriteString(`"."`)
	_, _ = out.WriteString(table)
	_, _ = out.WriteString(`" ADD CONSTRAINT "`)
	_, _ = out.WriteString(check.Name)
	_, _ = out.WriteString(`" CHECK `)
	_, _ = out.WriteString(check.Expression)
	_, _ = out.WriteString(`;`)
	_, _ = out.WriteString("\n")
}

func writeFunctionDiff(out *strings.Builder, funcDiff *schema.FunctionDiff) error {
	switch funcDiff.Action {
	case schema.MetadataDiffActionCreate:
		// CREATE new function
		_, _ = out.WriteString(funcDiff.NewFunction.Definition)
		if !strings.HasSuffix(strings.TrimSpace(funcDiff.NewFunction.Definition), ";") {
			_, _ = out.WriteString(";")
		}
		_, _ = out.WriteString("\n\n")

	case schema.MetadataDiffActionAlter:
		// ALTER function using CREATE OR REPLACE
		// The decision to use ALTER vs DROP/CREATE was already made in differ.go
		// If we reach here, it means we can safely use CREATE OR REPLACE
		if funcDiff.NewFunction != nil {
			// Use ANTLR parser to safely convert CREATE FUNCTION to CREATE OR REPLACE FUNCTION
			definition := convertToCreateOrReplace(funcDiff.NewFunction.Definition)

			_, _ = out.WriteString(definition)
			if !strings.HasSuffix(strings.TrimSpace(definition), ";") {
				_, _ = out.WriteString(";")
			}
			_, _ = out.WriteString("\n\n")
		}

	default:
		// Ignore other actions like DROP (handled elsewhere)
	}
	return nil
}

// Helper functions to check if diff contains certain types of changes
func hasCreateOrAlterObjects(diff *schema.MetadataDiff) bool {
	return hasCreateOrAlterTables(diff) || hasCreateViewsOrFunctions(diff)
}

func hasCreateOrAlterTables(diff *schema.MetadataDiff) bool {
	for _, tableDiff := range diff.TableChanges {
		if tableDiff.Action == schema.MetadataDiffActionCreate || tableDiff.Action == schema.MetadataDiffActionAlter {
			return true
		}
	}
	for _, seqDiff := range diff.SequenceChanges {
		if seqDiff.Action == schema.MetadataDiffActionCreate {
			return true
		}
	}
	return false
}

func hasCreateViewsOrFunctions(diff *schema.MetadataDiff) bool {
	for _, viewDiff := range diff.ViewChanges {
		if viewDiff.Action == schema.MetadataDiffActionCreate || viewDiff.Action == schema.MetadataDiffActionAlter {
			return true
		}
	}
	for _, mvDiff := range diff.MaterializedViewChanges {
		if mvDiff.Action == schema.MetadataDiffActionCreate || mvDiff.Action == schema.MetadataDiffActionAlter {
			return true
		}
	}
	for _, funcDiff := range diff.FunctionChanges {
		if funcDiff.Action == schema.MetadataDiffActionCreate || funcDiff.Action == schema.MetadataDiffActionAlter {
			return true
		}
	}
	return false
}

// getObjectID generates a unique identifier for database objects
func getMigrationObjectID(schema, name string) string {
	return fmt.Sprintf("%s.%s", schema, name)
}

// writeCreateSequence writes a CREATE SEQUENCE statement
func writeMigrationCreateSequence(out *strings.Builder, schema string, seq *storepb.SequenceMetadata) error {
	_, _ = out.WriteString(`CREATE SEQUENCE "`)
	_, _ = out.WriteString(schema)
	_, _ = out.WriteString(`"."`)
	_, _ = out.WriteString(seq.Name)
	_, _ = out.WriteString(`"`)

	if seq.DataType != "" {
		_, _ = out.WriteString(` AS `)
		_, _ = out.WriteString(seq.DataType)
	}
	if seq.Start != "" {
		_, _ = out.WriteString(` START WITH `)
		_, _ = out.WriteString(seq.Start)
	}
	if seq.Increment != "" {
		_, _ = out.WriteString(` INCREMENT BY `)
		_, _ = out.WriteString(seq.Increment)
	}
	if seq.MinValue != "" {
		_, _ = out.WriteString(` MINVALUE `)
		_, _ = out.WriteString(seq.MinValue)
	}
	if seq.MaxValue != "" {
		_, _ = out.WriteString(` MAXVALUE `)
		_, _ = out.WriteString(seq.MaxValue)
	}
	if seq.CacheSize != "" {
		_, _ = out.WriteString(` CACHE `)
		_, _ = out.WriteString(seq.CacheSize)
	}
	if seq.Cycle {
		_, _ = out.WriteString(` CYCLE`)
	}

	_, _ = out.WriteString(`;`)
	_, _ = out.WriteString("\n")
	return nil
}

// writeMigrationSequenceOwnership writes an ALTER SEQUENCE OWNED BY statement
func writeMigrationSequenceOwnership(out *strings.Builder, schema string, seq *storepb.SequenceMetadata) error {
	_, _ = out.WriteString(`ALTER SEQUENCE "`)
	_, _ = out.WriteString(schema)
	_, _ = out.WriteString(`"."`)
	_, _ = out.WriteString(seq.Name)
	_, _ = out.WriteString(`" OWNED BY "`)
	_, _ = out.WriteString(schema)
	_, _ = out.WriteString(`"."`)
	_, _ = out.WriteString(seq.OwnerTable)
	_, _ = out.WriteString(`"."`)
	_, _ = out.WriteString(seq.OwnerColumn)
	_, _ = out.WriteString(`";`)
	_, _ = out.WriteString("\n")
	return nil
}

// writeView writes a CREATE VIEW statement
func writeMigrationView(out *strings.Builder, schema string, view *storepb.ViewMetadata) error {
	_, _ = out.WriteString(`CREATE VIEW "`)
	_, _ = out.WriteString(schema)
	_, _ = out.WriteString(`"."`)
	_, _ = out.WriteString(view.Name)
	_, _ = out.WriteString(`" AS `)
	_, _ = out.WriteString(view.Definition)
	if !strings.HasSuffix(strings.TrimSpace(view.Definition), ";") {
		_, _ = out.WriteString(`;`)
	}
	_, _ = out.WriteString("\n")
	return nil
}

// writeMaterializedView writes a CREATE MATERIALIZED VIEW statement
func writeMigrationMaterializedView(out *strings.Builder, schema string, view *storepb.MaterializedViewMetadata) error {
	_, _ = out.WriteString(`CREATE MATERIALIZED VIEW "`)
	_, _ = out.WriteString(schema)
	_, _ = out.WriteString(`"."`)
	_, _ = out.WriteString(view.Name)
	_, _ = out.WriteString(`" AS `)
	_, _ = out.WriteString(view.Definition)
	if !strings.HasSuffix(strings.TrimSpace(view.Definition), ";") {
		_, _ = out.WriteString(`;`)
	}
	_, _ = out.WriteString("\n")
	return nil
}

// writeForeignKey writes an ALTER TABLE ADD CONSTRAINT statement for a foreign key
func writeMigrationForeignKey(out *strings.Builder, schema, table string, fk *storepb.ForeignKeyMetadata) error {
	_, _ = out.WriteString(`ALTER TABLE "`)
	_, _ = out.WriteString(schema)
	_, _ = out.WriteString(`"."`)
	_, _ = out.WriteString(table)
	_, _ = out.WriteString(`" ADD CONSTRAINT "`)
	_, _ = out.WriteString(fk.Name)
	_, _ = out.WriteString(`" FOREIGN KEY (`)

	for i, col := range fk.Columns {
		if i > 0 {
			_, _ = out.WriteString(`, `)
		}
		_, _ = out.WriteString(`"`)
		_, _ = out.WriteString(col)
		_, _ = out.WriteString(`"`)
	}

	_, _ = out.WriteString(`) REFERENCES "`)
	_, _ = out.WriteString(fk.ReferencedSchema)
	_, _ = out.WriteString(`"."`)
	_, _ = out.WriteString(fk.ReferencedTable)
	_, _ = out.WriteString(`" (`)

	for i, col := range fk.ReferencedColumns {
		if i > 0 {
			_, _ = out.WriteString(`, `)
		}
		_, _ = out.WriteString(`"`)
		_, _ = out.WriteString(col)
		_, _ = out.WriteString(`"`)
	}

	_, _ = out.WriteString(`)`)

	if fk.OnUpdate != "" {
		_, _ = out.WriteString(` ON UPDATE `)
		_, _ = out.WriteString(fk.OnUpdate)
	}
	if fk.OnDelete != "" {
		_, _ = out.WriteString(` ON DELETE `)
		_, _ = out.WriteString(fk.OnDelete)
	}

	_, _ = out.WriteString(`;`)
	_, _ = out.WriteString("\n")
	return nil
}

// writeMigrationTrigger writes a trigger creation statement
func writeMigrationTrigger(out *strings.Builder, trigger *storepb.TriggerMetadata) {
	if trigger == nil || trigger.Body == "" {
		return
	}
	_, _ = out.WriteString(trigger.Body)
	if !strings.HasSuffix(strings.TrimSpace(trigger.Body), ";") {
		_, _ = out.WriteString(";")
	}
	_, _ = out.WriteString("\n")
}

// writeMigrationMaterializedViewIndex writes an index creation statement for a materialized view
func writeMigrationMaterializedViewIndex(out *strings.Builder, _ string, _ string, index *storepb.IndexMetadata) {
	if index == nil || index.Definition == "" {
		return
	}
	_, _ = out.WriteString(index.Definition)
	if !strings.HasSuffix(strings.TrimSpace(index.Definition), ";") {
		_, _ = out.WriteString(";")
	}
	_, _ = out.WriteString("\n")
}

// writeCreateTable writes a CREATE TABLE statement
func writeMigrationCreateTable(out *strings.Builder, schema, table string, columns []*storepb.ColumnMetadata, checks []*storepb.CheckConstraintMetadata) {
	_, _ = out.WriteString(`CREATE TABLE "`)
	_, _ = out.WriteString(schema)
	_, _ = out.WriteString(`"."`)
	_, _ = out.WriteString(table)
	_, _ = out.WriteString(`" (`)
	_, _ = out.WriteString("\n")

	// Write columns
	for i, col := range columns {
		if i > 0 {
			_, _ = out.WriteString(",\n")
		}
		_, _ = out.WriteString("    \"")
		_, _ = out.WriteString(col.Name)
		_, _ = out.WriteString("\" ")
		_, _ = out.WriteString(col.Type)

		if !col.Nullable {
			_, _ = out.WriteString(" NOT NULL")
		}

		defaultExpr := getDefaultExpression(col)
		if defaultExpr != "" {
			_, _ = out.WriteString(" DEFAULT ")
			_, _ = out.WriteString(defaultExpr)
		}
	}

	// Write check constraints
	for _, check := range checks {
		_, _ = out.WriteString(",\n")
		_, _ = out.WriteString("    CONSTRAINT \"")
		_, _ = out.WriteString(check.Name)
		_, _ = out.WriteString("\" CHECK (")
		_, _ = out.WriteString(check.Expression)
		_, _ = out.WriteString(")")
	}

	_, _ = out.WriteString("\n)")
}

// writePartitionClause writes the partition clause for a table
func writeMigrationPartitionClause(out *strings.Builder, partition *storepb.TablePartitionMetadata) {
	_, _ = out.WriteString(" PARTITION BY ")
	_, _ = out.WriteString(partition.Expression)
}

// writePrimaryKey writes an ALTER TABLE ADD PRIMARY KEY statement
func writeMigrationPrimaryKey(out *strings.Builder, schema, table string, index *storepb.IndexMetadata) {
	_, _ = out.WriteString(`ALTER TABLE "`)
	_, _ = out.WriteString(schema)
	_, _ = out.WriteString(`"."`)
	_, _ = out.WriteString(table)
	_, _ = out.WriteString(`" ADD CONSTRAINT "`)
	_, _ = out.WriteString(index.Name)
	_, _ = out.WriteString(`" PRIMARY KEY (`)

	for i, expr := range index.Expressions {
		if i > 0 {
			_, _ = out.WriteString(`, `)
		}
		_, _ = out.WriteString(expr)
	}

	_, _ = out.WriteString(`);`)
	_, _ = out.WriteString("\n")
}

// writeUniqueKey writes an ALTER TABLE ADD UNIQUE statement
func writeMigrationUniqueKey(out *strings.Builder, schema, table string, index *storepb.IndexMetadata) {
	_, _ = out.WriteString(`ALTER TABLE "`)
	_, _ = out.WriteString(schema)
	_, _ = out.WriteString(`"."`)
	_, _ = out.WriteString(table)
	_, _ = out.WriteString(`" ADD CONSTRAINT "`)
	_, _ = out.WriteString(index.Name)
	_, _ = out.WriteString(`" UNIQUE (`)

	for i, expr := range index.Expressions {
		if i > 0 {
			_, _ = out.WriteString(`, `)
		}
		_, _ = out.WriteString(expr)
	}

	_, _ = out.WriteString(`);`)
	_, _ = out.WriteString("\n")
}

// writeIndex writes a CREATE INDEX statement
func writeMigrationIndex(out *strings.Builder, schema, table string, index *storepb.IndexMetadata) {
	// If index has a full definition, use it directly
	if index.Definition != "" {
		_, _ = out.WriteString(index.Definition)
		if !strings.HasSuffix(strings.TrimSpace(index.Definition), ";") {
			_, _ = out.WriteString(";")
		}
		_, _ = out.WriteString("\n")
		return
	}

	// Otherwise, construct the CREATE INDEX statement
	_, _ = out.WriteString(`CREATE `)
	if index.Unique {
		_, _ = out.WriteString(`UNIQUE `)
	}
	_, _ = out.WriteString(`INDEX "`)
	_, _ = out.WriteString(index.Name)
	_, _ = out.WriteString(`" ON "`)
	_, _ = out.WriteString(schema)
	_, _ = out.WriteString(`"."`)
	_, _ = out.WriteString(table)
	_, _ = out.WriteString(`" `)

	if index.Type != "" && index.Type != "BTREE" {
		_, _ = out.WriteString(`USING `)
		_, _ = out.WriteString(index.Type)
		_, _ = out.WriteString(` `)
	}

	_, _ = out.WriteString(`(`)
	for i, expr := range index.Expressions {
		if i > 0 {
			_, _ = out.WriteString(`, `)
		}
		_, _ = out.WriteString(expr)

		// Handle descending order if specified
		if i < len(index.Descending) && index.Descending[i] {
			_, _ = out.WriteString(` DESC`)
		}
	}
	_, _ = out.WriteString(`)`)

	_, _ = out.WriteString(`;`)
	_, _ = out.WriteString("\n")
}

// generateTableCommentChanges generates COMMENT ON TABLE statements for table comment changes
func generateTableCommentChanges(buf *strings.Builder, tableDiff *schema.TableDiff) error {
	if tableDiff.OldTable == nil || tableDiff.NewTable == nil {
		return nil
	}

	oldComment := tableDiff.OldTable.Comment
	newComment := tableDiff.NewTable.Comment

	// If comments are different, generate COMMENT ON TABLE statement
	if oldComment != newComment {
		writeCommentOnTable(buf, tableDiff.SchemaName, tableDiff.TableName, newComment)
	}

	return nil
}

// generateColumnCommentChanges generates COMMENT ON COLUMN statements for column comment changes
func generateColumnCommentChanges(buf *strings.Builder, tableDiff *schema.TableDiff) error {
	if tableDiff.OldTable == nil || tableDiff.NewTable == nil {
		return nil
	}

	// Build maps for efficient lookup
	oldColumnMap := make(map[string]*storepb.ColumnMetadata)
	for _, col := range tableDiff.OldTable.Columns {
		oldColumnMap[col.Name] = col
	}

	newColumnMap := make(map[string]*storepb.ColumnMetadata)
	for _, col := range tableDiff.NewTable.Columns {
		newColumnMap[col.Name] = col
	}

	// Check for comment changes in existing columns
	for _, newCol := range tableDiff.NewTable.Columns {
		if oldCol, exists := oldColumnMap[newCol.Name]; exists {
			// Column exists in both old and new - check if comment changed
			if oldCol.Comment != newCol.Comment {
				writeCommentOnColumn(buf, tableDiff.SchemaName, tableDiff.TableName, newCol.Name, newCol.Comment)
			}
		} else {
			// New column - if it has a comment, add it
			if newCol.Comment != "" {
				writeCommentOnColumn(buf, tableDiff.SchemaName, tableDiff.TableName, newCol.Name, newCol.Comment)
			}
		}
	}

	// Check for columns that were removed (comments need to be handled by column drop which is handled elsewhere)

	return nil
}

// writeCommentOnTable writes a COMMENT ON TABLE statement
func writeCommentOnTable(out *strings.Builder, schema, table, comment string) {
	_, _ = out.WriteString(`COMMENT ON TABLE "`)
	_, _ = out.WriteString(schema)
	_, _ = out.WriteString(`"."`)
	_, _ = out.WriteString(table)
	_, _ = out.WriteString(`" IS `)
	if comment == "" {
		_, _ = out.WriteString(`NULL`)
	} else {
		_, _ = out.WriteString(`'`)
		// Escape single quotes in the comment
		escapedComment := strings.ReplaceAll(comment, "'", "''")
		_, _ = out.WriteString(escapedComment)
		_, _ = out.WriteString(`'`)
	}
	_, _ = out.WriteString(`;`)
	_, _ = out.WriteString("\n")
}

// writeCommentOnColumn writes a COMMENT ON COLUMN statement
func writeCommentOnColumn(out *strings.Builder, schema, table, column, comment string) {
	_, _ = out.WriteString(`COMMENT ON COLUMN "`)
	_, _ = out.WriteString(schema)
	_, _ = out.WriteString(`"."`)
	_, _ = out.WriteString(table)
	_, _ = out.WriteString(`"."`)
	_, _ = out.WriteString(column)
	_, _ = out.WriteString(`" IS `)
	if comment == "" {
		_, _ = out.WriteString(`NULL`)
	} else {
		_, _ = out.WriteString(`'`)
		// Escape single quotes in the comment
		escapedComment := strings.ReplaceAll(comment, "'", "''")
		_, _ = out.WriteString(escapedComment)
		_, _ = out.WriteString(`'`)
	}
	_, _ = out.WriteString(`;`)
	_, _ = out.WriteString("\n")
}

// generateSchemaCommentChanges generates COMMENT ON SCHEMA statements for schema comment changes
func generateSchemaCommentChanges(buf *strings.Builder, diff *schema.MetadataDiff) error {
	for _, schemaDiff := range diff.SchemaChanges {
		if schemaDiff.Action == schema.MetadataDiffActionAlter {
			if schemaDiff.OldSchema == nil || schemaDiff.NewSchema == nil {
				continue
			}

			oldComment := schemaDiff.OldSchema.Comment
			newComment := schemaDiff.NewSchema.Comment

			// If comments are different, generate COMMENT ON SCHEMA statement
			if oldComment != newComment {
				writeCommentOnSchema(buf, schemaDiff.SchemaName, newComment)
			}
		}
	}
	return nil
}

// generateViewCommentChanges generates COMMENT ON VIEW statements for view comment changes
func generateViewCommentChanges(buf *strings.Builder, diff *schema.MetadataDiff) error {
	for _, viewDiff := range diff.ViewChanges {
		if viewDiff.Action == schema.MetadataDiffActionAlter {
			if viewDiff.OldView == nil || viewDiff.NewView == nil {
				continue
			}

			oldComment := viewDiff.OldView.Comment
			newComment := viewDiff.NewView.Comment

			// If comments are different, generate COMMENT ON VIEW statement
			if oldComment != newComment {
				writeCommentOnView(buf, viewDiff.SchemaName, viewDiff.ViewName, newComment)
			}
		}
	}
	return nil
}

// generateMaterializedViewCommentChanges generates COMMENT ON MATERIALIZED VIEW statements
func generateMaterializedViewCommentChanges(buf *strings.Builder, diff *schema.MetadataDiff) error {
	for _, mvDiff := range diff.MaterializedViewChanges {
		if mvDiff.Action == schema.MetadataDiffActionAlter {
			if mvDiff.OldMaterializedView == nil || mvDiff.NewMaterializedView == nil {
				continue
			}

			oldComment := mvDiff.OldMaterializedView.Comment
			newComment := mvDiff.NewMaterializedView.Comment

			// If comments are different, generate COMMENT ON MATERIALIZED VIEW statement
			if oldComment != newComment {
				writeCommentOnMaterializedView(buf, mvDiff.SchemaName, mvDiff.MaterializedViewName, newComment)
			}
		}
	}
	return nil
}

// generateFunctionCommentChanges generates COMMENT ON FUNCTION statements for function comment changes
func generateFunctionCommentChanges(buf *strings.Builder, diff *schema.MetadataDiff) error {
	for _, funcDiff := range diff.FunctionChanges {
		if funcDiff.Action == schema.MetadataDiffActionAlter {
			if funcDiff.OldFunction == nil || funcDiff.NewFunction == nil {
				continue
			}

			oldComment := funcDiff.OldFunction.Comment
			newComment := funcDiff.NewFunction.Comment

			// If comments are different, generate COMMENT ON FUNCTION statement
			if oldComment != newComment {
				writeCommentOnFunction(buf, funcDiff.SchemaName, funcDiff.NewFunction.Signature, newComment)
			}
		}
	}
	return nil
}

// generateSequenceCommentChanges generates COMMENT ON SEQUENCE statements for sequence comment changes
func generateSequenceCommentChanges(buf *strings.Builder, diff *schema.MetadataDiff) error {
	for _, seqDiff := range diff.SequenceChanges {
		if seqDiff.Action == schema.MetadataDiffActionAlter {
			if seqDiff.OldSequence == nil || seqDiff.NewSequence == nil {
				continue
			}

			oldComment := seqDiff.OldSequence.Comment
			newComment := seqDiff.NewSequence.Comment

			// If comments are different, generate COMMENT ON SEQUENCE statement
			if oldComment != newComment {
				writeCommentOnSequence(buf, seqDiff.SchemaName, seqDiff.SequenceName, newComment)
			}
		}
	}
	return nil
}

// generateIndexCommentChanges generates COMMENT ON INDEX statements for index comment changes within table diffs
func generateIndexCommentChanges(buf *strings.Builder, tableDiff *schema.TableDiff) error {
	for _, indexDiff := range tableDiff.IndexChanges {
		if indexDiff.Action == schema.MetadataDiffActionAlter {
			if indexDiff.OldIndex == nil || indexDiff.NewIndex == nil {
				continue
			}

			oldComment := indexDiff.OldIndex.Comment
			newComment := indexDiff.NewIndex.Comment

			// If comments are different, generate COMMENT ON INDEX statement
			if oldComment != newComment {
				writeCommentOnIndex(buf, tableDiff.SchemaName, indexDiff.NewIndex.Name, newComment)
			}
		}
	}
	return nil
}

// Helper functions to write comment statements for different object types

// writeCommentOnSchema writes a COMMENT ON SCHEMA statement
func writeCommentOnSchema(out *strings.Builder, schema, comment string) {
	_, _ = out.WriteString(`COMMENT ON SCHEMA "`)
	_, _ = out.WriteString(schema)
	_, _ = out.WriteString(`" IS `)
	if comment == "" {
		_, _ = out.WriteString(`NULL`)
	} else {
		_, _ = out.WriteString(`'`)
		// Escape single quotes in the comment
		escapedComment := strings.ReplaceAll(comment, "'", "''")
		_, _ = out.WriteString(escapedComment)
		_, _ = out.WriteString(`'`)
	}
	_, _ = out.WriteString(`;`)
	_, _ = out.WriteString("\n")
}

// writeCommentOnView writes a COMMENT ON VIEW statement
func writeCommentOnView(out *strings.Builder, schema, view, comment string) {
	_, _ = out.WriteString(`COMMENT ON VIEW "`)
	_, _ = out.WriteString(schema)
	_, _ = out.WriteString(`"."`)
	_, _ = out.WriteString(view)
	_, _ = out.WriteString(`" IS `)
	if comment == "" {
		_, _ = out.WriteString(`NULL`)
	} else {
		_, _ = out.WriteString(`'`)
		// Escape single quotes in the comment
		escapedComment := strings.ReplaceAll(comment, "'", "''")
		_, _ = out.WriteString(escapedComment)
		_, _ = out.WriteString(`'`)
	}
	_, _ = out.WriteString(`;`)
	_, _ = out.WriteString("\n")
}

// writeCommentOnMaterializedView writes a COMMENT ON MATERIALIZED VIEW statement
func writeCommentOnMaterializedView(out *strings.Builder, schema, view, comment string) {
	_, _ = out.WriteString(`COMMENT ON MATERIALIZED VIEW "`)
	_, _ = out.WriteString(schema)
	_, _ = out.WriteString(`"."`)
	_, _ = out.WriteString(view)
	_, _ = out.WriteString(`" IS `)
	if comment == "" {
		_, _ = out.WriteString(`NULL`)
	} else {
		_, _ = out.WriteString(`'`)
		// Escape single quotes in the comment
		escapedComment := strings.ReplaceAll(comment, "'", "''")
		_, _ = out.WriteString(escapedComment)
		_, _ = out.WriteString(`'`)
	}
	_, _ = out.WriteString(`;`)
	_, _ = out.WriteString("\n")
}

// writeCommentOnFunction writes a COMMENT ON FUNCTION statement
func writeCommentOnFunction(out *strings.Builder, schema, signature, comment string) {
	_, _ = out.WriteString(`COMMENT ON FUNCTION "`)
	_, _ = out.WriteString(schema)
	_, _ = out.WriteString(`".`)
	_, _ = out.WriteString(signature)
	_, _ = out.WriteString(` IS `)
	if comment == "" {
		_, _ = out.WriteString(`NULL`)
	} else {
		_, _ = out.WriteString(`'`)
		// Escape single quotes in the comment
		escapedComment := strings.ReplaceAll(comment, "'", "''")
		_, _ = out.WriteString(escapedComment)
		_, _ = out.WriteString(`'`)
	}
	_, _ = out.WriteString(`;`)
	_, _ = out.WriteString("\n")
}

// writeCommentOnSequence writes a COMMENT ON SEQUENCE statement
func writeCommentOnSequence(out *strings.Builder, schema, sequence, comment string) {
	_, _ = out.WriteString(`COMMENT ON SEQUENCE "`)
	_, _ = out.WriteString(schema)
	_, _ = out.WriteString(`"."`)
	_, _ = out.WriteString(sequence)
	_, _ = out.WriteString(`" IS `)
	if comment == "" {
		_, _ = out.WriteString(`NULL`)
	} else {
		_, _ = out.WriteString(`'`)
		// Escape single quotes in the comment
		escapedComment := strings.ReplaceAll(comment, "'", "''")
		_, _ = out.WriteString(escapedComment)
		_, _ = out.WriteString(`'`)
	}
	_, _ = out.WriteString(`;`)
	_, _ = out.WriteString("\n")
}

// writeCommentOnIndex writes a COMMENT ON INDEX statement
func writeCommentOnIndex(out *strings.Builder, schema, index, comment string) {
	_, _ = out.WriteString(`COMMENT ON INDEX "`)
	_, _ = out.WriteString(schema)
	_, _ = out.WriteString(`"."`)
	_, _ = out.WriteString(index)
	_, _ = out.WriteString(`" IS `)
	if comment == "" {
		_, _ = out.WriteString(`NULL`)
	} else {
		_, _ = out.WriteString(`'`)
		// Escape single quotes in the comment
		escapedComment := strings.ReplaceAll(comment, "'", "''")
		_, _ = out.WriteString(escapedComment)
		_, _ = out.WriteString(`'`)
	}
	_, _ = out.WriteString(`;`)
	_, _ = out.WriteString("\n")
}

// generateEnumTypeCommentChanges generates COMMENT ON TYPE statements for enum type comment changes
func generateEnumTypeCommentChanges(buf *strings.Builder, diff *schema.MetadataDiff) error {
	for _, enumDiff := range diff.EnumTypeChanges {
		if enumDiff.Action == schema.MetadataDiffActionAlter {
			if enumDiff.OldEnumType == nil || enumDiff.NewEnumType == nil {
				continue
			}

			oldComment := enumDiff.OldEnumType.Comment
			newComment := enumDiff.NewEnumType.Comment

			// If comments are different, generate COMMENT ON TYPE statement
			if oldComment != newComment {
				writeCommentOnType(buf, enumDiff.SchemaName, enumDiff.EnumTypeName, newComment)
			}
		}
	}
	return nil
}

// writeCommentOnType writes a COMMENT ON TYPE statement
func writeCommentOnType(out *strings.Builder, schema, typeName, comment string) {
	_, _ = out.WriteString(`COMMENT ON TYPE "`)
	_, _ = out.WriteString(schema)
	_, _ = out.WriteString(`"."`)
	_, _ = out.WriteString(typeName)
	_, _ = out.WriteString(`" IS `)
	if comment == "" {
		_, _ = out.WriteString(`NULL`)
	} else {
		_, _ = out.WriteString(`'`)
		// Escape single quotes in the comment
		escapedComment := strings.ReplaceAll(comment, "'", "''")
		_, _ = out.WriteString(escapedComment)
		_, _ = out.WriteString(`'`)
	}
	_, _ = out.WriteString(`;`)
	_, _ = out.WriteString("\n")
}

// convertToCreateOrReplace uses ANTLR parser to safely convert CREATE FUNCTION to CREATE OR REPLACE FUNCTION
func convertToCreateOrReplace(definition string) string {
	// Parse the SQL statement using ANTLR
	inputStream := antlr.NewInputStream(definition)
	lexer := pgparser.NewPostgreSQLLexer(inputStream)
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	parser := pgparser.NewPostgreSQLParser(stream)

	// Parse the root
	tree := parser.Root()
	if tree == nil {
		// If parsing fails, return original definition
		return definition
	}

	// Create a visitor to find and modify CREATE FUNCTION statements
	visitor := &CreateOrReplaceVisitor{
		tokens:     stream,
		rewriter:   antlr.NewTokenStreamRewriter(stream),
		definition: definition,
	}

	// Visit the tree
	visitor.Visit(tree)

	// Get the modified text
	interval := antlr.NewInterval(0, len(definition)-1)
	result := visitor.rewriter.GetText("", interval)
	if result == "" {
		// If rewriting fails, return original definition
		return definition
	}

	return result
}

// CreateOrReplaceVisitor visits the parse tree and modifies CREATE FUNCTION to CREATE OR REPLACE FUNCTION
type CreateOrReplaceVisitor struct {
	*pgparser.BasePostgreSQLParserVisitor
	tokens     *antlr.CommonTokenStream
	rewriter   *antlr.TokenStreamRewriter
	definition string
}

// Visit implements the visitor pattern
func (v *CreateOrReplaceVisitor) Visit(tree antlr.ParseTree) any {
	switch t := tree.(type) {
	case *pgparser.CreatefunctionstmtContext:
		return v.visitCreateFunctionStmt(t)
	default:
		// Continue visiting children
		return v.visitChildren(tree)
	}
}

// visitChildren visits all children of a node
func (v *CreateOrReplaceVisitor) visitChildren(node antlr.ParseTree) any {
	for i := 0; i < node.GetChildCount(); i++ {
		child := node.GetChild(i)
		if parseTree, ok := child.(antlr.ParseTree); ok {
			v.Visit(parseTree)
		}
	}
	return nil
}

// visitCreateFunctionStmt handles CREATE FUNCTION statements
func (v *CreateOrReplaceVisitor) visitCreateFunctionStmt(ctx *pgparser.CreatefunctionstmtContext) any {
	if ctx == nil {
		return nil
	}

	// Check if "OR REPLACE" already exists
	if v.hasOrReplace(ctx) {
		// Already has "OR REPLACE", no need to modify
		return nil
	}

	// Find the CREATE token
	createToken := ctx.GetStart()
	if createToken == nil {
		return nil
	}

	// Look for the FUNCTION keyword after CREATE
	functionToken := v.findFunctionToken(ctx)
	if functionToken == nil {
		return nil
	}

	// Insert "OR REPLACE" between CREATE and FUNCTION
	// We insert it right before the FUNCTION token
	v.rewriter.InsertBefore("", functionToken.GetTokenIndex(), "OR REPLACE ")

	return nil
}

// findFunctionToken finds the FUNCTION token in the CREATE FUNCTION statement
func (v *CreateOrReplaceVisitor) findFunctionToken(ctx *pgparser.CreatefunctionstmtContext) antlr.Token {
	// Get all tokens in the context range
	start := ctx.GetStart().GetTokenIndex()
	stop := ctx.GetStop().GetTokenIndex()

	for i := start; i <= stop; i++ {
		token := v.tokens.Get(i)
		if token.GetTokenType() == pgparser.PostgreSQLParserFUNCTION {
			return token
		}
	}

	return nil
}

// hasOrReplace checks if the CREATE FUNCTION statement already contains "OR REPLACE"
func (v *CreateOrReplaceVisitor) hasOrReplace(ctx *pgparser.CreatefunctionstmtContext) bool {
	// Get all tokens in the context range between CREATE and FUNCTION
	start := ctx.GetStart().GetTokenIndex()
	functionToken := v.findFunctionToken(ctx)
	if functionToken == nil {
		return false
	}
	stop := functionToken.GetTokenIndex()

	// Look for "OR" followed by "REPLACE" tokens, skipping whitespace
	for i := start; i < stop; i++ {
		token := v.tokens.Get(i)
		if token.GetTokenType() == pgparser.PostgreSQLParserOR {
			// Found OR, now look for REPLACE after it (skipping whitespace)
			for j := i + 1; j < stop; j++ {
				nextToken := v.tokens.Get(j)
				// Skip whitespace tokens (channel 1 is hidden channel based on debug output)
				if nextToken.GetChannel() == 1 {
					continue
				}
				// Check if the next non-whitespace token is REPLACE
				if nextToken.GetTokenType() == pgparser.PostgreSQLParserREPLACE {
					return true
				}
				// If it's not REPLACE, this OR is not our OR REPLACE pattern
				break
			}
		}
	}

	return false
}
