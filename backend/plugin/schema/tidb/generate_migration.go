package tidb

import (
	"fmt"
	"strings"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
)

func init() {
	schema.RegisterGenerateMigration(storepb.Engine_TIDB, generateMigration)
}

func generateMigration(diff *schema.MetadataDiff) (string, error) {
	var buf strings.Builder

	// TiDB doesn't have schemas like PostgreSQL, so we skip schema-level changes
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

	// Drop procedures (not supported in TiDB)
	for _, procDiff := range diff.ProcedureChanges {
		if procDiff.Action == schema.MetadataDiffActionDrop {
			if err := writeDropProcedure(buf, procDiff.ProcedureName); err != nil {
				return err
			}
		}
	}

	// Drop functions (not supported in TiDB)
	for _, funcDiff := range diff.FunctionChanges {
		if funcDiff.Action == schema.MetadataDiffActionDrop {
			if err := writeDropFunction(buf, funcDiff.FunctionName); err != nil {
				return err
			}
		}
	}

	// Drop sequences
	for _, seqDiff := range diff.SequenceChanges {
		if seqDiff.Action == schema.MetadataDiffActionDrop {
			if err := writeDropSequence(buf, seqDiff.SequenceName); err != nil {
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
			// TiDB supports CREATE OR REPLACE for views like MySQL
			if err := writeCreateOrReplaceView(buf, viewDiff.ViewName, viewDiff.NewView); err != nil {
				return err
			}
		default:
			// Ignore other view actions
		}
	}

	// Create sequences
	for _, seqDiff := range diff.SequenceChanges {
		if seqDiff.Action == schema.MetadataDiffActionCreate || seqDiff.Action == schema.MetadataDiffActionAlter {
			if err := writeSequenceDiff(buf, seqDiff); err != nil {
				return err
			}
		}
	}

	// Create functions (not supported in TiDB)
	for _, funcDiff := range diff.FunctionChanges {
		if funcDiff.Action == schema.MetadataDiffActionCreate || funcDiff.Action == schema.MetadataDiffActionAlter {
			if err := writeFunctionDiff(buf, funcDiff); err != nil {
				return err
			}
		}
	}

	// Create procedures (not supported in TiDB)
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
	// Handle table comment changes first
	if tableDiff.OldTable != nil && tableDiff.NewTable != nil {
		if tableDiff.OldTable.Comment != tableDiff.NewTable.Comment {
			if err := writeAlterTableComment(buf, tableDiff.TableName, tableDiff.NewTable.Comment); err != nil {
				return err
			}
		}
	}

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

	return nil
}

// Write functions for various DDL statements

func writeAlterTableComment(buf *strings.Builder, tableName, comment string) error {
	_, _ = buf.WriteString("ALTER TABLE `")
	_, _ = buf.WriteString(tableName)
	_, _ = buf.WriteString("` COMMENT = '")
	_, _ = buf.WriteString(comment)
	_, _ = buf.WriteString("';\n\n")
	return nil
}

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

		// Handle AUTO_INCREMENT and AUTO_RANDOM (TiDB specific)
		if hasAutoIncrement(col) {
			_, _ = buf.WriteString(" AUTO_INCREMENT")
		} else if hasAutoRandom(col) {
			_, _ = buf.WriteString(" /*T![auto_rand] ")
			_, _ = buf.WriteString(col.GetDefault())
			_, _ = buf.WriteString(" */")
		} else if hasDefaultValue(col) && !hasAutoIncrement(col) && !hasAutoRandom(col) {
			if e := getDefaultExpression(col); e != "" {
				_, _ = buf.WriteString(" DEFAULT ")
				_, _ = buf.WriteString(e)
			}
		}

		// Handle ON UPDATE
		if col.OnUpdate != "" {
			_, _ = buf.WriteString(" ON UPDATE ")
			_, _ = buf.WriteString(col.OnUpdate)
		}

		if col.Comment != "" {
			_, _ = buf.WriteString(" COMMENT '")
			_, _ = buf.WriteString(col.Comment)
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
			// Add TiDB specific clustered index comment
			if table.PrimaryKeyType != "" {
				_, _ = buf.WriteString(" /*T![clustered_index] ")
				_, _ = buf.WriteString(table.PrimaryKeyType)
				_, _ = buf.WriteString(" */")
			}
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

	// TiDB specific: Add SHARD_ROW_ID_BITS if present
	if strings.HasPrefix(table.ShardingInfo, "SHARD_BITS=") {
		_, _ = buf.WriteString(" /*T! ")
		_, _ = buf.WriteString(strings.ReplaceAll(table.ShardingInfo, "SHARD_BITS", "SHARD_ROW_ID_BITS"))
		_, _ = buf.WriteString(" */")
	}

	if table.Comment != "" {
		_, _ = buf.WriteString(" COMMENT='")
		_, _ = buf.WriteString(table.Comment)
		_, _ = buf.WriteString("'")
	}

	// Add partition clause if present
	if len(table.Partitions) > 0 {
		if err := writePartitionClause(buf, table.Partitions); err != nil {
			return err
		}
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

// TiDB specific partition clause (similar to MySQL but with TiDB extensions)
func writePartitionClause(buf *strings.Builder, partitions []*storepb.TablePartitionMetadata) error {
	if len(partitions) == 0 {
		return nil
	}

	_, _ = buf.WriteString("\n/*T![auto_rand] PARTITION BY ")

	switch partitions[0].Type {
	case storepb.TablePartitionMetadata_RANGE:
		_, _ = fmt.Fprintf(buf, "RANGE (%s)", partitions[0].Expression)
	case storepb.TablePartitionMetadata_RANGE_COLUMNS:
		_, _ = fmt.Fprintf(buf, "RANGE COLUMNS (%s)", partitions[0].Expression)
	case storepb.TablePartitionMetadata_LIST:
		_, _ = fmt.Fprintf(buf, "LIST (%s)", partitions[0].Expression)
	case storepb.TablePartitionMetadata_LIST_COLUMNS:
		_, _ = fmt.Fprintf(buf, "LIST COLUMNS (%s)", partitions[0].Expression)
	case storepb.TablePartitionMetadata_HASH:
		_, _ = fmt.Fprintf(buf, "HASH (%s)", partitions[0].Expression)
	case storepb.TablePartitionMetadata_KEY:
		_, _ = fmt.Fprintf(buf, "KEY (%s)", partitions[0].Expression)
	default:
		// Unsupported partition type
	}

	// Add partitions count if specified
	if partitions[0].UseDefault != "" && partitions[0].UseDefault != "0" {
		_, _ = fmt.Fprintf(buf, "\nPARTITIONS %s", partitions[0].UseDefault)
	} else {
		// Write individual partitions
		_, _ = buf.WriteString("\n(")
		for i, partition := range partitions {
			if i > 0 {
				_, _ = buf.WriteString(",\n ")
			}
			_, _ = fmt.Fprintf(buf, "PARTITION %s", partition.Name)
			if partition.Value != "" {
				switch partitions[0].Type {
				case storepb.TablePartitionMetadata_RANGE, storepb.TablePartitionMetadata_RANGE_COLUMNS:
					if partition.Value != "MAXVALUE" {
						_, _ = fmt.Fprintf(buf, " VALUES LESS THAN (%s)", partition.Value)
					} else {
						_, _ = fmt.Fprintf(buf, " VALUES LESS THAN %s", partition.Value)
					}
				case storepb.TablePartitionMetadata_LIST, storepb.TablePartitionMetadata_LIST_COLUMNS:
					_, _ = fmt.Fprintf(buf, " VALUES IN (%s)", partition.Value)
				default:
					// No VALUES clause for other partition types like HASH/KEY
				}
			}
		}
		_, _ = buf.WriteString(")")
	}

	_, _ = buf.WriteString(" */")
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

	// Handle AUTO_INCREMENT and AUTO_RANDOM (TiDB specific)
	if hasAutoIncrement(column) {
		_, _ = buf.WriteString(" AUTO_INCREMENT")
	} else if hasAutoRandom(column) {
		_, _ = buf.WriteString(" /*T![auto_rand] ")
		_, _ = buf.WriteString(column.GetDefault())
		_, _ = buf.WriteString(" */")
	} else if hasDefaultValue(column) && !hasAutoIncrement(column) && !hasAutoRandom(column) {
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
		_, _ = buf.WriteString(column.Comment)
		_, _ = buf.WriteString("'")
	}

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

	// Handle AUTO_INCREMENT and AUTO_RANDOM (TiDB specific)
	if hasAutoIncrement(column) {
		_, _ = buf.WriteString(" AUTO_INCREMENT")
	} else if hasAutoRandom(column) {
		_, _ = buf.WriteString(" /*T![auto_rand] ")
		_, _ = buf.WriteString(column.GetDefault())
		_, _ = buf.WriteString(" */")
	} else if hasDefaultValue(column) && !hasAutoIncrement(column) && !hasAutoRandom(column) {
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
		_, _ = buf.WriteString(column.Comment)
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
		_, _ = buf.WriteString(index.Comment)
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
	return nil
}

func writeFunctionDiff(buf *strings.Builder, funcDiff *schema.FunctionDiff) error {
	if funcDiff.Action == schema.MetadataDiffActionCreate {
		_, _ = buf.WriteString(funcDiff.NewFunction.Definition)
		if !strings.HasSuffix(strings.TrimSpace(funcDiff.NewFunction.Definition), ";") {
			_, _ = buf.WriteString(";")
		}
		_, _ = buf.WriteString("\n")
	}
	return nil
}

func writeProcedureDiff(buf *strings.Builder, procDiff *schema.ProcedureDiff) error {
	if procDiff.Action == schema.MetadataDiffActionCreate {
		_, _ = buf.WriteString(procDiff.NewProcedure.Definition)
		if !strings.HasSuffix(strings.TrimSpace(procDiff.NewProcedure.Definition), ";") {
			_, _ = buf.WriteString(";")
		}
		_, _ = buf.WriteString("\n")
	}
	return nil
}

func writeEventDiff(buf *strings.Builder, eventDiff *schema.EventDiff) error {
	if eventDiff.Action == schema.MetadataDiffActionCreate || eventDiff.Action == schema.MetadataDiffActionAlter {
		_, _ = buf.WriteString(eventDiff.NewEvent.Definition)
		if !strings.HasSuffix(strings.TrimSpace(eventDiff.NewEvent.Definition), ";") {
			_, _ = buf.WriteString(";")
		}
		_, _ = buf.WriteString("\n")
	}
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
	for _, seqDiff := range diff.SequenceChanges {
		if seqDiff.Action == schema.MetadataDiffActionCreate || seqDiff.Action == schema.MetadataDiffActionAlter {
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

	if column.GetDefault() != "" {
		return column.GetDefault()
	}

	return ""
}

func hasDefaultValue(column *storepb.ColumnMetadata) bool {
	if column == nil {
		return false
	}

	// Don't treat AUTO_INCREMENT as a default value
	if strings.EqualFold(column.GetDefault(), "AUTO_INCREMENT") || strings.HasPrefix(column.GetDefault(), "AUTO_RANDOM") {
		return false
	}

	return column.Default != ""
}

func hasAutoIncrement(column *storepb.ColumnMetadata) bool {
	if column == nil {
		return false
	}

	// Check if column has AUTO_INCREMENT in default field
	return strings.EqualFold(column.GetDefault(), "AUTO_INCREMENT")
}

func hasAutoRandom(column *storepb.ColumnMetadata) bool {
	if column == nil {
		return false
	}
	return strings.HasPrefix(column.GetDefault(), "AUTO_RANDOM")
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

func writeCreateTrigger(buf *strings.Builder, tableName string, trigger *storepb.TriggerMetadata) error {
	// Construct the complete trigger statement
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
	return nil
}

func writeDropSequence(buf *strings.Builder, sequence string) error {
	_, _ = buf.WriteString("DROP SEQUENCE IF EXISTS `")
	_, _ = buf.WriteString(sequence)
	_, _ = buf.WriteString("`;\n\n")
	return nil
}

func writeSequenceDiff(buf *strings.Builder, seqDiff *schema.SequenceDiff) error {
	switch seqDiff.Action {
	case schema.MetadataDiffActionCreate:
		return writeCreateSequence(buf, seqDiff.NewSequence)
	case schema.MetadataDiffActionAlter:
		return writeAlterSequence(buf, seqDiff.SequenceName, seqDiff.NewSequence)
	default:
		// Ignore other sequence actions
	}
	return nil
}

func writeCreateSequence(buf *strings.Builder, sequence *storepb.SequenceMetadata) error {
	_, _ = buf.WriteString("CREATE SEQUENCE `")
	_, _ = buf.WriteString(sequence.Name)
	_, _ = buf.WriteString("`")

	if sequence.Start != "" && sequence.Start != "1" {
		_, _ = buf.WriteString(" START ")
		_, _ = buf.WriteString(sequence.Start)
	}

	if sequence.Increment != "" && sequence.Increment != "1" {
		_, _ = buf.WriteString(" INCREMENT ")
		_, _ = buf.WriteString(sequence.Increment)
	}

	if sequence.MinValue != "" {
		_, _ = buf.WriteString(" MINVALUE ")
		_, _ = buf.WriteString(sequence.MinValue)
	}

	if sequence.MaxValue != "" {
		_, _ = buf.WriteString(" MAXVALUE ")
		_, _ = buf.WriteString(sequence.MaxValue)
	}

	if sequence.CacheSize != "" && sequence.CacheSize != "1" {
		_, _ = buf.WriteString(" CACHE ")
		_, _ = buf.WriteString(sequence.CacheSize)
	}

	if sequence.Cycle {
		_, _ = buf.WriteString(" CYCLE")
	} else {
		_, _ = buf.WriteString(" NOCYCLE")
	}

	if sequence.Comment != "" {
		_, _ = buf.WriteString(" COMMENT '")
		_, _ = buf.WriteString(sequence.Comment)
		_, _ = buf.WriteString("'")
	}

	_, _ = buf.WriteString(";\n")
	return nil
}

func writeAlterSequence(buf *strings.Builder, sequenceName string, sequence *storepb.SequenceMetadata) error {
	// TiDB does not support ALTER SEQUENCE, so we need to DROP and CREATE
	if err := writeDropSequence(buf, sequenceName); err != nil {
		return err
	}
	return writeCreateSequence(buf, sequence)
}
