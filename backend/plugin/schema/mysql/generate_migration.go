package mysql

import (
	"fmt"
	"slices"
	"strings"

	"github.com/bytebase/bytebase/backend/plugin/schema"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func init() {
	schema.RegisterGenerateMigration(storepb.Engine_MYSQL, generateMigration)
	schema.RegisterGenerateMigration(storepb.Engine_OCEANBASE, generateMigration)
}

func generateMigration(diff *schema.MetadataDiff) (string, error) {
	var buf strings.Builder

	// MySQL doesn't have schemas like PostgreSQL, so we skip schema-level changes
	// We'll focus on table-level changes

	// Phase 1: Drop dependent objects first
	if err := dropObjectsInOrder(diff, &buf); err != nil {
		return "", err
	}

	// Only add blank line if we have drops AND we're about to create something
	dropPhaseHasContent := buf.Len() > 0
	createPhaseWillHaveContent := hasCreateOrAlterObjects(diff)

	if dropPhaseHasContent && createPhaseWillHaveContent {
		_, _ = buf.WriteString("\n")
	}

	// Phase 2: Create/Alter objects
	if err := createObjectsInOrder(diff, &buf); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// dropObjectsInOrder drops all objects in the correct order
func dropObjectsInOrder(diff *schema.MetadataDiff, buf *strings.Builder) error {
	// Drop triggers first (they depend on tables)
	for _, tableDiff := range diff.TableChanges {
		if tableDiff.Action == schema.MetadataDiffActionDrop && tableDiff.OldTable != nil {
			for _, trigger := range tableDiff.OldTable.Triggers {
				if err := writeDropTrigger(buf, trigger.Name); err != nil {
					return err
				}
			}
		} else if tableDiff.Action == schema.MetadataDiffActionAlter {
			// Drop triggers that are being removed from altered tables
			for _, triggerDiff := range tableDiff.TriggerChanges {
				if triggerDiff.Action == schema.MetadataDiffActionDrop {
					if err := writeDropTrigger(buf, triggerDiff.OldTrigger.Name); err != nil {
						return err
					}
				}
			}
		}
	}

	// Drop foreign keys from tables being altered
	for _, tableDiff := range diff.TableChanges {
		if tableDiff.Action == schema.MetadataDiffActionAlter {
			for _, fkDiff := range tableDiff.ForeignKeyChanges {
				if fkDiff.Action == schema.MetadataDiffActionDrop {
					if err := writeDropForeignKey(buf, tableDiff.TableName, fkDiff.OldForeignKey.Name); err != nil {
						return err
					}
				}
			}
		}
	}

	// Create temporary views for views being dropped to handle dependencies
	for _, viewDiff := range diff.ViewChanges {
		if viewDiff.Action == schema.MetadataDiffActionDrop && viewDiff.OldView != nil {
			if err := writeTemporaryViewForDrop(buf, viewDiff.ViewName, viewDiff.OldView); err != nil {
				return err
			}
		}
	}

	// Drop events first (they can reference tables/views)
	for _, eventDiff := range diff.EventChanges {
		if eventDiff.Action == schema.MetadataDiffActionDrop {
			if err := writeDropEvent(buf, eventDiff.EventName); err != nil {
				return err
			}
		}
	}

	// Drop views
	for _, viewDiff := range diff.ViewChanges {
		if viewDiff.Action == schema.MetadataDiffActionDrop {
			if err := writeDropView(buf, viewDiff.ViewName); err != nil {
				return err
			}
		}
	}

	// Drop procedures
	for _, procDiff := range diff.ProcedureChanges {
		if procDiff.Action == schema.MetadataDiffActionDrop {
			if err := writeDropProcedure(buf, procDiff.ProcedureName); err != nil {
				return err
			}
		}
	}

	// Drop functions
	for _, funcDiff := range diff.FunctionChanges {
		if funcDiff.Action == schema.MetadataDiffActionDrop {
			if err := writeDropFunction(buf, funcDiff.FunctionName); err != nil {
				return err
			}
		}
	}

	// Drop tables
	for _, tableDiff := range diff.TableChanges {
		if tableDiff.Action == schema.MetadataDiffActionDrop {
			if err := writeDropTable(buf, tableDiff.TableName); err != nil {
				return err
			}
		}
	}

	// Handle ALTER table drops (constraints, indexes, columns)
	for _, tableDiff := range diff.TableChanges {
		if tableDiff.Action == schema.MetadataDiffActionAlter {
			// Drop check constraints
			for _, checkDiff := range tableDiff.CheckConstraintChanges {
				if checkDiff.Action == schema.MetadataDiffActionDrop {
					if err := writeDropCheckConstraint(buf, tableDiff.TableName, checkDiff.OldCheckConstraint.Name); err != nil {
						return err
					}
				}
			}

			// Drop indexes
			for _, indexDiff := range tableDiff.IndexChanges {
				if indexDiff.Action == schema.MetadataDiffActionDrop {
					if err := writeDropIndex(buf, tableDiff.TableName, indexDiff.OldIndex.Name); err != nil {
						return err
					}
				}
			}

			// Drop columns
			for _, colDiff := range tableDiff.ColumnChanges {
				if colDiff.Action == schema.MetadataDiffActionDrop {
					if err := writeDropColumn(buf, tableDiff.TableName, colDiff.OldColumn.Name); err != nil {
						return err
					}
				}
			}
		}
	}

	return nil
}

// createObjectsInOrder creates all objects in the correct order
func createObjectsInOrder(diff *schema.MetadataDiff, buf *strings.Builder) error {
	// Create tables (without foreign keys first)
	for _, tableDiff := range diff.TableChanges {
		if tableDiff.Action == schema.MetadataDiffActionCreate {
			if err := writeCreateTableWithoutForeignKeys(buf, tableDiff.TableName, tableDiff.NewTable); err != nil {
				return err
			}
		}
	}

	// Add foreign keys after all tables are created
	for _, tableDiff := range diff.TableChanges {
		if tableDiff.Action == schema.MetadataDiffActionCreate && tableDiff.NewTable != nil {
			for _, fk := range tableDiff.NewTable.ForeignKeys {
				if err := writeAddForeignKey(buf, tableDiff.TableName, fk); err != nil {
					return err
				}
			}
		}
	}

	// Handle ALTER table operations
	for _, tableDiff := range diff.TableChanges {
		if tableDiff.Action == schema.MetadataDiffActionAlter {
			if err := generateAlterTable(tableDiff, buf); err != nil {
				return err
			}
		}
	}

	// Create views
	for _, viewDiff := range diff.ViewChanges {
		switch viewDiff.Action {
		case schema.MetadataDiffActionCreate:
			if err := writeCreateView(buf, viewDiff.ViewName, viewDiff.NewView); err != nil {
				return err
			}
		case schema.MetadataDiffActionAlter:
			// MySQL requires CREATE OR REPLACE for views
			if err := writeCreateOrReplaceView(buf, viewDiff.ViewName, viewDiff.NewView); err != nil {
				return err
			}
		}
	}

	// Create functions
	for _, funcDiff := range diff.FunctionChanges {
		if funcDiff.Action == schema.MetadataDiffActionCreate || funcDiff.Action == schema.MetadataDiffActionAlter {
			if err := writeFunctionDiff(buf, funcDiff); err != nil {
				return err
			}
		}
	}

	// Create procedures
	for _, procDiff := range diff.ProcedureChanges {
		if procDiff.Action == schema.MetadataDiffActionCreate || procDiff.Action == schema.MetadataDiffActionAlter {
			if err := writeProcedureDiff(buf, procDiff); err != nil {
				return err
			}
		}
	}

	// Create events
	for _, eventDiff := range diff.EventChanges {
		if eventDiff.Action == schema.MetadataDiffActionCreate || eventDiff.Action == schema.MetadataDiffActionAlter {
			if err := writeEventDiff(buf, eventDiff); err != nil {
				return err
			}
		}
	}

	return nil
}

func generateAlterTable(tableDiff *schema.TableDiff, buf *strings.Builder) error {
	// Add columns first
	for _, colDiff := range tableDiff.ColumnChanges {
		if colDiff.Action == schema.MetadataDiffActionCreate {
			if err := writeAddColumn(buf, tableDiff.TableName, colDiff.NewColumn); err != nil {
				return err
			}
		}
	}

	// Modify columns
	for _, colDiff := range tableDiff.ColumnChanges {
		if colDiff.Action == schema.MetadataDiffActionAlter {
			if err := writeModifyColumn(buf, tableDiff.TableName, colDiff.NewColumn); err != nil {
				return err
			}
		}
	}

	// Add indexes
	for _, indexDiff := range tableDiff.IndexChanges {
		if indexDiff.Action == schema.MetadataDiffActionCreate {
			if indexDiff.NewIndex.Primary {
				if err := writeAddPrimaryKey(buf, tableDiff.TableName, indexDiff.NewIndex); err != nil {
					return err
				}
			} else if indexDiff.NewIndex.Unique {
				if err := writeAddUniqueKey(buf, tableDiff.TableName, indexDiff.NewIndex); err != nil {
					return err
				}
			} else {
				if err := writeCreateIndex(buf, tableDiff.TableName, indexDiff.NewIndex); err != nil {
					return err
				}
			}
		}
	}

	// Add check constraints
	for _, checkDiff := range tableDiff.CheckConstraintChanges {
		if checkDiff.Action == schema.MetadataDiffActionCreate {
			if err := writeAddCheckConstraint(buf, tableDiff.TableName, checkDiff.NewCheckConstraint); err != nil {
				return err
			}
		}
	}

	// Add foreign keys last
	for _, fkDiff := range tableDiff.ForeignKeyChanges {
		if fkDiff.Action == schema.MetadataDiffActionCreate {
			if err := writeAddForeignKey(buf, tableDiff.TableName, fkDiff.NewForeignKey); err != nil {
				return err
			}
		}
	}

	// Add triggers last
	for _, triggerDiff := range tableDiff.TriggerChanges {
		if triggerDiff.Action == schema.MetadataDiffActionCreate {
			if err := writeCreateTrigger(buf, tableDiff.TableName, triggerDiff.NewTrigger); err != nil {
				return err
			}
		}
	}

	// Handle table comment changes
	if tableDiff.OldTable != nil && tableDiff.NewTable != nil &&
		tableDiff.OldTable.Comment != tableDiff.NewTable.Comment {
		if err := writeAlterTableComment(buf, tableDiff.TableName, tableDiff.NewTable.Comment); err != nil {
			return err
		}
	}

	return nil
}

// Write functions for various DDL statements

func writeDropTrigger(buf *strings.Builder, trigger string) error {
	_, _ = buf.WriteString("DROP TRIGGER IF EXISTS `")
	_, _ = buf.WriteString(trigger)
	_, _ = buf.WriteString("`;\n\n")
	return nil
}

func writeDropForeignKey(buf *strings.Builder, table, constraint string) error {
	_, _ = buf.WriteString("ALTER TABLE `")
	_, _ = buf.WriteString(table)
	_, _ = buf.WriteString("` DROP FOREIGN KEY `")
	_, _ = buf.WriteString(constraint)
	_, _ = buf.WriteString("`;\n\n")
	return nil
}

func writeDropView(buf *strings.Builder, view string) error {
	_, _ = buf.WriteString("DROP VIEW IF EXISTS `")
	_, _ = buf.WriteString(view)
	_, _ = buf.WriteString("`;\n\n")
	return nil
}

func writeDropProcedure(buf *strings.Builder, procedure string) error {
	_, _ = buf.WriteString("DROP PROCEDURE IF EXISTS `")
	_, _ = buf.WriteString(procedure)
	_, _ = buf.WriteString("`;\n\n")
	return nil
}

func writeDropFunction(buf *strings.Builder, function string) error {
	_, _ = buf.WriteString("DROP FUNCTION IF EXISTS `")
	_, _ = buf.WriteString(function)
	_, _ = buf.WriteString("`;\n\n")
	return nil
}

func writeDropEvent(buf *strings.Builder, event string) error {
	_, _ = buf.WriteString("DROP EVENT IF EXISTS `")
	_, _ = buf.WriteString(event)
	_, _ = buf.WriteString("`;\n\n")
	return nil
}

func writeDropTable(buf *strings.Builder, table string) error {
	_, _ = buf.WriteString("DROP TABLE IF EXISTS `")
	_, _ = buf.WriteString(table)
	_, _ = buf.WriteString("`;\n\n")
	return nil
}

func writeDropCheckConstraint(buf *strings.Builder, table, constraint string) error {
	_, _ = buf.WriteString("ALTER TABLE `")
	_, _ = buf.WriteString(table)
	_, _ = buf.WriteString("` DROP CHECK `")
	_, _ = buf.WriteString(constraint)
	_, _ = buf.WriteString("`;\n\n")
	return nil
}

func writeDropIndex(buf *strings.Builder, table, index string) error {
	_, _ = buf.WriteString("DROP INDEX `")
	_, _ = buf.WriteString(index)
	_, _ = buf.WriteString("` ON `")
	_, _ = buf.WriteString(table)
	_, _ = buf.WriteString("`;\n\n")
	return nil
}

func writeDropColumn(buf *strings.Builder, table, column string) error {
	_, _ = buf.WriteString("ALTER TABLE `")
	_, _ = buf.WriteString(table)
	_, _ = buf.WriteString("` DROP COLUMN `")
	_, _ = buf.WriteString(column)
	_, _ = buf.WriteString("`;\n\n")
	return nil
}

func writeCreateTableWithoutForeignKeys(buf *strings.Builder, tableName string, table *storepb.TableMetadata) error {
	_, _ = buf.WriteString("CREATE TABLE IF NOT EXISTS `")
	_, _ = buf.WriteString(tableName)
	_, _ = buf.WriteString("` (\n")

	// Write columns
	for i, col := range table.Columns {
		if i > 0 {
			_, _ = buf.WriteString(",\n")
		}
		_, _ = buf.WriteString("  `")
		_, _ = buf.WriteString(col.Name)
		_, _ = buf.WriteString("` ")
		_, _ = buf.WriteString(col.Type)

		if col.CharacterSet != "" {
			_, _ = buf.WriteString(" CHARACTER SET ")
			_, _ = buf.WriteString(col.CharacterSet)
		}
		if col.Collation != "" {
			_, _ = buf.WriteString(" COLLATE ")
			_, _ = buf.WriteString(col.Collation)
		}

		if !col.Nullable {
			_, _ = buf.WriteString(" NOT NULL")
		}

		// Handle AUTO_INCREMENT before default value
		if hasAutoIncrement(col) {
			_, _ = buf.WriteString(" AUTO_INCREMENT")
		} else if hasDefaultValue(col) && !hasAutoIncrement(col) {
			_, _ = buf.WriteString(" DEFAULT ")
			_, _ = buf.WriteString(getDefaultExpression(col))
		}

		// Handle ON UPDATE
		if col.OnUpdate != "" {
			_, _ = buf.WriteString(" ON UPDATE ")
			_, _ = buf.WriteString(col.OnUpdate)
		}

		if col.Comment != "" {
			_, _ = buf.WriteString(" COMMENT '")
			_, _ = buf.WriteString(escapeString(col.Comment))
			_, _ = buf.WriteString("'")
		}
	}

	// Write primary key constraint inline if exists
	for _, index := range table.Indexes {
		if index.Primary {
			_, _ = buf.WriteString(",\n  PRIMARY KEY (")
			for i, expr := range index.Expressions {
				if i > 0 {
					_, _ = buf.WriteString(", ")
				}
				_, _ = buf.WriteString("`")
				_, _ = buf.WriteString(expr)
				_, _ = buf.WriteString("`")
			}
			_, _ = buf.WriteString(")")
			break
		}
	}

	// Write unique constraints inline
	for _, index := range table.Indexes {
		if index.Unique && !index.Primary {
			_, _ = buf.WriteString(",\n  UNIQUE KEY `")
			_, _ = buf.WriteString(index.Name)
			_, _ = buf.WriteString("` (")
			for i, expr := range index.Expressions {
				if i > 0 {
					_, _ = buf.WriteString(", ")
				}
				_, _ = buf.WriteString("`")
				_, _ = buf.WriteString(expr)
				_, _ = buf.WriteString("`")
			}
			_, _ = buf.WriteString(")")
		}
	}

	// Write check constraints
	for _, check := range table.CheckConstraints {
		_, _ = buf.WriteString(",\n  CONSTRAINT `")
		_, _ = buf.WriteString(check.Name)
		_, _ = buf.WriteString("` CHECK (")
		_, _ = buf.WriteString(check.Expression)
		_, _ = buf.WriteString(")")
	}

	_, _ = buf.WriteString("\n)")

	// Write table options
	if table.Engine != "" {
		_, _ = buf.WriteString(" ENGINE=")
		_, _ = buf.WriteString(table.Engine)
	}
	if table.Charset != "" {
		_, _ = buf.WriteString(" DEFAULT CHARSET=")
		_, _ = buf.WriteString(table.Charset)
	}
	if table.Collation != "" {
		_, _ = buf.WriteString(" COLLATE=")
		_, _ = buf.WriteString(table.Collation)
	}
	if table.Comment != "" {
		_, _ = buf.WriteString(" COMMENT='")
		_, _ = buf.WriteString(escapeString(table.Comment))
		_, _ = buf.WriteString("'")
	}

	_, _ = buf.WriteString(";\n")

	// Create non-unique indexes separately
	for _, index := range table.Indexes {
		if !index.Primary && !index.Unique {
			if err := writeCreateIndex(buf, tableName, index); err != nil {
				return err
			}
		}
	}

	// Note: Foreign keys are NOT created here - they will be added separately

	return nil
}

func writeAddColumn(buf *strings.Builder, table string, column *storepb.ColumnMetadata) error {
	_, _ = buf.WriteString("ALTER TABLE `")
	_, _ = buf.WriteString(table)
	_, _ = buf.WriteString("` ADD COLUMN `")
	_, _ = buf.WriteString(column.Name)
	_, _ = buf.WriteString("` ")
	_, _ = buf.WriteString(column.Type)

	if column.CharacterSet != "" {
		_, _ = buf.WriteString(" CHARACTER SET ")
		_, _ = buf.WriteString(column.CharacterSet)
	}
	if column.Collation != "" {
		_, _ = buf.WriteString(" COLLATE ")
		_, _ = buf.WriteString(column.Collation)
	}

	if !column.Nullable {
		_, _ = buf.WriteString(" NOT NULL")
	}

	// Handle AUTO_INCREMENT before default value
	if hasAutoIncrement(column) {
		_, _ = buf.WriteString(" AUTO_INCREMENT")
	} else if hasDefaultValue(column) && !hasAutoIncrement(column) {
		_, _ = buf.WriteString(" DEFAULT ")
		_, _ = buf.WriteString(getDefaultExpression(column))
	}

	// Handle ON UPDATE
	if column.OnUpdate != "" {
		_, _ = buf.WriteString(" ON UPDATE ")
		_, _ = buf.WriteString(column.OnUpdate)
	}

	if column.Comment != "" {
		_, _ = buf.WriteString(" COMMENT '")
		_, _ = buf.WriteString(escapeString(column.Comment))
		_, _ = buf.WriteString("'")
	}

	// TODO: Add column position support (FIRST, AFTER column_name)

	_, _ = buf.WriteString(";\n")
	return nil
}

func writeModifyColumn(buf *strings.Builder, table string, column *storepb.ColumnMetadata) error {
	_, _ = buf.WriteString("ALTER TABLE `")
	_, _ = buf.WriteString(table)
	_, _ = buf.WriteString("` MODIFY COLUMN `")
	_, _ = buf.WriteString(column.Name)
	_, _ = buf.WriteString("` ")
	_, _ = buf.WriteString(column.Type)

	if column.CharacterSet != "" {
		_, _ = buf.WriteString(" CHARACTER SET ")
		_, _ = buf.WriteString(column.CharacterSet)
	}
	if column.Collation != "" {
		_, _ = buf.WriteString(" COLLATE ")
		_, _ = buf.WriteString(column.Collation)
	}

	if !column.Nullable {
		_, _ = buf.WriteString(" NOT NULL")
	}

	// Handle AUTO_INCREMENT before default value
	if hasAutoIncrement(column) {
		_, _ = buf.WriteString(" AUTO_INCREMENT")
	} else if hasDefaultValue(column) && !hasAutoIncrement(column) {
		_, _ = buf.WriteString(" DEFAULT ")
		_, _ = buf.WriteString(getDefaultExpression(column))
	}

	// Handle ON UPDATE
	if column.OnUpdate != "" {
		_, _ = buf.WriteString(" ON UPDATE ")
		_, _ = buf.WriteString(column.OnUpdate)
	}

	if column.Comment != "" {
		_, _ = buf.WriteString(" COMMENT '")
		_, _ = buf.WriteString(escapeString(column.Comment))
		_, _ = buf.WriteString("'")
	}

	_, _ = buf.WriteString(";\n")
	return nil
}

func writeCreateIndex(buf *strings.Builder, table string, index *storepb.IndexMetadata) error {
	_, _ = buf.WriteString("CREATE ")
	// Handle special index types
	if strings.ToUpper(index.Type) == "FULLTEXT" {
		_, _ = buf.WriteString("FULLTEXT ")
	} else if strings.ToUpper(index.Type) == "SPATIAL" {
		_, _ = buf.WriteString("SPATIAL ")
	}

	_, _ = buf.WriteString("INDEX `")
	_, _ = buf.WriteString(index.Name)
	_, _ = buf.WriteString("` ON `")
	_, _ = buf.WriteString(table)
	_, _ = buf.WriteString("` (")

	for i, expr := range index.Expressions {
		if i > 0 {
			_, _ = buf.WriteString(", ")
		}
		_, _ = buf.WriteString("`")
		_, _ = buf.WriteString(expr)
		_, _ = buf.WriteString("`")

		// Handle column length for text/blob columns
		if i < len(index.KeyLength) && index.KeyLength[i] > 0 {
			_, _ = fmt.Fprintf(buf, "(%d)", index.KeyLength[i])
		}
	}

	_, _ = buf.WriteString(")")

	// Add index type if specified and not default
	if index.Type != "" && index.Type != "BTREE" &&
		strings.ToUpper(index.Type) != "FULLTEXT" &&
		strings.ToUpper(index.Type) != "SPATIAL" {
		_, _ = buf.WriteString(" USING ")
		_, _ = buf.WriteString(index.Type)
	}

	if index.Comment != "" {
		_, _ = buf.WriteString(" COMMENT '")
		_, _ = buf.WriteString(escapeString(index.Comment))
		_, _ = buf.WriteString("'")
	}

	_, _ = buf.WriteString(";\n")
	return nil
}

func writeAddPrimaryKey(buf *strings.Builder, table string, index *storepb.IndexMetadata) error {
	_, _ = buf.WriteString("ALTER TABLE `")
	_, _ = buf.WriteString(table)
	_, _ = buf.WriteString("` ADD PRIMARY KEY (")

	for i, expr := range index.Expressions {
		if i > 0 {
			_, _ = buf.WriteString(", ")
		}
		_, _ = buf.WriteString("`")
		_, _ = buf.WriteString(expr)
		_, _ = buf.WriteString("`")
	}

	_, _ = buf.WriteString(");\n")
	return nil
}

func writeAddUniqueKey(buf *strings.Builder, table string, index *storepb.IndexMetadata) error {
	_, _ = buf.WriteString("ALTER TABLE `")
	_, _ = buf.WriteString(table)
	_, _ = buf.WriteString("` ADD UNIQUE KEY `")
	_, _ = buf.WriteString(index.Name)
	_, _ = buf.WriteString("` (")

	for i, expr := range index.Expressions {
		if i > 0 {
			_, _ = buf.WriteString(", ")
		}
		_, _ = buf.WriteString("`")
		_, _ = buf.WriteString(expr)
		_, _ = buf.WriteString("`")
	}

	_, _ = buf.WriteString(");\n")
	return nil
}

func writeAddCheckConstraint(buf *strings.Builder, table string, check *storepb.CheckConstraintMetadata) error {
	_, _ = buf.WriteString("ALTER TABLE `")
	_, _ = buf.WriteString(table)
	_, _ = buf.WriteString("` ADD CONSTRAINT `")
	_, _ = buf.WriteString(check.Name)
	_, _ = buf.WriteString("` CHECK (")
	_, _ = buf.WriteString(check.Expression)
	_, _ = buf.WriteString(");\n")
	return nil
}

func writeAddForeignKey(buf *strings.Builder, table string, fk *storepb.ForeignKeyMetadata) error {
	_, _ = buf.WriteString("ALTER TABLE `")
	_, _ = buf.WriteString(table)
	_, _ = buf.WriteString("` ADD CONSTRAINT `")
	_, _ = buf.WriteString(fk.Name)
	_, _ = buf.WriteString("` FOREIGN KEY (")

	for i, col := range fk.Columns {
		if i > 0 {
			_, _ = buf.WriteString(", ")
		}
		_, _ = buf.WriteString("`")
		_, _ = buf.WriteString(col)
		_, _ = buf.WriteString("`")
	}

	_, _ = buf.WriteString(") REFERENCES `")
	_, _ = buf.WriteString(fk.ReferencedTable)
	_, _ = buf.WriteString("` (")

	for i, col := range fk.ReferencedColumns {
		if i > 0 {
			_, _ = buf.WriteString(", ")
		}
		_, _ = buf.WriteString("`")
		_, _ = buf.WriteString(col)
		_, _ = buf.WriteString("`")
	}

	_, _ = buf.WriteString(")")

	if fk.OnUpdate != "" {
		_, _ = buf.WriteString(" ON UPDATE ")
		_, _ = buf.WriteString(fk.OnUpdate)
	}
	if fk.OnDelete != "" {
		_, _ = buf.WriteString(" ON DELETE ")
		_, _ = buf.WriteString(fk.OnDelete)
	}

	_, _ = buf.WriteString(";\n")
	return nil
}

func writeCreateView(buf *strings.Builder, viewName string, view *storepb.ViewMetadata) error {
	_, _ = buf.WriteString("CREATE VIEW `")
	_, _ = buf.WriteString(viewName)
	_, _ = buf.WriteString("` AS ")
	_, _ = buf.WriteString(view.Definition)

	if !strings.HasSuffix(strings.TrimSpace(view.Definition), ";") {
		_, _ = buf.WriteString(";")
	}
	_, _ = buf.WriteString("\n")

	// Note: MySQL doesn't support adding comments directly to views via DDL
	// View comments are stored in information_schema.VIEWS but cannot be set via CREATE VIEW
	// The comment field is read-only
	return nil
}

func writeCreateOrReplaceView(buf *strings.Builder, viewName string, view *storepb.ViewMetadata) error {
	_, _ = buf.WriteString("CREATE OR REPLACE VIEW `")
	_, _ = buf.WriteString(viewName)
	_, _ = buf.WriteString("` AS ")
	_, _ = buf.WriteString(view.Definition)

	if !strings.HasSuffix(strings.TrimSpace(view.Definition), ";") {
		_, _ = buf.WriteString(";")
	}
	_, _ = buf.WriteString("\n")

	// Note: MySQL doesn't support adding comments directly to views via DDL
	// View comments are stored in information_schema.VIEWS but cannot be set via CREATE VIEW
	// The comment field is read-only
	return nil
}

func writeFunctionDiff(buf *strings.Builder, funcDiff *schema.FunctionDiff) error {
	if funcDiff.Action == schema.MetadataDiffActionCreate || funcDiff.Action == schema.MetadataDiffActionAlter {
		// Don't add DELIMITER statements - the definition should already be complete
		_, _ = buf.WriteString(funcDiff.NewFunction.Definition)
		if !strings.HasSuffix(strings.TrimSpace(funcDiff.NewFunction.Definition), ";") {
			_, _ = buf.WriteString(";")
		}
		_, _ = buf.WriteString("\n")

		// Note: MySQL doesn't support ALTER FUNCTION ... COMMENT syntax
		// Function comments must be set during CREATE FUNCTION statement
		// The definition should already include the comment if needed
	}
	return nil
}

func writeProcedureDiff(buf *strings.Builder, procDiff *schema.ProcedureDiff) error {
	if procDiff.Action == schema.MetadataDiffActionCreate || procDiff.Action == schema.MetadataDiffActionAlter {
		// Don't add DELIMITER statements - the definition should already be complete
		_, _ = buf.WriteString(procDiff.NewProcedure.Definition)
		if !strings.HasSuffix(strings.TrimSpace(procDiff.NewProcedure.Definition), ";") {
			_, _ = buf.WriteString(";")
		}
		_, _ = buf.WriteString("\n")

		// Note: MySQL doesn't support ALTER PROCEDURE ... COMMENT syntax
		// Procedure comments must be set during CREATE PROCEDURE statement
		// The definition should already include the comment if needed
	}
	return nil
}

func writeEventDiff(buf *strings.Builder, eventDiff *schema.EventDiff) error {
	if eventDiff.Action == schema.MetadataDiffActionCreate || eventDiff.Action == schema.MetadataDiffActionAlter {
		// Write the event definition
		_, _ = buf.WriteString(eventDiff.NewEvent.Definition)
		if !strings.HasSuffix(strings.TrimSpace(eventDiff.NewEvent.Definition), ";") {
			_, _ = buf.WriteString(";")
		}
		_, _ = buf.WriteString("\n")

		// Note: MySQL doesn't support ALTER EVENT ... COMMENT syntax
		// Event comments must be set during CREATE EVENT statement
		// The definition should already include the comment if needed
	}
	return nil
}

func writeAlterTableComment(buf *strings.Builder, tableName, comment string) error {
	_, _ = buf.WriteString("ALTER TABLE `")
	_, _ = buf.WriteString(tableName)
	_, _ = buf.WriteString("` COMMENT = '")
	_, _ = buf.WriteString(escapeString(comment))
	_, _ = buf.WriteString("';\n")
	return nil
}

// Helper functions

func hasCreateOrAlterObjects(diff *schema.MetadataDiff) bool {
	for _, tableDiff := range diff.TableChanges {
		if tableDiff.Action == schema.MetadataDiffActionCreate || tableDiff.Action == schema.MetadataDiffActionAlter {
			return true
		}
	}
	for _, viewDiff := range diff.ViewChanges {
		if viewDiff.Action == schema.MetadataDiffActionCreate || viewDiff.Action == schema.MetadataDiffActionAlter {
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
	for _, eventDiff := range diff.EventChanges {
		if eventDiff.Action == schema.MetadataDiffActionCreate || eventDiff.Action == schema.MetadataDiffActionAlter {
			return true
		}
	}
	return false
}

func getDefaultExpression(column *storepb.ColumnMetadata) string {
	if column == nil {
		return ""
	}

	// Check for expression-based default first
	if column.DefaultExpression != "" {
		return column.DefaultExpression
	}

	// Check for string default value
	if column.Default != "" {
		// Check if it's a numeric value or needs quotes
		if isNumeric(column.Default) || isKeyword(column.Default) {
			return column.Default
		}
		return fmt.Sprintf("'%s'", escapeString(column.Default))
	}

	// Check for NULL default
	if column.DefaultNull {
		return "NULL"
	}

	return ""
}

func hasDefaultValue(column *storepb.ColumnMetadata) bool {
	if column == nil {
		return false
	}
	return column.DefaultNull || column.DefaultExpression != "" || (column.Default != "")
}

func hasAutoIncrement(column *storepb.ColumnMetadata) bool {
	if column == nil {
		return false
	}
	// Check if column has AUTO_INCREMENT in default expression
	return strings.EqualFold(column.GetDefaultExpression(), "AUTO_INCREMENT")
}

func escapeString(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "'", "\\'")
	return s
}

func writeTemporaryViewForDrop(buf *strings.Builder, viewName string, view *storepb.ViewMetadata) error {
	// Create a temporary view with SELECT 1 AS column_name structure
	// to satisfy other views that depend on this view
	_, _ = buf.WriteString("CREATE OR REPLACE VIEW `")
	_, _ = buf.WriteString(viewName)
	_, _ = buf.WriteString("` AS SELECT")

	for i, column := range view.Columns {
		if i > 0 {
			_, _ = buf.WriteString(",")
		}
		_, _ = buf.WriteString(" 1 AS `")
		_, _ = buf.WriteString(column.Name)
		_, _ = buf.WriteString("`")
	}

	// If no columns, create a dummy view
	if len(view.Columns) == 0 {
		_, _ = buf.WriteString(" 1")
	}

	_, _ = buf.WriteString(";\n")
	return nil
}

func isNumeric(s string) bool {
	// Simple check for numeric values
	_, err := fmt.Sscanf(s, "%f", new(float64))
	return err == nil
}

func isKeyword(s string) bool {
	// Check for common MySQL keywords that don't need quotes
	keywords := []string{"CURRENT_TIMESTAMP", "CURRENT_DATE", "CURRENT_TIME", "NULL"}
	upper := strings.ToUpper(s)
	return slices.Contains(keywords, upper)
}

func writeCreateTrigger(buf *strings.Builder, tableName string, trigger *storepb.TriggerMetadata) error {
	// Don't add DELIMITER statements - construct the complete trigger statement
	_, _ = buf.WriteString("CREATE TRIGGER `")
	_, _ = buf.WriteString(trigger.Name)
	_, _ = buf.WriteString("` ")
	_, _ = buf.WriteString(trigger.Timing)
	_, _ = buf.WriteString(" ")
	_, _ = buf.WriteString(trigger.Event)
	_, _ = buf.WriteString(" ON `")
	_, _ = buf.WriteString(tableName)
	_, _ = buf.WriteString("` FOR EACH ROW ")
	_, _ = buf.WriteString(trigger.Body)
	if !strings.HasSuffix(strings.TrimSpace(trigger.Body), ";") {
		_, _ = buf.WriteString(";")
	}
	_, _ = buf.WriteString("\n")

	// Note: MySQL doesn't support specifying comments in CREATE TRIGGER syntax
	// Trigger comments are retrieved from INFORMATION_SCHEMA.TRIGGERS but cannot be set via DDL
	// The comment field is read-only and set by MySQL itself
	return nil
}
