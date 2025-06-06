package mssql

import (
	"fmt"
	"strings"

	"github.com/bytebase/bytebase/backend/plugin/schema"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func init() {
	schema.RegisterGenerateMigration(storepb.Engine_MSSQL, generateMigration)
}

func generateMigration(diff *schema.MetadataDiff) (string, error) {
	var buf strings.Builder

	// Safe order for migrations:
	// 1. Drop dependent objects first (in reverse dependency order)
	//    - Drop foreign keys
	//    - Drop indexes
	//    - Drop views (might depend on tables/columns)
	//    - Drop functions/procedures (might depend on tables)
	//    - Drop tables
	//    - Drop schemas
	// 2. Create/Alter objects (in dependency order)
	//    - Create schemas
	//    - Create/Alter tables and columns
	//    - Create indexes
	//    - Create foreign keys
	//    - Create views
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

	// 1.4 Drop views (might depend on tables/columns)
	for _, viewDiff := range diff.ViewChanges {
		if viewDiff.Action == schema.MetadataDiffActionDrop {
			_, _ = buf.WriteString("DROP VIEW [")
			_, _ = buf.WriteString(viewDiff.SchemaName)
			_, _ = buf.WriteString("].[")
			_, _ = buf.WriteString(viewDiff.ViewName)
			_, _ = buf.WriteString("];\nGO\n")
		}
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

	// Add blank line between drop and create phases if we had any drops
	if buf.Len() > 0 {
		_, _ = buf.WriteString("\n")
	}

	// Phase 2: Create/Alter objects
	// 2.1 Create schemas first
	for _, schemaDiff := range diff.SchemaChanges {
		if schemaDiff.Action == schema.MetadataDiffActionCreate {
			// Skip creating dbo schema as it already exists by default
			if strings.ToLower(schemaDiff.SchemaName) == "dbo" {
				continue
			}
			_, _ = buf.WriteString("CREATE SCHEMA [")
			_, _ = buf.WriteString(schemaDiff.SchemaName)
			_, _ = buf.WriteString("];\nGO\n")
		}
	}

	// Add blank line after schema creation if any
	schemasCreated := false
	for _, schemaDiff := range diff.SchemaChanges {
		if schemaDiff.Action == schema.MetadataDiffActionCreate && strings.ToLower(schemaDiff.SchemaName) != "dbo" {
			schemasCreated = true
			break
		}
	}
	if schemasCreated {
		_, _ = buf.WriteString("\n")
	}

	// 2.2 Create new tables
	for _, tableDiff := range diff.TableChanges {
		if tableDiff.Action == schema.MetadataDiffActionCreate {
			createTableSQL, err := generateCreateTable(tableDiff.SchemaName, tableDiff.TableName, tableDiff.NewTable)
			if err != nil {
				return "", err
			}
			_, _ = buf.WriteString(createTableSQL)
			_, _ = buf.WriteString("\n")
		}
	}

	// 2.3 Alter existing tables (add columns, alter columns)
	for _, tableDiff := range diff.TableChanges {
		if tableDiff.Action == schema.MetadataDiffActionAlter {
			alterTableSQL, err := generateAlterTable(tableDiff)
			if err != nil {
				return "", err
			}
			_, _ = buf.WriteString(alterTableSQL)
		}
	}

	// 2.4 Create views (after tables are ready)
	for _, viewDiff := range diff.ViewChanges {
		switch viewDiff.Action {
		case schema.MetadataDiffActionCreate:
			_, _ = buf.WriteString(viewDiff.NewView.Definition)
			if !strings.HasSuffix(strings.TrimSpace(viewDiff.NewView.Definition), ";") {
				_, _ = buf.WriteString(";")
			}
			_, _ = buf.WriteString("\nGO\n")
		case schema.MetadataDiffActionAlter:
			// MSSQL requires dropping and recreating views
			_, _ = buf.WriteString("DROP VIEW [")
			_, _ = buf.WriteString(viewDiff.SchemaName)
			_, _ = buf.WriteString("].[")
			_, _ = buf.WriteString(viewDiff.ViewName)
			_, _ = buf.WriteString("];\nGO\n")
			_, _ = buf.WriteString(viewDiff.NewView.Definition)
			if !strings.HasSuffix(strings.TrimSpace(viewDiff.NewView.Definition), ";") {
				_, _ = buf.WriteString(";")
			}
			_, _ = buf.WriteString("\nGO\n")
		}
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

	// Add foreign keys (after table and referenced tables exist)
	for _, fk := range table.ForeignKeys {
		fkSQL, err := generateAddForeignKey(schemaName, tableName, fk)
		if err != nil {
			return "", err
		}
		_, _ = buf.WriteString("\n")
		_, _ = buf.WriteString(fkSQL)
		_, _ = buf.WriteString(";")
	}

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
