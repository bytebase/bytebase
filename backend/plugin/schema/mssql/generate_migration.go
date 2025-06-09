package mssql

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/tsql-parser"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/plugin/parser/tsql"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/store/model"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
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
	if err := dropViewsInOrder(diff, &buf); err != nil {
		return "", err
	}

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
			for _, colDiff := range tableDiff.ColumnChanges {
				if colDiff.Action == schema.MetadataDiffActionDrop {
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

	// 1.6 Drop tables
	for _, tableDiff := range diff.TableChanges {
		if tableDiff.Action == schema.MetadataDiffActionDrop {
			_, _ = buf.WriteString("DROP TABLE [")
			_, _ = buf.WriteString(tableDiff.SchemaName)
			_, _ = buf.WriteString("].[")
			_, _ = buf.WriteString(tableDiff.TableName)
			_, _ = buf.WriteString("];\n")
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

	// 2.2 Create new tables WITHOUT foreign keys (sorted for consistent output)
	var tablesToCreate []*schema.TableDiff
	for _, tableDiff := range diff.TableChanges {
		if tableDiff.Action == schema.MetadataDiffActionCreate {
			tablesToCreate = append(tablesToCreate, tableDiff)
		}
	}
	// Sort by schema.table name for consistent output
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
		createTableSQL, err := generateCreateTable(tableDiff.SchemaName, tableDiff.TableName, tableDiff.NewTable)
		if err != nil {
			return "", err
		}
		_, _ = buf.WriteString(createTableSQL)
		_, _ = buf.WriteString("\n")
	}

	// 2.3 Alter existing tables (add columns, alter columns)
	for _, tableDiff := range diff.TableChanges {
		if tableDiff.Action == schema.MetadataDiffActionAlter {
			alterTableSQL, err := generateAlterTable(tableDiff)
			if err != nil {
				return "", err
			}
			_, _ = buf.WriteString(alterTableSQL)
			if alterTableSQL != "" {
				_, _ = buf.WriteString("\n")
			}
		}
	}

	// 2.4 Add foreign keys for newly created tables (after all tables exist)
	for _, tableDiff := range tablesToCreate {
		for _, fk := range tableDiff.NewTable.ForeignKeys {
			fkSQL, err := generateAddForeignKey(tableDiff.SchemaName, tableDiff.TableName, fk)
			if err != nil {
				return "", err
			}
			_, _ = buf.WriteString(fkSQL)
			_, _ = buf.WriteString(";\n")
		}
	}

	// Add a GO statement to separate table creation/alteration from the next phase
	// Only add if we have tables/alters and we have views/functions/procedures to follow
	if hasCreateOrAlterTables(diff) && (hasCreateViews(diff) || hasCreateFunctions(diff) || hasCreateProcedures(diff)) {
		_, _ = buf.WriteString("\nGO\n")
	}

	// 2.4 Create views in topological order (dependencies first)
	if err := createViewsInOrder(diff, &buf); err != nil {
		return "", err
	}

	// 2.5 Create functions
	for _, funcDiff := range diff.FunctionChanges {
		if funcDiff.Action == schema.MetadataDiffActionCreate {
			_, _ = buf.WriteString(funcDiff.NewFunction.Definition)
			if !strings.HasSuffix(strings.TrimSpace(funcDiff.NewFunction.Definition), ";") {
				_, _ = buf.WriteString(";")
			}
			_, _ = buf.WriteString("\nGO\n")
		}
	}

	// 2.6 Create procedures
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
		}
	}

	return buf.String(), nil
}

func generateCreateTable(schemaName, tableName string, table *storepb.TableMetadata) (string, error) {
	var buf strings.Builder

	_, _ = buf.WriteString("CREATE TABLE [")
	_, _ = buf.WriteString(schemaName)
	_, _ = buf.WriteString("].[")
	_, _ = buf.WriteString(tableName)
	_, _ = buf.WriteString("] (\n")

	// Add columns
	for i, column := range table.Columns {
		columnDef, err := generateColumnDefinition(column)
		if err != nil {
			return "", err
		}
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
			indexSQL, err := generateCreateIndex(schemaName, tableName, idx)
			if err != nil {
				return "", err
			}
			_, _ = buf.WriteString("\n")
			_, _ = buf.WriteString(indexSQL)
			_, _ = buf.WriteString(";")
		}
	}

	// Note: Foreign keys are now added in a separate phase after all tables are created

	return buf.String(), nil
}

func generateAlterTable(tableDiff *schema.TableDiff) (string, error) {
	var buf strings.Builder

	// Add columns first (other operations might depend on them)
	for _, colDiff := range tableDiff.ColumnChanges {
		if colDiff.Action == schema.MetadataDiffActionCreate {
			columnDef, err := generateColumnDefinition(colDiff.NewColumn)
			if err != nil {
				return "", err
			}
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
			alterColSQL, err := generateAlterColumn(tableDiff.SchemaName, tableDiff.TableName, colDiff)
			if err != nil {
				return "", err
			}
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
				indexSQL, err := generateCreateIndex(tableDiff.SchemaName, tableDiff.TableName, indexDiff.NewIndex)
				if err != nil {
					return "", err
				}
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
			fkSQL, err := generateAddForeignKey(tableDiff.SchemaName, tableDiff.TableName, fkDiff.NewForeignKey)
			if err != nil {
				return "", err
			}
			_, _ = buf.WriteString(fkSQL)
			_, _ = buf.WriteString(";\n")
		}
	}

	return buf.String(), nil
}

func generateColumnDefinition(column *storepb.ColumnMetadata) (string, error) {
	var buf strings.Builder

	_, _ = buf.WriteString("[")
	_, _ = buf.WriteString(column.Name)
	_, _ = buf.WriteString("] ")
	_, _ = buf.WriteString(column.Type)

	// Add nullability
	if column.Nullable {
		_, _ = buf.WriteString(" NULL")
	} else {
		_, _ = buf.WriteString(" NOT NULL")
	}

	// Add default value
	defaultExpr := getDefaultExpression(column)
	if defaultExpr != "" {
		_, _ = buf.WriteString(" DEFAULT ")
		_, _ = buf.WriteString(defaultExpr)
	}

	return buf.String(), nil
}

func generateAlterColumn(schemaName, tableName string, colDiff *schema.ColumnDiff) (string, error) {
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
	oldHasDefault := hasDefaultValue(colDiff.OldColumn)
	newHasDefault := hasDefaultValue(colDiff.NewColumn)
	if oldHasDefault || newHasDefault {
		if !defaultValuesEqual(colDiff.OldColumn, colDiff.NewColumn) {
			// First drop the old default constraint if it exists
			if oldHasDefault {
				// We need to find the default constraint name
				// In practice, this might require querying system tables
				// For now, we'll use a naming convention
				constraintName := fmt.Sprintf("DF_%s_%s", tableName, colDiff.OldColumn.Name)
				_, _ = buf.WriteString("ALTER TABLE [")
				_, _ = buf.WriteString(schemaName)
				_, _ = buf.WriteString("].[")
				_, _ = buf.WriteString(tableName)
				_, _ = buf.WriteString("] DROP CONSTRAINT [")
				_, _ = buf.WriteString(constraintName)
				_, _ = buf.WriteString("];\n")
			}

			// Add new default constraint if needed
			if newHasDefault {
				constraintName := fmt.Sprintf("DF_%s_%s", tableName, colDiff.NewColumn.Name)
				defaultExpr := getDefaultExpression(colDiff.NewColumn)
				_, _ = buf.WriteString("ALTER TABLE [")
				_, _ = buf.WriteString(schemaName)
				_, _ = buf.WriteString("].[")
				_, _ = buf.WriteString(tableName)
				_, _ = buf.WriteString("] ADD CONSTRAINT [")
				_, _ = buf.WriteString(constraintName)
				_, _ = buf.WriteString("] DEFAULT ")
				_, _ = buf.WriteString(defaultExpr)
				_, _ = buf.WriteString(" FOR [")
				_, _ = buf.WriteString(colDiff.NewColumn.Name)
				_, _ = buf.WriteString("];\n")
			}
		}
	}

	return buf.String(), nil
}

func generateCreateIndex(schemaName, tableName string, index *storepb.IndexMetadata) (string, error) {
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
		return buf.String(), nil
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
		return buf.String(), nil
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

		return buf.String(), nil
	}
}

func generateAddForeignKey(schemaName, tableName string, fk *storepb.ForeignKeyMetadata) (string, error) {
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

	return buf.String(), nil
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

	parseResult, err := tsql.ParseTSQL(viewDef)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse view definition")
	}

	// Extract the query part after the CREATE VIEW statement
	// This assumes the viewDef is a valid CREATE VIEW statement
	// and that it contains a valid T-SQL query.
	l := &queryClauseListener{}
	antlr.ParseTreeWalkerDefault.Walk(l, parseResult.Tree)
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
				dbMetadata := model.NewDatabaseMetadata(metadata, false, false)
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
func dropViewsInOrder(diff *schema.MetadataDiff, buf *strings.Builder) error {
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
		return nil
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

	return nil
}

// createViewsInOrder creates views in topological order (dependencies first)
func createViewsInOrder(diff *schema.MetadataDiff, buf *strings.Builder) error {
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
		}
		return nil
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
		}
	}

	return nil
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
