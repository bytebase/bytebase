package pg

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	pgparser "github.com/bytebase/parser/postgresql"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	pgpluginparser "github.com/bytebase/bytebase/backend/plugin/parser/pg"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/plugin/schema/pg/ast"
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
	// Drop event triggers first (before everything else)
	// Event triggers are database-level objects that depend on functions
	for _, etDiff := range diff.EventTriggerChanges {
		if etDiff.Action == schema.MetadataDiffActionDrop {
			writeDropEventTrigger(buf, etDiff.EventTriggerName)
		}
	}

	// Next, drop all triggers that might depend on functions we're about to drop
	// This is necessary because PostgreSQL doesn't allow dropping functions that are used by triggers
	functionsBeingDropped := make(map[string]bool)
	for _, funcDiff := range diff.FunctionChanges {
		if funcDiff.Action == schema.MetadataDiffActionDrop {
			funcName := strings.ToLower(funcDiff.FunctionName)
			functionsBeingDropped[funcName] = true
		}
	}

	// Drop all triggers first (before dropping functions or tables they depend on)
	// This avoids dependency errors when dropping functions or tables
	for _, tableDiff := range diff.TableChanges {
		// Use TriggerChanges for AST-only mode, OldTable.Triggers for metadata mode
		if len(tableDiff.TriggerChanges) > 0 {
			// AST-only mode: use TriggerChanges
			for _, triggerDiff := range tableDiff.TriggerChanges {
				if triggerDiff.Action == schema.MetadataDiffActionDrop {
					writeDropTrigger(buf, tableDiff.SchemaName, tableDiff.TableName, triggerDiff.TriggerName)
				}
			}
		} else if tableDiff.OldTable != nil && tableDiff.Action == schema.MetadataDiffActionDrop {
			// Metadata mode: use OldTable.Triggers
			// Only drop triggers for tables being dropped
			for _, trigger := range tableDiff.OldTable.Triggers {
				writeDropTrigger(buf, tableDiff.SchemaName, tableDiff.TableName, trigger.Name)
			}
		}
	}

	// Also drop triggers for ALTER table operations
	for _, tableDiff := range diff.TableChanges {
		if tableDiff.Action == schema.MetadataDiffActionAlter {
			for _, triggerDiff := range tableDiff.TriggerChanges {
				if triggerDiff.Action == schema.MetadataDiffActionDrop {
					writeDropTrigger(buf, tableDiff.SchemaName, tableDiff.TableName, triggerDiff.TriggerName)
				}
			}
		}
	}

	// Build dependency graph for all objects being dropped or altered
	graph := base.NewGraph()

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

	// Build temporary metadata for AST-only mode dependency extraction
	// This metadata only contains tables and views that are being dropped,
	// allowing GetQuerySpan to find them when extracting view dependencies
	tempMetadata := buildTempMetadataForDrop(diff)

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
		var dependencies []*storepb.DependencyColumn

		if viewDiff.OldView != nil {
			// Use metadata if available
			dependencies = viewDiff.OldView.DependencyColumns
		} else if viewDiff.OldASTNode != nil {
			// Extract dependencies from AST node for AST-only mode
			// Use the temporary metadata containing objects being dropped
			dependencies = getViewDependenciesFromAST(viewDiff.OldASTNode, viewDiff.SchemaName, tempMetadata)
		}

		for _, dep := range dependencies {
			depID := getMigrationObjectID(dep.Schema, dep.Table)
			if allObjects[depID] {
				// Edge from dependent to dependency (view depends on table/view)
				graph.AddEdge(viewID, depID)
			}
		}
	}

	// For materialized views depending on tables/views
	for mvID, mvDiff := range materializedViewMap {
		var dependencies []*storepb.DependencyColumn

		if mvDiff.OldMaterializedView != nil {
			// Use metadata if available
			dependencies = mvDiff.OldMaterializedView.DependencyColumns
		} else if mvDiff.OldASTNode != nil {
			// Extract dependencies from AST node for AST-only mode
			// Use the temporary metadata containing objects being dropped
			dependencies = getMaterializedViewDependenciesFromAST(mvDiff.OldASTNode, mvDiff.SchemaName, tempMetadata)
		}

		for _, dep := range dependencies {
			depID := getMigrationObjectID(dep.Schema, dep.Table)
			if allObjects[depID] {
				// Edge from dependent to dependency (materialized view depends on table/view)
				// For DROP: mv -> view/table means mv should be dropped before view/table
				graph.AddEdge(mvID, depID)
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

	// For tables with foreign keys depending on other tables (DROP operations)
	for tableID, tableDiff := range tableMap {
		var foreignKeys []*storepb.ForeignKeyMetadata

		if tableDiff.OldTable != nil {
			// Metadata mode: use ForeignKeys from metadata
			foreignKeys = tableDiff.OldTable.ForeignKeys
		} else if tableDiff.OldASTNode != nil {
			// AST-only mode: extract foreign keys from AST node
			foreignKeys = extractForeignKeysFromAST(tableDiff.OldASTNode, tableDiff.SchemaName)
		}

		for _, fk := range foreignKeys {
			depID := getMigrationObjectID(fk.ReferencedSchema, fk.ReferencedTable)
			if allObjects[depID] && depID != tableID {
				// Edge from table with FK to referenced table
				// For DROP: table1 (with FK) -> table2 (referenced)
				// This ensures table1 is dropped before table2
				graph.AddEdge(tableID, depID)
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
						if fkDiff.OldForeignKey != nil {
							// Metadata mode: use constraint metadata
							writeDropForeignKey(buf, tableDiff.SchemaName, tableDiff.TableName, fkDiff.OldForeignKey.Name)
						} else if fkDiff.OldASTNode != nil {
							// AST-only mode: extract from AST node
							if constraintAST, ok := fkDiff.OldASTNode.(pgparser.ITableconstraintContext); ok {
								writeDropForeignKeyFromAST(buf, tableDiff.SchemaName, tableDiff.TableName, constraintAST)
							}
						}
					}
				}
			}
		}

		// Triggers have already been dropped at the beginning of generateDrops function
		// to avoid dependency errors with functions and tables

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
			if viewDiff.Action == schema.MetadataDiffActionDrop {
				writeDropView(buf, viewDiff.SchemaName, viewDiff.ViewName)
			}
		}

		// Drop materialized views
		for _, mvDiff := range materializedViewMap {
			if mvDiff.Action == schema.MetadataDiffActionDrop {
				writeDropMaterializedView(buf, mvDiff.SchemaName, mvDiff.MaterializedViewName)
			}
		}

		// Drop functions
		for _, funcDiff := range functionMap {
			definition := getFunctionDefinitionForDrop(funcDiff)
			writeDropFunction(buf, funcDiff.SchemaName, funcDiff.FunctionName, funcDiff.OldASTNode, definition)
		}

		// Drop tables
		for _, tableDiff := range tableMap {
			writeDropTable(buf, tableDiff.SchemaName, tableDiff.TableName)
		}

		// Handle remaining ALTER table operations (constraints, indexes, columns)
		for _, tableDiff := range diff.TableChanges {
			if tableDiff.Action == schema.MetadataDiffActionAlter {
				// Drop triggers
				for _, triggerDiff := range tableDiff.TriggerChanges {
					if triggerDiff.Action == schema.MetadataDiffActionDrop {
						// Skip triggers with empty names (defensive check)
						if triggerDiff.TriggerName == "" {
							continue
						}
						writeDropTrigger(buf, tableDiff.SchemaName, tableDiff.TableName, triggerDiff.TriggerName)
					}
				}

				// Drop check constraints
				for _, checkDiff := range tableDiff.CheckConstraintChanges {
					if checkDiff.Action == schema.MetadataDiffActionDrop {
						if checkDiff.OldCheckConstraint != nil {
							// Metadata mode: use constraint metadata
							writeDropCheckConstraint(buf, tableDiff.SchemaName, tableDiff.TableName, checkDiff.OldCheckConstraint.Name)
						} else if checkDiff.OldASTNode != nil {
							// AST-only mode: extract from AST node
							if constraintAST, ok := checkDiff.OldASTNode.(pgparser.ITableconstraintContext); ok {
								writeDropCheckConstraintFromAST(buf, tableDiff.SchemaName, tableDiff.TableName, constraintAST)
							}
						}
					}
				}
				// Drop EXCLUDE constraints
				for _, excludeDiff := range tableDiff.ExcludeConstraintChanges {
					if excludeDiff.Action == schema.MetadataDiffActionDrop {
						if excludeDiff.OldExcludeConstraint != nil {
							// Metadata mode: use constraint metadata
							writeDropConstraint(buf, tableDiff.SchemaName, tableDiff.TableName, excludeDiff.OldExcludeConstraint.Name)
						} else if excludeDiff.OldASTNode != nil {
							// AST-only mode: extract from AST node
							if constraintAST, ok := excludeDiff.OldASTNode.(pgparser.ITableconstraintContext); ok {
								writeDropExcludeConstraintFromAST(buf, tableDiff.SchemaName, tableDiff.TableName, constraintAST)
							}
						}
					}
				}
				// Drop primary key constraints
				for _, pkDiff := range tableDiff.PrimaryKeyChanges {
					if pkDiff.Action == schema.MetadataDiffActionDrop {
						if pkDiff.OldPrimaryKey != nil {
							// Metadata mode: use constraint metadata
							writeDropConstraint(buf, tableDiff.SchemaName, tableDiff.TableName, pkDiff.OldPrimaryKey.Name)
						} else if pkDiff.OldASTNode != nil {
							// AST-only mode: extract from AST node
							if constraintAST, ok := pkDiff.OldASTNode.(pgparser.ITableconstraintContext); ok {
								writeDropPrimaryKeyFromAST(buf, tableDiff.SchemaName, tableDiff.TableName, constraintAST)
							}
						}
					}
				}

				// Drop indexes
				for _, indexDiff := range tableDiff.IndexChanges {
					if indexDiff.Action == schema.MetadataDiffActionDrop {
						if indexDiff.OldIndex != nil {
							// Metadata mode: use index metadata
							if indexDiff.OldIndex.IsConstraint {
								writeDropConstraint(buf, tableDiff.SchemaName, tableDiff.TableName, indexDiff.OldIndex.Name)
							} else {
								writeDropIndex(buf, tableDiff.SchemaName, indexDiff.OldIndex.Name)
							}
						} else if indexDiff.OldASTNode != nil {
							// AST-only mode: extract from AST node
							if indexAST, ok := indexDiff.OldASTNode.(*pgparser.IndexstmtContext); ok {
								writeDropIndexFromAST(buf, tableDiff.SchemaName, indexAST)
							}
						}
					}
				}

				// Drop columns
				for _, colDiff := range tableDiff.ColumnChanges {
					if colDiff.Action == schema.MetadataDiffActionDrop {
						if colDiff.OldColumn != nil {
							// Metadata mode - use column metadata
							writeDropColumn(buf, tableDiff.SchemaName, tableDiff.TableName, colDiff.OldColumn.Name)
						} else if colDiff.OldASTNode != nil {
							// AST-only mode - use AST functions
							_ = writeDropColumnFromAST(buf, tableDiff.SchemaName, tableDiff.TableName, colDiff.OldASTNode)
						}
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
						if fkDiff.OldForeignKey != nil {
							// Metadata mode: use constraint metadata
							writeDropForeignKey(buf, tableDiff.SchemaName, tableDiff.TableName, fkDiff.OldForeignKey.Name)
						} else if fkDiff.OldASTNode != nil {
							// AST-only mode: extract from AST node
							if constraintAST, ok := fkDiff.OldASTNode.(pgparser.ITableconstraintContext); ok {
								writeDropForeignKeyFromAST(buf, tableDiff.SchemaName, tableDiff.TableName, constraintAST)
							}
						}
					}
				}
			}
		}

		// Drop in topological order (most dependent first)
		// Note: Triggers have already been dropped at the beginning of generateDrops
		for _, objID := range orderedList {
			// Drop the object itself
			if viewDiff, ok := viewMap[objID]; ok {
				if viewDiff.Action == schema.MetadataDiffActionDrop {
					writeDropView(buf, viewDiff.SchemaName, viewDiff.ViewName)
				}
			} else if mvDiff, ok := materializedViewMap[objID]; ok {
				if mvDiff.Action == schema.MetadataDiffActionDrop {
					writeDropMaterializedView(buf, mvDiff.SchemaName, mvDiff.MaterializedViewName)
				}
			} else if funcDiff, ok := functionMap[objID]; ok {
				definition := getFunctionDefinitionForDrop(funcDiff)
				writeDropFunction(buf, funcDiff.SchemaName, funcDiff.FunctionName, funcDiff.OldASTNode, definition)
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
				// Drop triggers
				for _, triggerDiff := range tableDiff.TriggerChanges {
					if triggerDiff.Action == schema.MetadataDiffActionDrop {
						// Skip triggers with empty names (defensive check)
						if triggerDiff.TriggerName == "" {
							continue
						}
						writeDropTrigger(buf, tableDiff.SchemaName, tableDiff.TableName, triggerDiff.TriggerName)
					}
				}

				// Drop check constraints
				for _, checkDiff := range tableDiff.CheckConstraintChanges {
					if checkDiff.Action == schema.MetadataDiffActionDrop {
						if checkDiff.OldCheckConstraint != nil {
							// Metadata mode: use constraint metadata
							writeDropCheckConstraint(buf, tableDiff.SchemaName, tableDiff.TableName, checkDiff.OldCheckConstraint.Name)
						} else if checkDiff.OldASTNode != nil {
							// AST-only mode: extract from AST node
							if constraintAST, ok := checkDiff.OldASTNode.(pgparser.ITableconstraintContext); ok {
								writeDropCheckConstraintFromAST(buf, tableDiff.SchemaName, tableDiff.TableName, constraintAST)
							}
						}
					}
				}
				// Drop EXCLUDE constraints
				for _, excludeDiff := range tableDiff.ExcludeConstraintChanges {
					if excludeDiff.Action == schema.MetadataDiffActionDrop {
						if excludeDiff.OldExcludeConstraint != nil {
							// Metadata mode: use constraint metadata
							writeDropConstraint(buf, tableDiff.SchemaName, tableDiff.TableName, excludeDiff.OldExcludeConstraint.Name)
						} else if excludeDiff.OldASTNode != nil {
							// AST-only mode: extract from AST node
							if constraintAST, ok := excludeDiff.OldASTNode.(pgparser.ITableconstraintContext); ok {
								writeDropExcludeConstraintFromAST(buf, tableDiff.SchemaName, tableDiff.TableName, constraintAST)
							}
						}
					}
				}
				// Drop primary key constraints
				for _, pkDiff := range tableDiff.PrimaryKeyChanges {
					if pkDiff.Action == schema.MetadataDiffActionDrop {
						if pkDiff.OldPrimaryKey != nil {
							// Metadata mode: use constraint metadata
							writeDropConstraint(buf, tableDiff.SchemaName, tableDiff.TableName, pkDiff.OldPrimaryKey.Name)
						} else if pkDiff.OldASTNode != nil {
							// AST-only mode: extract from AST node
							if constraintAST, ok := pkDiff.OldASTNode.(pgparser.ITableconstraintContext); ok {
								writeDropPrimaryKeyFromAST(buf, tableDiff.SchemaName, tableDiff.TableName, constraintAST)
							}
						}
					}
				}

				// Drop indexes
				for _, indexDiff := range tableDiff.IndexChanges {
					if indexDiff.Action == schema.MetadataDiffActionDrop {
						if indexDiff.OldIndex != nil {
							// Metadata mode: use index metadata
							if indexDiff.OldIndex.IsConstraint {
								writeDropConstraint(buf, tableDiff.SchemaName, tableDiff.TableName, indexDiff.OldIndex.Name)
							} else {
								writeDropIndex(buf, tableDiff.SchemaName, indexDiff.OldIndex.Name)
							}
						} else if indexDiff.OldASTNode != nil {
							// AST-only mode: extract from AST node
							if indexAST, ok := indexDiff.OldASTNode.(*pgparser.IndexstmtContext); ok {
								writeDropIndexFromAST(buf, tableDiff.SchemaName, indexAST)
							}
						}
					}
				}

				// Drop columns
				for _, colDiff := range tableDiff.ColumnChanges {
					if colDiff.Action == schema.MetadataDiffActionDrop {
						if colDiff.OldColumn != nil {
							// Metadata mode - use column metadata
							writeDropColumn(buf, tableDiff.SchemaName, tableDiff.TableName, colDiff.OldColumn.Name)
						} else if colDiff.OldASTNode != nil {
							// AST-only mode - use AST functions
							_ = writeDropColumnFromAST(buf, tableDiff.SchemaName, tableDiff.TableName, colDiff.OldASTNode)
						}
					}
				}
			}
		}

		// Handle remaining ALTER materialized view drops (indexes)
		for _, mvDiff := range diff.MaterializedViewChanges {
			if mvDiff.Action == schema.MetadataDiffActionAlter {
				// Drop indexes
				for _, indexDiff := range mvDiff.IndexChanges {
					if indexDiff.Action == schema.MetadataDiffActionDrop {
						if indexDiff.OldIndex != nil {
							// Metadata mode: use index metadata
							if indexDiff.OldIndex.IsConstraint {
								writeDropConstraint(buf, mvDiff.SchemaName, mvDiff.MaterializedViewName, indexDiff.OldIndex.Name)
							} else {
								writeDropIndex(buf, mvDiff.SchemaName, indexDiff.OldIndex.Name)
							}
						} else if indexDiff.OldASTNode != nil {
							// AST-only mode: extract from AST node
							if indexAST, ok := indexDiff.OldASTNode.(*pgparser.IndexstmtContext); ok {
								writeDropIndexFromAST(buf, mvDiff.SchemaName, indexAST)
							}
						}
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

	// Drop extensions (after enum types and tables)
	for _, extDiff := range diff.ExtensionChanges {
		if extDiff.Action == schema.MetadataDiffActionDrop {
			writeDropExtension(buf, extDiff.ExtensionName)
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
					// In metadata mode, no AST node is available, so pass nil
					writeDropFunction(buf, schemaDiff.SchemaName, fn.Name, nil, fn.Definition)
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

	// Create extensions (before enum types and tables as they might provide types used in definitions)
	for _, extDiff := range diff.ExtensionChanges {
		if extDiff.Action == schema.MetadataDiffActionCreate {
			// Support both metadata and AST-only modes
			if extDiff.NewExtension != nil {
				// Metadata mode: use extension metadata
				if err := writeCreateExtension(buf, extDiff.NewExtension); err != nil {
					return err
				}
			} else if extDiff.NewASTNode != nil {
				// AST-only mode: extract SQL from AST node
				if err := writeMigrationExtensionFromAST(buf, extDiff.NewASTNode); err != nil {
					return err
				}
			}
		}
	}

	// Create enum types (before tables as they might be used in column definitions)
	for _, enumDiff := range diff.EnumTypeChanges {
		if enumDiff.Action == schema.MetadataDiffActionCreate {
			// Support both metadata and AST-only modes
			if enumDiff.NewEnumType != nil {
				// Metadata mode: use enum type metadata
				if err := writeCreateEnumType(buf, enumDiff.SchemaName, enumDiff.NewEnumType); err != nil {
					return err
				}
			} else if enumDiff.NewASTNode != nil {
				// AST-only mode: extract SQL from AST node
				if err := writeMigrationEnumTypeFromAST(buf, enumDiff.NewASTNode); err != nil {
					return err
				}
			}
		}
	}

	// Create sequences (before tables as they might be used in column defaults)
	for _, seqDiff := range diff.SequenceChanges {
		if seqDiff.Action == schema.MetadataDiffActionCreate {
			// Support both metadata and AST-only modes
			if seqDiff.NewSequence != nil {
				// Metadata mode: use sequence metadata
				if err := writeMigrationCreateSequence(buf, seqDiff.SchemaName, seqDiff.NewSequence); err != nil {
					return err
				}
			} else if seqDiff.NewASTNode != nil {
				// AST-only mode: extract SQL from AST node
				if err := writeMigrationSequenceFromAST(buf, seqDiff.NewASTNode); err != nil {
					return err
				}
			} else {
				return errors.Errorf("sequence diff for %s.%s has neither metadata nor AST node", seqDiff.SchemaName, seqDiff.SequenceName)
			}
		}
	}

	// Build dependency graph for all objects being created or altered
	graph := base.NewGraph()

	// Build temporary metadata for AST-only mode dependency extraction
	// This metadata contains tables and views that are being created,
	// allowing GetQuerySpan to find them when extracting view dependencies
	tempMetadata := buildTempMetadataForCreate(diff)

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

	// Add triggers to graph (only for CREATE table operations)
	// Triggers on ALTER tables are handled separately via generateAlterTableTriggers
	triggerMap := make(map[string]*schema.TriggerDiff)
	for _, tableDiff := range diff.TableChanges {
		// Only add triggers for CREATE table operations
		// ALTER table triggers are handled in generateAlterTableTriggers to avoid duplicates
		if tableDiff.Action == schema.MetadataDiffActionCreate {
			for _, triggerDiff := range tableDiff.TriggerChanges {
				if triggerDiff.Action == schema.MetadataDiffActionCreate {
					triggerID := getTriggerObjectID(triggerDiff)
					graph.AddNode(triggerID)
					triggerMap[triggerID] = triggerDiff
					allObjects[triggerID] = true
				}
			}
		}
	}

	// Add dependency edges
	// For tables with foreign keys depending on other tables (only for CREATE operations)
	// Note: ALTER FK additions are handled after all tables are created, so they don't need topological sorting
	for tableID, tableDiff := range tableMap {
		if tableDiff.Action == schema.MetadataDiffActionCreate {
			// For CREATE: extract all FKs from the table
			var foreignKeys []*storepb.ForeignKeyMetadata

			if tableDiff.NewTable != nil {
				// Metadata mode: use ForeignKeys from metadata
				foreignKeys = tableDiff.NewTable.ForeignKeys
			} else if tableDiff.NewASTNode != nil {
				// AST-only mode: extract foreign keys from AST node
				foreignKeys = extractForeignKeysFromAST(tableDiff.NewASTNode, tableDiff.SchemaName)
			}

			for _, fk := range foreignKeys {
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
		var dependencies []*storepb.DependencyColumn

		if viewDiff.NewView != nil {
			// Use metadata if available
			dependencies = viewDiff.NewView.DependencyColumns
		} else if viewDiff.NewASTNode != nil {
			// Extract dependencies from AST node for AST-only mode
			// Use the temporary metadata containing objects being created
			dependencies = getViewDependenciesFromAST(viewDiff.NewASTNode, viewDiff.SchemaName, tempMetadata)
		}

		for _, dep := range dependencies {
			depID := getMigrationObjectID(dep.Schema, dep.Table)
			if allObjects[depID] {
				// Edge from dependency to dependent (table/view to view)
				graph.AddEdge(depID, viewID)
			}
		}
	}

	// For materialized views depending on tables/views
	for mvID, mvDiff := range materializedViewMap {
		var dependencies []*storepb.DependencyColumn

		if mvDiff.NewMaterializedView != nil {
			// Use metadata if available
			dependencies = mvDiff.NewMaterializedView.DependencyColumns
		} else if mvDiff.NewASTNode != nil {
			// Extract dependencies from AST node for AST-only mode
			// Use the temporary metadata containing objects being created
			dependencies = getMaterializedViewDependenciesFromAST(mvDiff.NewASTNode, mvDiff.SchemaName, tempMetadata)
		}

		for _, dep := range dependencies {
			depID := getMigrationObjectID(dep.Schema, dep.Table)
			if allObjects[depID] {
				// Edge from dependency to dependent (table/view to materialized view)
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

	// For triggers depending on tables and functions
	for triggerID, triggerDiff := range triggerMap {
		// Trigger depends on table
		tableID := getMigrationObjectID(triggerDiff.SchemaName, triggerDiff.TableName)
		if allObjects[tableID] {
			graph.AddEdge(tableID, triggerID)
		}

		// Trigger depends on trigger function
		if triggerDiff.NewASTNode != nil {
			functionName := extractTriggerFunctionName(triggerDiff.NewASTNode)
			if functionName != "" {
				parts := strings.Split(functionName, ".")
				var functionSchemaName, functionNameOnly string
				if len(parts) == 2 {
					functionSchemaName, functionNameOnly = parts[0], parts[1]
				} else {
					functionSchemaName, functionNameOnly = triggerDiff.SchemaName, functionName
				}
				functionID := getMigrationObjectID(functionSchemaName, functionNameOnly)
				if allObjects[functionID] {
					graph.AddEdge(functionID, triggerID)
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
				// Support both metadata mode and AST-only mode
				if tableDiff.NewTable != nil {
					// Metadata mode: use generateCreateTable
					createTableSQL, err := generateCreateTable(tableDiff.SchemaName, tableDiff.TableName, tableDiff.NewTable, false)
					if err != nil {
						return err
					}
					_, _ = buf.WriteString(createTableSQL)
					if createTableSQL != "" {
						_, _ = buf.WriteString("\n")
					}

					// Add table and column comments for newly created tables
					if tableDiff.NewTable.Comment != "" {
						writeCommentOnTable(buf, tableDiff.SchemaName, tableDiff.TableName, tableDiff.NewTable.Comment)
					}
					for _, col := range tableDiff.NewTable.Columns {
						if col.Comment != "" {
							writeCommentOnColumn(buf, tableDiff.SchemaName, tableDiff.TableName, col.Name, col.Comment)
						}
					}
				} else if tableDiff.NewASTNode != nil {
					// AST-only mode: extract SQL from AST node
					if err := writeMigrationTableFromAST(buf, tableDiff.NewASTNode); err != nil {
						return err
					}
				} else {
					return errors.Errorf("table CREATE action requires either NewTable metadata or NewASTNode for table %s.%s", tableDiff.SchemaName, tableDiff.TableName)
				}

				// Add indexes for newly created tables immediately after table creation
				// This is necessary because later tables might have FK that reference indexed columns
				for _, indexDiff := range tableDiff.IndexChanges {
					if indexDiff.Action == schema.MetadataDiffActionCreate {
						if indexDiff.NewIndex != nil {
							// Metadata mode: use index metadata
							if indexDiff.NewIndex.IsConstraint {
								// Skip constraint-based indexes (primary key, unique, exclude)
								// They are already included in CREATE TABLE statement
								continue
							}
							writeMigrationIndex(buf, tableDiff.SchemaName, tableDiff.TableName, indexDiff.NewIndex)
						} else if indexDiff.NewASTNode != nil {
							// AST-only mode: extract from AST node
							if indexAST, ok := indexDiff.NewASTNode.(*pgparser.IndexstmtContext); ok {
								if err := writeCreateIndexFromAST(buf, indexAST); err != nil {
									// If AST extraction fails, log error but continue (non-fatal)
									_, _ = fmt.Fprintf(buf, "-- Error creating index: %v\n", err)
								}
							}
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
				// Support AST-only mode: if metadata is nil but AST node exists, use AST
				if viewDiff.NewView != nil {
					if err := writeMigrationView(buf, viewDiff.SchemaName, viewDiff.NewView); err != nil {
						return err
					}
				} else if viewDiff.NewASTNode != nil {
					if err := writeMigrationViewFromAST(buf, viewDiff.NewASTNode); err != nil {
						return err
					}
				} else {
					return errors.Errorf("view diff for %s.%s has neither metadata nor AST node", viewDiff.SchemaName, viewDiff.ViewName)
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
				// Support AST-only mode: if metadata is nil but AST node exists, use AST
				if mvDiff.NewMaterializedView != nil {
					if err := writeMigrationMaterializedView(buf, mvDiff.SchemaName, mvDiff.NewMaterializedView); err != nil {
						return err
					}
				} else if mvDiff.NewASTNode != nil {
					if err := writeMigrationMaterializedViewFromAST(buf, mvDiff.NewASTNode); err != nil {
						return err
					}
				} else {
					return errors.Errorf("materialized view diff for %s.%s has neither metadata nor AST node", mvDiff.SchemaName, mvDiff.MaterializedViewName)
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
				writeCommentOnFunction(buf, funcDiff.SchemaName, funcDiff.NewFunction.Signature, funcDiff.NewFunction.Comment, funcDiff.NewASTNode, funcDiff.NewFunction.Definition)
			}
		}

		// Set sequence ownership after all tables are created
		for _, seqDiff := range diff.SequenceChanges {
			if seqDiff.Action == schema.MetadataDiffActionCreate && seqDiff.NewSequence != nil && seqDiff.NewSequence.OwnerTable != "" && seqDiff.NewSequence.OwnerColumn != "" {
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
					// Support both metadata mode and AST-only mode
					if tableDiff.NewTable != nil {
						// Metadata mode: use generateCreateTable
						createTableSQL, err := generateCreateTable(tableDiff.SchemaName, tableDiff.TableName, tableDiff.NewTable, false)
						if err != nil {
							return err
						}
						_, _ = buf.WriteString(createTableSQL)
						if createTableSQL != "" {
							_, _ = buf.WriteString("\n")
						}

						// Add table and column comments for newly created tables
						if tableDiff.NewTable.Comment != "" {
							writeCommentOnTable(buf, tableDiff.SchemaName, tableDiff.TableName, tableDiff.NewTable.Comment)
						}
						for _, col := range tableDiff.NewTable.Columns {
							if col.Comment != "" {
								writeCommentOnColumn(buf, tableDiff.SchemaName, tableDiff.TableName, col.Name, col.Comment)
							}
						}
					} else if tableDiff.NewASTNode != nil {
						// AST-only mode: extract SQL from AST node
						if err := writeMigrationTableFromAST(buf, tableDiff.NewASTNode); err != nil {
							return err
						}
					} else {
						return errors.Errorf("table CREATE action requires either NewTable metadata or NewASTNode for table %s.%s", tableDiff.SchemaName, tableDiff.TableName)
					}

					// Add indexes for newly created tables immediately after table creation
					// This is necessary because later tables might have FK that reference indexed columns
					for _, indexDiff := range tableDiff.IndexChanges {
						if indexDiff.Action == schema.MetadataDiffActionCreate {
							if indexDiff.NewIndex != nil {
								// Metadata mode: use index metadata
								if indexDiff.NewIndex.IsConstraint {
									// Skip constraint-based indexes (primary key, unique, exclude)
									// They are already included in CREATE TABLE statement
									continue
								}
								writeMigrationIndex(buf, tableDiff.SchemaName, tableDiff.TableName, indexDiff.NewIndex)
							} else if indexDiff.NewASTNode != nil {
								// AST-only mode: extract from AST node
								if indexAST, ok := indexDiff.NewASTNode.(*pgparser.IndexstmtContext); ok {
									if err := writeCreateIndexFromAST(buf, indexAST); err != nil {
										// If AST extraction fails, log error but continue (non-fatal)
										_, _ = fmt.Fprintf(buf, "-- Error creating index: %v\n", err)
									}
								}
							}
						}
					}
				case schema.MetadataDiffActionAlter:
					// Handle ALTER table operations with generateAlterTableWithOptions
					// Skip foreign keys in the first pass to avoid dependency issues
					alterTableSQL, err := generateAlterTableWithOptions(tableDiff, true)
					if err != nil {
						return err
					}
					_, _ = buf.WriteString(alterTableSQL)
					if alterTableSQL != "" {
						_, _ = buf.WriteString("\n")
					}
				default:
					// No action needed for other operations
				}
			} else if viewDiff, ok := viewMap[objID]; ok {
				switch viewDiff.Action {
				case schema.MetadataDiffActionCreate, schema.MetadataDiffActionAlter:
					// Support AST-only mode: if metadata is nil but AST node exists, use AST
					if viewDiff.NewView != nil {
						if err := writeMigrationView(buf, viewDiff.SchemaName, viewDiff.NewView); err != nil {
							return err
						}
					} else if viewDiff.NewASTNode != nil {
						if err := writeMigrationViewFromAST(buf, viewDiff.NewASTNode); err != nil {
							return err
						}
					} else {
						return errors.Errorf("view diff for %s.%s has neither metadata nor AST node", viewDiff.SchemaName, viewDiff.ViewName)
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
					// Support AST-only mode: if metadata is nil but AST node exists, use AST
					if mvDiff.NewMaterializedView != nil {
						if err := writeMigrationMaterializedView(buf, mvDiff.SchemaName, mvDiff.NewMaterializedView); err != nil {
							return err
						}
					} else if mvDiff.NewASTNode != nil {
						if err := writeMigrationMaterializedViewFromAST(buf, mvDiff.NewASTNode); err != nil {
							return err
						}
					} else {
						return errors.Errorf("materialized view diff for %s.%s has neither metadata nor AST node", mvDiff.SchemaName, mvDiff.MaterializedViewName)
					}
				case schema.MetadataDiffActionAlter:
					// Check if this is an index-only change (no MV definition change)
					hasIndexOnlyChanges := len(mvDiff.IndexChanges) > 0 && mvDiff.NewMaterializedView == nil && mvDiff.NewASTNode == nil

					// For index-only changes, don't alter the MV itself
					// Index changes are processed separately later (see lines ~1107-1127)
					if !hasIndexOnlyChanges {
						// For PostgreSQL materialized views, we need to drop and recreate
						// since ALTER MATERIALIZED VIEW doesn't support changing the definition
						writeDropMaterializedView(buf, mvDiff.SchemaName, mvDiff.MaterializedViewName)
						// Support AST-only mode
						if mvDiff.NewMaterializedView != nil {
							if err := writeMigrationMaterializedView(buf, mvDiff.SchemaName, mvDiff.NewMaterializedView); err != nil {
								return err
							}
						} else if mvDiff.NewASTNode != nil {
							if err := writeMigrationMaterializedViewFromAST(buf, mvDiff.NewASTNode); err != nil {
								return err
							}
						} else {
							return errors.Errorf("materialized view ALTER for %s.%s has neither metadata nor AST node", mvDiff.SchemaName, mvDiff.MaterializedViewName)
						}
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
					writeCommentOnFunction(buf, funcDiff.SchemaName, funcDiff.NewFunction.Signature, funcDiff.NewFunction.Comment, funcDiff.NewASTNode, funcDiff.NewFunction.Definition)
				}
			} else if triggerDiff, ok := triggerMap[objID]; ok {
				// Handle triggers (both CREATE and ALTER use CREATE OR REPLACE)
				if triggerDiff.Action == schema.MetadataDiffActionCreate || triggerDiff.Action == schema.MetadataDiffActionAlter {
					if err := writeCreateTrigger(buf, triggerDiff); err != nil {
						return err
					}
				}
			}
		}

		// Set sequence ownership after all tables are created
		for _, seqDiff := range diff.SequenceChanges {
			if seqDiff.Action == schema.MetadataDiffActionCreate && seqDiff.NewSequence != nil && seqDiff.NewSequence.OwnerTable != "" && seqDiff.NewSequence.OwnerColumn != "" {
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

		// Add foreign keys for ALTER table operations after all tables are created
		for _, tableDiff := range tableMap {
			if tableDiff.Action == schema.MetadataDiffActionAlter {
				// Only add foreign keys in this second pass
				fkSQL, err := generateAlterTableForeignKeys(tableDiff)
				if err != nil {
					return err
				}
				_, _ = buf.WriteString(fkSQL)
				if fkSQL != "" {
					_, _ = buf.WriteString("\n")
				}
			}
		}

		// Add triggers for ALTER table operations after foreign keys
		for _, tableDiff := range tableMap {
			if tableDiff.Action == schema.MetadataDiffActionAlter {
				triggerSQL := generateAlterTableTriggers(tableDiff)
				_, _ = buf.WriteString(triggerSQL)
				if triggerSQL != "" {
					_, _ = buf.WriteString("\n")
				}
			}
		}

		// Create triggers for CREATE table operations that aren't in TriggerChanges
		// This handles metadata mode where triggers are in NewTable.Triggers but not in TriggerChanges
		for _, tableDiff := range tableMap {
			if tableDiff.Action == schema.MetadataDiffActionCreate && tableDiff.NewTable != nil {
				// Only create triggers that weren't already created from TriggerChanges
				triggersInChanges := make(map[string]bool)
				for _, triggerDiff := range tableDiff.TriggerChanges {
					if triggerDiff.Action == schema.MetadataDiffActionCreate {
						triggersInChanges[triggerDiff.TriggerName] = true
					}
				}

				for _, trigger := range tableDiff.NewTable.Triggers {
					// Skip if already created from TriggerChanges
					if !triggersInChanges[trigger.Name] {
						writeMigrationTrigger(buf, trigger)
					}
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

		// Add indexes for materialized view ALTER operations
		for _, mvDiff := range diff.MaterializedViewChanges {
			if mvDiff.Action == schema.MetadataDiffActionAlter {
				for _, indexDiff := range mvDiff.IndexChanges {
					if indexDiff.Action == schema.MetadataDiffActionCreate {
						if indexDiff.NewIndex != nil {
							// Metadata mode: use index metadata
							writeMigrationMaterializedViewIndex(buf, mvDiff.SchemaName, mvDiff.MaterializedViewName, indexDiff.NewIndex)
						} else if indexDiff.NewASTNode != nil {
							// AST-only mode: extract from AST node
							if indexAST, ok := indexDiff.NewASTNode.(*pgparser.IndexstmtContext); ok {
								if err := writeCreateIndexFromAST(buf, indexAST); err != nil {
									// If AST extraction fails, log error but continue (non-fatal)
									_, _ = fmt.Fprintf(buf, "-- Error creating index: %v\n", err)
								}
							}
						}
					}
				}
			}
		}
	}

	// ALTER table operations are now handled earlier in the topological order
	for _, tableDiff := range diff.TableChanges {
		if tableDiff.Action == schema.MetadataDiffActionAlter {
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

	// Handle materialized view index comment changes
	for _, mvDiff := range diff.MaterializedViewChanges {
		if mvDiff.Action == schema.MetadataDiffActionAlter {
			if err := generateMaterializedViewIndexCommentChanges(buf, mvDiff); err != nil {
				return err
			}
		}
	}

	// Handle function comment changes
	if err := generateFunctionCommentChanges(buf, diff); err != nil {
		return err
	}

	// Handle sequence comment changes
	if err := generateSequenceCommentChanges(buf, diff); err != nil {
		return err
	}

	// Handle sequence ownership changes (must be after all tables are created)
	if err := generateSequenceOwnershipChanges(buf, diff); err != nil {
		return err
	}

	// Create event triggers (last, as they depend on functions which may have been created)
	for _, etDiff := range diff.EventTriggerChanges {
		if etDiff.Action == schema.MetadataDiffActionCreate {
			// Support both metadata and AST-only modes
			if etDiff.NewEventTrigger != nil {
				// Metadata mode: use event trigger metadata
				if err := writeCreateEventTrigger(buf, etDiff.NewEventTrigger); err != nil {
					return err
				}
			} else if etDiff.NewASTNode != nil {
				// AST-only mode: extract SQL from AST node
				if err := writeMigrationEventTriggerFromAST(buf, etDiff.NewASTNode); err != nil {
					return err
				}
			}
		}
	}

	// Handle enum type comment changes
	if err := generateEnumTypeCommentChanges(buf, diff); err != nil {
		return err
	}

	// Handle SDL-based comment changes (from SDL diff mode)
	return generateCommentChangesFromSDL(buf, diff)
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
				// Support both metadata mode and AST-only mode
				if colDiff.NewColumn != nil {
					// Metadata mode: use existing writeAddColumn
					writeAddColumn(&buf, tableDiff.SchemaName, tableDiff.TableName, colDiff.NewColumn)
				} else if colDiff.NewASTNode != nil {
					// AST-only mode: extract SQL from AST node
					if err := writeAddColumnFromAST(&buf, tableDiff.SchemaName, tableDiff.TableName, colDiff.NewASTNode); err != nil {
						return "", err
					}
				}
			}
		}
	}

	// Alter columns
	for _, colDiff := range tableDiff.ColumnChanges {
		if colDiff.Action == schema.MetadataDiffActionAlter {
			// Support both metadata mode and AST-only mode
			if colDiff.OldColumn != nil && colDiff.NewColumn != nil {
				// Metadata mode: use existing generateAlterColumn
				alterColSQL := generateAlterColumn(tableDiff.SchemaName, tableDiff.TableName, colDiff)
				_, _ = buf.WriteString(alterColSQL)
			} else if colDiff.OldASTNode != nil && colDiff.NewASTNode != nil {
				// AST-only mode: use ALTER COLUMN approach
				if err := writeAlterColumnFromAST(&buf, tableDiff.SchemaName, tableDiff.TableName, colDiff.OldASTNode, colDiff.NewASTNode); err != nil {
					return "", err
				}
			}
		}
	}

	// Add indexes
	for _, indexDiff := range tableDiff.IndexChanges {
		if indexDiff.Action == schema.MetadataDiffActionCreate {
			if indexDiff.NewIndex != nil {
				// Metadata mode: use index metadata
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
			} else if indexDiff.NewASTNode != nil {
				// AST-only mode: extract from AST node
				if indexAST, ok := indexDiff.NewASTNode.(*pgparser.IndexstmtContext); ok {
					if err := writeCreateIndexFromAST(&buf, indexAST); err != nil {
						// If AST extraction fails, log error but continue (non-fatal)
						_, _ = buf.WriteString(fmt.Sprintf("-- Error creating index: %v\n", err))
					}
				}
			}
		}
	}

	// Add check constraints
	for _, checkDiff := range tableDiff.CheckConstraintChanges {
		if checkDiff.Action == schema.MetadataDiffActionCreate {
			if checkDiff.NewCheckConstraint != nil {
				// Metadata mode: use constraint metadata
				writeAddCheckConstraint(&buf, tableDiff.SchemaName, tableDiff.TableName, checkDiff.NewCheckConstraint)
			} else if checkDiff.NewASTNode != nil {
				// AST-only mode: extract from AST node
				if constraintAST, ok := checkDiff.NewASTNode.(pgparser.ITableconstraintContext); ok {
					if err := writeAddCheckConstraintFromAST(&buf, tableDiff.SchemaName, tableDiff.TableName, constraintAST); err != nil {
						// If AST extraction fails, log error but continue (non-fatal)
						_, _ = buf.WriteString(fmt.Sprintf("-- Error adding check constraint: %v\n", err))
					}
				}
			}
		}
	}

	// Add EXCLUDE constraints
	for _, excludeDiff := range tableDiff.ExcludeConstraintChanges {
		if excludeDiff.Action == schema.MetadataDiffActionCreate {
			if excludeDiff.NewExcludeConstraint != nil {
				// Metadata mode: use constraint metadata
				writeAddExcludeConstraint(&buf, tableDiff.SchemaName, tableDiff.TableName, excludeDiff.NewExcludeConstraint)
			} else if excludeDiff.NewASTNode != nil {
				// AST-only mode: extract from AST node
				if constraintAST, ok := excludeDiff.NewASTNode.(pgparser.ITableconstraintContext); ok {
					if err := writeAddExcludeConstraintFromAST(&buf, tableDiff.SchemaName, tableDiff.TableName, constraintAST); err != nil {
						// If AST extraction fails, log error but continue (non-fatal)
						_, _ = buf.WriteString(fmt.Sprintf("-- Error adding EXCLUDE constraint: %v\n", err))
					}
				}
			}
		}
	}

	// Add unique constraints
	for _, uniqueDiff := range tableDiff.UniqueConstraintChanges {
		if uniqueDiff.Action == schema.MetadataDiffActionCreate {
			if uniqueDiff.NewUniqueConstraint != nil {
				// Metadata mode: use constraint metadata
				writeMigrationUniqueKey(&buf, tableDiff.SchemaName, tableDiff.TableName, uniqueDiff.NewUniqueConstraint)
			} else if uniqueDiff.NewASTNode != nil {
				// AST-only mode: extract from AST node
				if constraintAST, ok := uniqueDiff.NewASTNode.(pgparser.ITableconstraintContext); ok {
					if err := writeAddUniqueConstraintFromAST(&buf, tableDiff.SchemaName, tableDiff.TableName, constraintAST); err != nil {
						// If AST extraction fails, log error but continue (non-fatal)
						_, _ = buf.WriteString(fmt.Sprintf("-- Error adding unique constraint: %v\n", err))
					}
				}
			}
		}
	}

	// Add primary key constraints
	for _, pkDiff := range tableDiff.PrimaryKeyChanges {
		if pkDiff.Action == schema.MetadataDiffActionCreate {
			if pkDiff.NewPrimaryKey != nil {
				// Metadata mode: use constraint metadata
				writeMigrationPrimaryKey(&buf, tableDiff.SchemaName, tableDiff.TableName, pkDiff.NewPrimaryKey)
			} else if pkDiff.NewASTNode != nil {
				// AST-only mode: extract from AST node
				if constraintAST, ok := pkDiff.NewASTNode.(pgparser.ITableconstraintContext); ok {
					if err := writeAddPrimaryKeyFromAST(&buf, tableDiff.SchemaName, tableDiff.TableName, constraintAST); err != nil {
						// If AST extraction fails, log error but continue (non-fatal)
						_, _ = buf.WriteString(fmt.Sprintf("-- Error adding primary key constraint: %v\n", err))
					}
				}
			}
		}
	}

	return buf.String(), nil
}

// generateAlterTableForeignKeys generates foreign key statements for a table diff
func generateAlterTableForeignKeys(tableDiff *schema.TableDiff) (string, error) {
	var buf strings.Builder

	for _, fkDiff := range tableDiff.ForeignKeyChanges {
		if fkDiff.Action == schema.MetadataDiffActionCreate {
			if fkDiff.NewForeignKey != nil {
				// Metadata mode: use constraint metadata
				if err := writeMigrationForeignKey(&buf, tableDiff.SchemaName, tableDiff.TableName, fkDiff.NewForeignKey); err != nil {
					return "", err
				}
			} else if fkDiff.NewASTNode != nil {
				// AST-only mode: extract from AST node
				if constraintAST, ok := fkDiff.NewASTNode.(pgparser.ITableconstraintContext); ok {
					if err := writeAddForeignKeyFromAST(&buf, tableDiff.SchemaName, tableDiff.TableName, constraintAST); err != nil {
						// If AST extraction fails, log error but continue (non-fatal)
						_, _ = buf.WriteString(fmt.Sprintf("-- Error adding foreign key constraint: %v\n", err))
					}
				}
			}
		}
	}

	return buf.String(), nil
}

// generateAlterTableTriggers generates trigger statements for a table diff
func generateAlterTableTriggers(tableDiff *schema.TableDiff) string {
	var buf strings.Builder

	for _, triggerDiff := range tableDiff.TriggerChanges {
		if triggerDiff.Action == schema.MetadataDiffActionCreate || triggerDiff.Action == schema.MetadataDiffActionAlter {
			// Use writeCreateTrigger which supports both AST and metadata modes
			// Both CREATE and ALTER use CREATE OR REPLACE
			if err := writeCreateTrigger(&buf, triggerDiff); err != nil {
				// Log error but continue - this is a best-effort operation
				// The error will be caught in the caller if needed
				continue
			}
		}
	}

	return buf.String()
}

func generateAlterColumn(schemaName, tableName string, colDiff *schema.ColumnDiff) string {
	var buf strings.Builder

	// In PostgreSQL, we need to handle different aspects of column changes separately

	// If type changed, alter the column type
	if colDiff.OldColumn.Type != colDiff.NewColumn.Type {
		writeAlterColumnType(&buf, schemaName, tableName, colDiff.NewColumn.Name, colDiff.OldColumn.Type, colDiff.NewColumn.Type)
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

	// Use semantic comparison for default values
	// This handles cases like 'ARRAY[]::TEXT[]' vs 'ARRAY[]::text[]'
	if def1 == def2 {
		return true
	}

	// If they're not identical strings, try semantic comparison
	if def1 != "" || def2 != "" {
		// Use the expression comparer from the ast package
		comparer := ast.NewPostgreSQLExpressionComparer()
		equal, err := comparer.CompareExpressions(def1, def2)
		if err != nil {
			// If comparison fails, fall back to string comparison
			return def1 == def2
		}
		return equal
	}

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

// isFunctionProcedure checks if the AST node or definition represents a PROCEDURE (not a FUNCTION)
// Returns true if it's a PROCEDURE, false if it's a FUNCTION or cannot be determined
func isFunctionProcedure(astNode any, definition string) bool {
	// First, try to determine from AST node (most reliable)
	if astNode != nil {
		// Check if it's a CreatefunctionstmtContext with PROCEDURE keyword
		if ctx, ok := astNode.(*pgparser.CreatefunctionstmtContext); ok {
			// If PROCEDURE() returns non-nil, it's a PROCEDURE
			// If FUNCTION() returns non-nil, it's a FUNCTION
			return ctx.PROCEDURE() != nil
		}
		// Check if it's a CommentstmtContext (COMMENT ON FUNCTION/PROCEDURE)
		if ctx, ok := astNode.(*pgparser.CommentstmtContext); ok {
			// If PROCEDURE() returns non-nil, it's a COMMENT ON PROCEDURE
			// If FUNCTION() returns non-nil (and PROCEDURE is nil), it's a COMMENT ON FUNCTION
			return ctx.PROCEDURE() != nil
		}
	}

	// Fall back to checking definition string (for metadata mode or when AST is not available)
	if definition != "" {
		upperDef := strings.ToUpper(definition)
		// Check for PROCEDURE keyword in CREATE statement
		return strings.Contains(upperDef, " PROCEDURE ") ||
			strings.HasPrefix(upperDef, "CREATE PROCEDURE") ||
			strings.HasPrefix(upperDef, "CREATE OR REPLACE PROCEDURE")
	}

	return false
}

// getFunctionDefinitionForDrop extracts the function definition from either metadata or AST node
func getFunctionDefinitionForDrop(funcDiff *schema.FunctionDiff) string {
	// Try metadata first (if available)
	if funcDiff.OldFunction != nil {
		return funcDiff.OldFunction.Definition
	}

	// Fall back to extracting from AST node (AST-only mode)
	if funcDiff.OldASTNode != nil {
		// Use generic interface to extract text from any parser rule context
		type parserContext interface {
			GetParser() antlr.Parser
			GetStart() antlr.Token
			GetStop() antlr.Token
			GetText() string
		}

		if ruleContext, ok := funcDiff.OldASTNode.(parserContext); ok {
			// Try to use token stream first (preserves formatting)
			if parser := ruleContext.GetParser(); parser != nil {
				if tokenStream := parser.GetTokenStream(); tokenStream != nil {
					start := ruleContext.GetStart()
					stop := ruleContext.GetStop()
					if start != nil && stop != nil {
						text := tokenStream.GetTextFromTokens(start, stop)
						if text != "" {
							return text
						}
					}
				}
			}
			// Fallback to GetText() method
			return ruleContext.GetText()
		}
	}

	// No definition available
	return ""
}

func writeDropFunction(out *strings.Builder, schema, function string, astNode any, definition string) {
	// Determine if this is a PROCEDURE or FUNCTION using AST-based detection
	objectType := "FUNCTION"
	if isFunctionProcedure(astNode, definition) {
		objectType = "PROCEDURE"
	}

	_, _ = out.WriteString(`DROP `)
	_, _ = out.WriteString(objectType)
	_, _ = out.WriteString(` IF EXISTS "`)
	_, _ = out.WriteString(schema)
	_, _ = out.WriteString(`"."`)

	// Parse function signature to extract function name and parameters
	// Format: function_name(param1 type1, param2 type2)
	parenIndex := strings.Index(function, "(")
	if parenIndex > 0 {
		// Extract function name and parameters separately
		funcName := function[:parenIndex]
		params := function[parenIndex:] // includes parentheses
		_, _ = out.WriteString(funcName)
		_, _ = out.WriteString(`"`)
		_, _ = out.WriteString(params)
	} else {
		// Fallback for functions without parameters
		_, _ = out.WriteString(function)
		_, _ = out.WriteString(`"()`)
	}

	_, _ = out.WriteString(`;`)
	_, _ = out.WriteString("\n")
}

// writeMigrationSequenceFromAST writes CREATE SEQUENCE statement from AST node
func writeMigrationSequenceFromAST(out *strings.Builder, astNode any) error {
	if astNode == nil {
		return errors.New("AST node is nil")
	}

	// Extract the original SQL text from the AST node
	var sequenceSQL string

	// Try to cast to PostgreSQL CreateseqstmtContext
	if ctx, ok := astNode.(*pgparser.CreateseqstmtContext); ok {
		// First try to get text using token stream
		if tokenStream := ctx.GetParser().GetTokenStream(); tokenStream != nil {
			start := ctx.GetStart()
			stop := ctx.GetStop()
			if start != nil && stop != nil {
				sequenceSQL = tokenStream.GetTextFromTokens(start, stop)
			}
		}

		// Fallback to GetText() if token stream approach failed
		if sequenceSQL == "" {
			sequenceSQL = ctx.GetText()
		}
	} else {
		return errors.Errorf("unsupported AST node type for sequence: %T", astNode)
	}

	if sequenceSQL == "" {
		return errors.New("failed to extract sequence SQL from AST node")
	}

	// Write the sequence SQL
	_, _ = out.WriteString(sequenceSQL)
	// Ensure statement ends with semicolon
	if !strings.HasSuffix(strings.TrimSpace(sequenceSQL), ";") {
		_, _ = out.WriteString(";")
	}
	_, _ = out.WriteString("\n\n")

	return nil
}

// writeMigrationEnumTypeFromAST writes CREATE TYPE AS ENUM statement from AST node
func writeMigrationEnumTypeFromAST(out *strings.Builder, astNode any) error {
	if astNode == nil {
		return errors.New("AST node is nil")
	}

	// Extract the original SQL text from the AST node
	var enumSQL string

	// Try to cast to PostgreSQL DefinestmtContext (CREATE TYPE AS ENUM)
	if ctx, ok := astNode.(*pgparser.DefinestmtContext); ok {
		// First try to get text using token stream
		if tokenStream := ctx.GetParser().GetTokenStream(); tokenStream != nil {
			start := ctx.GetStart()
			stop := ctx.GetStop()
			if start != nil && stop != nil {
				enumSQL = tokenStream.GetTextFromTokens(start, stop)
			}
		}

		// Fallback to GetText() if token stream approach failed
		if enumSQL == "" {
			enumSQL = ctx.GetText()
		}
	} else {
		return errors.Errorf("unsupported AST node type for enum type: %T", astNode)
	}

	if enumSQL == "" {
		return errors.New("failed to extract enum type SQL from AST node")
	}

	// Write the enum type SQL
	_, _ = out.WriteString(enumSQL)
	// Ensure statement ends with semicolon
	if !strings.HasSuffix(strings.TrimSpace(enumSQL), ";") {
		_, _ = out.WriteString(";")
	}
	_, _ = out.WriteString("\n\n")

	return nil
}

// writeMigrationTableFromAST writes CREATE TABLE statement from AST node
func writeMigrationTableFromAST(out *strings.Builder, astNode any) error {
	if astNode == nil {
		return errors.New("AST node is nil")
	}

	// Extract the original SQL text from the AST node
	var tableSQL string

	// Try to cast to PostgreSQL CreatestmtContext
	if ctx, ok := astNode.(*pgparser.CreatestmtContext); ok {
		// First try to get text using token stream
		if tokenStream := ctx.GetParser().GetTokenStream(); tokenStream != nil {
			start := ctx.GetStart()
			stop := ctx.GetStop()
			if start != nil && stop != nil {
				tableSQL = tokenStream.GetTextFromTokens(start, stop)
			}
		}

		// If token stream failed, fall back to GetText() method
		if tableSQL == "" {
			tableSQL = ctx.GetText()
		}
	}

	// If we couldn't extract SQL, return an error
	if tableSQL == "" {
		return errors.New("could not extract table SQL from AST node")
	}

	// Write the table SQL
	_, _ = out.WriteString(tableSQL)
	// Ensure statement ends with semicolon
	if !strings.HasSuffix(strings.TrimSpace(tableSQL), ";") {
		_, _ = out.WriteString(";")
	}
	_, _ = out.WriteString("\n\n")

	return nil
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

func writeDropPrimaryKey(out *strings.Builder, schema, table, constraint string) {
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

func writeDropExtension(out *strings.Builder, extensionName string) {
	_, _ = out.WriteString(`DROP EXTENSION IF EXISTS "`)
	_, _ = out.WriteString(extensionName)
	_, _ = out.WriteString(`";`)
	_, _ = out.WriteString("\n")
}

func writeCreateExtension(out *strings.Builder, extension *storepb.ExtensionMetadata) error {
	_, _ = out.WriteString(`CREATE EXTENSION IF NOT EXISTS "`)
	_, _ = out.WriteString(extension.Name)
	_, _ = out.WriteString(`"`)

	// Add WITH SCHEMA clause if schema is specified
	if extension.Schema != "" {
		_, _ = out.WriteString(` WITH SCHEMA "`)
		_, _ = out.WriteString(extension.Schema)
		_, _ = out.WriteString(`"`)
	}

	// Add VERSION clause if version is specified
	if extension.Version != "" {
		_, _ = out.WriteString(` VERSION '`)
		_, _ = out.WriteString(extension.Version)
		_, _ = out.WriteString(`'`)
	}

	_, _ = out.WriteString(`;`)
	_, _ = out.WriteString("\n")

	// Write description (comment) if present
	if extension.Description != "" {
		writeCommentOnExtension(out, extension.Name, extension.Description)
	}

	return nil
}

func writeMigrationExtensionFromAST(out *strings.Builder, astNode any) error {
	if astNode == nil {
		return errors.New("AST node is nil")
	}

	// Try to cast to PostgreSQL CreateextensionstmtContext
	if ctx, ok := astNode.(*pgparser.CreateextensionstmtContext); ok {
		// Get text using token stream
		if stream := ctx.GetParser().GetTokenStream(); stream != nil {
			text := stream.GetTextFromInterval(antlr.Interval{
				Start: ctx.GetStart().GetTokenIndex(),
				Stop:  ctx.GetStop().GetTokenIndex(),
			})
			_, _ = out.WriteString(text)
			_, _ = out.WriteString(";")
			_, _ = out.WriteString("\n")
			return nil
		}
		return errors.New("token stream not available for extension AST node")
	}

	return errors.Errorf("unexpected AST node type for extension: %T", astNode)
}

func writeDropEventTrigger(out *strings.Builder, eventTriggerName string) {
	_, _ = out.WriteString(`DROP EVENT TRIGGER IF EXISTS "`)
	_, _ = out.WriteString(eventTriggerName)
	_, _ = out.WriteString(`";`)
	_, _ = out.WriteString("\n")
}

func writeCreateEventTrigger(out *strings.Builder, eventTrigger *storepb.EventTriggerMetadata) error {
	// Use the stored definition if available
	if eventTrigger.Definition != "" {
		_, _ = out.WriteString(eventTrigger.Definition)
		_, _ = out.WriteString(";")
		_, _ = out.WriteString("\n")
		return nil
	}

	// Otherwise, build the CREATE EVENT TRIGGER statement
	_, _ = out.WriteString(`CREATE EVENT TRIGGER "`)
	_, _ = out.WriteString(eventTrigger.Name)
	_, _ = out.WriteString(`" ON `)
	_, _ = out.WriteString(eventTrigger.Event)

	// Add WHEN TAG IN clause if tags are specified
	if len(eventTrigger.Tags) > 0 {
		_, _ = out.WriteString("\n  WHEN TAG IN (")
		for i, tag := range eventTrigger.Tags {
			if i > 0 {
				_, _ = out.WriteString(", ")
			}
			_, _ = out.WriteString("'")
			_, _ = out.WriteString(tag)
			_, _ = out.WriteString("'")
		}
		_, _ = out.WriteString(")")
	}

	// Add EXECUTE FUNCTION clause
	_, _ = out.WriteString("\n  EXECUTE FUNCTION ")
	if eventTrigger.FunctionSchema != "" {
		_, _ = out.WriteString(`"`)
		_, _ = out.WriteString(eventTrigger.FunctionSchema)
		_, _ = out.WriteString(`".`)
	}
	_, _ = out.WriteString(`"`)
	_, _ = out.WriteString(eventTrigger.FunctionName)
	_, _ = out.WriteString(`"()`)
	_, _ = out.WriteString(";")
	_, _ = out.WriteString("\n")

	return nil
}

func writeMigrationEventTriggerFromAST(out *strings.Builder, astNode any) error {
	if astNode == nil {
		return errors.New("AST node is nil")
	}

	// Try to cast to PostgreSQL CreateeventtrigstmtContext
	if ctx, ok := astNode.(*pgparser.CreateeventtrigstmtContext); ok {
		// Get text using token stream
		if stream := ctx.GetParser().GetTokenStream(); stream != nil {
			text := stream.GetTextFromInterval(antlr.Interval{
				Start: ctx.GetStart().GetTokenIndex(),
				Stop:  ctx.GetStop().GetTokenIndex(),
			})
			_, _ = out.WriteString(text)
			_, _ = out.WriteString(";")
			_, _ = out.WriteString("\n")
			return nil
		}
		return errors.New("token stream not available for event trigger AST node")
	}

	return errors.Errorf("unexpected AST node type for event trigger: %T", astNode)
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

// requiresExplicitCasting determines if a PostgreSQL type conversion requires a USING clause
func requiresExplicitCasting(oldType, newType string) bool {
	// Types are already normalized by get_database_metadata.go, no need to normalize again

	// Extract base types without parameters
	oldBaseType := extractBaseType(oldType)
	newBaseType := extractBaseType(newType)

	// If same base type, usually no USING clause needed (except for precision reductions)
	if oldBaseType == newBaseType {
		return requiresUsingForSameType(oldType, newType)
	}

	// Define incompatible type conversions that require USING clause
	incompatibleConversions := map[string][]string{
		"text":                        {"integer", "numeric", "real", "double precision", "smallint", "bigint", "boolean"},
		"character varying":           {"integer", "numeric", "real", "double precision", "smallint", "bigint", "boolean"},
		"character":                   {"integer", "numeric", "real", "double precision", "smallint", "bigint", "boolean"},
		"jsonb":                       {"text", "character varying", "character"},
		"json":                        {"text", "character varying", "character"},
		"integer[]":                   {"text", "character varying", "character"},
		"text[]":                      {"text", "character varying", "character"},
		"bigint":                      {"smallint", "integer"},                                       // Potential overflow
		"double precision":            {"real"},                                                      // Precision loss
		"timestamp with time zone":    {"date"},                                                      // Time component loss
		"timestamp without time zone": {"date"},                                                      // Time component loss
		"numeric":                     {"integer", "smallint", "bigint", "real", "double precision"}, // When precision specified
	}

	if targets, exists := incompatibleConversions[oldBaseType]; exists {
		for _, target := range targets {
			if target == newBaseType {
				return true
			}
		}
	}

	return false
}

// extractBaseType removes precision/scale parameters from PostgreSQL types
func extractBaseType(typeName string) string {
	// Handle common patterns: numeric(10,2) -> numeric, varchar(100) -> character varying
	if idx := strings.Index(typeName, "("); idx != -1 {
		return strings.TrimSpace(typeName[:idx])
	}
	return typeName
}

// requiresUsingForSameType checks if same base type requires USING (e.g., precision reductions)
func requiresUsingForSameType(oldType, newType string) bool {
	oldBase := extractBaseType(oldType)
	newBase := extractBaseType(newType)

	if oldBase != newBase {
		return false
	}

	// For character varying, check if reducing length significantly
	if strings.Contains(oldType, "character varying") {
		oldLen := extractLength(oldType)
		newLen := extractLength(newType)
		if oldLen > 0 && newLen > 0 && oldLen > newLen*2 {
			return true // Significant reduction might truncate data
		}
	}

	return false
}

// extractLength extracts length parameter from varchar(n) or char(n)
func extractLength(typeName string) int {
	start := strings.Index(typeName, "(")
	end := strings.Index(typeName, ")")
	if start == -1 || end == -1 || end <= start {
		return 0
	}

	lengthStr := strings.TrimSpace(typeName[start+1 : end])
	if length, err := strconv.Atoi(lengthStr); err == nil {
		return length
	}
	return 0
}

func writeAlterColumnType(out *strings.Builder, schema, table, column, oldType, newType string) {
	_, _ = out.WriteString(`ALTER TABLE "`)
	_, _ = out.WriteString(schema)
	_, _ = out.WriteString(`"."`)
	_, _ = out.WriteString(table)
	_, _ = out.WriteString(`" ALTER COLUMN "`)
	_, _ = out.WriteString(column)
	_, _ = out.WriteString(`" TYPE `)
	_, _ = out.WriteString(newType)

	// Add USING clause for incompatible type conversions
	if requiresExplicitCasting(oldType, newType) {
		_, _ = out.WriteString(` USING "`)
		_, _ = out.WriteString(column)
		_, _ = out.WriteString(`"::`)
		_, _ = out.WriteString(newType)
	}

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

// writeAddColumnFromAST writes ALTER TABLE ADD COLUMN statement from AST node
func writeAddColumnFromAST(out *strings.Builder, schema, table string, columnASTNode pgparser.IColumnDefContext) error {
	if columnASTNode == nil {
		return errors.New("Column AST node is nil")
	}

	columnDefinition := extractColumnDefinitionFromAST(columnASTNode)
	if columnDefinition == "" {
		return errors.New("could not extract column definition from AST node")
	}

	_, _ = out.WriteString(`ALTER TABLE "`)
	_, _ = out.WriteString(schema)
	_, _ = out.WriteString(`"."`)
	_, _ = out.WriteString(table)
	_, _ = out.WriteString(`" ADD COLUMN `)
	_, _ = out.WriteString(columnDefinition)
	_, _ = out.WriteString(";\n")

	return nil
}

// writeDropColumnFromAST writes ALTER TABLE DROP COLUMN statement from AST node
func writeDropColumnFromAST(out *strings.Builder, schema, table string, columnASTNode pgparser.IColumnDefContext) error {
	if columnASTNode == nil {
		return errors.New("Column AST node is nil")
	}

	columnName := extractColumnNameFromAST(columnASTNode)
	if columnName == "" {
		return errors.New("could not extract column name from AST node")
	}

	_, _ = out.WriteString(`ALTER TABLE "`)
	_, _ = out.WriteString(schema)
	_, _ = out.WriteString(`"."`)
	_, _ = out.WriteString(table)
	_, _ = out.WriteString(`" DROP COLUMN IF EXISTS "`)
	_, _ = out.WriteString(columnName)
	_, _ = out.WriteString(`";`)
	_, _ = out.WriteString("\n")

	return nil
}

// writeAlterColumnFromAST writes ALTER TABLE ALTER COLUMN statement from AST nodes
func writeAlterColumnFromAST(out *strings.Builder, schema, table string, oldColumnASTNode, newColumnASTNode pgparser.IColumnDefContext) error {
	if oldColumnASTNode == nil || newColumnASTNode == nil {
		return errors.New("Column AST nodes are nil")
	}

	// Extract column metadata from both AST nodes (without normalization except for column name)
	oldMetadata := extractColumnMetadataFromAST(oldColumnASTNode)
	newMetadata := extractColumnMetadataFromAST(newColumnASTNode)

	if oldMetadata == nil || newMetadata == nil {
		return errors.New("could not extract column metadata from AST nodes")
	}

	if oldMetadata.Name == "" {
		return errors.New("could not extract column name from AST node")
	}

	// Check for type changes
	if oldMetadata.Type != newMetadata.Type {
		_, _ = out.WriteString(`ALTER TABLE "`)
		_, _ = out.WriteString(schema)
		_, _ = out.WriteString(`"."`)
		_, _ = out.WriteString(table)
		_, _ = out.WriteString(`" ALTER COLUMN "`)
		_, _ = out.WriteString(oldMetadata.Name)
		_, _ = out.WriteString(`" TYPE `)
		_, _ = out.WriteString(newMetadata.Type)
		_, _ = out.WriteString(";\n")
	}

	// Check for nullable changes
	if oldMetadata.Nullable != newMetadata.Nullable {
		_, _ = out.WriteString(`ALTER TABLE "`)
		_, _ = out.WriteString(schema)
		_, _ = out.WriteString(`"."`)
		_, _ = out.WriteString(table)
		_, _ = out.WriteString(`" ALTER COLUMN "`)
		_, _ = out.WriteString(oldMetadata.Name)
		if newMetadata.Nullable {
			_, _ = out.WriteString(`" DROP NOT NULL`)
		} else {
			_, _ = out.WriteString(`" SET NOT NULL`)
		}
		_, _ = out.WriteString(";\n")
	}

	// Check for default value changes
	if oldMetadata.Default != newMetadata.Default {
		_, _ = out.WriteString(`ALTER TABLE "`)
		_, _ = out.WriteString(schema)
		_, _ = out.WriteString(`"."`)
		_, _ = out.WriteString(table)
		_, _ = out.WriteString(`" ALTER COLUMN "`)
		_, _ = out.WriteString(oldMetadata.Name)
		if newMetadata.Default == "" {
			_, _ = out.WriteString(`" DROP DEFAULT`)
		} else {
			_, _ = out.WriteString(`" SET DEFAULT `)
			_, _ = out.WriteString(newMetadata.Default)
		}
		_, _ = out.WriteString(";\n")
	}

	return nil
}

func writeAddCheckConstraint(out *strings.Builder, schema, table string, check *storepb.CheckConstraintMetadata) {
	_, _ = out.WriteString(`ALTER TABLE "`)
	_, _ = out.WriteString(schema)
	_, _ = out.WriteString(`"."`)
	_, _ = out.WriteString(table)
	_, _ = out.WriteString(`" ADD CONSTRAINT "`)
	_, _ = out.WriteString(check.Name)
	_, _ = out.WriteString(`" CHECK (`)
	_, _ = out.WriteString(check.Expression)
	_, _ = out.WriteString(`);`)
	_, _ = out.WriteString("\n")
}

func writeFunctionDiff(out *strings.Builder, funcDiff *schema.FunctionDiff) error {
	switch funcDiff.Action {
	case schema.MetadataDiffActionCreate:
		// CREATE new function - support both metadata and AST-only modes
		if funcDiff.NewFunction != nil {
			// Metadata mode: use function definition
			_, _ = out.WriteString(funcDiff.NewFunction.Definition)
			if !strings.HasSuffix(strings.TrimSpace(funcDiff.NewFunction.Definition), ";") {
				_, _ = out.WriteString(";")
			}
			_, _ = out.WriteString("\n\n")
		} else if funcDiff.NewASTNode != nil {
			// AST-only mode: extract SQL from AST node
			if err := writeMigrationFunctionFromAST(out, funcDiff.NewASTNode); err != nil {
				return err
			}
		} else {
			return errors.Errorf("function diff for %s.%s has neither metadata nor AST node", funcDiff.SchemaName, funcDiff.FunctionName)
		}

	case schema.MetadataDiffActionAlter:
		// ALTER function using CREATE OR REPLACE
		// The decision to use ALTER vs DROP/CREATE was already made in differ.go
		// If we reach here, it means we can safely use CREATE OR REPLACE
		if funcDiff.NewFunction != nil {
			// Metadata mode: use ANTLR parser to safely convert CREATE FUNCTION to CREATE OR REPLACE FUNCTION
			definition := convertToCreateOrReplace(funcDiff.NewFunction.Definition)

			_, _ = out.WriteString(definition)
			if !strings.HasSuffix(strings.TrimSpace(definition), ";") {
				_, _ = out.WriteString(";")
			}
			_, _ = out.WriteString("\n\n")
		} else if funcDiff.NewASTNode != nil {
			// AST-only mode: extract SQL and convert to CREATE OR REPLACE
			if err := writeMigrationFunctionFromASTWithReplace(out, funcDiff.NewASTNode); err != nil {
				return err
			}
		} else {
			return errors.Errorf("function diff for %s.%s has neither metadata nor AST node", funcDiff.SchemaName, funcDiff.FunctionName)
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

// buildTempMetadataForCreate builds a temporary DatabaseSchemaMetadata containing
// the tables and views that are being created or altered. This allows GetQuerySpan to
// find these objects when extracting view dependencies in AST-only mode.
func buildTempMetadataForCreate(diff *schema.MetadataDiff) *storepb.DatabaseSchemaMetadata {
	// Group objects by schema
	schemaMap := make(map[string]*storepb.SchemaMetadata)

	// Collect tables being created or altered
	for _, tableDiff := range diff.TableChanges {
		if tableDiff.Action == schema.MetadataDiffActionCreate || tableDiff.Action == schema.MetadataDiffActionAlter {
			schemaName := tableDiff.SchemaName
			if schemaMap[schemaName] == nil {
				schemaMap[schemaName] = &storepb.SchemaMetadata{
					Name: schemaName,
				}
			}
			// Add table with just the name - that's all GetQuerySpan needs
			schemaMap[schemaName].Tables = append(schemaMap[schemaName].Tables, &storepb.TableMetadata{
				Name: tableDiff.TableName,
			})
		}
	}

	// Collect views being created or altered
	for _, viewDiff := range diff.ViewChanges {
		if viewDiff.Action == schema.MetadataDiffActionCreate || viewDiff.Action == schema.MetadataDiffActionAlter {
			schemaName := viewDiff.SchemaName
			if schemaMap[schemaName] == nil {
				schemaMap[schemaName] = &storepb.SchemaMetadata{
					Name: schemaName,
				}
			}
			// Add view with just the name
			schemaMap[schemaName].Views = append(schemaMap[schemaName].Views, &storepb.ViewMetadata{
				Name: viewDiff.ViewName,
			})
		}
	}

	// Convert map to slice
	var schemas []*storepb.SchemaMetadata
	for _, schemaMeta := range schemaMap {
		schemas = append(schemas, schemaMeta)
	}

	return &storepb.DatabaseSchemaMetadata{
		Schemas: schemas,
	}
}

// buildTempMetadataForDrop builds a temporary DatabaseSchemaMetadata containing only
// the tables and views that are being dropped. This allows GetQuerySpan to find these
// objects when extracting view dependencies in AST-only mode.
func buildTempMetadataForDrop(diff *schema.MetadataDiff) *storepb.DatabaseSchemaMetadata {
	// Group objects by schema
	schemaMap := make(map[string]*storepb.SchemaMetadata)

	// Collect tables being dropped
	for _, tableDiff := range diff.TableChanges {
		if tableDiff.Action == schema.MetadataDiffActionDrop {
			schemaName := tableDiff.SchemaName
			if schemaMap[schemaName] == nil {
				schemaMap[schemaName] = &storepb.SchemaMetadata{
					Name: schemaName,
				}
			}
			// Add table with just the name - that's all GetQuerySpan needs
			schemaMap[schemaName].Tables = append(schemaMap[schemaName].Tables, &storepb.TableMetadata{
				Name: tableDiff.TableName,
			})
		}
	}

	// Collect views being dropped
	for _, viewDiff := range diff.ViewChanges {
		if viewDiff.Action == schema.MetadataDiffActionDrop || viewDiff.Action == schema.MetadataDiffActionAlter {
			schemaName := viewDiff.SchemaName
			if schemaMap[schemaName] == nil {
				schemaMap[schemaName] = &storepb.SchemaMetadata{
					Name: schemaName,
				}
			}
			// Add view with just the name
			schemaMap[schemaName].Views = append(schemaMap[schemaName].Views, &storepb.ViewMetadata{
				Name: viewDiff.ViewName,
			})
		}
	}

	// Convert map to slice
	var schemas []*storepb.SchemaMetadata
	for _, schemaMeta := range schemaMap {
		schemas = append(schemas, schemaMeta)
	}

	return &storepb.DatabaseSchemaMetadata{
		Schemas: schemas,
	}
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

// writeView writes a CREATE VIEW statement from AST node or metadata
func writeMigrationViewFromAST(out *strings.Builder, astNode any) error {
	if astNode == nil {
		return errors.New("AST node is nil")
	}

	// Extract the original SQL text from the AST node
	var viewSQL string

	// Try to cast to PostgreSQL ViewstmtContext first
	if ctx, ok := astNode.(*pgparser.ViewstmtContext); ok {
		// First try to get text using token stream
		if tokenStream := ctx.GetParser().GetTokenStream(); tokenStream != nil {
			start := ctx.GetStart()
			stop := ctx.GetStop()
			if start != nil && stop != nil {
				viewSQL = tokenStream.GetTextFromTokens(start, stop)
			}
		}

		// Fallback to GetText() if token stream approach failed
		if viewSQL == "" {
			viewSQL = ctx.GetText()
		}
	} else {
		// Generic fallback - try to get text using token approach first
		if tree, ok := astNode.(antlr.ParseTree); ok {
			viewSQL = getTextFromAST(tree)
		}
	}

	if viewSQL != "" {
		_, _ = out.WriteString(viewSQL)
		if !strings.HasSuffix(strings.TrimSpace(viewSQL), ";") {
			_, _ = out.WriteString(`;`)
		}
		_, _ = out.WriteString("\n\n")
		return nil
	}

	return errors.New("failed to extract SQL from AST node")
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

// writeMigrationFunctionFromAST writes a CREATE FUNCTION statement from AST node
func writeMigrationFunctionFromAST(out *strings.Builder, astNode any) error {
	if astNode == nil {
		return errors.New("AST node is nil")
	}

	// Extract the original SQL text from the AST node
	var functionSQL string

	// Try to cast to PostgreSQL CreatefunctionstmtContext first
	if ctx, ok := astNode.(*pgparser.CreatefunctionstmtContext); ok {
		// First try to get text using token stream
		if tokenStream := ctx.GetParser().GetTokenStream(); tokenStream != nil {
			start := ctx.GetStart()
			stop := ctx.GetStop()
			if start != nil && stop != nil {
				functionSQL = tokenStream.GetTextFromTokens(start, stop)
			}
		}

		// Fallback to GetText() if token stream approach failed
		if functionSQL == "" {
			functionSQL = ctx.GetText()
		}
	} else {
		return errors.Errorf("unsupported AST node type for function: %T", astNode)
	}

	if functionSQL == "" {
		return errors.New("could not extract function SQL from AST node")
	}

	// Write the function SQL
	_, _ = out.WriteString(functionSQL)
	if !strings.HasSuffix(strings.TrimSpace(functionSQL), ";") {
		_, _ = out.WriteString(";")
	}
	_, _ = out.WriteString("\n\n")

	return nil
}

// writeMigrationFunctionFromASTWithReplace writes a CREATE OR REPLACE FUNCTION statement from AST node
func writeMigrationFunctionFromASTWithReplace(out *strings.Builder, astNode any) error {
	if astNode == nil {
		return errors.New("AST node is nil")
	}

	// Extract the original SQL text from the AST node
	var functionSQL string

	// Try to cast to PostgreSQL CreatefunctionstmtContext first
	if ctx, ok := astNode.(*pgparser.CreatefunctionstmtContext); ok {
		// First try to get text using token stream
		if tokenStream := ctx.GetParser().GetTokenStream(); tokenStream != nil {
			start := ctx.GetStart()
			stop := ctx.GetStop()
			if start != nil && stop != nil {
				functionSQL = tokenStream.GetTextFromTokens(start, stop)
			}
		}

		// Fallback to GetText() if token stream approach failed
		if functionSQL == "" {
			functionSQL = ctx.GetText()
		}
	} else {
		return errors.Errorf("unsupported AST node type for function: %T", astNode)
	}

	if functionSQL == "" {
		return errors.New("could not extract function SQL from AST node")
	}

	// Convert CREATE FUNCTION to CREATE OR REPLACE FUNCTION
	functionSQL = convertToCreateOrReplace(functionSQL)

	// Write the function SQL
	_, _ = out.WriteString(functionSQL)
	if !strings.HasSuffix(strings.TrimSpace(functionSQL), ";") {
		_, _ = out.WriteString(";")
	}
	_, _ = out.WriteString("\n\n")

	return nil
}

// writeMigrationMaterializedViewFromAST writes a CREATE MATERIALIZED VIEW statement from AST node
func writeMigrationMaterializedViewFromAST(out *strings.Builder, astNode any) error {
	if astNode == nil {
		return errors.New("AST node is nil")
	}

	// Extract the original SQL text from the AST node
	var mvSQL string

	// Try to cast to PostgreSQL CreatematviewstmtContext first
	if ctx, ok := astNode.(*pgparser.CreatematviewstmtContext); ok {
		// First try to get text using token stream
		if tokenStream := ctx.GetParser().GetTokenStream(); tokenStream != nil {
			start := ctx.GetStart()
			stop := ctx.GetStop()
			if start != nil && stop != nil {
				mvSQL = tokenStream.GetTextFromTokens(start, stop)
			}
		}

		// Fallback to GetText() if token stream approach failed
		if mvSQL == "" {
			mvSQL = ctx.GetText()
		}
	} else {
		// Generic fallback - try to get text using token approach first
		if tree, ok := astNode.(antlr.ParseTree); ok {
			mvSQL = getTextFromAST(tree)
		}
	}

	if mvSQL != "" {
		_, _ = out.WriteString(mvSQL)
		if !strings.HasSuffix(strings.TrimSpace(mvSQL), ";") {
			_, _ = out.WriteString(`;`)
		}
		_, _ = out.WriteString("\n\n")
		return nil
	}

	return errors.New("failed to extract SQL from materialized view AST node")
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
func writeMigrationMaterializedViewIndex(out *strings.Builder, schema, mvName string, index *storepb.IndexMetadata) {
	if index == nil {
		return
	}

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
	_, _ = out.WriteString(mvName)
	_, _ = out.WriteString(`" `)

	if index.Type != "" && index.Type != "BTREE" && index.Type != "btree" {
		_, _ = out.WriteString(`USING `)
		_, _ = out.WriteString(strings.ToUpper(index.Type))
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
				writeCommentOnFunction(buf, funcDiff.SchemaName, funcDiff.NewFunction.Signature, newComment, funcDiff.NewASTNode, funcDiff.NewFunction.Definition)
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

// generateSequenceOwnershipChanges generates ALTER SEQUENCE OWNED BY statements for sequence ownership changes
func generateSequenceOwnershipChanges(buf *strings.Builder, diff *schema.MetadataDiff) error {
	for _, seqDiff := range diff.SequenceChanges {
		if seqDiff.Action == schema.MetadataDiffActionAlter {
			// Handle ownership changes using AST nodes
			if seqDiff.NewASTNode != nil {
				// Adding or modifying ownership - extract SQL from NewASTNode
				if err := writeMigrationAlterSequenceFromAST(buf, seqDiff.NewASTNode); err != nil {
					return errors.Wrapf(err, "failed to generate ALTER SEQUENCE for %s.%s",
						seqDiff.SchemaName, seqDiff.SequenceName)
				}
			} else if seqDiff.OldASTNode != nil {
				// Removing ownership - generate ALTER SEQUENCE ... OWNED BY NONE
				if err := writeMigrationRemoveSequenceOwnership(buf, seqDiff.SchemaName, seqDiff.SequenceName); err != nil {
					return errors.Wrapf(err, "failed to remove ownership for %s.%s",
						seqDiff.SchemaName, seqDiff.SequenceName)
				}
			}
		}
	}
	return nil
}

// writeMigrationAlterSequenceFromAST writes ALTER SEQUENCE statement from AST node
func writeMigrationAlterSequenceFromAST(out *strings.Builder, astNode any) error {
	if astNode == nil {
		return errors.New("AST node is nil")
	}

	// Extract the original SQL text from the AST node
	var alterSQL string

	// Try to cast to PostgreSQL AlterseqstmtContext
	if ctx, ok := astNode.(*pgparser.AlterseqstmtContext); ok {
		// First try to get text using token stream
		if tokenStream := ctx.GetParser().GetTokenStream(); tokenStream != nil {
			start := ctx.GetStart()
			stop := ctx.GetStop()
			if start != nil && stop != nil {
				alterSQL = tokenStream.GetTextFromTokens(start, stop)
			}
		}

		// Fallback to GetText() if token stream approach failed
		if alterSQL == "" {
			alterSQL = ctx.GetText()
		}
	} else {
		return errors.Errorf("unsupported AST node type for ALTER SEQUENCE: %T", astNode)
	}

	if alterSQL == "" {
		return errors.New("failed to extract ALTER SEQUENCE SQL from AST node")
	}

	// Write the ALTER SQL
	_, _ = out.WriteString(alterSQL)
	// Ensure statement ends with semicolon
	if !strings.HasSuffix(strings.TrimSpace(alterSQL), ";") {
		_, _ = out.WriteString(";")
	}
	_, _ = out.WriteString("\n\n")

	return nil
}

// writeMigrationRemoveSequenceOwnership writes ALTER SEQUENCE OWNED BY NONE statement
//
//nolint:unparam
func writeMigrationRemoveSequenceOwnership(out *strings.Builder, schema, sequence string) error {
	_, _ = out.WriteString(`ALTER SEQUENCE "`)
	_, _ = out.WriteString(schema)
	_, _ = out.WriteString(`"."`)
	_, _ = out.WriteString(sequence)
	_, _ = out.WriteString(`" OWNED BY NONE;`)
	_, _ = out.WriteString("\n\n")
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

// generateMaterializedViewIndexCommentChanges generates COMMENT ON INDEX statements for index comment changes within materialized view diffs
func generateMaterializedViewIndexCommentChanges(buf *strings.Builder, mvDiff *schema.MaterializedViewDiff) error {
	for _, indexDiff := range mvDiff.IndexChanges {
		if indexDiff.Action == schema.MetadataDiffActionAlter {
			if indexDiff.OldIndex == nil || indexDiff.NewIndex == nil {
				continue
			}

			oldComment := indexDiff.OldIndex.Comment
			newComment := indexDiff.NewIndex.Comment

			// If comments are different, generate COMMENT ON INDEX statement
			if oldComment != newComment {
				writeCommentOnIndex(buf, mvDiff.SchemaName, indexDiff.NewIndex.Name, newComment)
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

// writeCommentOnFunction writes a COMMENT ON FUNCTION/PROCEDURE statement
func writeCommentOnFunction(out *strings.Builder, schema, signature, comment string, astNode any, definition string) {
	// Determine if this is a PROCEDURE or FUNCTION using AST-based detection
	objectType := "FUNCTION"
	if isFunctionProcedure(astNode, definition) {
		objectType = "PROCEDURE"
	}

	_, _ = out.WriteString(`COMMENT ON `)
	_, _ = out.WriteString(objectType)
	_, _ = out.WriteString(` "`)
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

func writeCommentOnExtension(out *strings.Builder, extensionName, comment string) {
	_, _ = out.WriteString(`COMMENT ON EXTENSION "`)
	_, _ = out.WriteString(extensionName)
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

func writeCommentOnEventTrigger(out *strings.Builder, eventTriggerName, comment string) {
	_, _ = out.WriteString(`COMMENT ON EVENT TRIGGER "`)
	_, _ = out.WriteString(eventTriggerName)
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

func writeCommentOnTrigger(out *strings.Builder, schemaName, tableName, triggerName, comment string) {
	// Parse table name to get schema and table separately
	parts := strings.Split(tableName, ".")
	var tableSchemaName, tableNameOnly string
	if len(parts) == 2 {
		tableSchemaName, tableNameOnly = parts[0], parts[1]
	} else {
		tableSchemaName, tableNameOnly = schemaName, tableName
	}

	_, _ = out.WriteString(`COMMENT ON TRIGGER "`)
	_, _ = out.WriteString(triggerName)
	_, _ = out.WriteString(`" ON "`)
	_, _ = out.WriteString(tableSchemaName)
	_, _ = out.WriteString(`"."`)
	_, _ = out.WriteString(tableNameOnly)
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

// generateCommentChangesFromSDL generates COMMENT ON statements from SDL-based comment diffs
// This handles comment changes detected from SDL diff mode (AST-based)
func generateCommentChangesFromSDL(buf *strings.Builder, diff *schema.MetadataDiff) error {
	if len(diff.CommentChanges) == 0 {
		return nil
	}

	// Build a set of tables being dropped
	// When a table is dropped, its comments are automatically removed
	tablesBeingDropped := make(map[string]bool)
	for _, tableDiff := range diff.TableChanges {
		if tableDiff.Action == schema.MetadataDiffActionDrop {
			tableKey := tableDiff.SchemaName + "." + tableDiff.TableName
			tablesBeingDropped[tableKey] = true
		}
	}

	// Build a set of columns being dropped
	// When a column is dropped, its comment is automatically removed
	columnsBeingDropped := make(map[string]bool)
	for _, tableDiff := range diff.TableChanges {
		if tableDiff.Action == schema.MetadataDiffActionAlter {
			for _, colDiff := range tableDiff.ColumnChanges {
				if colDiff.Action == schema.MetadataDiffActionDrop {
					var columnName string
					if colDiff.OldColumn != nil {
						columnName = colDiff.OldColumn.Name
					} else if colDiff.OldASTNode != nil {
						columnName = extractColumnNameFromAST(colDiff.OldASTNode)
					}
					if columnName != "" {
						columnKey := tableDiff.SchemaName + "." + tableDiff.TableName + "." + columnName
						columnsBeingDropped[columnKey] = true
					}
				}
			}
		}
	}

	for _, commentDiff := range diff.CommentChanges {
		// Extract the new comment text from the AST node
		newComment := extractCommentFromDiff(commentDiff)

		switch commentDiff.ObjectType {
		case schema.CommentObjectTypeSchema:
			writeCommentOnSchema(buf, commentDiff.SchemaName, newComment)

		case schema.CommentObjectTypeTable:
			// Skip comment generation for tables being dropped
			tableKey := commentDiff.SchemaName + "." + commentDiff.ObjectName
			if tablesBeingDropped[tableKey] {
				continue
			}
			writeCommentOnTable(buf, commentDiff.SchemaName, commentDiff.ObjectName, newComment)

		case schema.CommentObjectTypeColumn:
			// Skip comment generation for columns being dropped
			columnKey := commentDiff.SchemaName + "." + commentDiff.ObjectName + "." + commentDiff.ColumnName
			if columnsBeingDropped[columnKey] {
				continue
			}
			writeCommentOnColumn(buf, commentDiff.SchemaName, commentDiff.ObjectName, commentDiff.ColumnName, newComment)

		case schema.CommentObjectTypeView:
			writeCommentOnView(buf, commentDiff.SchemaName, commentDiff.ObjectName, newComment)

		case schema.CommentObjectTypeMaterializedView:
			writeCommentOnMaterializedView(buf, commentDiff.SchemaName, commentDiff.ObjectName, newComment)

		case schema.CommentObjectTypeFunction:
			// For functions, ObjectName contains the function signature
			// Try to find the function definition and AST node to determine if it's a FUNCTION or PROCEDURE
			var functionDefinition string
			var functionASTNode any
			functionKey := commentDiff.SchemaName + "." + commentDiff.ObjectName
			for _, funcDiff := range diff.FunctionChanges {
				// Check both NewFunction and OldFunction to handle comment removal cases
				if funcDiff.NewFunction != nil {
					funcKey := funcDiff.SchemaName + "." + funcDiff.NewFunction.Signature
					if funcKey == functionKey {
						functionDefinition = funcDiff.NewFunction.Definition
						functionASTNode = funcDiff.NewASTNode
						break
					}
				}
				if funcDiff.OldFunction != nil {
					funcKey := funcDiff.SchemaName + "." + funcDiff.OldFunction.Signature
					if funcKey == functionKey {
						// Only use OldFunction if we haven't found NewFunction
						if functionDefinition == "" {
							functionDefinition = funcDiff.OldFunction.Definition
							functionASTNode = funcDiff.OldASTNode
						}
						break
					}
				}
			}
			// If we didn't find the function AST node from FunctionChanges, use the comment AST node
			// to determine if it's a FUNCTION or PROCEDURE
			// Check both NewASTNode (for adding comments) and OldASTNode (for removing comments)
			if functionASTNode == nil {
				if commentDiff.NewASTNode != nil {
					functionASTNode = commentDiff.NewASTNode
				} else if commentDiff.OldASTNode != nil {
					functionASTNode = commentDiff.OldASTNode
				}
			}
			writeCommentOnFunction(buf, commentDiff.SchemaName, commentDiff.ObjectName, newComment, functionASTNode, functionDefinition)

		case schema.CommentObjectTypeSequence:
			writeCommentOnSequence(buf, commentDiff.SchemaName, commentDiff.ObjectName, newComment)

		case schema.CommentObjectTypeIndex:
			// Index name is stored in IndexName field
			indexName := commentDiff.IndexName
			if indexName == "" {
				// Fallback to ObjectName if IndexName is not set
				indexName = commentDiff.ObjectName
			}
			writeCommentOnIndex(buf, commentDiff.SchemaName, indexName, newComment)

		case schema.CommentObjectTypeType:
			// COMMENT ON TYPE (for enum types)
			writeCommentOnType(buf, commentDiff.SchemaName, commentDiff.ObjectName, newComment)

		case schema.CommentObjectTypeExtension:
			// COMMENT ON EXTENSION
			writeCommentOnExtension(buf, commentDiff.ObjectName, newComment)

		case schema.CommentObjectTypeEventTrigger:
			// COMMENT ON EVENT TRIGGER
			writeCommentOnEventTrigger(buf, commentDiff.ObjectName, newComment)

		case schema.CommentObjectTypeTrigger:
			// COMMENT ON TRIGGER trigger_name ON table_name
			writeCommentOnTrigger(buf, commentDiff.SchemaName, commentDiff.TableName, commentDiff.ObjectName, newComment)

		default:
			// Unknown object type, skip
			continue
		}
	}

	return nil
}

// extractCommentFromDiff extracts the comment text from a CommentDiff
// It uses the NewComment field which contains the extracted comment text
func extractCommentFromDiff(commentDiff *schema.CommentDiff) string {
	return commentDiff.NewComment
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

	// Look for the FUNCTION/PROCEDURE keyword after CREATE
	functionToken := v.findFunctionOrProcedureToken(ctx)
	if functionToken == nil {
		return nil
	}

	// Insert "OR REPLACE" between CREATE and FUNCTION/PROCEDURE
	// We insert it right before the FUNCTION token
	v.rewriter.InsertBefore("", functionToken.GetTokenIndex(), "OR REPLACE ")

	return nil
}

// extractColumnNameFromAST extracts column name from column definition AST node
func extractColumnNameFromAST(columnASTNode pgparser.IColumnDefContext) string {
	if columnASTNode == nil {
		return ""
	}

	// Use the proper normalization function for PostgreSQL column identifiers
	if columnASTNode.Colid() != nil {
		return pgpluginparser.NormalizePostgreSQLColid(columnASTNode.Colid())
	}

	return ""
}

// extractColumnDefinitionFromAST extracts the full column definition from AST node
func extractColumnDefinitionFromAST(columnASTNode pgparser.IColumnDefContext) string {
	if columnASTNode == nil {
		return ""
	}

	// Use the getTextFromAST helper to extract text properly
	return getTextFromAST(columnASTNode)
}

// findFunctionOrProcedureToken finds the FUNCTION or PROCEDURE token in the CREATE FUNCTION/PROCEDURE statement
func (v *CreateOrReplaceVisitor) findFunctionOrProcedureToken(ctx *pgparser.CreatefunctionstmtContext) antlr.Token {
	// Get all tokens in the context range
	start := ctx.GetStart().GetTokenIndex()
	stop := ctx.GetStop().GetTokenIndex()

	for i := start; i <= stop; i++ {
		token := v.tokens.Get(i)
		// Check for both FUNCTION and PROCEDURE tokens since they share the same grammar rule
		if token.GetTokenType() == pgparser.PostgreSQLParserFUNCTION ||
			token.GetTokenType() == pgparser.PostgreSQLParserPROCEDURE {
			return token
		}
	}

	return nil
}

// hasOrReplace checks if the CREATE FUNCTION/PROCEDURE statement already contains "OR REPLACE"
func (v *CreateOrReplaceVisitor) hasOrReplace(ctx *pgparser.CreatefunctionstmtContext) bool {
	// Get all tokens in the context range between CREATE and FUNCTION/PROCEDURE
	start := ctx.GetStart().GetTokenIndex()
	functionToken := v.findFunctionOrProcedureToken(ctx)
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

// getViewDependenciesFromAST extracts view dependencies from AST node using lightweight access table extraction.
// This avoids the complexity and potential panics of GetQuerySpan by only extracting table/view references.
// The caller is responsible for filtering dependencies against the set of objects being migrated.
func getViewDependenciesFromAST(astNode any, schemaName string, _ *storepb.DatabaseSchemaMetadata) []*storepb.DependencyColumn {
	if astNode == nil {
		return []*storepb.DependencyColumn{}
	}

	var selectStatement string

	if ctx, ok := astNode.(*pgparser.ViewstmtContext); ok {
		if ctx.Selectstmt() != nil {
			// Try to get text using token stream first
			if tokenStream := ctx.GetParser().GetTokenStream(); tokenStream != nil {
				start := ctx.Selectstmt().GetStart()
				stop := ctx.Selectstmt().GetStop()
				if start != nil && stop != nil {
					selectStatement = tokenStream.GetTextFromTokens(start, stop)
				}
			}

			// Fallback to token-based approach if failed
			if selectStatement == "" {
				selectStatement = getTextFromAST(ctx.Selectstmt())
			}
		}
	}

	if selectStatement == "" {
		return []*storepb.DependencyColumn{}
	}

	queryStatement := strings.TrimSpace(selectStatement)

	accessTables, err := pgpluginparser.ExtractAccessTables(queryStatement, pgpluginparser.ExtractAccessTablesOption{
		DefaultDatabase:        "",
		DefaultSchema:          schemaName,
		SkipMetadataValidation: true,
	})
	if err != nil {
		return []*storepb.DependencyColumn{}
	}

	// The caller will filter against allObjects
	dependencyMap := make(map[string]*storepb.DependencyColumn)
	for _, resource := range accessTables {
		if resource.Schema == "pg_catalog" || resource.Schema == "information_schema" {
			continue
		}

		resourceSchema := resource.Schema
		if resourceSchema == "" {
			resourceSchema = schemaName
		}

		key := fmt.Sprintf("%s.%s", resourceSchema, resource.Table)
		if _, exists := dependencyMap[key]; !exists {
			dependencyMap[key] = &storepb.DependencyColumn{
				Schema: resourceSchema,
				Table:  resource.Table,
				Column: "*", // Table-level dependencies
			}
		}
	}

	var dependencies []*storepb.DependencyColumn
	for _, dep := range dependencyMap {
		dependencies = append(dependencies, dep)
	}

	return dependencies
}

// getMaterializedViewDependenciesFromAST extracts table/view dependencies from a materialized view's AST node
func getMaterializedViewDependenciesFromAST(astNode any, schemaName string, _ *storepb.DatabaseSchemaMetadata) []*storepb.DependencyColumn {
	if astNode == nil {
		return []*storepb.DependencyColumn{}
	}

	var selectStatement string

	if ctx, ok := astNode.(*pgparser.CreatematviewstmtContext); ok {
		if ctx.Selectstmt() != nil {
			// Try to get text using token stream first
			if tokenStream := ctx.GetParser().GetTokenStream(); tokenStream != nil {
				start := ctx.Selectstmt().GetStart()
				stop := ctx.Selectstmt().GetStop()
				if start != nil && stop != nil {
					selectStatement = tokenStream.GetTextFromTokens(start, stop)
				}
			}

			// Fallback to token-based approach if failed
			if selectStatement == "" {
				selectStatement = getTextFromAST(ctx.Selectstmt())
			}
		}
	}

	if selectStatement == "" {
		return []*storepb.DependencyColumn{}
	}

	queryStatement := strings.TrimSpace(selectStatement)

	accessTables, err := pgpluginparser.ExtractAccessTables(queryStatement, pgpluginparser.ExtractAccessTablesOption{
		DefaultDatabase:        "",
		DefaultSchema:          schemaName,
		SkipMetadataValidation: true,
	})
	if err != nil {
		return []*storepb.DependencyColumn{}
	}

	// The caller will filter against allObjects
	dependencyMap := make(map[string]*storepb.DependencyColumn)
	for _, resource := range accessTables {
		if resource.Schema == "pg_catalog" || resource.Schema == "information_schema" {
			continue
		}

		resourceSchema := resource.Schema
		if resourceSchema == "" {
			resourceSchema = schemaName
		}

		key := fmt.Sprintf("%s.%s", resourceSchema, resource.Table)
		if _, exists := dependencyMap[key]; !exists {
			dependencyMap[key] = &storepb.DependencyColumn{
				Schema: resourceSchema,
				Table:  resource.Table,
				Column: "*", // Table-level dependencies
			}
		}
	}

	var dependencies []*storepb.DependencyColumn
	for _, dep := range dependencyMap {
		dependencies = append(dependencies, dep)
	}

	return dependencies
}

// extractColumnMetadataFromAST extracts column metadata from AST node without normalization (except column name)
// This is used in AST-only mode where we need the raw values for comparison
func extractColumnMetadataFromAST(columnDef pgparser.IColumnDefContext) *columnMetadataAST {
	if columnDef == nil {
		return nil
	}

	// Only normalize the column name - everything else stays as raw text
	columnName := ""
	if columnDef.Colid() != nil {
		columnName = pgpluginparser.NormalizePostgreSQLColid(columnDef.Colid())
	}

	metadata := &columnMetadataAST{
		Name: columnName,
	}

	// Extract type as raw text
	if columnDef.Typename() != nil {
		metadata.Type = getTextFromAST(columnDef.Typename())
	}

	// Extract nullable - check AST directly for NOT NULL constraint
	metadata.Nullable = true // Default is nullable
	if columnDef.Colquallist() != nil {
		for _, qualifier := range columnDef.Colquallist().AllColconstraint() {
			if qualifier != nil && qualifier.Colconstraintelem() != nil {
				elem := qualifier.Colconstraintelem()
				// Check for NOT NULL constraint using AST inspection
				if elem.NOT() != nil && elem.NULL_P() != nil {
					metadata.Nullable = false
					break
				}
				// Check for explicit NULL constraint
				if elem.NULL_P() != nil && elem.NOT() == nil {
					metadata.Nullable = true
				}
			}
		}
	}

	// Extract default value using AST inspection
	if columnDef.Colquallist() != nil {
		for _, qualifier := range columnDef.Colquallist().AllColconstraint() {
			if qualifier != nil && qualifier.Colconstraintelem() != nil {
				elem := qualifier.Colconstraintelem()
				// Check for DEFAULT constraint using AST inspection
				if elem.DEFAULT() != nil && elem.B_expr() != nil {
					metadata.Default = getTextFromAST(elem.B_expr())
					break
				}
			}
		}
	}

	return metadata
}

// columnMetadataAST holds column metadata extracted from AST without normalization
type columnMetadataAST struct {
	Name     string
	Type     string
	Nullable bool
	Default  string
}

// getTextFromAST extracts text from AST node using tokens when available, fallback to GetText()
func getTextFromAST(ctx antlr.ParseTree) string {
	if ctx == nil {
		return ""
	}

	// Try to get text from tokens first (preferred method)
	// Check for interfaces that have the required methods
	type parserContext interface {
		GetParser() antlr.Parser
		GetStart() antlr.Token
		GetStop() antlr.Token
	}

	if ruleContext, ok := ctx.(parserContext); ok {
		if parser := ruleContext.GetParser(); parser != nil {
			if tokenStream := parser.GetTokenStream(); tokenStream != nil {
				start := ruleContext.GetStart()
				stop := ruleContext.GetStop()
				if start != nil && stop != nil {
					return tokenStream.GetTextFromTokens(start, stop)
				}
			}
		}
	}

	// Fallback to GetText() if tokens are not available
	return ctx.GetText()
}

// extractConstraintNameFromAST extracts constraint name from table constraint AST node
func extractConstraintNameFromAST(constraintAST pgparser.ITableconstraintContext) string {
	if constraintAST == nil {
		return ""
	}

	if constraintAST.Name() != nil {
		return pgpluginparser.NormalizePostgreSQLName(constraintAST.Name())
	}

	// If no explicit name, generate one based on constraint type and content
	if constraintAST.Constraintelem() != nil {
		elem := constraintAST.Constraintelem()

		// For CHECK constraints, try to generate a meaningful name
		if elem.CHECK() != nil {
			// Extract table name from context if possible, otherwise use a generic name
			return "check_constraint"
		}

		// For FOREIGN KEY constraints
		if elem.FOREIGN() != nil && elem.KEY() != nil {
			return "foreign_key_constraint"
		}

		// For UNIQUE constraints
		if elem.UNIQUE() != nil {
			return "unique_constraint"
		}

		// For PRIMARY KEY constraints
		if elem.PRIMARY() != nil && elem.KEY() != nil {
			return "primary_key_constraint"
		}
	}

	// Fallback: use the full constraint text as identifier
	return getTextFromAST(constraintAST)
}

// writeDropCheckConstraintFromAST drops a check constraint using AST node
func writeDropCheckConstraintFromAST(out *strings.Builder, schema, table string, constraintAST pgparser.ITableconstraintContext) {
	constraintName := extractConstraintNameFromAST(constraintAST)
	if constraintName == "" {
		// If we can't extract a name, use the constraint text as fallback
		constraintName = getTextFromAST(constraintAST)
	}
	writeDropCheckConstraint(out, schema, table, constraintName)
}

// writeDropForeignKeyFromAST drops a foreign key constraint using AST node
func writeDropForeignKeyFromAST(out *strings.Builder, schema, table string, constraintAST pgparser.ITableconstraintContext) {
	constraintName := extractConstraintNameFromAST(constraintAST)
	if constraintName == "" {
		// If we can't extract a name, use the constraint text as fallback
		constraintName = getTextFromAST(constraintAST)
	}
	writeDropForeignKey(out, schema, table, constraintName)
}

// writeDropPrimaryKeyFromAST drops a primary key constraint using AST node
func writeDropPrimaryKeyFromAST(out *strings.Builder, schema, table string, constraintAST pgparser.ITableconstraintContext) {
	constraintName := extractConstraintNameFromAST(constraintAST)
	if constraintName == "" {
		// If we can't extract a name, generate a default primary key name
		constraintName = fmt.Sprintf("%s_pkey", table)
	}
	writeDropPrimaryKey(out, schema, table, constraintName)
}

// writeAddCheckConstraintFromAST adds a check constraint using AST node
func writeAddCheckConstraintFromAST(out *strings.Builder, schema, table string, constraintAST pgparser.ITableconstraintContext) error {
	if constraintAST == nil {
		return errors.New("constraint AST node is nil")
	}

	constraintName := extractConstraintNameFromAST(constraintAST)

	// Extract CHECK expression
	if constraintAST.Constraintelem() != nil {
		elem := constraintAST.Constraintelem()
		if elem.CHECK() != nil && elem.A_expr() != nil {
			checkExpr := getTextFromAST(elem.A_expr())

			// Generate constraint name if not provided
			if constraintName == "" || constraintName == "check_constraint" {
				constraintName = fmt.Sprintf("%s_check", table)
			}

			_, _ = out.WriteString(`ALTER TABLE "`)
			_, _ = out.WriteString(schema)
			_, _ = out.WriteString(`"."`)
			_, _ = out.WriteString(table)
			_, _ = out.WriteString(`" ADD CONSTRAINT "`)
			_, _ = out.WriteString(constraintName)
			_, _ = out.WriteString(`" CHECK (`)
			_, _ = out.WriteString(checkExpr)
			_, _ = out.WriteString(`);`)
			_, _ = out.WriteString("\n")
			return nil
		}
	}

	return errors.New("could not extract CHECK constraint from AST node")
}

// writeAddExcludeConstraint adds an EXCLUDE constraint using constraint metadata
func writeAddExcludeConstraint(out *strings.Builder, schema, table string, exclude *storepb.ExcludeConstraintMetadata) {
	_, _ = out.WriteString(`ALTER TABLE "`)
	_, _ = out.WriteString(schema)
	_, _ = out.WriteString(`"."`)
	_, _ = out.WriteString(table)
	_, _ = out.WriteString(`" ADD CONSTRAINT "`)
	_, _ = out.WriteString(exclude.Name)
	_, _ = out.WriteString(`" `)
	_, _ = out.WriteString(exclude.Expression) // Already includes "EXCLUDE USING ..."
	_, _ = out.WriteString(`;`)
	_, _ = out.WriteString("\n")
}

// writeAddExcludeConstraintFromAST adds an EXCLUDE constraint using AST node
func writeAddExcludeConstraintFromAST(out *strings.Builder, schema, table string, constraintAST pgparser.ITableconstraintContext) error {
	if constraintAST == nil {
		return errors.New("constraint AST node is nil")
	}

	constraintName := extractConstraintNameFromAST(constraintAST)

	// Extract EXCLUDE constraint definition
	if constraintAST.Constraintelem() != nil {
		elem := constraintAST.Constraintelem()
		if elem.EXCLUDE() != nil {
			// Get the full EXCLUDE expression (EXCLUDE USING ...)
			excludeExpr := getTextFromAST(elem)

			// Generate constraint name if not provided
			if constraintName == "" || constraintName == "exclude_constraint" {
				constraintName = fmt.Sprintf("%s_exclude", table)
			}

			_, _ = out.WriteString(`ALTER TABLE "`)
			_, _ = out.WriteString(schema)
			_, _ = out.WriteString(`"."`)
			_, _ = out.WriteString(table)
			_, _ = out.WriteString(`" ADD CONSTRAINT "`)
			_, _ = out.WriteString(constraintName)
			_, _ = out.WriteString(`" `)
			_, _ = out.WriteString(excludeExpr) // Full expression including "EXCLUDE USING ..."
			_, _ = out.WriteString(`;`)
			_, _ = out.WriteString("\n")
			return nil
		}
	}

	return errors.New("could not extract EXCLUDE constraint from AST node")
}

// writeDropExcludeConstraintFromAST drops an EXCLUDE constraint using AST node
func writeDropExcludeConstraintFromAST(out *strings.Builder, schema, table string, constraintAST pgparser.ITableconstraintContext) {
	constraintName := extractConstraintNameFromAST(constraintAST)
	if constraintName == "" {
		// If we can't extract a name, use the constraint text as fallback
		constraintName = getTextFromAST(constraintAST)
	}
	writeDropConstraint(out, schema, table, constraintName)
}

// writeAddForeignKeyFromAST adds a foreign key constraint using AST node
func writeAddForeignKeyFromAST(out *strings.Builder, schema, table string, constraintAST pgparser.ITableconstraintContext) error {
	if constraintAST == nil {
		return errors.New("constraint AST node is nil")
	}

	constraintName := extractConstraintNameFromAST(constraintAST)

	// Extract FOREIGN KEY definition
	if constraintAST.Constraintelem() != nil {
		elem := constraintAST.Constraintelem()
		if elem.FOREIGN() != nil && elem.KEY() != nil {
			// Generate constraint name if not provided
			if constraintName == "" || constraintName == "foreign_key_constraint" {
				constraintName = fmt.Sprintf("%s_fkey", table)
			}

			// Extract the FOREIGN KEY part from the full constraint
			// This is a simplified approach - we'll use the original constraint text
			// but replace the constraint name if needed
			_, _ = out.WriteString(`ALTER TABLE "`)
			_, _ = out.WriteString(schema)
			_, _ = out.WriteString(`"."`)
			_, _ = out.WriteString(table)
			_, _ = out.WriteString(`" ADD CONSTRAINT "`)
			_, _ = out.WriteString(constraintName)
			_, _ = out.WriteString(`" `)

			// Extract just the FOREIGN KEY part (after constraint name)
			if constraintAST.Constraintelem() != nil {
				fkElem := constraintAST.Constraintelem()
				fkText := getTextFromAST(fkElem)
				_, _ = out.WriteString(fkText)
			}

			_, _ = out.WriteString(`;`)
			_, _ = out.WriteString("\n")
			return nil
		}
	}

	return errors.New("could not extract FOREIGN KEY constraint from AST node")
}

// writeAddUniqueConstraintFromAST adds a unique constraint using AST node
func writeAddUniqueConstraintFromAST(out *strings.Builder, schema, table string, constraintAST pgparser.ITableconstraintContext) error {
	if constraintAST == nil {
		return errors.New("constraint AST node is nil")
	}

	constraintName := extractConstraintNameFromAST(constraintAST)

	// Extract UNIQUE definition
	if constraintAST.Constraintelem() != nil {
		elem := constraintAST.Constraintelem()
		if elem.UNIQUE() != nil {
			// Generate constraint name if not provided
			if constraintName == "" || constraintName == "unique_constraint" {
				constraintName = fmt.Sprintf("%s_unique", table)
			}

			_, _ = out.WriteString(`ALTER TABLE "`)
			_, _ = out.WriteString(schema)
			_, _ = out.WriteString(`"."`)
			_, _ = out.WriteString(table)
			_, _ = out.WriteString(`" ADD CONSTRAINT "`)
			_, _ = out.WriteString(constraintName)
			_, _ = out.WriteString(`" `)

			// Extract the UNIQUE part
			uniqueText := getTextFromAST(elem)
			_, _ = out.WriteString(uniqueText)

			_, _ = out.WriteString(`;`)
			_, _ = out.WriteString("\n")
			return nil
		}
	}

	return errors.New("could not extract UNIQUE constraint from AST node")
}

// writeAddPrimaryKeyFromAST adds a primary key constraint using AST node
func writeAddPrimaryKeyFromAST(out *strings.Builder, schema, table string, constraintAST pgparser.ITableconstraintContext) error {
	if constraintAST == nil {
		return errors.New("constraint AST node is nil")
	}

	constraintName := extractConstraintNameFromAST(constraintAST)

	// Extract PRIMARY KEY definition
	if constraintAST.Constraintelem() != nil {
		elem := constraintAST.Constraintelem()
		if elem.PRIMARY() != nil && elem.KEY() != nil {
			// Generate constraint name if not provided
			if constraintName == "" || constraintName == "primary_key_constraint" {
				constraintName = fmt.Sprintf("%s_pkey", table)
			}

			_, _ = out.WriteString(`ALTER TABLE "`)
			_, _ = out.WriteString(schema)
			_, _ = out.WriteString(`"."`)
			_, _ = out.WriteString(table)
			_, _ = out.WriteString(`" ADD CONSTRAINT "`)
			_, _ = out.WriteString(constraintName)
			_, _ = out.WriteString(`" `)

			// Extract the PRIMARY KEY part
			pkText := getTextFromAST(elem)
			_, _ = out.WriteString(pkText)

			_, _ = out.WriteString(`;`)
			_, _ = out.WriteString("\n")
			return nil
		}
	}

	return errors.New("could not extract PRIMARY KEY constraint from AST node")
}

// Index helper functions for AST-only mode

// extractIndexNameFromAST extracts index name from standalone CREATE INDEX AST node
func extractIndexNameFromAST(indexAST *pgparser.IndexstmtContext) string {
	if indexAST == nil {
		return ""
	}

	if indexAST.Name() != nil {
		return pgpluginparser.NormalizePostgreSQLName(indexAST.Name())
	}

	return ""
}

// writeDropIndexFromAST drops an index using AST node
func writeDropIndexFromAST(out *strings.Builder, schema string, indexAST *pgparser.IndexstmtContext) {
	indexName := extractIndexNameFromAST(indexAST)
	if indexName == "" {
		// If we can't extract a name, use a fallback
		indexName = "unknown_index"
	}
	writeDropIndex(out, schema, indexName)
}

// writeCreateIndexFromAST creates an index using AST node
func writeCreateIndexFromAST(out *strings.Builder, indexAST *pgparser.IndexstmtContext) error {
	if indexAST == nil {
		return errors.New("index AST node is nil")
	}

	// Extract the original SQL text from the AST node
	var indexSQL string

	// Try to get text using token stream first
	if tokenStream := indexAST.GetParser().GetTokenStream(); tokenStream != nil {
		start := indexAST.GetStart()
		stop := indexAST.GetStop()
		if start != nil && stop != nil {
			indexSQL = tokenStream.GetTextFromTokens(start, stop)
		}
	}

	// Fallback to GetText() if token stream approach failed
	if indexSQL == "" {
		indexSQL = indexAST.GetText()
	}

	if indexSQL == "" {
		return errors.New("could not extract index SQL from AST node")
	}

	// Write the index SQL
	_, _ = out.WriteString(indexSQL)
	if !strings.HasSuffix(strings.TrimSpace(indexSQL), ";") {
		_, _ = out.WriteString(";")
	}
	_, _ = out.WriteString("\n")

	return nil
}

// extractForeignKeysFromAST extracts foreign key constraints from a CREATE TABLE AST node
// and returns foreign key metadata for dependency graph construction
func extractForeignKeysFromAST(createStmt *pgparser.CreatestmtContext, defaultSchema string) []*storepb.ForeignKeyMetadata {
	if createStmt == nil || createStmt.Opttableelementlist() == nil {
		return nil
	}

	tableElementList := createStmt.Opttableelementlist().Tableelementlist()
	if tableElementList == nil {
		return nil
	}

	var foreignKeys []*storepb.ForeignKeyMetadata

	for _, element := range tableElementList.AllTableelement() {
		if element.Tableconstraint() == nil {
			continue
		}

		constraint := element.Tableconstraint()
		if constraint.Constraintelem() == nil {
			continue
		}

		elem := constraint.Constraintelem()
		if elem.FOREIGN() == nil || elem.KEY() == nil {
			continue
		}

		// This is a foreign key constraint
		// Extract referenced table from qualified_name using AST and normalize functions
		// Grammar: FOREIGN KEY (...) REFERENCES qualified_name opt_column_list? ...
		qualifiedName := elem.Qualified_name()
		if qualifiedName == nil {
			continue
		}

		// Use NormalizePostgreSQLQualifiedName to get schema and table name
		// Returns []string: [table] or [schema, table]
		nameParts := pgpluginparser.NormalizePostgreSQLQualifiedName(qualifiedName)
		if len(nameParts) == 0 {
			continue
		}

		var referencedSchema, referencedTable string
		if len(nameParts) == 1 {
			// No schema specified, use default
			referencedSchema = defaultSchema
			referencedTable = nameParts[0]
		} else if len(nameParts) == 2 {
			// Schema and table specified
			referencedSchema = nameParts[0]
			referencedTable = nameParts[1]
		} else {
			// Unexpected format, skip
			continue
		}

		if referencedTable != "" {
			fk := &storepb.ForeignKeyMetadata{
				ReferencedSchema: referencedSchema,
				ReferencedTable:  referencedTable,
			}
			foreignKeys = append(foreignKeys, fk)
		}
	}

	return foreignKeys
}

// getTriggerObjectID generates a unique identifier for trigger objects
// Triggers are table-scoped in PostgreSQL, so identifier is schema.table.trigger_name
func getTriggerObjectID(triggerDiff *schema.TriggerDiff) string {
	return fmt.Sprintf("trigger:%s.%s.%s", triggerDiff.SchemaName, triggerDiff.TableName, triggerDiff.TriggerName)
}

// writeCreateTrigger writes a CREATE TRIGGER or CREATE OR REPLACE TRIGGER statement
func writeCreateTrigger(buf *strings.Builder, triggerDiff *schema.TriggerDiff) error {
	switch triggerDiff.Action {
	case schema.MetadataDiffActionCreate, schema.MetadataDiffActionAlter:
		// For both CREATE and ALTER, use CREATE OR REPLACE TRIGGER
		if triggerDiff.NewASTNode != nil {
			// AST mode: extract SQL from AST node
			triggerStmt, ok := triggerDiff.NewASTNode.(*pgparser.CreatetrigstmtContext)
			if !ok {
				return errors.New("invalid AST node type for trigger")
			}

			// Extract SQL from token stream
			var sqlText string
			if tokenStream := triggerStmt.GetParser().GetTokenStream(); tokenStream != nil {
				start := triggerStmt.GetStart()
				stop := triggerStmt.GetStop()
				if start != nil && stop != nil {
					sqlText = tokenStream.GetTextFromTokens(start, stop)
				}
			}

			if sqlText == "" {
				sqlText = triggerStmt.GetText()
			}

			// Convert to CREATE OR REPLACE for idempotency
			sqlText = convertTriggerToCreateOrReplace(sqlText)

			sqlText = strings.TrimSpace(sqlText)
			if !strings.HasSuffix(sqlText, ";") {
				sqlText += ";"
			}

			buf.WriteString(sqlText)
			buf.WriteString("\n\n")

			return nil
		} else if triggerDiff.NewTrigger != nil {
			// Metadata mode: convert trigger body to CREATE OR REPLACE
			triggerSQL := convertTriggerToCreateOrReplace(triggerDiff.NewTrigger.Body)
			buf.WriteString(triggerSQL)
			if !strings.HasSuffix(strings.TrimSpace(triggerSQL), ";") {
				buf.WriteString(";")
			}
			buf.WriteString("\n\n")
			return nil
		}

		return errors.Errorf("trigger %s requires either NewASTNode or NewTrigger", triggerDiff.Action)

	default:
		// Ignore other actions like DROP (handled elsewhere)
	}

	return nil
}

// extractTriggerFunctionName extracts the function name from EXECUTE FUNCTION clause
func extractTriggerFunctionName(astNode any) string {
	triggerStmt, ok := astNode.(*pgparser.CreatetrigstmtContext)
	if !ok || triggerStmt == nil {
		return ""
	}

	// Get text from token stream for more reliable parsing
	var text string
	if tokenStream := triggerStmt.GetParser().GetTokenStream(); tokenStream != nil {
		start := triggerStmt.GetStart()
		stop := triggerStmt.GetStop()
		if start != nil && stop != nil {
			text = tokenStream.GetTextFromTokens(start, stop)
		}
	}

	if text == "" {
		text = triggerStmt.GetText()
	}

	// Look for "EXECUTE FUNCTION function_name" or "EXECUTE PROCEDURE function_name"
	upperText := strings.ToUpper(text)
	idx := strings.Index(upperText, "EXECUTE FUNCTION")
	if idx == -1 {
		idx = strings.Index(upperText, "EXECUTE PROCEDURE")
		if idx == -1 {
			return ""
		}
		idx += len("EXECUTE PROCEDURE")
	} else {
		idx += len("EXECUTE FUNCTION")
	}

	remaining := strings.TrimSpace(text[idx:])
	// Extract function name (first word, possibly schema-qualified)
	endIdx := strings.IndexAny(remaining, "();")
	if endIdx > 0 {
		funcName := strings.TrimSpace(remaining[:endIdx])
		// Remove quotes if present
		funcName = strings.Trim(funcName, "\"")
		return funcName
	}

	return ""
}

// convertTriggerToCreateOrReplace converts CREATE TRIGGER to CREATE OR REPLACE TRIGGER using ANTLR
func convertTriggerToCreateOrReplace(definition string) string {
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

	// Create a visitor to find and modify CREATE TRIGGER statements
	visitor := &CreateOrReplaceTriggerVisitor{
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

// CreateOrReplaceTriggerVisitor visits the parse tree and modifies CREATE TRIGGER to CREATE OR REPLACE TRIGGER
type CreateOrReplaceTriggerVisitor struct {
	*pgparser.BasePostgreSQLParserVisitor
	tokens     *antlr.CommonTokenStream
	rewriter   *antlr.TokenStreamRewriter
	definition string
}

// Visit implements the visitor pattern
func (v *CreateOrReplaceTriggerVisitor) Visit(tree antlr.ParseTree) any {
	switch t := tree.(type) {
	case *pgparser.CreatetrigstmtContext:
		return v.visitCreateTriggerStmt(t)
	default:
		// Continue visiting children
		return v.visitChildren(tree)
	}
}

// visitChildren visits all children of a node
func (v *CreateOrReplaceTriggerVisitor) visitChildren(node antlr.ParseTree) any {
	for i := 0; i < node.GetChildCount(); i++ {
		child := node.GetChild(i)
		if parseTree, ok := child.(antlr.ParseTree); ok {
			v.Visit(parseTree)
		}
	}
	return nil
}

// visitCreateTriggerStmt handles CREATE TRIGGER statements
func (v *CreateOrReplaceTriggerVisitor) visitCreateTriggerStmt(ctx *pgparser.CreatetrigstmtContext) any {
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

	// Look for the TRIGGER keyword after CREATE
	triggerToken := v.findTriggerToken(ctx)
	if triggerToken == nil {
		return nil
	}

	// Insert "OR REPLACE" between CREATE and TRIGGER
	// We insert it right before the TRIGGER token
	v.rewriter.InsertBefore("", triggerToken.GetTokenIndex(), "OR REPLACE ")

	return nil
}

// findTriggerToken finds the TRIGGER token in the CREATE TRIGGER statement
func (v *CreateOrReplaceTriggerVisitor) findTriggerToken(ctx *pgparser.CreatetrigstmtContext) antlr.Token {
	// Get all tokens in the context range
	start := ctx.GetStart().GetTokenIndex()
	stop := ctx.GetStop().GetTokenIndex()

	for i := start; i <= stop; i++ {
		token := v.tokens.Get(i)
		if token.GetTokenType() == pgparser.PostgreSQLParserTRIGGER {
			return token
		}
	}

	return nil
}

// hasOrReplace checks if the CREATE TRIGGER statement already contains "OR REPLACE"
func (v *CreateOrReplaceTriggerVisitor) hasOrReplace(ctx *pgparser.CreatetrigstmtContext) bool {
	// Get all tokens in the context range between CREATE and TRIGGER
	start := ctx.GetStart().GetTokenIndex()
	triggerToken := v.findTriggerToken(ctx)
	if triggerToken == nil {
		return false
	}
	stop := triggerToken.GetTokenIndex()

	// Look for "OR" followed by "REPLACE" tokens, skipping whitespace
	for i := start; i < stop; i++ {
		token := v.tokens.Get(i)
		if token.GetTokenType() == pgparser.PostgreSQLParserOR {
			// Found OR, now look for REPLACE after it (skipping whitespace)
			for j := i + 1; j < stop; j++ {
				nextToken := v.tokens.Get(j)
				// Skip whitespace tokens (channel 1 is hidden channel)
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
