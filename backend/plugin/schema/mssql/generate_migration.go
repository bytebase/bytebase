package mssql

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/tsql"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/plugin/parser/tsql"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/store/model"
)

func init() {
	schema.RegisterGenerateMigration(storepb.Engine_MSSQL, generateMigration)
}

func generateMigration(diff *schema.MetadataDiff) (string, error) {
	var buf strings.Builder

	// Collect schemas to create first (needed for checking if create phase will have content)
	var schemasToCreate []string
	for _, schemaDiff := range diff.SchemaChanges {
		if schemaDiff.Action == schema.MetadataDiffActionCreate {
			// Skip creating dbo schema as it already exists by default
			if strings.ToLower(schemaDiff.SchemaName) != "dbo" {
				schemasToCreate = append(schemasToCreate, schemaDiff.SchemaName)
			}
		}
	}
	slices.Sort(schemasToCreate)

	// Safe order for migrations:
	// 1. Drop dependent objects first (in reverse dependency order)
	//    - Drop foreign keys
	//    - Drop indexes
	//    - Drop views (in reverse topological order)
	//    - Drop functions/procedures (might depend on tables)
	//    - Drop tables
	//    - Drop schemas
	// 2. Create/Alter objects (in dependency order)
	//    - Create schemas
	//    - Create/Alter tables and columns
	//    - Create indexes
	//    - Create foreign keys
	//    - Create views (in topological order)
	//    - Create functions/procedures

	// Phase 1: Drop dependent objects
	// 1.1 Drop foreign keys first (they depend on tables)
	for _, tableDiff := range diff.TableChanges {
		switch tableDiff.Action {
		case schema.MetadataDiffActionAlter:
			for _, fkDiff := range tableDiff.ForeignKeyChanges {
				if fkDiff.Action == schema.MetadataDiffActionDrop {
					_, _ = buf.WriteString("ALTER TABLE [")
					_, _ = buf.WriteString(tableDiff.SchemaName)
					_, _ = buf.WriteString("].[")
					_, _ = buf.WriteString(tableDiff.TableName)
					_, _ = buf.WriteString("] DROP CONSTRAINT [")
					_, _ = buf.WriteString(fkDiff.OldForeignKey.Name)
					_, _ = buf.WriteString("];\nGO\n")
				}
			}
		case schema.MetadataDiffActionDrop:
			// Drop foreign keys before dropping table
			for _, fk := range tableDiff.OldTable.ForeignKeys {
				_, _ = buf.WriteString("ALTER TABLE [")
				_, _ = buf.WriteString(tableDiff.SchemaName)
				_, _ = buf.WriteString("].[")
				_, _ = buf.WriteString(tableDiff.TableName)
				_, _ = buf.WriteString("] DROP CONSTRAINT [")
				_, _ = buf.WriteString(fk.Name)
				_, _ = buf.WriteString("];\n")
			}
		default:
			// Ignore other actions for this phase
		}
	}

	// 1.2 Drop procedures (might depend on tables)
	for _, procDiff := range diff.ProcedureChanges {
		if procDiff.Action == schema.MetadataDiffActionDrop {
			_, _ = buf.WriteString("DROP PROCEDURE [")
			_, _ = buf.WriteString(procDiff.SchemaName)
			_, _ = buf.WriteString("].[")
			_, _ = buf.WriteString(procDiff.ProcedureName)
			_, _ = buf.WriteString("];\nGO\n")
		}
	}

	// 1.3 Drop functions (might depend on tables)
	for _, funcDiff := range diff.FunctionChanges {
		if funcDiff.Action == schema.MetadataDiffActionDrop {
			_, _ = buf.WriteString("DROP FUNCTION [")
			_, _ = buf.WriteString(funcDiff.SchemaName)
			_, _ = buf.WriteString("].[")
			_, _ = buf.WriteString(funcDiff.FunctionName)
			_, _ = buf.WriteString("];\nGO\n")
		}
	}

	// 1.4 Drop views in reverse topological order (dependent views first)
	dropViewsInOrder(diff, &buf)

	// 1.5 Drop indexes and constraints from tables being altered
	for _, tableDiff := range diff.TableChanges {
		if tableDiff.Action == schema.MetadataDiffActionAlter {
			// Drop check constraints
			for _, checkDiff := range tableDiff.CheckConstraintChanges {
				if checkDiff.Action == schema.MetadataDiffActionDrop {
					_, _ = buf.WriteString("ALTER TABLE [")
					_, _ = buf.WriteString(tableDiff.SchemaName)
					_, _ = buf.WriteString("].[")
					_, _ = buf.WriteString(tableDiff.TableName)
					_, _ = buf.WriteString("] DROP CONSTRAINT [")
					_, _ = buf.WriteString(checkDiff.OldCheckConstraint.Name)
					_, _ = buf.WriteString("];\n")
				}
			}

			// Drop indexes
			for _, indexDiff := range tableDiff.IndexChanges {
				if indexDiff.Action == schema.MetadataDiffActionDrop {
					if indexDiff.OldIndex.IsConstraint {
						_, _ = buf.WriteString("ALTER TABLE [")
						_, _ = buf.WriteString(tableDiff.SchemaName)
						_, _ = buf.WriteString("].[")
						_, _ = buf.WriteString(tableDiff.TableName)
						_, _ = buf.WriteString("] DROP CONSTRAINT [")
						_, _ = buf.WriteString(indexDiff.OldIndex.Name)
						_, _ = buf.WriteString("];\n")
					} else {
						_, _ = buf.WriteString("DROP INDEX [")
						_, _ = buf.WriteString(indexDiff.OldIndex.Name)
						_, _ = buf.WriteString("] ON [")
						_, _ = buf.WriteString(tableDiff.SchemaName)
						_, _ = buf.WriteString("].[")
						_, _ = buf.WriteString(tableDiff.TableName)
						_, _ = buf.WriteString("];\n")
					}
				}
			}

			// Drop columns
			hasDroppedIndex := false
			for _, indexDiff := range tableDiff.IndexChanges {
				if indexDiff.Action == schema.MetadataDiffActionDrop {
					hasDroppedIndex = true
					break
				}
			}

			for i, colDiff := range tableDiff.ColumnChanges {
				if colDiff.Action == schema.MetadataDiffActionDrop {
					// Add GO before first column drop if we dropped indexes
					if i == 0 && hasDroppedIndex {
						_, _ = buf.WriteString("GO\n")
					}

					// If the column has a default constraint, drop it first
					if getColumnDefaultValue(colDiff.OldColumn) != "" {
						if colDiff.OldColumn.DefaultConstraintName != "" {
							// Use the known constraint name directly
							_, _ = buf.WriteString("ALTER TABLE [")
							_, _ = buf.WriteString(tableDiff.SchemaName)
							_, _ = buf.WriteString("].[")
							_, _ = buf.WriteString(tableDiff.TableName)
							_, _ = buf.WriteString("] DROP CONSTRAINT [")
							_, _ = buf.WriteString(colDiff.OldColumn.DefaultConstraintName)
							_, _ = buf.WriteString("];\n")
						}
						// Note: If DefaultConstraintName is empty, we cannot drop the constraint automatically
					}

					_, _ = buf.WriteString("ALTER TABLE [")
					_, _ = buf.WriteString(tableDiff.SchemaName)
					_, _ = buf.WriteString("].[")
					_, _ = buf.WriteString(tableDiff.TableName)
					_, _ = buf.WriteString("] DROP COLUMN [")
					_, _ = buf.WriteString(colDiff.OldColumn.Name)
					_, _ = buf.WriteString("];\n")
				}
			}
		}
	}

	// Add GO after table alterations if we had any and we're about to drop tables
	hasTableAlterations := false
	for _, tableDiff := range diff.TableChanges {
		if tableDiff.Action == schema.MetadataDiffActionAlter {
			for _, colDiff := range tableDiff.ColumnChanges {
				if colDiff.Action == schema.MetadataDiffActionDrop {
					hasTableAlterations = true
					break
				}
			}
		}
	}

	// 1.6 Drop tables
	hasTableDrops := false
	for _, tableDiff := range diff.TableChanges {
		if tableDiff.Action == schema.MetadataDiffActionDrop {
			if !hasTableDrops && hasTableAlterations {
				_, _ = buf.WriteString("GO\n")
			}
			hasTableDrops = true
			_, _ = buf.WriteString("DROP TABLE [")
			_, _ = buf.WriteString(tableDiff.SchemaName)
			_, _ = buf.WriteString("].[")
			_, _ = buf.WriteString(tableDiff.TableName)
			_, _ = buf.WriteString("];\n")
		}
	}

	// 1.6.1 Drop all tables in schemas that are being dropped
	// This handles cases where the schema differ doesn't detect tables in dropped schemas
	for _, schemaDiff := range diff.SchemaChanges {
		if schemaDiff.Action == schema.MetadataDiffActionDrop && schemaDiff.OldSchema != nil {
			// Drop all tables in this schema
			for _, table := range schemaDiff.OldSchema.Tables {
				// Check if this table is already handled by tableDiff
				alreadyHandled := false
				for _, tableDiff := range diff.TableChanges {
					if tableDiff.Action == schema.MetadataDiffActionDrop &&
						tableDiff.SchemaName == schemaDiff.SchemaName &&
						tableDiff.TableName == table.Name {
						alreadyHandled = true
						break
					}
				}
				if !alreadyHandled {
					_, _ = buf.WriteString("DROP TABLE [")
					_, _ = buf.WriteString(schemaDiff.SchemaName)
					_, _ = buf.WriteString("].[")
					_, _ = buf.WriteString(table.Name)
					_, _ = buf.WriteString("];\n")
				}
			}
		}
	}

	// 1.7 Drop schemas (must be empty)
	for _, schemaDiff := range diff.SchemaChanges {
		if schemaDiff.Action == schema.MetadataDiffActionDrop {
			// Skip dropping dbo schema as it's a system schema
			if strings.ToLower(schemaDiff.SchemaName) == "dbo" {
				continue
			}
			_, _ = buf.WriteString("DROP SCHEMA [")
			_, _ = buf.WriteString(schemaDiff.SchemaName)
			_, _ = buf.WriteString("];\nGO\n")
		}
	}

	// Only add blank line if we have drops AND we're about to create something
	dropPhaseHasContent := buf.Len() > 0
	createPhaseWillHaveContent := len(schemasToCreate) > 0 ||
		hasCreateOrAlterTables(diff) ||
		hasCreateViews(diff) ||
		hasCreateFunctions(diff) ||
		hasCreateProcedures(diff)

	if dropPhaseHasContent && createPhaseWillHaveContent {
		_, _ = buf.WriteString("\n")
	}

	// Phase 2: Create/Alter objects
	// 2.1 Create schemas first (already sorted)
	for _, schemaName := range schemasToCreate {
		_, _ = buf.WriteString("CREATE SCHEMA [")
		_, _ = buf.WriteString(schemaName)
		_, _ = buf.WriteString("];\nGO\n")
	}

	// Add blank line after schema creation only if we have schemas and more content follows
	if len(schemasToCreate) > 0 && (hasCreateOrAlterTables(diff) || hasCreateViews(diff) || hasCreateFunctions(diff) || hasCreateProcedures(diff)) {
		_, _ = buf.WriteString("\n")
	}

	// 2.2 Create new tables WITHOUT foreign keys (in topological order based on FK dependencies)
	createTablesInOrder(diff, &buf)

	// 2.3 Alter existing tables (add columns, alter columns)
	for _, tableDiff := range diff.TableChanges {
		if tableDiff.Action == schema.MetadataDiffActionAlter {
			alterTableSQL := generateAlterTable(tableDiff)
			_, _ = buf.WriteString(alterTableSQL)
			if alterTableSQL != "" {
				_, _ = buf.WriteString("\n")
			}
		}
	}

	// 2.4 Add foreign keys for newly created tables (after all tables exist)
	for _, tableDiff := range diff.TableChanges {
		if tableDiff.Action == schema.MetadataDiffActionCreate && tableDiff.NewTable != nil {
			for _, fk := range tableDiff.NewTable.ForeignKeys {
				fkSQL := generateAddForeignKey(tableDiff.SchemaName, tableDiff.TableName, fk)
				_, _ = buf.WriteString(fkSQL)
				_, _ = buf.WriteString(";\n")
			}
		}
	}

	// 2.5 Handle table and column comments
	hasCommentChanges := false
	for _, tableDiff := range diff.TableChanges {
		if tableDiff.Action == schema.MetadataDiffActionCreate && tableDiff.NewTable != nil {
			// Add table comment if present
			if tableDiff.NewTable.Comment != "" {
				commentSQL := generateTableCommentSQL("ADD", tableDiff.SchemaName, tableDiff.TableName, tableDiff.NewTable.Comment)
				_, _ = buf.WriteString(commentSQL)
				_, _ = buf.WriteString(";\n")
				hasCommentChanges = true
			}
			// Add column comments if present
			for _, column := range tableDiff.NewTable.Columns {
				if column.Comment != "" {
					commentSQL := generateColumnCommentSQL("ADD", tableDiff.SchemaName, tableDiff.TableName, column.Name, column.Comment)
					_, _ = buf.WriteString(commentSQL)
					_, _ = buf.WriteString(";\n")
					hasCommentChanges = true
				}
			}
		} else if tableDiff.Action == schema.MetadataDiffActionAlter {
			// Handle table comment changes
			oldTableComment := ""
			newTableComment := ""
			if tableDiff.OldTable != nil {
				oldTableComment = tableDiff.OldTable.Comment
			}
			if tableDiff.NewTable != nil {
				newTableComment = tableDiff.NewTable.Comment
			}
			if oldTableComment != newTableComment {
				if oldTableComment == "" && newTableComment != "" {
					// Add new table comment
					commentSQL := generateTableCommentSQL("ADD", tableDiff.SchemaName, tableDiff.TableName, newTableComment)
					_, _ = buf.WriteString(commentSQL)
					_, _ = buf.WriteString(";\n")
					hasCommentChanges = true
				} else if oldTableComment != "" && newTableComment == "" {
					// Drop table comment
					commentSQL := generateTableCommentSQL("DROP", tableDiff.SchemaName, tableDiff.TableName, "")
					_, _ = buf.WriteString(commentSQL)
					_, _ = buf.WriteString(";\n")
					hasCommentChanges = true
				} else if oldTableComment != "" && newTableComment != "" {
					// Update table comment
					commentSQL := generateTableCommentSQL("UPDATE", tableDiff.SchemaName, tableDiff.TableName, newTableComment)
					_, _ = buf.WriteString(commentSQL)
					_, _ = buf.WriteString(";\n")
					hasCommentChanges = true
				}
			}
			// Handle column comment changes
			for _, colDiff := range tableDiff.ColumnChanges {
				if colDiff.Action == schema.MetadataDiffActionCreate && colDiff.NewColumn != nil {
					if colDiff.NewColumn.Comment != "" {
						commentSQL := generateColumnCommentSQL("ADD", tableDiff.SchemaName, tableDiff.TableName, colDiff.NewColumn.Name, colDiff.NewColumn.Comment)
						_, _ = buf.WriteString(commentSQL)
						_, _ = buf.WriteString(";\n")
						hasCommentChanges = true
					}
				} else if colDiff.Action == schema.MetadataDiffActionAlter {
					oldColComment := ""
					newColComment := ""
					if colDiff.OldColumn != nil {
						oldColComment = colDiff.OldColumn.Comment
					}
					if colDiff.NewColumn != nil {
						newColComment = colDiff.NewColumn.Comment
					}
					if oldColComment != newColComment {
						if oldColComment == "" && newColComment != "" {
							// Add new column comment
							commentSQL := generateColumnCommentSQL("ADD", tableDiff.SchemaName, tableDiff.TableName, colDiff.NewColumn.Name, newColComment)
							_, _ = buf.WriteString(commentSQL)
							_, _ = buf.WriteString(";\n")
							hasCommentChanges = true
						} else if oldColComment != "" && newColComment == "" {
							// Drop column comment
							commentSQL := generateColumnCommentSQL("DROP", tableDiff.SchemaName, tableDiff.TableName, colDiff.NewColumn.Name, "")
							_, _ = buf.WriteString(commentSQL)
							_, _ = buf.WriteString(";\n")
							hasCommentChanges = true
						} else if oldColComment != "" && newColComment != "" {
							// Update column comment
							commentSQL := generateColumnCommentSQL("UPDATE", tableDiff.SchemaName, tableDiff.TableName, colDiff.NewColumn.Name, newColComment)
							_, _ = buf.WriteString(commentSQL)
							_, _ = buf.WriteString(";\n")
							hasCommentChanges = true
						}
					}
				}
			}
		}
	}

	// Add GO after comment changes if we have any
	if hasCommentChanges {
		_, _ = buf.WriteString("GO\n")
	}

	// Add a GO statement to separate table creation/alteration from the next phase
	// Only add if we have tables/alters and we have views/functions/procedures to follow
	if hasCreateOrAlterTables(diff) && (hasCreateViews(diff) || hasCreateFunctions(diff) || hasCreateProcedures(diff)) {
		_, _ = buf.WriteString("\nGO\n")
	}

	// 2.6 Create views in topological order (dependencies first)
	createViewsInOrder(diff, &buf)

	// 2.7 Create functions
	for _, funcDiff := range diff.FunctionChanges {
		if funcDiff.Action == schema.MetadataDiffActionCreate {
			_, _ = buf.WriteString(funcDiff.NewFunction.Definition)
			if !strings.HasSuffix(strings.TrimSpace(funcDiff.NewFunction.Definition), ";") {
				_, _ = buf.WriteString(";")
			}
			_, _ = buf.WriteString("\nGO\n")

			// Add function comment if present
			if funcDiff.NewFunction.Comment != "" {
				commentSQL := generateFunctionCommentSQL("ADD", funcDiff.SchemaName, funcDiff.FunctionName, funcDiff.NewFunction.Comment)
				_, _ = buf.WriteString(commentSQL)
				_, _ = buf.WriteString(";\nGO\n")
			}
		}
	}

	// 2.8 Create procedures
	for _, procDiff := range diff.ProcedureChanges {
		switch procDiff.Action {
		case schema.MetadataDiffActionCreate:
			_, _ = buf.WriteString(procDiff.NewProcedure.Definition)
			if !strings.HasSuffix(strings.TrimSpace(procDiff.NewProcedure.Definition), ";") {
				_, _ = buf.WriteString(";")
			}
			_, _ = buf.WriteString("\nGO\n")
		case schema.MetadataDiffActionAlter:
			// MSSQL requires dropping and recreating procedures
			_, _ = buf.WriteString("DROP PROCEDURE [")
			_, _ = buf.WriteString(procDiff.SchemaName)
			_, _ = buf.WriteString("].[")
			_, _ = buf.WriteString(procDiff.ProcedureName)
			_, _ = buf.WriteString("];\nGO\n")
			_, _ = buf.WriteString(procDiff.NewProcedure.Definition)
			if !strings.HasSuffix(strings.TrimSpace(procDiff.NewProcedure.Definition), ";") {
				_, _ = buf.WriteString(";")
			}
			_, _ = buf.WriteString("\nGO\n")
		default:
			// Ignore other actions
		}
	}

	// 2.9 Create sequences
	for _, sequenceDiff := range diff.SequenceChanges {
		if sequenceDiff.Action == schema.MetadataDiffActionCreate {
			// Generate CREATE SEQUENCE statement
			_, _ = buf.WriteString("CREATE SEQUENCE [")
			_, _ = buf.WriteString(sequenceDiff.SchemaName)
			_, _ = buf.WriteString("].[")
			_, _ = buf.WriteString(sequenceDiff.SequenceName)
			_, _ = buf.WriteString("]")

			if sequenceDiff.NewSequence.DataType != "" {
				_, _ = buf.WriteString(" AS ")
				_, _ = buf.WriteString(sequenceDiff.NewSequence.DataType)
			}

			_, _ = buf.WriteString(";\nGO\n")

			// Add sequence comment if present
			if sequenceDiff.NewSequence.Comment != "" {
				commentSQL := generateSequenceCommentSQL("ADD", sequenceDiff.SchemaName, sequenceDiff.SequenceName, sequenceDiff.NewSequence.Comment)
				_, _ = buf.WriteString(commentSQL)
				_, _ = buf.WriteString(";\nGO\n")
			}
		}
	}

	return buf.String(), nil
}

func generateCreateTable(schemaName, tableName string, table *storepb.TableMetadata) string {
	var buf strings.Builder

	_, _ = buf.WriteString("CREATE TABLE [")
	_, _ = buf.WriteString(schemaName)
	_, _ = buf.WriteString("].[")
	_, _ = buf.WriteString(tableName)
	_, _ = buf.WriteString("] (\n")

	// Add columns
	for i, column := range table.Columns {
		columnDef := generateColumnDefinition(column)
		_, _ = buf.WriteString("  ")
		_, _ = buf.WriteString(columnDef)

		// Add comma if not last column or if there are constraints
		if i < len(table.Columns)-1 || hasConstraintsInTable(table) || len(table.CheckConstraints) > 0 {
			_, _ = buf.WriteString(",")
		}
		_, _ = buf.WriteString("\n")
	}

	// Add constraints (inline with table definition)
	constraintAdded := false
	for _, idx := range table.Indexes {
		if idx.IsConstraint {
			if constraintAdded {
				_, _ = buf.WriteString(",\n")
			}
			_, _ = buf.WriteString("  CONSTRAINT [")
			_, _ = buf.WriteString(idx.Name)
			if idx.Primary {
				_, _ = buf.WriteString("] PRIMARY KEY")
			} else if idx.Unique {
				_, _ = buf.WriteString("] UNIQUE")
			} else {
				_, _ = buf.WriteString("]")
			}
			switch idx.Type {
			case "CLUSTERED":
				_, _ = buf.WriteString(" CLUSTERED")
			case "NONCLUSTERED":
				_, _ = buf.WriteString(" NONCLUSTERED")
			default:
				// Default to "" if not specified
			}
			_, _ = buf.WriteString(" (")
			for j, expr := range idx.Expressions {
				if j > 0 {
					_, _ = buf.WriteString(", ")
				}
				_, _ = buf.WriteString("[")
				_, _ = buf.WriteString(expr)
				_, _ = buf.WriteString("]")
			}
			_, _ = buf.WriteString(")")
			constraintAdded = true
		}
	}

	if constraintAdded && len(table.CheckConstraints) > 0 {
		_, _ = buf.WriteString(",\n")
	} else if constraintAdded {
		_, _ = buf.WriteString("\n")
	}

	// Add check constraints (inline with table definition)
	for i, check := range table.CheckConstraints {
		_, _ = buf.WriteString("  CONSTRAINT [")
		_, _ = buf.WriteString(check.Name)
		_, _ = buf.WriteString("] CHECK (")
		_, _ = buf.WriteString(check.Expression)
		_, _ = buf.WriteString(")")
		if i < len(table.CheckConstraints)-1 {
			_, _ = buf.WriteString(",")
		}
		_, _ = buf.WriteString("\n")
	}

	_, _ = buf.WriteString(");")

	// Add non-primary indexes (after table creation)
	for _, idx := range table.Indexes {
		if !idx.IsConstraint {
			indexSQL := generateCreateIndex(schemaName, tableName, idx)
			_, _ = buf.WriteString("\n")
			_, _ = buf.WriteString(indexSQL)
			_, _ = buf.WriteString(";")
		}
	}

	// Note: Foreign keys are now added in a separate phase after all tables are created

	return buf.String()
}

func generateAlterTable(tableDiff *schema.TableDiff) string {
	var buf strings.Builder

	// Add columns first (other operations might depend on them)
	for _, colDiff := range tableDiff.ColumnChanges {
		if colDiff.Action == schema.MetadataDiffActionCreate {
			columnDef := generateColumnDefinition(colDiff.NewColumn)
			_, _ = buf.WriteString("ALTER TABLE [")
			_, _ = buf.WriteString(tableDiff.SchemaName)
			_, _ = buf.WriteString("].[")
			_, _ = buf.WriteString(tableDiff.TableName)
			_, _ = buf.WriteString("] ADD ")
			_, _ = buf.WriteString(columnDef)
			_, _ = buf.WriteString(";\n")
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
				_, _ = buf.WriteString("ALTER TABLE [")
				_, _ = buf.WriteString(tableDiff.SchemaName)
				_, _ = buf.WriteString("].[")
				_, _ = buf.WriteString(tableDiff.TableName)
				_, _ = buf.WriteString("] ADD CONSTRAINT [")
				_, _ = buf.WriteString(indexDiff.NewIndex.Name)
				if indexDiff.NewIndex.Primary {
					_, _ = buf.WriteString("] PRIMARY KEY")
				} else if indexDiff.NewIndex.Unique {
					_, _ = buf.WriteString("] UNIQUE")
				} else {
					_, _ = buf.WriteString("]")
				}
				switch indexDiff.NewIndex.Type {
				case "CLUSTERED":
					_, _ = buf.WriteString(" CLUSTERED")
				case "NONCLUSTERED":
					_, _ = buf.WriteString(" NONCLUSTERED")
				default:
					// Other index types
				}
				_, _ = buf.WriteString(" (")
				for j, expr := range indexDiff.NewIndex.Expressions {
					if j > 0 {
						_, _ = buf.WriteString(", ")
					}
					_, _ = buf.WriteString("[")
					_, _ = buf.WriteString(expr)
					_, _ = buf.WriteString("]")
				}
				_, _ = buf.WriteString(");\n")
			} else {
				indexSQL := generateCreateIndex(tableDiff.SchemaName, tableDiff.TableName, indexDiff.NewIndex)
				_, _ = buf.WriteString(indexSQL)
				_, _ = buf.WriteString(";\n")
			}
		}
	}

	// Add check constraints
	for _, checkDiff := range tableDiff.CheckConstraintChanges {
		if checkDiff.Action == schema.MetadataDiffActionCreate {
			_, _ = buf.WriteString("ALTER TABLE [")
			_, _ = buf.WriteString(tableDiff.SchemaName)
			_, _ = buf.WriteString("].[")
			_, _ = buf.WriteString(tableDiff.TableName)
			_, _ = buf.WriteString("] ADD CONSTRAINT [")
			_, _ = buf.WriteString(checkDiff.NewCheckConstraint.Name)
			_, _ = buf.WriteString("] CHECK (")
			_, _ = buf.WriteString(checkDiff.NewCheckConstraint.Expression)
			_, _ = buf.WriteString(");\n")
		}
	}

	// Add foreign keys last (they depend on other tables/columns)
	for _, fkDiff := range tableDiff.ForeignKeyChanges {
		if fkDiff.Action == schema.MetadataDiffActionCreate {
			fkSQL := generateAddForeignKey(tableDiff.SchemaName, tableDiff.TableName, fkDiff.NewForeignKey)
			_, _ = buf.WriteString(fkSQL)
			_, _ = buf.WriteString(";\n")
		}
	}

	return buf.String()
}

func generateColumnDefinition(column *storepb.ColumnMetadata) string {
	var buf strings.Builder
	_, _ = buf.WriteString("[")
	_, _ = buf.WriteString(column.Name)
	_, _ = buf.WriteString("] ")

	// Check if this is a computed column by examining the type field
	isComputedColumn := strings.Contains(column.Type, " AS (") || strings.HasPrefix(column.Type, "AS (")

	if isComputedColumn {
		// For computed columns, just use the type as-is (it contains the AS expression)
		_, _ = buf.WriteString(column.Type)
		// Note: Computed columns don't have explicit nullability, IDENTITY, or defaults
		// unless they are PERSISTED, which would be included in the type definition
	} else {
		// Regular column: add type, identity, nullability, and default
		_, _ = buf.WriteString(column.Type)

		// Add IDENTITY if applicable
		if column.IsIdentity {
			_, _ = buf.WriteString(" IDENTITY(")
			_, _ = buf.WriteString(fmt.Sprintf("%d,%d", column.IdentitySeed, column.IdentityIncrement))
			_, _ = buf.WriteString(")")
		}

		// Add nullability
		if column.Nullable {
			_, _ = buf.WriteString(" NULL")
		} else {
			_, _ = buf.WriteString(" NOT NULL")
		}

		// Add default value if present
		if column.GetDefault() != "" {
			_, _ = buf.WriteString(" DEFAULT ")
			_, _ = buf.WriteString(column.GetDefault())
		}
	}
	return buf.String()
}

func generateAlterColumn(schemaName, tableName string, colDiff *schema.ColumnDiff) string {
	var buf strings.Builder

	// In MSSQL, we need to handle different aspects of column changes separately

	// If type changed, alter the column type
	if colDiff.OldColumn.Type != colDiff.NewColumn.Type {
		_, _ = buf.WriteString("ALTER TABLE [")
		_, _ = buf.WriteString(schemaName)
		_, _ = buf.WriteString("].[")
		_, _ = buf.WriteString(tableName)
		_, _ = buf.WriteString("] ALTER COLUMN [")
		_, _ = buf.WriteString(colDiff.NewColumn.Name)
		_, _ = buf.WriteString("] ")
		_, _ = buf.WriteString(colDiff.NewColumn.Type)
		if colDiff.NewColumn.Nullable {
			_, _ = buf.WriteString(" NULL")
		} else {
			_, _ = buf.WriteString(" NOT NULL")
		}
		_, _ = buf.WriteString(";\n")
	} else if colDiff.OldColumn.Nullable != colDiff.NewColumn.Nullable {
		// If only nullability changed
		_, _ = buf.WriteString("ALTER TABLE [")
		_, _ = buf.WriteString(schemaName)
		_, _ = buf.WriteString("].[")
		_, _ = buf.WriteString(tableName)
		_, _ = buf.WriteString("] ALTER COLUMN [")
		_, _ = buf.WriteString(colDiff.NewColumn.Name)
		_, _ = buf.WriteString("] ")
		_, _ = buf.WriteString(colDiff.NewColumn.Type)
		if colDiff.NewColumn.Nullable {
			_, _ = buf.WriteString(" NULL")
		} else {
			_, _ = buf.WriteString(" NOT NULL")
		}
		_, _ = buf.WriteString(";\n")
	}

	// Handle default value changes
	oldDefault := getColumnDefaultValue(colDiff.OldColumn)
	newDefault := getColumnDefaultValue(colDiff.NewColumn)

	if oldDefault != newDefault {
		// First, drop the existing default constraint if it exists
		if oldDefault != "" {
			if colDiff.OldColumn.DefaultConstraintName != "" {
				// Use the known constraint name directly (when synced from database)
				_, _ = buf.WriteString("ALTER TABLE [")
				_, _ = buf.WriteString(schemaName)
				_, _ = buf.WriteString("].[")
				_, _ = buf.WriteString(tableName)
				_, _ = buf.WriteString("] DROP CONSTRAINT [")
				_, _ = buf.WriteString(colDiff.OldColumn.DefaultConstraintName)
				_, _ = buf.WriteString("];\n")
			}
			// Note: If DefaultConstraintName is empty (e.g., when parsed from SQL), we cannot drop the constraint
			// as we don't know its name. The user needs to drop it manually or sync from the database first.
		}
		// Then, add the new default constraint if specified
		if newDefault != "" {
			_, _ = buf.WriteString("ALTER TABLE [")
			_, _ = buf.WriteString(schemaName)
			_, _ = buf.WriteString("].[")
			_, _ = buf.WriteString(tableName)
			_, _ = buf.WriteString("] ADD CONSTRAINT [DF_")
			_, _ = buf.WriteString(tableName)
			_, _ = buf.WriteString("_")
			_, _ = buf.WriteString(colDiff.NewColumn.Name)
			_, _ = buf.WriteString("] DEFAULT ")
			_, _ = buf.WriteString(newDefault)
			_, _ = buf.WriteString(" FOR [")
			_, _ = buf.WriteString(colDiff.NewColumn.Name)
			_, _ = buf.WriteString("];\n")
		}
	}
	return buf.String()
}

// getColumnDefaultValue extracts the default value from a column
func getColumnDefaultValue(column *storepb.ColumnMetadata) string {
	if column == nil {
		return ""
	}
	if column.GetDefault() != "" {
		return column.GetDefault()
	}
	return ""
}

func generateCreateIndex(schemaName, tableName string, index *storepb.IndexMetadata) string {
	var buf strings.Builder

	_, _ = buf.WriteString("CREATE")

	// Handle columnstore indexes specially
	indexType := strings.ToUpper(index.Type)
	switch indexType {
	case "CLUSTERED COLUMNSTORE":
		_, _ = buf.WriteString(" CLUSTERED COLUMNSTORE INDEX [")
		_, _ = buf.WriteString(index.Name)
		_, _ = buf.WriteString("] ON [")
		_, _ = buf.WriteString(schemaName)
		_, _ = buf.WriteString("].[")
		_, _ = buf.WriteString(tableName)
		_, _ = buf.WriteString("]")
		// Clustered columnstore indexes don't specify columns
		return buf.String()
	case "NONCLUSTERED COLUMNSTORE":
		_, _ = buf.WriteString(" NONCLUSTERED COLUMNSTORE INDEX [")
		_, _ = buf.WriteString(index.Name)
		_, _ = buf.WriteString("] ON [")
		_, _ = buf.WriteString(schemaName)
		_, _ = buf.WriteString("].[")
		_, _ = buf.WriteString(tableName)
		_, _ = buf.WriteString("]")
		// Nonclustered columnstore indexes may specify columns
		if len(index.Expressions) > 0 {
			_, _ = buf.WriteString(" (")
			for i, expr := range index.Expressions {
				if i > 0 {
					_, _ = buf.WriteString(", ")
				}
				_, _ = buf.WriteString("[")
				_, _ = buf.WriteString(expr)
				_, _ = buf.WriteString("]")
			}
			_, _ = buf.WriteString(")")
		}
		return buf.String()
	case "SPATIAL":
		return generateSpatialIndexDDL(index, schemaName, tableName)
	default:
		// Regular indexes
		if index.Unique {
			_, _ = buf.WriteString(" UNIQUE")
		}

		switch index.Type {
		case "CLUSTERED":
			_, _ = buf.WriteString(" CLUSTERED")
		case "NONCLUSTERED":
			_, _ = buf.WriteString(" NONCLUSTERED")
		default:
			// Other index types
		}

		_, _ = buf.WriteString(" INDEX [")
		_, _ = buf.WriteString(index.Name)
		_, _ = buf.WriteString("] ON [")
		_, _ = buf.WriteString(schemaName)
		_, _ = buf.WriteString("].[")
		_, _ = buf.WriteString(tableName)
		_, _ = buf.WriteString("] (")

		for i, expr := range index.Expressions {
			if i > 0 {
				_, _ = buf.WriteString(", ")
			}
			_, _ = buf.WriteString("[")
			_, _ = buf.WriteString(expr)
			_, _ = buf.WriteString("]")
		}

		_, _ = buf.WriteString(")")

		return buf.String()
	}
}

func generateAddForeignKey(schemaName, tableName string, fk *storepb.ForeignKeyMetadata) string {
	var buf strings.Builder

	_, _ = buf.WriteString("ALTER TABLE [")
	_, _ = buf.WriteString(schemaName)
	_, _ = buf.WriteString("].[")
	_, _ = buf.WriteString(tableName)
	_, _ = buf.WriteString("] ADD CONSTRAINT [")
	_, _ = buf.WriteString(fk.Name)
	_, _ = buf.WriteString("] FOREIGN KEY (")

	for i, col := range fk.Columns {
		if i > 0 {
			_, _ = buf.WriteString(", ")
		}
		_, _ = buf.WriteString("[")
		_, _ = buf.WriteString(col)
		_, _ = buf.WriteString("]")
	}

	_, _ = buf.WriteString(") REFERENCES ")

	if fk.ReferencedSchema != "" {
		_, _ = buf.WriteString("[")
		_, _ = buf.WriteString(fk.ReferencedSchema)
		_, _ = buf.WriteString("].")
	}

	_, _ = buf.WriteString("[")
	_, _ = buf.WriteString(fk.ReferencedTable)
	_, _ = buf.WriteString("] (")

	for i, col := range fk.ReferencedColumns {
		if i > 0 {
			_, _ = buf.WriteString(", ")
		}
		_, _ = buf.WriteString("[")
		_, _ = buf.WriteString(col)
		_, _ = buf.WriteString("]")
	}

	_, _ = buf.WriteString(")")

	// Add ON DELETE action
	if fk.OnDelete != "" {
		_, _ = buf.WriteString(" ON DELETE ")
		_, _ = buf.WriteString(fk.OnDelete)
	}

	// Add ON UPDATE action
	if fk.OnUpdate != "" {
		_, _ = buf.WriteString(" ON UPDATE ")
		_, _ = buf.WriteString(fk.OnUpdate)
	}

	return buf.String()
}

// hasConstraintsInTable checks if the table has any constraints (primary key or unique)
func hasConstraintsInTable(table *storepb.TableMetadata) bool {
	for _, idx := range table.Indexes {
		if idx.IsConstraint {
			return true
		}
	}
	return false
}

// generateExtendedPropertySQL generates SQL for adding, updating, or dropping extended properties (comments)
func generateExtendedPropertySQL(action, objectType, schemaName, objectName, comment string) string {
	var buf strings.Builder

	switch action {
	case "ADD":
		_, _ = buf.WriteString("EXEC sp_addextendedproperty 'MS_Description', '")
		_, _ = buf.WriteString(strings.ReplaceAll(comment, "'", "''")) // Escape single quotes
		_, _ = buf.WriteString("', 'SCHEMA', '")
		_, _ = buf.WriteString(schemaName)
		_, _ = buf.WriteString("', '")
		_, _ = buf.WriteString(objectType)
		_, _ = buf.WriteString("', '")
		_, _ = buf.WriteString(objectName)
		_, _ = buf.WriteString("'")
	case "UPDATE":
		_, _ = buf.WriteString("EXEC sp_updateextendedproperty 'MS_Description', '")
		_, _ = buf.WriteString(strings.ReplaceAll(comment, "'", "''")) // Escape single quotes
		_, _ = buf.WriteString("', 'SCHEMA', '")
		_, _ = buf.WriteString(schemaName)
		_, _ = buf.WriteString("', '")
		_, _ = buf.WriteString(objectType)
		_, _ = buf.WriteString("', '")
		_, _ = buf.WriteString(objectName)
		_, _ = buf.WriteString("'")
	case "DROP":
		_, _ = buf.WriteString("EXEC sp_dropextendedproperty 'MS_Description', 'SCHEMA', '")
		_, _ = buf.WriteString(schemaName)
		_, _ = buf.WriteString("', '")
		_, _ = buf.WriteString(objectType)
		_, _ = buf.WriteString("', '")
		_, _ = buf.WriteString(objectName)
		_, _ = buf.WriteString("'")
	default:
		// Other actions
	}

	return buf.String()
}

// generateViewCommentSQL generates SQL for view comment changes
func generateViewCommentSQL(action, schemaName, viewName, comment string) string {
	return generateExtendedPropertySQL(action, "VIEW", schemaName, viewName, comment)
}

// generateFunctionCommentSQL generates SQL for function comment changes
func generateFunctionCommentSQL(action, schemaName, functionName, comment string) string {
	return generateExtendedPropertySQL(action, "FUNCTION", schemaName, functionName, comment)
}

// generateSequenceCommentSQL generates SQL for sequence comment changes
func generateSequenceCommentSQL(action, schemaName, sequenceName, comment string) string {
	return generateExtendedPropertySQL(action, "SEQUENCE", schemaName, sequenceName, comment)
}

// generateTableCommentSQL generates SQL for table comment changes
func generateTableCommentSQL(action, schemaName, tableName, comment string) string {
	return generateExtendedPropertySQL(action, "TABLE", schemaName, tableName, comment)
}

// generateColumnCommentSQL generates SQL for column comment changes
func generateColumnCommentSQL(action, schemaName, tableName, columnName, comment string) string {
	var buf strings.Builder

	switch action {
	case "ADD":
		_, _ = buf.WriteString("EXEC sp_addextendedproperty 'MS_Description', '")
		_, _ = buf.WriteString(strings.ReplaceAll(comment, "'", "''")) // Escape single quotes
		_, _ = buf.WriteString("', 'SCHEMA', '")
		_, _ = buf.WriteString(schemaName)
		_, _ = buf.WriteString("', 'TABLE', '")
		_, _ = buf.WriteString(tableName)
		_, _ = buf.WriteString("', 'COLUMN', '")
		_, _ = buf.WriteString(columnName)
		_, _ = buf.WriteString("'")
	case "UPDATE":
		_, _ = buf.WriteString("EXEC sp_updateextendedproperty 'MS_Description', '")
		_, _ = buf.WriteString(strings.ReplaceAll(comment, "'", "''")) // Escape single quotes
		_, _ = buf.WriteString("', 'SCHEMA', '")
		_, _ = buf.WriteString(schemaName)
		_, _ = buf.WriteString("', 'TABLE', '")
		_, _ = buf.WriteString(tableName)
		_, _ = buf.WriteString("', 'COLUMN', '")
		_, _ = buf.WriteString(columnName)
		_, _ = buf.WriteString("'")
	case "DROP":
		_, _ = buf.WriteString("EXEC sp_dropextendedproperty 'MS_Description', 'SCHEMA', '")
		_, _ = buf.WriteString(schemaName)
		_, _ = buf.WriteString("', 'TABLE', '")
		_, _ = buf.WriteString(tableName)
		_, _ = buf.WriteString("', 'COLUMN', '")
		_, _ = buf.WriteString(columnName)
		_, _ = buf.WriteString("'")
	default:
		// Other actions
	}

	return buf.String()
}

type queryClauseListener struct {
	*parser.BaseTSqlParserListener

	result string
}

func (l *queryClauseListener) EnterCreate_view(ctx *parser.Create_viewContext) {
	if l.result != "" {
		return
	}

	l.result = ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx.Select_statement_standalone())
}

// getViewDependencies extracts the tables that a view depends on
func getViewDependencies(viewDef string, schemaName string) ([]string, error) {
	// Parse the CREATE VIEW statement to extract the query properly
	// We need to find the AS keyword that's part of CREATE VIEW, not column aliases

	parseResults, err := tsql.ParseTSQL(viewDef)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse view definition")
	}

	if len(parseResults) != 1 {
		return nil, errors.Errorf("expected exactly 1 statement, got %d", len(parseResults))
	}

	// Extract the query part after the CREATE VIEW statement
	// This assumes the viewDef is a valid CREATE VIEW statement
	// and that it contains a valid T-SQL query.
	l := &queryClauseListener{}
	antlr.ParseTreeWalkerDefault.Walk(l, parseResults[0].Tree)
	if l.result == "" {
		return []string{}, nil
	}

	// Use GetQuerySpan with mock functions to avoid nil pointer dereference
	span, err := tsql.GetQuerySpan(
		context.Background(),
		base.GetQuerySpanContext{
			GetDatabaseMetadataFunc: func(_ context.Context, _, databaseName string) (string, *model.DatabaseMetadata, error) {
				// Return minimal metadata - we only need table references, not column info
				metadata := &storepb.DatabaseSchemaMetadata{
					Name: databaseName,
					Schemas: []*storepb.SchemaMetadata{
						{
							Name:   schemaName,
							Tables: []*storepb.TableMetadata{},
						},
					},
				}
				dbMetadata := model.NewDatabaseMetadata(metadata, nil, nil, storepb.Engine_MSSQL, false)
				return databaseName, dbMetadata, nil
			},
			ListDatabaseNamesFunc: func(_ context.Context, _ string) ([]string, error) {
				// Return empty list - we don't need actual database names for dependency extraction
				return []string{}, nil
			},
		},
		l.result,
		"", // database
		schemaName,
		false, // case sensitive
	)

	// If error parsing query span, return empty dependencies to allow migration to proceed
	// This is intentional - we prefer to proceed with no dependencies rather than block the migration
	if err != nil {
		return []string{}, nil // nolint:nilerr
	}

	// Collect unique dependencies
	dependencyMap := make(map[string]bool)
	for sourceColumn := range span.SourceColumns {
		// Create dependency ID in format: schema.table
		depID := getObjectID(sourceColumn.Schema, sourceColumn.Table)
		dependencyMap[depID] = true
	}

	// Convert map to slice
	var dependencies []string
	for dep := range dependencyMap {
		dependencies = append(dependencies, dep)
	}

	return dependencies, nil
}

// getObjectID creates a unique identifier for a database object
func getObjectID(schema, name string) string {
	if schema == "" {
		return name
	}
	return fmt.Sprintf("%s.%s", schema, name)
}

// dropViewsInOrder drops views in reverse topological order (dependent views first)
func dropViewsInOrder(diff *schema.MetadataDiff, buf *strings.Builder) {
	// Build dependency graph for views being dropped or altered
	graph := base.NewGraph()
	viewMap := make(map[string]*schema.ViewDiff)

	// First pass: Add all views to be dropped or altered to the graph and viewMap
	// Sort for deterministic processing order
	var viewsToProcess []*schema.ViewDiff
	for _, viewDiff := range diff.ViewChanges {
		if viewDiff.Action == schema.MetadataDiffActionDrop || viewDiff.Action == schema.MetadataDiffActionAlter {
			viewsToProcess = append(viewsToProcess, viewDiff)
		}
	}
	slices.SortFunc(viewsToProcess, func(i, j *schema.ViewDiff) int {
		iFullName := getObjectID(i.SchemaName, i.ViewName)
		jFullName := getObjectID(j.SchemaName, j.ViewName)
		if iFullName < jFullName {
			return -1
		}
		if iFullName > jFullName {
			return 1
		}
		return 0
	})

	for _, viewDiff := range viewsToProcess {
		viewID := getObjectID(viewDiff.SchemaName, viewDiff.ViewName)
		graph.AddNode(viewID)
		viewMap[viewID] = viewDiff
	}

	// Second pass: Add dependency edges now that all views are in viewMap
	for _, viewDiff := range viewsToProcess {
		viewID := getObjectID(viewDiff.SchemaName, viewDiff.ViewName)

		// Get dependencies from the old view definition
		if viewDiff.OldView != nil && viewDiff.OldView.Definition != "" {
			deps, err := getViewDependencies(viewDiff.OldView.Definition, viewDiff.SchemaName)
			if err != nil {
				// If we can't parse dependencies, we'll just drop in original order
				continue
			}

			// Add edges from this view to its dependencies
			for _, dep := range deps {
				// Only add edge if the dependency is also being dropped/altered
				if _, exists := viewMap[dep]; exists {
					graph.AddEdge(viewID, dep)
				}
			}
		}
	}

	// Get topological order
	orderedList, err := graph.TopologicalSort()
	if err != nil {
		// If there's a cycle or error, fall back to original order
		var fallbackViews []*schema.ViewDiff
		for _, viewDiff := range diff.ViewChanges {
			if viewDiff.Action == schema.MetadataDiffActionDrop || viewDiff.Action == schema.MetadataDiffActionAlter {
				fallbackViews = append(fallbackViews, viewDiff)
			}
		}
		// Sort alphabetically for deterministic output
		slices.SortFunc(fallbackViews, func(i, j *schema.ViewDiff) int {
			iFullName := getObjectID(i.SchemaName, i.ViewName)
			jFullName := getObjectID(j.SchemaName, j.ViewName)
			if iFullName < jFullName {
				return -1
			}
			if iFullName > jFullName {
				return 1
			}
			return 0
		})
		for _, viewDiff := range fallbackViews {
			_, _ = buf.WriteString("DROP VIEW [")
			_, _ = buf.WriteString(viewDiff.SchemaName)
			_, _ = buf.WriteString("].[")
			_, _ = buf.WriteString(viewDiff.ViewName)
			_, _ = buf.WriteString("];\nGO\n")
		}
		return
	}

	// Drop views in order (most dependent first due to edge direction)
	for _, viewID := range orderedList {
		if viewDiff, ok := viewMap[viewID]; ok {
			_, _ = buf.WriteString("DROP VIEW [")
			_, _ = buf.WriteString(viewDiff.SchemaName)
			_, _ = buf.WriteString("].[")
			_, _ = buf.WriteString(viewDiff.ViewName)
			_, _ = buf.WriteString("];\nGO\n")
		}
	}
}

// createTablesInOrder creates tables in topological order (dependencies first based on foreign keys)
func createTablesInOrder(diff *schema.MetadataDiff, buf *strings.Builder) {
	// Build dependency graph for tables being created
	graph := base.NewGraph()
	tableMap := make(map[string]*schema.TableDiff)

	// First pass: Add all tables to be created to the graph and tableMap
	// Sort for deterministic processing order
	var tablesToCreate []*schema.TableDiff
	for _, tableDiff := range diff.TableChanges {
		if tableDiff.Action == schema.MetadataDiffActionCreate {
			tablesToCreate = append(tablesToCreate, tableDiff)
		}
	}
	slices.SortFunc(tablesToCreate, func(i, j *schema.TableDiff) int {
		iFullName := getObjectID(i.SchemaName, i.TableName)
		jFullName := getObjectID(j.SchemaName, j.TableName)
		if iFullName < jFullName {
			return -1
		}
		if iFullName > jFullName {
			return 1
		}
		return 0
	})

	for _, tableDiff := range tablesToCreate {
		tableID := getObjectID(tableDiff.SchemaName, tableDiff.TableName)
		graph.AddNode(tableID)
		tableMap[tableID] = tableDiff
	}

	// Second pass: Add dependency edges based on foreign keys
	for _, tableDiff := range tablesToCreate {
		tableID := getObjectID(tableDiff.SchemaName, tableDiff.TableName)

		// Add edges based on foreign key dependencies
		if tableDiff.NewTable != nil {
			for _, fk := range tableDiff.NewTable.ForeignKeys {
				// Create dependency ID for the referenced table
				refTableID := getObjectID(fk.ReferencedSchema, fk.ReferencedTable)

				// Only add edge if the referenced table is also being created
				if _, exists := tableMap[refTableID]; exists {
					// Edge from referenced table to this table (referenced must be created first)
					graph.AddEdge(refTableID, tableID)
				}
			}
		}
	}

	// Get topological order
	orderedList, err := graph.TopologicalSort()
	if err != nil {
		// If there's a cycle or error, fall back to alphabetical order
		for _, tableDiff := range tablesToCreate {
			createTableSQL := generateCreateTable(tableDiff.SchemaName, tableDiff.TableName, tableDiff.NewTable)
			_, _ = buf.WriteString(createTableSQL)
			_, _ = buf.WriteString("\n")
		}
		return
	}

	// Create tables in order
	for _, tableID := range orderedList {
		if tableDiff, ok := tableMap[tableID]; ok {
			createTableSQL := generateCreateTable(tableDiff.SchemaName, tableDiff.TableName, tableDiff.NewTable)
			_, _ = buf.WriteString(createTableSQL)
			_, _ = buf.WriteString("\n")
		}
	}
}

// createViewsInOrder creates views in topological order (dependencies first)
func createViewsInOrder(diff *schema.MetadataDiff, buf *strings.Builder) {
	// Build dependency graph for views being created or altered
	graph := base.NewGraph()
	viewMap := make(map[string]*schema.ViewDiff)

	// First pass: Add all views to be created or altered to the graph and viewMap
	// Sort for deterministic processing order
	var viewsToProcess []*schema.ViewDiff
	for _, viewDiff := range diff.ViewChanges {
		if viewDiff.Action == schema.MetadataDiffActionCreate || viewDiff.Action == schema.MetadataDiffActionAlter {
			viewsToProcess = append(viewsToProcess, viewDiff)
		}
	}
	slices.SortFunc(viewsToProcess, func(i, j *schema.ViewDiff) int {
		iFullName := getObjectID(i.SchemaName, i.ViewName)
		jFullName := getObjectID(j.SchemaName, j.ViewName)
		if iFullName < jFullName {
			return -1
		}
		if iFullName > jFullName {
			return 1
		}
		return 0
	})

	for _, viewDiff := range viewsToProcess {
		viewID := getObjectID(viewDiff.SchemaName, viewDiff.ViewName)
		graph.AddNode(viewID)
		viewMap[viewID] = viewDiff
	}

	// Second pass: Add dependency edges now that all views are in viewMap
	for _, viewDiff := range viewsToProcess {
		viewID := getObjectID(viewDiff.SchemaName, viewDiff.ViewName)

		// Get dependencies from the new view definition
		if viewDiff.NewView != nil && viewDiff.NewView.Definition != "" {
			deps, err := getViewDependencies(viewDiff.NewView.Definition, viewDiff.SchemaName)
			if err != nil {
				// If we can't parse dependencies, we'll just create in original order
				continue
			}

			// Add edges from dependencies to this view
			for _, dep := range deps {
				// Only add edge if the dependency is also being created/altered
				if _, exists := viewMap[dep]; exists {
					graph.AddEdge(dep, viewID)
				}
			}
		}
	}

	// Get topological order
	orderedList, err := graph.TopologicalSort()
	if err != nil {
		// If there's a cycle or error, fall back to original order
		var fallbackViews []*schema.ViewDiff
		for _, viewDiff := range diff.ViewChanges {
			// Only handle CREATE and ALTER (as create) in this phase
			if viewDiff.Action == schema.MetadataDiffActionCreate || viewDiff.Action == schema.MetadataDiffActionAlter {
				fallbackViews = append(fallbackViews, viewDiff)
			}
		}
		// Sort alphabetically for deterministic output
		slices.SortFunc(fallbackViews, func(i, j *schema.ViewDiff) int {
			iFullName := getObjectID(i.SchemaName, i.ViewName)
			jFullName := getObjectID(j.SchemaName, j.ViewName)
			if iFullName < jFullName {
				return -1
			}
			if iFullName > jFullName {
				return 1
			}
			return 0
		})
		for _, viewDiff := range fallbackViews {
			_, _ = buf.WriteString(viewDiff.NewView.Definition)
			if !strings.HasSuffix(strings.TrimSpace(viewDiff.NewView.Definition), ";") {
				_, _ = buf.WriteString(";")
			}
			_, _ = buf.WriteString("\nGO\n")

			// Add view comment if present
			if viewDiff.NewView.Comment != "" {
				commentSQL := generateViewCommentSQL("ADD", viewDiff.SchemaName, viewDiff.ViewName, viewDiff.NewView.Comment)
				_, _ = buf.WriteString(commentSQL)
				_, _ = buf.WriteString(";\nGO\n")
			}
		}
		return
	}

	// Create views in order
	for _, viewID := range orderedList {
		if viewDiff, ok := viewMap[viewID]; ok {
			// Both CREATE and ALTER actions create the view in this phase
			// (ALTER views were already dropped in the drop phase)
			_, _ = buf.WriteString(viewDiff.NewView.Definition)
			if !strings.HasSuffix(strings.TrimSpace(viewDiff.NewView.Definition), ";") {
				_, _ = buf.WriteString(";")
			}
			_, _ = buf.WriteString("\nGO\n")

			// Add view comment if present
			if viewDiff.NewView.Comment != "" {
				commentSQL := generateViewCommentSQL("ADD", viewDiff.SchemaName, viewDiff.ViewName, viewDiff.NewView.Comment)
				_, _ = buf.WriteString(commentSQL)
				_, _ = buf.WriteString(";\nGO\n")
			}
		}
	}
}

// Helper functions to check if diff contains certain types of changes
func hasCreateOrAlterTables(diff *schema.MetadataDiff) bool {
	for _, tableDiff := range diff.TableChanges {
		if tableDiff.Action == schema.MetadataDiffActionCreate || tableDiff.Action == schema.MetadataDiffActionAlter {
			return true
		}
	}
	return false
}

func hasCreateViews(diff *schema.MetadataDiff) bool {
	for _, viewDiff := range diff.ViewChanges {
		if viewDiff.Action == schema.MetadataDiffActionCreate || viewDiff.Action == schema.MetadataDiffActionAlter {
			return true
		}
	}
	return false
}

func hasCreateFunctions(diff *schema.MetadataDiff) bool {
	for _, funcDiff := range diff.FunctionChanges {
		if funcDiff.Action == schema.MetadataDiffActionCreate {
			return true
		}
	}
	return false
}

func hasCreateProcedures(diff *schema.MetadataDiff) bool {
	for _, procDiff := range diff.ProcedureChanges {
		if procDiff.Action == schema.MetadataDiffActionCreate || procDiff.Action == schema.MetadataDiffActionAlter {
			return true
		}
	}
	return false
}

func generateSpatialIndexDDL(index *storepb.IndexMetadata, schemaName, tableName string) string {
	var buf strings.Builder

	// Build the CREATE SPATIAL INDEX statement
	buf.WriteString("CREATE SPATIAL INDEX [")
	buf.WriteString(index.Name)
	buf.WriteString("] ON [")
	buf.WriteString(schemaName)
	buf.WriteString("].[")
	buf.WriteString(tableName)
	buf.WriteString("] (")

	// Add column expressions
	for i, expr := range index.Expressions {
		if i > 0 {
			buf.WriteString(", ")
		}
		buf.WriteString("[")
		buf.WriteString(expr)
		buf.WriteString("]")
	}
	buf.WriteString(")")

	// Check if spatial configuration exists
	if index.SpatialConfig == nil || index.SpatialConfig.Tessellation == nil {
		// Add warning comment for incomplete configuration
		buf.WriteString("\n/* WARNING: Spatial index configuration may be incomplete. ")
		buf.WriteString("Please verify BOUNDING_BOX, GRIDS, and other parameters manually. */")
		return buf.String()
	}

	tessellation := index.SpatialConfig.Tessellation

	// Add USING clause for tessellation scheme
	if tessellation.Scheme != "" {
		buf.WriteString("\nUSING ")
		buf.WriteString(tessellation.Scheme)
	}

	// Build WITH clause parameters
	withParams := []string{}

	// Add tessellation parameters
	withParams = append(withParams, buildMigrationTessellationParams(tessellation)...)

	// Add storage parameters
	if index.SpatialConfig.Storage != nil {
		withParams = append(withParams, buildMigrationStorageParams(index.SpatialConfig.Storage)...)
	}

	// Add WITH clause if we have parameters
	if len(withParams) > 0 {
		buf.WriteString("\nWITH (\n    ")
		buf.WriteString(strings.Join(withParams, ",\n    "))
		buf.WriteString("\n)")
	}

	return buf.String()
}

func buildMigrationTessellationParams(tessellation *storepb.TessellationConfig) []string {
	params := []string{}

	// BOUNDING_BOX for GEOMETRY indexes
	if tessellation.BoundingBox != nil {
		bbox := tessellation.BoundingBox
		params = append(params, fmt.Sprintf("BOUNDING_BOX = (%g, %g, %g, %g)",
			bbox.Xmin, bbox.Ymin, bbox.Xmax, bbox.Ymax))
	}

	// GRIDS configuration
	if len(tessellation.GridLevels) > 0 {
		gridParts := []string{}
		for _, level := range tessellation.GridLevels {
			gridParts = append(gridParts, fmt.Sprintf("LEVEL_%d = %s", level.Level, level.Density))
		}
		if len(gridParts) > 0 {
			params = append(params, fmt.Sprintf("GRIDS = (%s)", strings.Join(gridParts, ", ")))
		}
	}

	// CELLS_PER_OBJECT
	if tessellation.CellsPerObject > 0 {
		params = append(params, fmt.Sprintf("CELLS_PER_OBJECT = %d", tessellation.CellsPerObject))
	}

	return params
}

func buildMigrationStorageParams(storage *storepb.StorageConfig) []string {
	params := []string{}

	// PAD_INDEX (defaults to OFF, so only output when ON)
	if storage.PadIndex {
		params = append(params, "PAD_INDEX = ON")
	}

	// FILLFACTOR
	if storage.Fillfactor > 0 {
		params = append(params, fmt.Sprintf("FILLFACTOR = %d", storage.Fillfactor))
	}

	// SORT_IN_TEMPDB
	if storage.SortInTempdb != "" {
		params = append(params, fmt.Sprintf("SORT_IN_TEMPDB = %s", storage.SortInTempdb))
	}

	// DROP_EXISTING
	if storage.DropExisting {
		params = append(params, "DROP_EXISTING = ON")
	}

	// ONLINE
	if storage.Online {
		params = append(params, "ONLINE = ON")
	}

	// ALLOW_ROW_LOCKS and ALLOW_PAGE_LOCKS
	// Note: For migration DDL, we output them when they are true
	// This differs from definition DDL where we handle defaults differently
	if storage.AllowRowLocks {
		params = append(params, "ALLOW_ROW_LOCKS = ON")
	}

	if storage.AllowPageLocks {
		params = append(params, "ALLOW_PAGE_LOCKS = ON")
	}

	// MAXDOP
	if storage.Maxdop > 0 {
		params = append(params, fmt.Sprintf("MAXDOP = %d", storage.Maxdop))
	}

	// DATA_COMPRESSION
	if storage.DataCompression != "" && storage.DataCompression != "NONE" {
		params = append(params, fmt.Sprintf("DATA_COMPRESSION = %s", storage.DataCompression))
	}

	return params
}
