package oracle

import (
	"fmt"
	"slices"
	"strings"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	parserbase "github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/plugin/schema"
)

func init() {
	schema.RegisterGenerateMigration(storepb.Engine_ORACLE, generateMigration)
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
	createObjectsInOrder(diff, &buf)

	return buf.String(), nil
}

// dropObjectsInOrder drops all objects in reverse topological order (most dependent first)
func dropObjectsInOrder(diff *schema.MetadataDiff, buf *strings.Builder) {
	// Build dependency graph for all objects being dropped or altered
	graph := parserbase.NewGraph()

	// Maps to store different object types
	viewMap := make(map[string]*schema.ViewDiff)
	materializedViewMap := make(map[string]*schema.MaterializedViewDiff)
	tableMap := make(map[string]*schema.TableDiff)
	functionMap := make(map[string]*schema.FunctionDiff)
	procedureMap := make(map[string]*schema.ProcedureDiff)
	sequenceMap := make(map[string]*schema.SequenceDiff)

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

	// Add procedures to graph
	for _, procDiff := range diff.ProcedureChanges {
		if procDiff.Action == schema.MetadataDiffActionDrop {
			procID := getMigrationObjectID(procDiff.SchemaName, procDiff.ProcedureName)
			graph.AddNode(procID)
			procedureMap[procID] = procDiff
			allObjects[procID] = true
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

	// Add sequences to graph
	for _, seqDiff := range diff.SequenceChanges {
		if seqDiff.Action == schema.MetadataDiffActionDrop {
			// Skip system-generated sequences that cannot be manually dropped
			if isSystemGeneratedSequence(seqDiff.SequenceName) {
				continue
			}
			seqID := getMigrationObjectID(seqDiff.SchemaName, seqDiff.SequenceName)
			graph.AddNode(seqID)
			sequenceMap[seqID] = seqDiff
			allObjects[seqID] = true
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

		// Drop procedures
		for _, procDiff := range procedureMap {
			writeDropProcedure(buf, procDiff.SchemaName, procDiff.ProcedureName)
		}

		// Drop tables
		for _, tableDiff := range tableMap {
			writeDropTable(buf, tableDiff.SchemaName, tableDiff.TableName)
		}

		// Drop sequences
		for _, seqDiff := range sequenceMap {
			writeDropSequence(buf, seqDiff.SchemaName, seqDiff.SequenceName)
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

				// Drop triggers
				for _, triggerDiff := range tableDiff.TriggerChanges {
					if triggerDiff.Action == schema.MetadataDiffActionDrop {
						writeDropTrigger(buf, triggerDiff.OldTrigger.Name)
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
			// Drop the object itself
			if viewDiff, ok := viewMap[objID]; ok {
				writeDropView(buf, viewDiff.SchemaName, viewDiff.ViewName)
			} else if mvDiff, ok := materializedViewMap[objID]; ok {
				writeDropMaterializedView(buf, mvDiff.SchemaName, mvDiff.MaterializedViewName)
			} else if funcDiff, ok := functionMap[objID]; ok {
				writeDropFunction(buf, funcDiff.SchemaName, funcDiff.FunctionName)
			} else if procDiff, ok := procedureMap[objID]; ok {
				writeDropProcedure(buf, procDiff.SchemaName, procDiff.ProcedureName)
			} else if tableDiff, ok := tableMap[objID]; ok {
				// Drop foreign keys before table
				if tableDiff.OldTable != nil {
					for _, fk := range tableDiff.OldTable.ForeignKeys {
						writeDropForeignKey(buf, tableDiff.SchemaName, tableDiff.TableName, fk.Name)
					}
				}
				writeDropTable(buf, tableDiff.SchemaName, tableDiff.TableName)
			} else if seqDiff, ok := sequenceMap[objID]; ok {
				writeDropSequence(buf, seqDiff.SchemaName, seqDiff.SequenceName)
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

				// Drop triggers
				for _, triggerDiff := range tableDiff.TriggerChanges {
					if triggerDiff.Action == schema.MetadataDiffActionDrop {
						writeDropTrigger(buf, triggerDiff.OldTrigger.Name)
					}
				}
			}
		}
	}

	// Drop schemas (must be empty)
	for _, schemaDiff := range diff.SchemaChanges {
		if schemaDiff.Action == schema.MetadataDiffActionDrop {
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

				// Drop procedures
				for _, proc := range schemaDiff.OldSchema.Procedures {
					writeDropProcedure(buf, schemaDiff.SchemaName, proc.Name)
				}

				// Drop tables (this will handle foreign keys internally)
				for _, table := range schemaDiff.OldSchema.Tables {
					// Drop the table
					writeDropTable(buf, schemaDiff.SchemaName, table.Name)
				}

				// Drop sequences
				for _, seq := range schemaDiff.OldSchema.Sequences {
					writeDropSequence(buf, schemaDiff.SchemaName, seq.Name)
				}
			}

			writeDropSchema(buf, schemaDiff.SchemaName)
		}
	}
}

// createObjectsInOrder creates all objects in topological order (dependencies first)
func createObjectsInOrder(diff *schema.MetadataDiff, buf *strings.Builder) {
	// First create schemas (they don't have dependencies)
	var schemasToCreate []string
	for _, schemaDiff := range diff.SchemaChanges {
		if schemaDiff.Action == schema.MetadataDiffActionCreate {
			schemasToCreate = append(schemasToCreate, schemaDiff.SchemaName)
		}
	}
	slices.Sort(schemasToCreate)
	for _, schemaName := range schemasToCreate {
		writeCreateSchema(buf, schemaName)
	}

	// Add blank line after schema creation only if we have schemas and more content follows
	if len(schemasToCreate) > 0 && (hasCreateOrAlterTables(diff) || hasCreateViewsOrFunctions(diff)) {
		_, _ = buf.WriteString("\n")
	}

	// Create sequences (before tables as they might be used in column defaults)
	for _, seqDiff := range diff.SequenceChanges {
		if seqDiff.Action == schema.MetadataDiffActionCreate {
			// Skip system-generated sequences that are automatically created by Oracle
			if isSystemGeneratedSequence(seqDiff.SequenceName) {
				continue
			}
			writeMigrationCreateSequence(buf, seqDiff.SchemaName, seqDiff.NewSequence)
		}
	}

	// Build dependency graph for all objects being created or altered
	graph := parserbase.NewGraph()

	// Maps to store different object types
	viewMap := make(map[string]*schema.ViewDiff)
	materializedViewMap := make(map[string]*schema.MaterializedViewDiff)
	tableMap := make(map[string]*schema.TableDiff)
	functionMap := make(map[string]*schema.FunctionDiff)
	procedureMap := make(map[string]*schema.ProcedureDiff)

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

	// Add procedures to graph for creation
	for _, procDiff := range diff.ProcedureChanges {
		if procDiff.Action == schema.MetadataDiffActionCreate || procDiff.Action == schema.MetadataDiffActionAlter {
			procID := getMigrationObjectID(procDiff.SchemaName, procDiff.ProcedureName)
			graph.AddNode(procID)
			procedureMap[procID] = procDiff
			allObjects[procID] = true
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
				// Only add edge if the dependency exists in our objects being created
				if allObjects[depID] {
					// Edge from dependency to dependent (table/view to view)
					graph.AddEdge(depID, viewID)
				} else {
					// Try with empty schema if the dependency schema doesn't match
					depIDNoSchema := getMigrationObjectID("", dep.Table)
					if allObjects[depIDNoSchema] {
						// Edge from dependency to dependent (table/view to view)
						graph.AddEdge(depIDNoSchema, viewID)
					}
				}
			}
		}
	}

	// For materialized views depending on tables/views
	for mvID, mvDiff := range materializedViewMap {
		if mvDiff.NewMaterializedView != nil {
			for _, dep := range mvDiff.NewMaterializedView.DependencyColumns {
				depID := getMigrationObjectID(dep.Schema, dep.Table)
				// Only add edge if the dependency exists in our objects being created
				if allObjects[depID] {
					// Edge from dependency to dependent
					graph.AddEdge(depID, mvID)
				}
			}
		}
	}

	// For functions depending on tables
	for funcID, funcDiff := range functionMap {
		if funcDiff.NewFunction != nil {
			for _, dep := range funcDiff.NewFunction.DependencyTables {
				depID := getMigrationObjectID(dep.Schema, dep.Table)
				// Only add edge if the dependency exists in our objects being created
				if allObjects[depID] {
					// Edge from table to function
					graph.AddEdge(depID, funcID)
				}
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
					return
				}
				_, _ = buf.WriteString(createTableSQL)
				if createTableSQL != "" {
					_, _ = buf.WriteString("\n")
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
				writeMigrationView(buf, viewDiff.SchemaName, viewDiff.NewView)
			default:
				// No action needed
			}
		}

		// Create materialized views
		for _, mvDiff := range materializedViewMap {
			switch mvDiff.Action {
			case schema.MetadataDiffActionCreate, schema.MetadataDiffActionAlter:
				writeMigrationMaterializedView(buf, mvDiff.SchemaName, mvDiff.NewMaterializedView)
			default:
				// No action needed
			}
		}

		// Create functions
		for _, funcDiff := range functionMap {
			writeFunctionDiff(buf, funcDiff)
		}

		// Create procedures
		for _, procDiff := range procedureMap {
			writeProcedureDiff(buf, procDiff)
		}

		// Add foreign keys (only for CREATE operations)
		for _, tableDiff := range tableMap {
			if tableDiff.Action == schema.MetadataDiffActionCreate && tableDiff.NewTable != nil {
				for _, fk := range tableDiff.NewTable.ForeignKeys {
					writeMigrationForeignKey(buf, tableDiff.SchemaName, tableDiff.TableName, fk)
				}
			}
		}
	} else {
		// Create in topological order (dependencies first)
		for _, objID := range orderedList {
			if tableDiff, ok := tableMap[objID]; ok {
				switch tableDiff.Action {
				case schema.MetadataDiffActionCreate:
					// Create table without foreign keys
					createTableSQL, err := generateCreateTable(tableDiff.SchemaName, tableDiff.TableName, tableDiff.NewTable, false)
					if err != nil {
						return
					}
					_, _ = buf.WriteString(createTableSQL)
					if createTableSQL != "" {
						_, _ = buf.WriteString("\n")
					}
				case schema.MetadataDiffActionAlter:
					// Handle column additions for ALTER operations (only columns in topological order)
					for _, colDiff := range tableDiff.ColumnChanges {
						if colDiff.Action == schema.MetadataDiffActionCreate {
							writeAddColumn(buf, tableDiff.SchemaName, tableDiff.TableName, colDiff.NewColumn)
						}
					}
				default:
					// No action needed
				}
			} else if viewDiff, ok := viewMap[objID]; ok {
				switch viewDiff.Action {
				case schema.MetadataDiffActionCreate, schema.MetadataDiffActionAlter:
					writeMigrationView(buf, viewDiff.SchemaName, viewDiff.NewView)
				default:
					// No action needed
				}
			} else if mvDiff, ok := materializedViewMap[objID]; ok {
				switch mvDiff.Action {
				case schema.MetadataDiffActionCreate, schema.MetadataDiffActionAlter:
					writeMigrationMaterializedView(buf, mvDiff.SchemaName, mvDiff.NewMaterializedView)
				default:
					// No action needed
				}
			} else if funcDiff, ok := functionMap[objID]; ok {
				writeFunctionDiff(buf, funcDiff)
			} else if procDiff, ok := procedureMap[objID]; ok {
				writeProcedureDiff(buf, procDiff)
			}
		}

		// Add foreign keys after all tables are created (only for CREATE operations)
		for _, tableDiff := range tableMap {
			if tableDiff.Action == schema.MetadataDiffActionCreate && tableDiff.NewTable != nil {
				for _, fk := range tableDiff.NewTable.ForeignKeys {
					writeMigrationForeignKey(buf, tableDiff.SchemaName, tableDiff.TableName, fk)
				}
			}
		}
	}

	// Handle remaining ALTER table operations (not columns, which are handled in topological order)
	for _, tableDiff := range diff.TableChanges {
		if tableDiff.Action == schema.MetadataDiffActionAlter {
			// Skip column additions (already handled in topological order)
			// Handle other ALTER operations

			// Alter columns (type changes, nullability, etc.)
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
							writeMigrationPrimaryKey(buf, tableDiff.SchemaName, tableDiff.TableName, indexDiff.NewIndex)
						} else if indexDiff.NewIndex.Unique {
							writeMigrationUniqueKey(buf, tableDiff.SchemaName, tableDiff.TableName, indexDiff.NewIndex)
						}
					} else {
						writeMigrationIndex(buf, tableDiff.SchemaName, tableDiff.TableName, indexDiff.NewIndex)
					}
				}
			}

			// Add check constraints
			for _, checkDiff := range tableDiff.CheckConstraintChanges {
				if checkDiff.Action == schema.MetadataDiffActionCreate {
					writeAddCheckConstraint(buf, tableDiff.SchemaName, tableDiff.TableName, checkDiff.NewCheckConstraint)
				}
			}

			// Add foreign keys
			for _, fkDiff := range tableDiff.ForeignKeyChanges {
				if fkDiff.Action == schema.MetadataDiffActionCreate {
					writeMigrationForeignKey(buf, tableDiff.SchemaName, tableDiff.TableName, fkDiff.NewForeignKey)
				}
			}

			// Add triggers
			for _, triggerDiff := range tableDiff.TriggerChanges {
				if triggerDiff.Action == schema.MetadataDiffActionCreate {
					writeMigrationTrigger(buf, triggerDiff.NewTrigger)
				}
			}

			// Handle table comment changes
			generateTableCommentChanges(buf, tableDiff)

			// Handle column comment changes
			generateColumnCommentChanges(buf, tableDiff)

			// Handle index comment changes
			generateIndexCommentChanges(buf, tableDiff)
		}
	}

	// Handle comment changes for other object types
	generateViewCommentChanges(buf, diff)
	generateMaterializedViewCommentChanges(buf, diff)
	generateFunctionCommentChanges(buf, diff)
	generateSequenceCommentChanges(buf, diff)
}

func generateCreateTable(schemaName, tableName string, table *storepb.TableMetadata, includeForeignKeys bool) (string, error) {
	var buf strings.Builder

	writeMigrationCreateTable(&buf, schemaName, tableName, table.Columns, table.CheckConstraints)

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

	// Add table comment if present
	if table.Comment != "" {
		writeTableComment(&buf, schemaName, tableName, table.Comment)
	}

	// Add column comments if present
	for _, col := range table.Columns {
		if col.Comment != "" {
			writeColumnComment(&buf, schemaName, tableName, col.Name, col.Comment)
		}
	}

	// Optionally add foreign keys
	if includeForeignKeys && table != nil {
		for _, fk := range table.ForeignKeys {
			writeMigrationForeignKey(&buf, schemaName, tableName, fk)
		}
	}

	// Add triggers
	for _, trigger := range table.Triggers {
		writeMigrationTrigger(&buf, trigger)
	}

	return buf.String(), nil
}

func generateAlterColumn(schemaName, tableName string, colDiff *schema.ColumnDiff) string {
	var buf strings.Builder

	// In Oracle, we need to handle different aspects of column changes separately

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

func writeDropForeignKey(out *strings.Builder, schema, table, constraint string) {
	_, _ = out.WriteString(`ALTER TABLE `)
	if schema != "" {
		_, _ = out.WriteString(`"`)
		_, _ = out.WriteString(schema)
		_, _ = out.WriteString(`".`)
	}
	_, _ = out.WriteString(`"`)
	_, _ = out.WriteString(table)
	_, _ = out.WriteString(`" DROP CONSTRAINT "`)
	_, _ = out.WriteString(constraint)
	_, _ = out.WriteString(`";`)
	_, _ = out.WriteString("\n")
}

func writeDropFunction(out *strings.Builder, schema, function string) {
	_, _ = out.WriteString(`DROP FUNCTION `)
	if schema != "" {
		_, _ = out.WriteString(`"`)
		_, _ = out.WriteString(schema)
		_, _ = out.WriteString(`".`)
	}
	_, _ = out.WriteString(`"`)
	_, _ = out.WriteString(function)
	_, _ = out.WriteString(`";`)
	_, _ = out.WriteString("\n")
}

func writeDropProcedure(out *strings.Builder, schema, procedure string) {
	_, _ = out.WriteString(`DROP PROCEDURE `)
	if schema != "" {
		_, _ = out.WriteString(`"`)
		_, _ = out.WriteString(schema)
		_, _ = out.WriteString(`".`)
	}
	_, _ = out.WriteString(`"`)
	_, _ = out.WriteString(procedure)
	_, _ = out.WriteString(`";`)
	_, _ = out.WriteString("\n")
}

func writeDropView(out *strings.Builder, schema, view string) {
	_, _ = out.WriteString(`DROP VIEW `)
	if schema != "" {
		_, _ = out.WriteString(`"`)
		_, _ = out.WriteString(schema)
		_, _ = out.WriteString(`".`)
	}
	_, _ = out.WriteString(`"`)
	_, _ = out.WriteString(view)
	_, _ = out.WriteString(`";`)
	_, _ = out.WriteString("\n")
}

func writeDropMaterializedView(out *strings.Builder, schema, view string) {
	_, _ = out.WriteString(`DROP MATERIALIZED VIEW `)
	if schema != "" {
		_, _ = out.WriteString(`"`)
		_, _ = out.WriteString(schema)
		_, _ = out.WriteString(`".`)
	}
	_, _ = out.WriteString(`"`)
	_, _ = out.WriteString(view)
	_, _ = out.WriteString(`";`)
	_, _ = out.WriteString("\n")
}

func writeDropCheckConstraint(out *strings.Builder, schema, table, constraint string) {
	_, _ = out.WriteString(`ALTER TABLE `)
	if schema != "" {
		_, _ = out.WriteString(`"`)
		_, _ = out.WriteString(schema)
		_, _ = out.WriteString(`".`)
	}
	_, _ = out.WriteString(`"`)
	_, _ = out.WriteString(table)
	_, _ = out.WriteString(`" DROP CONSTRAINT "`)
	_, _ = out.WriteString(constraint)
	_, _ = out.WriteString(`";`)
	_, _ = out.WriteString("\n")
}

func writeDropConstraint(out *strings.Builder, schema, table, constraint string) {
	_, _ = out.WriteString(`ALTER TABLE `)
	if schema != "" {
		_, _ = out.WriteString(`"`)
		_, _ = out.WriteString(schema)
		_, _ = out.WriteString(`".`)
	}
	_, _ = out.WriteString(`"`)
	_, _ = out.WriteString(table)
	_, _ = out.WriteString(`" DROP CONSTRAINT "`)
	_, _ = out.WriteString(constraint)
	_, _ = out.WriteString(`";`)
	_, _ = out.WriteString("\n")
}

func writeDropIndex(out *strings.Builder, schema, index string) {
	_, _ = out.WriteString(`DROP INDEX `)
	if schema != "" {
		_, _ = out.WriteString(`"`)
		_, _ = out.WriteString(schema)
		_, _ = out.WriteString(`".`)
	}
	_, _ = out.WriteString(`"`)
	_, _ = out.WriteString(index)
	_, _ = out.WriteString(`";`)
	_, _ = out.WriteString("\n")
}

func writeDropColumn(out *strings.Builder, schema, table, column string) {
	_, _ = out.WriteString(`ALTER TABLE `)
	if schema != "" {
		_, _ = out.WriteString(`"`)
		_, _ = out.WriteString(schema)
		_, _ = out.WriteString(`".`)
	}
	_, _ = out.WriteString(`"`)
	_, _ = out.WriteString(table)
	_, _ = out.WriteString(`" DROP COLUMN "`)
	_, _ = out.WriteString(column)
	_, _ = out.WriteString(`";`)
	_, _ = out.WriteString("\n")
}

func writeDropTable(out *strings.Builder, schema, table string) {
	_, _ = out.WriteString(`DROP TABLE `)
	if schema != "" {
		_, _ = out.WriteString(`"`)
		_, _ = out.WriteString(schema)
		_, _ = out.WriteString(`".`)
	}
	_, _ = out.WriteString(`"`)
	_, _ = out.WriteString(table)
	_, _ = out.WriteString(`";`)
	_, _ = out.WriteString("\n")
}

func writeDropSequence(out *strings.Builder, schema, sequence string) {
	_, _ = out.WriteString(`DROP SEQUENCE `)
	if schema != "" {
		_, _ = out.WriteString(`"`)
		_, _ = out.WriteString(schema)
		_, _ = out.WriteString(`".`)
	}
	_, _ = out.WriteString(`"`)
	_, _ = out.WriteString(sequence)
	_, _ = out.WriteString(`";`)
	_, _ = out.WriteString("\n")
}

func writeDropSchema(out *strings.Builder, schema string) {
	_, _ = out.WriteString(`DROP USER "`)
	_, _ = out.WriteString(schema)
	_, _ = out.WriteString(`" CASCADE;`)
	_, _ = out.WriteString("\n")
}

func writeCreateSchema(out *strings.Builder, schema string) {
	_, _ = out.WriteString(`CREATE USER "`)
	_, _ = out.WriteString(schema)
	_, _ = out.WriteString(`";`)
	_, _ = out.WriteString("\n")
}

func writeAddColumn(out *strings.Builder, schema, table string, column *storepb.ColumnMetadata) {
	_, _ = out.WriteString(`ALTER TABLE `)
	if schema != "" {
		_, _ = out.WriteString(`"`)
		_, _ = out.WriteString(schema)
		_, _ = out.WriteString(`".`)
	}
	_, _ = out.WriteString(`"`)
	_, _ = out.WriteString(table)
	_, _ = out.WriteString(`" ADD "`)
	_, _ = out.WriteString(column.Name)
	_, _ = out.WriteString(`" `)
	_, _ = out.WriteString(column.Type)

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
	_, _ = out.WriteString(`ALTER TABLE `)
	if schema != "" {
		_, _ = out.WriteString(`"`)
		_, _ = out.WriteString(schema)
		_, _ = out.WriteString(`".`)
	}
	_, _ = out.WriteString(`"`)
	_, _ = out.WriteString(table)
	_, _ = out.WriteString(`" MODIFY "`)
	_, _ = out.WriteString(column)
	_, _ = out.WriteString(`" `)
	_, _ = out.WriteString(newType)
	_, _ = out.WriteString(`;`)
	_, _ = out.WriteString("\n")
}

func writeAlterColumnSetNotNull(out *strings.Builder, schema, table, column string) {
	_, _ = out.WriteString(`ALTER TABLE `)
	if schema != "" {
		_, _ = out.WriteString(`"`)
		_, _ = out.WriteString(schema)
		_, _ = out.WriteString(`".`)
	}
	_, _ = out.WriteString(`"`)
	_, _ = out.WriteString(table)
	_, _ = out.WriteString(`" MODIFY "`)
	_, _ = out.WriteString(column)
	_, _ = out.WriteString(`" NOT NULL;`)
	_, _ = out.WriteString("\n")
}

func writeAlterColumnDropNotNull(out *strings.Builder, schema, table, column string) {
	_, _ = out.WriteString(`ALTER TABLE `)
	if schema != "" {
		_, _ = out.WriteString(`"`)
		_, _ = out.WriteString(schema)
		_, _ = out.WriteString(`".`)
	}
	_, _ = out.WriteString(`"`)
	_, _ = out.WriteString(table)
	_, _ = out.WriteString(`" MODIFY "`)
	_, _ = out.WriteString(column)
	_, _ = out.WriteString(`" NULL;`)
	_, _ = out.WriteString("\n")
}

func writeAlterColumnSetDefault(out *strings.Builder, schema, table, column, defaultExpr string) {
	_, _ = out.WriteString(`ALTER TABLE `)
	if schema != "" {
		_, _ = out.WriteString(`"`)
		_, _ = out.WriteString(schema)
		_, _ = out.WriteString(`".`)
	}
	_, _ = out.WriteString(`"`)
	_, _ = out.WriteString(table)
	_, _ = out.WriteString(`" MODIFY "`)
	_, _ = out.WriteString(column)
	_, _ = out.WriteString(`" DEFAULT `)
	_, _ = out.WriteString(defaultExpr)
	_, _ = out.WriteString(`;`)
	_, _ = out.WriteString("\n")
}

func writeAlterColumnDropDefault(out *strings.Builder, schema, table, column string) {
	_, _ = out.WriteString(`ALTER TABLE `)
	if schema != "" {
		_, _ = out.WriteString(`"`)
		_, _ = out.WriteString(schema)
		_, _ = out.WriteString(`".`)
	}
	_, _ = out.WriteString(`"`)
	_, _ = out.WriteString(table)
	_, _ = out.WriteString(`" MODIFY "`)
	_, _ = out.WriteString(column)
	_, _ = out.WriteString(`" DEFAULT NULL;`)
	_, _ = out.WriteString("\n")
}

func writeAddCheckConstraint(out *strings.Builder, schema, table string, check *storepb.CheckConstraintMetadata) {
	_, _ = out.WriteString(`ALTER TABLE `)
	if schema != "" {
		_, _ = out.WriteString(`"`)
		_, _ = out.WriteString(schema)
		_, _ = out.WriteString(`".`)
	}
	_, _ = out.WriteString(`"`)
	_, _ = out.WriteString(table)
	_, _ = out.WriteString(`" ADD CONSTRAINT "`)
	_, _ = out.WriteString(check.Name)
	_, _ = out.WriteString(`" CHECK (`)
	_, _ = out.WriteString(check.Expression)
	_, _ = out.WriteString(`);`)
	_, _ = out.WriteString("\n")
}

func writeFunctionDiff(out *strings.Builder, funcDiff *schema.FunctionDiff) {
	switch funcDiff.Action {
	case schema.MetadataDiffActionCreate:
		definition := funcDiff.NewFunction.Definition
		// If the definition doesn't start with CREATE, add the CREATE OR REPLACE prefix
		if !strings.HasPrefix(strings.ToUpper(strings.TrimSpace(definition)), "CREATE") {
			_, _ = out.WriteString("CREATE OR REPLACE ")
		}
		_, _ = out.WriteString(definition)
		if !strings.HasSuffix(strings.TrimSpace(definition), ";") {
			_, _ = out.WriteString(";")
		}
		_, _ = out.WriteString("\n")

		// Add comment if present
		if funcDiff.NewFunction.Comment != "" {
			writeFunctionComment(out, funcDiff.SchemaName, funcDiff.FunctionName, funcDiff.NewFunction.Comment)
		}
		_, _ = out.WriteString("\n")
	case schema.MetadataDiffActionAlter:
		// Oracle requires CREATE OR REPLACE for functions
		definition := funcDiff.NewFunction.Definition
		// If the definition doesn't start with CREATE, add the CREATE OR REPLACE prefix
		if !strings.HasPrefix(strings.ToUpper(strings.TrimSpace(definition)), "CREATE") {
			_, _ = out.WriteString("CREATE OR REPLACE ")
		}
		_, _ = out.WriteString(definition)
		if !strings.HasSuffix(strings.TrimSpace(definition), ";") {
			_, _ = out.WriteString(";")
		}
		_, _ = out.WriteString("\n")

		// Add comment if present
		if funcDiff.NewFunction.Comment != "" {
			writeFunctionComment(out, funcDiff.SchemaName, funcDiff.FunctionName, funcDiff.NewFunction.Comment)
		}
		_, _ = out.WriteString("\n")
	default:
		// Handle other actions if needed
	}
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
			// Skip system-generated sequences
			if !isSystemGeneratedSequence(seqDiff.SequenceName) {
				return true
			}
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
	for _, procDiff := range diff.ProcedureChanges {
		if procDiff.Action == schema.MetadataDiffActionCreate || procDiff.Action == schema.MetadataDiffActionAlter {
			return true
		}
	}
	return false
}

// getObjectID generates a unique identifier for database objects
func getMigrationObjectID(schema, name string) string {
	return fmt.Sprintf("%s.%s", schema, name)
}

// isSystemGeneratedSequence checks if a sequence is system-generated and should not be included in migration DDL
func isSystemGeneratedSequence(sequenceName string) bool {
	// Oracle automatically creates sequences for IDENTITY columns with names like ISEQ$$_*
	return strings.HasPrefix(sequenceName, "ISEQ$$_")
}

// writeCreateSequence writes a CREATE SEQUENCE statement
func writeMigrationCreateSequence(out *strings.Builder, schema string, seq *storepb.SequenceMetadata) {
	_, _ = out.WriteString(`CREATE SEQUENCE `)
	if schema != "" {
		_, _ = out.WriteString(`"`)
		_, _ = out.WriteString(schema)
		_, _ = out.WriteString(`".`)
	}
	_, _ = out.WriteString(`"`)
	_, _ = out.WriteString(seq.Name)
	_, _ = out.WriteString(`"`)

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

	// Add comment if present
	if seq.Comment != "" {
		writeSequenceComment(out, schema, seq.Name, seq.Comment)
	}
}

func writeProcedureDiff(out *strings.Builder, procDiff *schema.ProcedureDiff) {
	switch procDiff.Action {
	case schema.MetadataDiffActionCreate:
		definition := procDiff.NewProcedure.Definition
		// If the definition doesn't start with CREATE, add the CREATE OR REPLACE prefix
		if !strings.HasPrefix(strings.ToUpper(strings.TrimSpace(definition)), "CREATE") {
			_, _ = out.WriteString("CREATE OR REPLACE ")
		}
		_, _ = out.WriteString(definition)
		if !strings.HasSuffix(strings.TrimSpace(definition), ";") {
			_, _ = out.WriteString(";")
		}
		_, _ = out.WriteString("\n")
		// Note: ProcedureMetadata doesn't have a comment field in the protobuf,
		// so we don't add procedure comments here
		_, _ = out.WriteString("\n")
	case schema.MetadataDiffActionAlter:
		// Oracle requires CREATE OR REPLACE for procedures
		definition := procDiff.NewProcedure.Definition
		// If the definition doesn't start with CREATE, add the CREATE OR REPLACE prefix
		if !strings.HasPrefix(strings.ToUpper(strings.TrimSpace(definition)), "CREATE") {
			_, _ = out.WriteString("CREATE OR REPLACE ")
		}
		_, _ = out.WriteString(definition)
		if !strings.HasSuffix(strings.TrimSpace(definition), ";") {
			_, _ = out.WriteString(";")
		}
		_, _ = out.WriteString("\n")
		// Note: ProcedureMetadata doesn't have a comment field in the protobuf,
		// so we don't add procedure comments here
		_, _ = out.WriteString("\n")
	default:
		// Handle other actions if needed
	}
}

// writeView writes a CREATE VIEW statement
func writeMigrationView(out *strings.Builder, schema string, view *storepb.ViewMetadata) {
	_, _ = out.WriteString(`CREATE VIEW `)
	if schema != "" {
		_, _ = out.WriteString(`"`)
		_, _ = out.WriteString(schema)
		_, _ = out.WriteString(`".`)
	}
	_, _ = out.WriteString(`"`)
	_, _ = out.WriteString(view.Name)
	_, _ = out.WriteString(`" AS `)
	_, _ = out.WriteString(view.Definition)
	if !strings.HasSuffix(strings.TrimSpace(view.Definition), ";") {
		_, _ = out.WriteString(`;`)
	}
	_, _ = out.WriteString("\n")

	// Add comment if present
	if view.Comment != "" {
		writeViewComment(out, schema, view.Name, view.Comment)
	}
}

// writeMaterializedView writes a CREATE MATERIALIZED VIEW statement
func writeMigrationMaterializedView(out *strings.Builder, schema string, view *storepb.MaterializedViewMetadata) {
	_, _ = out.WriteString(`CREATE MATERIALIZED VIEW `)
	if schema != "" {
		_, _ = out.WriteString(`"`)
		_, _ = out.WriteString(schema)
		_, _ = out.WriteString(`".`)
	}
	_, _ = out.WriteString(`"`)
	_, _ = out.WriteString(view.Name)
	_, _ = out.WriteString(`" AS `)
	_, _ = out.WriteString(view.Definition)
	if !strings.HasSuffix(strings.TrimSpace(view.Definition), ";") {
		_, _ = out.WriteString(`;`)
	}
	_, _ = out.WriteString("\n")

	// Add comment if present
	if view.Comment != "" {
		writeMaterializedViewComment(out, schema, view.Name, view.Comment)
	}
}

// writeForeignKey writes an ALTER TABLE ADD CONSTRAINT statement for a foreign key
func writeMigrationForeignKey(out *strings.Builder, schema, table string, fk *storepb.ForeignKeyMetadata) {
	_, _ = out.WriteString(`ALTER TABLE `)
	if schema != "" {
		_, _ = out.WriteString(`"`)
		_, _ = out.WriteString(schema)
		_, _ = out.WriteString(`".`)
	}
	_, _ = out.WriteString(`"`)
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

	_, _ = out.WriteString(`) REFERENCES `)
	if fk.ReferencedSchema != "" {
		_, _ = out.WriteString(`"`)
		_, _ = out.WriteString(fk.ReferencedSchema)
		_, _ = out.WriteString(`".`)
	}
	_, _ = out.WriteString(`"`)
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
}

// writeCreateTable writes a CREATE TABLE statement
func writeMigrationCreateTable(out *strings.Builder, schema, table string, columns []*storepb.ColumnMetadata, checks []*storepb.CheckConstraintMetadata) {
	_, _ = out.WriteString(`CREATE TABLE `)
	if schema != "" {
		_, _ = out.WriteString(`"`)
		_, _ = out.WriteString(schema)
		_, _ = out.WriteString(`".`)
	}
	_, _ = out.WriteString(`"`)
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

		defaultExpr := getDefaultExpression(col)
		if defaultExpr != "" {
			_, _ = out.WriteString(" DEFAULT ")
			_, _ = out.WriteString(defaultExpr)
		}

		if !col.Nullable {
			_, _ = out.WriteString(" NOT NULL")
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

// writePrimaryKey writes an ALTER TABLE ADD PRIMARY KEY statement
func writeMigrationPrimaryKey(out *strings.Builder, schema, table string, index *storepb.IndexMetadata) {
	_, _ = out.WriteString(`ALTER TABLE `)
	if schema != "" {
		_, _ = out.WriteString(`"`)
		_, _ = out.WriteString(schema)
		_, _ = out.WriteString(`".`)
	}
	_, _ = out.WriteString(`"`)
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
	_, _ = out.WriteString(`ALTER TABLE `)
	if schema != "" {
		_, _ = out.WriteString(`"`)
		_, _ = out.WriteString(schema)
		_, _ = out.WriteString(`".`)
	}
	_, _ = out.WriteString(`"`)
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
	_, _ = out.WriteString(`" ON `)
	if schema != "" {
		_, _ = out.WriteString(`"`)
		_, _ = out.WriteString(schema)
		_, _ = out.WriteString(`".`)
	}
	_, _ = out.WriteString(`"`)
	_, _ = out.WriteString(table)
	_, _ = out.WriteString(`" (`)

	// Handle FUNCTION-BASED indexes differently from normal indexes
	if strings.Contains(index.Type, "FUNCTION-BASED") {
		for i, expression := range index.Expressions {
			if i > 0 {
				_, _ = out.WriteString(`, `)
			}
			_, _ = out.WriteString(expression)

			// For function-based indexes, always add explicit ASC/DESC
			if i < len(index.Descending) && index.Descending[i] {
				_, _ = out.WriteString(` DESC`)
			} else {
				_, _ = out.WriteString(` ASC`)
			}
		}
	} else {
		for i, column := range index.Expressions {
			if i > 0 {
				_, _ = out.WriteString(`, `)
			}
			// Remove quotes if they already exist to avoid double quoting
			cleanColumn := column
			if strings.HasPrefix(column, `"`) && strings.HasSuffix(column, `"`) {
				cleanColumn = column[1 : len(column)-1]
			}
			_, _ = out.WriteString(`"`)
			_, _ = out.WriteString(cleanColumn)
			_, _ = out.WriteString(`"`)

			// For normal indexes, add explicit ASC/DESC
			if i < len(index.Descending) && index.Descending[i] {
				_, _ = out.WriteString(` DESC`)
			} else {
				_, _ = out.WriteString(` ASC`)
			}
		}
	}
	_, _ = out.WriteString(`)`)

	_, _ = out.WriteString(`;`)
	_, _ = out.WriteString("\n")

	// Add comment if present
	if index.Comment != "" {
		writeIndexComment(out, schema, index.Name, index.Comment)
	}
}

// Comment writing helper functions

// writeViewComment writes a COMMENT ON VIEW statement
func writeViewComment(out *strings.Builder, schema, viewName, comment string) {
	_, _ = out.WriteString(`COMMENT ON VIEW `)
	if schema != "" {
		_, _ = out.WriteString(`"`)
		_, _ = out.WriteString(schema)
		_, _ = out.WriteString(`".`)
	}
	_, _ = out.WriteString(`"`)
	_, _ = out.WriteString(viewName)
	_, _ = out.WriteString(`" IS `)
	if comment == "" {
		_, _ = out.WriteString(`''`)
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

// writeMaterializedViewComment writes a COMMENT ON MATERIALIZED VIEW statement
func writeMaterializedViewComment(out *strings.Builder, schema, viewName, comment string) {
	_, _ = out.WriteString(`COMMENT ON MATERIALIZED VIEW `)
	if schema != "" {
		_, _ = out.WriteString(`"`)
		_, _ = out.WriteString(schema)
		_, _ = out.WriteString(`".`)
	}
	_, _ = out.WriteString(`"`)
	_, _ = out.WriteString(viewName)
	_, _ = out.WriteString(`" IS `)
	if comment == "" {
		_, _ = out.WriteString(`''`)
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

// writeFunctionComment writes a COMMENT ON FUNCTION statement
func writeFunctionComment(out *strings.Builder, schema, functionName, comment string) {
	_, _ = out.WriteString(`COMMENT ON FUNCTION `)
	if schema != "" {
		_, _ = out.WriteString(`"`)
		_, _ = out.WriteString(schema)
		_, _ = out.WriteString(`".`)
	}
	_, _ = out.WriteString(`"`)
	_, _ = out.WriteString(functionName)
	_, _ = out.WriteString(`" IS `)
	if comment == "" {
		_, _ = out.WriteString(`''`)
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

// writeSequenceComment writes a COMMENT ON SEQUENCE statement
func writeSequenceComment(out *strings.Builder, schema, sequenceName, comment string) {
	_, _ = out.WriteString(`COMMENT ON SEQUENCE `)
	if schema != "" {
		_, _ = out.WriteString(`"`)
		_, _ = out.WriteString(schema)
		_, _ = out.WriteString(`".`)
	}
	_, _ = out.WriteString(`"`)
	_, _ = out.WriteString(sequenceName)
	_, _ = out.WriteString(`" IS `)
	if comment == "" {
		_, _ = out.WriteString(`''`)
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

// writeIndexComment writes a COMMENT ON INDEX statement
func writeIndexComment(out *strings.Builder, schema, indexName, comment string) {
	_, _ = out.WriteString(`COMMENT ON INDEX `)
	if schema != "" {
		_, _ = out.WriteString(`"`)
		_, _ = out.WriteString(schema)
		_, _ = out.WriteString(`".`)
	}
	_, _ = out.WriteString(`"`)
	_, _ = out.WriteString(indexName)
	_, _ = out.WriteString(`" IS `)
	if comment == "" {
		_, _ = out.WriteString(`''`)
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

// Comment generation functions for Oracle migration DDL

// generateTableCommentChanges generates COMMENT ON TABLE statements for table comment changes
func generateTableCommentChanges(buf *strings.Builder, tableDiff *schema.TableDiff) {
	if tableDiff.OldTable == nil || tableDiff.NewTable == nil {
		return
	}

	oldComment := tableDiff.OldTable.Comment
	newComment := tableDiff.NewTable.Comment

	// If comments are different, generate COMMENT ON TABLE statement
	if oldComment != newComment {
		writeTableComment(buf, tableDiff.SchemaName, tableDiff.TableName, newComment)
	}
}

// generateColumnCommentChanges generates COMMENT ON COLUMN statements for column comment changes
func generateColumnCommentChanges(buf *strings.Builder, tableDiff *schema.TableDiff) {
	if tableDiff.OldTable == nil || tableDiff.NewTable == nil {
		return
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

	// Check for columns that exist in both old and new (column modifications/comment changes)
	for _, newCol := range tableDiff.NewTable.Columns {
		if oldCol, exists := oldColumnMap[newCol.Name]; exists {
			// Column exists in both, check if comment changed
			if oldCol.Comment != newCol.Comment {
				writeColumnComment(buf, tableDiff.SchemaName, tableDiff.TableName, newCol.Name, newCol.Comment)
			}
		} else {
			// New column, add comment if it has one
			if newCol.Comment != "" {
				writeColumnComment(buf, tableDiff.SchemaName, tableDiff.TableName, newCol.Name, newCol.Comment)
			}
		}
	}

	// Handle explicit column changes in the diff
	for _, colDiff := range tableDiff.ColumnChanges {
		if colDiff.Action == schema.MetadataDiffActionCreate && colDiff.NewColumn != nil && colDiff.NewColumn.Comment != "" {
			writeColumnComment(buf, tableDiff.SchemaName, tableDiff.TableName, colDiff.NewColumn.Name, colDiff.NewColumn.Comment)
		} else if colDiff.Action == schema.MetadataDiffActionAlter && colDiff.OldColumn != nil && colDiff.NewColumn != nil {
			if colDiff.OldColumn.Comment != colDiff.NewColumn.Comment {
				writeColumnComment(buf, tableDiff.SchemaName, tableDiff.TableName, colDiff.NewColumn.Name, colDiff.NewColumn.Comment)
			}
		}
	}
}

// generateIndexCommentChanges generates COMMENT ON INDEX statements for index comment changes within table diffs
func generateIndexCommentChanges(buf *strings.Builder, tableDiff *schema.TableDiff) {
	for _, indexDiff := range tableDiff.IndexChanges {
		if indexDiff.Action == schema.MetadataDiffActionAlter {
			if indexDiff.OldIndex == nil || indexDiff.NewIndex == nil {
				continue
			}

			oldComment := indexDiff.OldIndex.Comment
			newComment := indexDiff.NewIndex.Comment

			// If comments are different, generate COMMENT ON INDEX statement
			if oldComment != newComment {
				writeIndexComment(buf, tableDiff.SchemaName, indexDiff.NewIndex.Name, newComment)
			}
		} else if indexDiff.Action == schema.MetadataDiffActionCreate && indexDiff.NewIndex != nil && indexDiff.NewIndex.Comment != "" {
			// New index with comment
			writeIndexComment(buf, tableDiff.SchemaName, indexDiff.NewIndex.Name, indexDiff.NewIndex.Comment)
		}
	}
}

// generateViewCommentChanges generates COMMENT ON VIEW statements for view comment changes
func generateViewCommentChanges(buf *strings.Builder, diff *schema.MetadataDiff) {
	for _, viewDiff := range diff.ViewChanges {
		if viewDiff.Action == schema.MetadataDiffActionAlter {
			if viewDiff.OldView == nil || viewDiff.NewView == nil {
				continue
			}

			oldComment := viewDiff.OldView.Comment
			newComment := viewDiff.NewView.Comment

			// If comments are different, generate COMMENT ON VIEW statement
			if oldComment != newComment {
				writeViewComment(buf, viewDiff.SchemaName, viewDiff.ViewName, newComment)
			}
		} else if viewDiff.Action == schema.MetadataDiffActionCreate && viewDiff.NewView != nil && viewDiff.NewView.Comment != "" {
			// New view with comment
			writeViewComment(buf, viewDiff.SchemaName, viewDiff.ViewName, viewDiff.NewView.Comment)
		}
	}
}

// generateMaterializedViewCommentChanges generates COMMENT ON MATERIALIZED VIEW statements
func generateMaterializedViewCommentChanges(buf *strings.Builder, diff *schema.MetadataDiff) {
	for _, mvDiff := range diff.MaterializedViewChanges {
		if mvDiff.Action == schema.MetadataDiffActionAlter {
			if mvDiff.OldMaterializedView == nil || mvDiff.NewMaterializedView == nil {
				continue
			}

			oldComment := mvDiff.OldMaterializedView.Comment
			newComment := mvDiff.NewMaterializedView.Comment

			// If comments are different, generate COMMENT ON MATERIALIZED VIEW statement
			if oldComment != newComment {
				writeMaterializedViewComment(buf, mvDiff.SchemaName, mvDiff.MaterializedViewName, newComment)
			}
		} else if mvDiff.Action == schema.MetadataDiffActionCreate && mvDiff.NewMaterializedView != nil && mvDiff.NewMaterializedView.Comment != "" {
			// New materialized view with comment
			writeMaterializedViewComment(buf, mvDiff.SchemaName, mvDiff.MaterializedViewName, mvDiff.NewMaterializedView.Comment)
		}
	}
}

// generateFunctionCommentChanges generates COMMENT ON FUNCTION statements for function comment changes
func generateFunctionCommentChanges(buf *strings.Builder, diff *schema.MetadataDiff) {
	for _, funcDiff := range diff.FunctionChanges {
		if funcDiff.Action == schema.MetadataDiffActionAlter {
			if funcDiff.OldFunction == nil || funcDiff.NewFunction == nil {
				continue
			}

			oldComment := funcDiff.OldFunction.Comment
			newComment := funcDiff.NewFunction.Comment

			// If comments are different, generate COMMENT ON FUNCTION statement
			if oldComment != newComment {
				writeFunctionComment(buf, funcDiff.SchemaName, funcDiff.FunctionName, newComment)
			}
		} else if funcDiff.Action == schema.MetadataDiffActionCreate && funcDiff.NewFunction != nil && funcDiff.NewFunction.Comment != "" {
			// New function with comment
			writeFunctionComment(buf, funcDiff.SchemaName, funcDiff.FunctionName, funcDiff.NewFunction.Comment)
		}
	}
}

// generateSequenceCommentChanges generates COMMENT ON SEQUENCE statements for sequence comment changes
func generateSequenceCommentChanges(buf *strings.Builder, diff *schema.MetadataDiff) {
	for _, seqDiff := range diff.SequenceChanges {
		if seqDiff.Action == schema.MetadataDiffActionAlter {
			if seqDiff.OldSequence == nil || seqDiff.NewSequence == nil {
				continue
			}

			oldComment := seqDiff.OldSequence.Comment
			newComment := seqDiff.NewSequence.Comment

			// If comments are different, generate COMMENT ON SEQUENCE statement
			if oldComment != newComment {
				writeSequenceComment(buf, seqDiff.SchemaName, seqDiff.SequenceName, newComment)
			}
		} else if seqDiff.Action == schema.MetadataDiffActionCreate && seqDiff.NewSequence != nil && seqDiff.NewSequence.Comment != "" {
			// New sequence with comment
			writeSequenceComment(buf, seqDiff.SchemaName, seqDiff.SequenceName, seqDiff.NewSequence.Comment)
		}
	}
}

// Helper functions to write comment statements for different object types

// writeTableComment writes a COMMENT ON TABLE statement
func writeTableComment(out *strings.Builder, schema, table, comment string) {
	_, _ = out.WriteString(`COMMENT ON TABLE `)
	if schema != "" {
		_, _ = out.WriteString(`"`)
		_, _ = out.WriteString(schema)
		_, _ = out.WriteString(`".`)
	}
	_, _ = out.WriteString(`"`)
	_, _ = out.WriteString(table)
	_, _ = out.WriteString(`" IS `)
	if comment == "" {
		_, _ = out.WriteString(`''`)
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

// writeColumnComment writes a COMMENT ON COLUMN statement
func writeColumnComment(out *strings.Builder, schema, table, column, comment string) {
	_, _ = out.WriteString(`COMMENT ON COLUMN `)
	if schema != "" {
		_, _ = out.WriteString(`"`)
		_, _ = out.WriteString(schema)
		_, _ = out.WriteString(`".`)
	}
	_, _ = out.WriteString(`"`)
	_, _ = out.WriteString(table)
	_, _ = out.WriteString(`"."`)
	_, _ = out.WriteString(column)
	_, _ = out.WriteString(`" IS `)
	if comment == "" {
		_, _ = out.WriteString(`''`)
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

// writeMigrationTrigger writes a CREATE TRIGGER statement for migration
func writeMigrationTrigger(out *strings.Builder, trigger *storepb.TriggerMetadata) {
	if trigger == nil {
		return
	}
	_, _ = out.WriteString(trigger.Body)
	if !strings.HasSuffix(strings.TrimSpace(trigger.Body), ";") {
		_, _ = out.WriteString(";")
	}
	_, _ = out.WriteString("\n")
}

// writeDropTrigger writes a DROP TRIGGER statement
func writeDropTrigger(out *strings.Builder, triggerName string) {
	_, _ = out.WriteString(`DROP TRIGGER "`)
	_, _ = out.WriteString(triggerName)
	_, _ = out.WriteString(`";`)
	_, _ = out.WriteString("\n")
}
