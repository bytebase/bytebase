package pg

import (
	"fmt"
	"slices"
	"strings"

	parserbase "github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
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
	for _, tableDiff := range diff.TableChanges {
		if tableDiff.Action != schema.MetadataDiffActionDrop && tableDiff.OldTable != nil {
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

	// Add tables to graph
	for _, tableDiff := range diff.TableChanges {
		if tableDiff.Action == schema.MetadataDiffActionCreate {
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
	// For tables with foreign keys depending on other tables
	for tableID, tableDiff := range tableMap {
		if tableDiff.NewTable != nil {
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
			createTableSQL, err := generateCreateTable(tableDiff.SchemaName, tableDiff.TableName, tableDiff.NewTable, false)
			if err != nil {
				return err
			}
			_, _ = buf.WriteString(createTableSQL)
			if createTableSQL != "" {
				_, _ = buf.WriteString("\n")
			}
		}

		// Create views
		for _, viewDiff := range viewMap {
			if err := writeMigrationView(buf, viewDiff.SchemaName, viewDiff.NewView); err != nil {
				return err
			}
		}

		// Create materialized views
		for _, mvDiff := range materializedViewMap {
			if err := writeMigrationMaterializedView(buf, mvDiff.SchemaName, mvDiff.NewMaterializedView); err != nil {
				return err
			}
		}

		// Create functions
		for _, funcDiff := range functionMap {
			if err := writeFunctionDiff(buf, funcDiff); err != nil {
				return err
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

		// Add foreign keys
		for _, tableDiff := range tableMap {
			if tableDiff.NewTable != nil {
				for _, fk := range tableDiff.NewTable.ForeignKeys {
					if err := writeMigrationForeignKey(buf, tableDiff.SchemaName, tableDiff.TableName, fk); err != nil {
						return err
					}
				}
			}
		}
	} else {
		// Create in topological order (dependencies first)
		for _, objID := range orderedList {
			if tableDiff, ok := tableMap[objID]; ok {
				// Create table without foreign keys
				createTableSQL, err := generateCreateTable(tableDiff.SchemaName, tableDiff.TableName, tableDiff.NewTable, false)
				if err != nil {
					return err
				}
				_, _ = buf.WriteString(createTableSQL)
				if createTableSQL != "" {
					_, _ = buf.WriteString("\n")
				}
			} else if viewDiff, ok := viewMap[objID]; ok {
				if err := writeMigrationView(buf, viewDiff.SchemaName, viewDiff.NewView); err != nil {
					return err
				}
			} else if mvDiff, ok := materializedViewMap[objID]; ok {
				if err := writeMigrationMaterializedView(buf, mvDiff.SchemaName, mvDiff.NewMaterializedView); err != nil {
					return err
				}
			} else if funcDiff, ok := functionMap[objID]; ok {
				if err := writeFunctionDiff(buf, funcDiff); err != nil {
					return err
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

		// Add foreign keys after all tables are created
		for _, tableDiff := range tableMap {
			if tableDiff.NewTable != nil {
				for _, fk := range tableDiff.NewTable.ForeignKeys {
					if err := writeMigrationForeignKey(buf, tableDiff.SchemaName, tableDiff.TableName, fk); err != nil {
						return err
					}
				}
			}
		}
	}

	// Handle ALTER table operations
	for _, tableDiff := range diff.TableChanges {
		if tableDiff.Action == schema.MetadataDiffActionAlter {
			alterTableSQL, err := generateAlterTable(tableDiff)
			if err != nil {
				return err
			}
			_, _ = buf.WriteString(alterTableSQL)
			if alterTableSQL != "" {
				_, _ = buf.WriteString("\n")
			}
		}
	}

	return nil
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

func generateAlterTable(tableDiff *schema.TableDiff) (string, error) {
	var buf strings.Builder

	// Add columns first (other operations might depend on them)
	for _, colDiff := range tableDiff.ColumnChanges {
		if colDiff.Action == schema.MetadataDiffActionCreate {
			writeAddColumn(&buf, tableDiff.SchemaName, tableDiff.TableName, colDiff.NewColumn)
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
	return column.GetDefaultExpression() != "" ||
		(column.GetDefault() != nil && column.GetDefault().Value != "") ||
		column.GetDefaultNull()
}

// defaultValuesEqual checks if two columns have the same default value
func defaultValuesEqual(col1, col2 *storepb.ColumnMetadata) bool {
	if col1 == nil || col2 == nil {
		return col1 == col2
	}

	// Check default expression
	if col1.GetDefaultExpression() != col2.GetDefaultExpression() {
		return false
	}

	// Check default value
	def1 := col1.GetDefault()
	def2 := col2.GetDefault()
	if (def1 == nil) != (def2 == nil) {
		return false
	}
	if def1 != nil && def1.Value != def2.Value {
		return false
	}

	// Check default null
	if col1.GetDefaultNull() != col2.GetDefaultNull() {
		return false
	}

	return true
}

// getDefaultExpression returns the SQL expression for a column's default value
func getDefaultExpression(column *storepb.ColumnMetadata) string {
	if column == nil {
		return ""
	}

	if expr := column.GetDefaultExpression(); expr != "" {
		return expr
	}

	if def := column.GetDefault(); def != nil && def.Value != "" {
		// Quote string literals
		return fmt.Sprintf("'%s'", def.Value)
	}

	if column.GetDefaultNull() {
		return "NULL"
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

	if column.DefaultValue != nil {
		if defaultValue, ok := column.DefaultValue.(*storepb.ColumnMetadata_DefaultExpression); ok {
			_, _ = out.WriteString(` DEFAULT `)
			_, _ = out.WriteString(defaultValue.DefaultExpression)
		}
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
		_, _ = out.WriteString(funcDiff.NewFunction.Definition)
		if !strings.HasSuffix(strings.TrimSpace(funcDiff.NewFunction.Definition), ";") {
			_, _ = out.WriteString(";")
		}
		_, _ = out.WriteString("\n\n")
	case schema.MetadataDiffActionAlter:
		// PostgreSQL requires CREATE OR REPLACE for functions
		_, _ = out.WriteString(funcDiff.NewFunction.Definition)
		if !strings.HasSuffix(strings.TrimSpace(funcDiff.NewFunction.Definition), ";") {
			_, _ = out.WriteString(";")
		}
		_, _ = out.WriteString("\n\n")
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

		if col.DefaultValue != nil {
			_, _ = out.WriteString(" DEFAULT ")
			_, _ = out.WriteString(getDefaultExpression(col))
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

	// Note: IndexMetadata doesn't have a Where field in the current protobuf definition
	// This would need to be added if partial indexes need to be supported

	_, _ = out.WriteString(`;`)
	_, _ = out.WriteString("\n")
}
