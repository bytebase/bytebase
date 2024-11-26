package mssql

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log/slog"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/db/util"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

const (
	defaultSchema = "dbo"
)

// Dump dumps the database.
func (driver *Driver) Dump(ctx context.Context, out io.Writer, _ *storepb.DatabaseSchemaMetadata) error {
	txn, err := driver.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return err
	}
	defer txn.Rollback()

	if err := driver.dumpDatabaseTxn(ctx, txn, out); err != nil {
		return errors.Wrapf(err, "failed to dump database %q", driver.databaseName)
	}

	return txn.Commit()
}

const (
	dumpTableSQL = `
	SELECT
	    o.name,
	    o.object_id,
	    o.type_desc,
	    o.create_date,
	    o.modify_date,
	    t.lock_escalation,
	    ct.is_track_columns_updated_on,
	    st.row_count AS rows,
	    CAST(ep.value AS NVARCHAR(MAX)) AS comment,
	    s.name AS schemaname ,
	    IDENT_CURRENT(QUOTENAME(s.name) + '.' + QUOTENAME(o.name)) AS current_value
	FROM sys.objects o
	    LEFT JOIN sys.schemas s ON o.schema_id = s.schema_id
	    LEFT JOIN sys.tables t ON o.object_id = t.object_id
	    LEFT JOIN sys.extended_properties ep ON (o.object_id = ep.major_id AND ep.class = 1 AND ep.minor_id = 0 AND ep.name = 'MS_Description')
	    LEFT JOIN (SELECT object_id, SUM(ROWS) row_count FROM sys.partitions WHERE index_id < 2 GROUP BY object_id) st ON o.object_id = st.object_id
	    LEFT JOIN sys.change_tracking_tables ct ON ct.object_id = o.object_id
	WHERE o.type = 'U' AND s.name in (%s)
	ORDER BY s.name ASC, o.name ASC
	`
	dumpColumnSQL = `
	SELECT
		s.name AS schema_name,
		OBJECT_NAME(c.object_id) AS table_name,
		c.name AS column_name,
		t.name AS type_name,
		(SELECT ts.name FROM sys.schemas ts WHERE ts.schema_id = t.schema_id) AS type_schema,
		t.is_user_defined,
		c.is_computed,
		cc.definition,
		cc.is_persisted,
		c.max_length,
		c.precision AS precision,
		c.scale,
		c.collation_name,
		c.is_nullable,
		c.is_rowguidcol,
		c.is_identity,
		c.is_filestream,
		d.default_schema,
		d.default_name,
		d.definition AS default_value,
		c.is_sparse,
		c.is_column_set,
		CAST(p.[value] AS nvarchar(4000)) AS comment,
		id.seed_value AS seed_value,
		id.increment_value AS increment_value ,
		COLUMNPROPERTY(o.object_id, (SELECT TOP 1 name FROM sys.columns c WHERE c.object_id = o.object_id AND c.is_identity = 1), 'IsIdNotForRepl') AS identity_not_for_replication,
		CAST(OBJECTPROPERTY(d.object_id, 'IsDefaultCnst') AS bit) AS is_defaultcnst
	FROM sys.columns c
		LEFT JOIN sys.computed_columns cc ON cc.object_id = c.object_id AND cc.column_id = c.column_id
		LEFT JOIN sys.types t ON c.user_type_id = t.user_type_id
		LEFT JOIN (SELECT so.object_id, sc.name as default_schema, so.name AS default_name, dc.definition FROM sys.objects so LEFT JOIN sys.schemas sc ON sc.schema_id = so.schema_id LEFT JOIN sys.default_constraints dc ON dc.object_id = so.object_id WHERE so.type = 'D') d ON d.object_id = c.default_object_id
		LEFT JOIN sys.objects o ON o.object_id = c.object_id
		LEFT JOIN sys.schemas s ON s.schema_id = o.schema_id
		LEFT JOIN sys.identity_columns id ON c.object_id = id.object_id AND c.column_id = id.column_id
		LEFT JOIN sys.extended_properties p ON p.major_id = c.object_id AND p.minor_id = c.column_id AND p.class = 1 AND p.name = 'MS_Description'
	WHERE s.name in (%s)
	ORDER BY s.name ASC, c.object_id ASC, c.column_id ASC 
	`
	dumpForeignKeySQL = `
	SELECT
		t.schema_name,
	    t.name AS table_name,
	    f.name,
	    co.type AS constraint_type,
	    t.name AS parent_table,
	    t.type AS parent_table_type,
	    f.referenced_schema,
	    f.referenced_table,
	    f.is_disabled,
	    f.is_not_for_replication,
	    f.comment,
	    f.delete_referential_action,
	    f.update_referential_action,
	    f.parent_column,
	    f.referenced_column
	FROM (SELECT s.name AS schema_name, o.name, o.object_id, o.type FROM sys.all_objects o LEFT JOIN sys.schemas s ON s.schema_id = o.schema_id WHERE s.name in (%s) ) t
	    INNER JOIN (SELECT fk.object_id, fk.parent_object_id, fk.name, OBJECT_SCHEMA_NAME(fk.referenced_object_id) AS referenced_schema, OBJECT_NAME(fk.referenced_object_id) AS referenced_table, fk.is_disabled, fk.is_not_for_replication, fk.delete_referential_action, fk.update_referential_action, fc.parent_column, CAST(p.[value] AS nvarchar(4000)) AS comment, fc.referenced_column FROM sys.foreign_keys fk LEFT JOIN (SELECT fkc.constraint_object_id, pc.name AS parent_column, rc.name AS referenced_column FROM sys.foreign_key_columns fkc LEFT JOIN sys.all_columns pc ON pc.object_id = fkc.parent_object_id AND pc.column_id = fkc.parent_column_id LEFT JOIN sys.all_columns rc ON rc.object_id = fkc.referenced_object_id AND rc.column_id = fkc.referenced_column_id) fc ON fc.constraint_object_id = fk.object_id LEFT JOIN sys.extended_properties p ON p.major_id = fk.object_id AND p.minor_id = 0 AND p.name = 'MS_Description' ) f ON f.parent_object_id = t.object_id
	    LEFT JOIN sys.objects co ON co.object_id = f.object_id
	ORDER BY t.schema_name ASC, t.object_id ASC, f.object_id ASC
	`
	dumpCheckConstraintSQL = `
	SELECT
		t.schema_name,
	    t.name AS table_name,
	    c.name,
	    co.type AS constraint_type,
	    t.type AS table_type,
	    c.object_id,
	    c.comment,
	    c.is_disabled,
	    c.is_not_for_replication,
	    c.definition
	FROM
	    (SELECT s.name as schema_name, o.name, o.object_id, o.type FROM sys.all_objects o LEFT JOIN sys.schemas s ON s.schema_id = o.schema_id WHERE s.name in (%s) ) t
	        INNER JOIN (SELECT ch.name, ch.object_id, ch.parent_object_id, ch.is_disabled, CAST(p.[value] AS nvarchar(4000)) AS comment, ch.is_not_for_replication, ch.definition FROM sys.check_constraints ch LEFT JOIN sys.extended_properties p ON p.major_id = ch.object_id AND p.minor_id = 0 AND p.name = 'MS_Description') c ON c.parent_object_id = t.object_id
	        LEFT JOIN sys.objects co ON co.object_id = c.object_id
	ORDER BY t.schema_name ASC, t.object_id ASC, c.object_id ASC
	`
	dumpKeySQL = `
	SELECT
		s.name AS schema_name,
	    o.name AS table_name,
	    i.name,
	    co.type AS constraint_type,
	    c.name AS column_name,
	    ic.partition_ordinal,
	    ic.is_descending_key,
	    i.is_primary_key,
	    i.is_unique_constraint,
	    i.type_desc,
	    i.is_disabled,
	    i.is_padded,
	    i.fill_factor,
	    i.ignore_dup_key,
	    CAST(CASE WHEN o.is_ms_shipped = 1 THEN 1 WHEN (SELECT major_id FROM sys.extended_properties WHERE major_id = o.object_id AND minor_id = 0 AND class = 1 AND name = 'microsoft_database_tools_support') IS NOT NULL THEN 1 ELSE 0 END AS bit) AS is_system_object,
	    d.name AS data_space_name,
	    d.type AS data_space_type,
	    CAST(p.[value] AS nvarchar(4000)) AS comment,
	    i.allow_row_locks,
	    i.allow_page_locks,
	    st.no_recompute
	FROM
	    sys.indexes i
	        LEFT JOIN sys.index_columns ic ON ic.object_id = i.object_id AND ic.index_id = i.index_id
	        LEFT JOIN sys.columns c ON c.object_id = ic.object_id AND c.column_id = ic.column_id
	        LEFT JOIN sys.stats st ON st.object_id = i.object_id AND st.name = i.name
	        LEFT JOIN sys.objects co ON co.parent_object_id = i.object_id AND co.name = i.name LEFT JOIN sys.objects o ON o.object_id = i.object_id
	        LEFT JOIN sys.schemas s ON s.schema_id = o.schema_id
	        LEFT JOIN sys.data_spaces d on d.data_space_id = i.data_space_id
	        LEFT JOIN sys.extended_properties p ON p.major_id = co.object_id AND p.class = 1 AND p.name = 'MS_Description'
	WHERE i.index_id > 0 AND (i.is_primary_key = 1 OR i.is_unique_constraint = 1) AND o.type IN ('U', 'V') AND s.name in (%s)
	ORDER BY s.name ASC, i.name ASC, ic.key_ordinal ASC
	`
	dumpIndexSQL = `
	SELECT
		s.name AS schema_name,
	    o.name AS table_name,
	    o.type AS table_type,
	    CAST (CASE WHEN o.is_ms_shipped = 1 THEN 1 WHEN (SELECT major_id FROM sys.extended_properties WHERE major_id = o.object_id AND minor_id = 0 AND class = 1 AND name = 'microsoft_database_tools_support') IS NOT NULL THEN 1 ELSE 0 END AS bit) AS is_system_object,
	    o.object_id,
	    i.name,
	    ic.partition_ordinal,
	    i.type_desc,
	    i.index_id,
	    i.is_unique,
	    i.is_primary_key,
	    i.is_unique_constraint,
	    i.fill_factor,
	    i.data_space_id,
	    i.ignore_dup_key,
	    stat.no_recompute,
	    i.is_padded,
	    i.is_disabled,
	    i.is_hypothetical,
	    i.allow_row_locks,
	    i.allow_page_locks,
	    col.name AS column_name,
	    ic.is_descending_key,
	    ic.key_ordinal,
	    ic.is_included_column,
	    i.has_filter,
	    i.filter_definition,
	    si.spatial_index_type,
	    sit.bounding_box_xmin,
	    sit.bounding_box_ymin,
	    sit.bounding_box_xmax,
	    sit.bounding_box_ymax,
	    sit.level_1_grid_desc,
	    sit.level_2_grid_desc,
	    sit.level_3_grid_desc,
	    sit.level_4_grid_desc,
	    sit.cells_per_object,
	    xi.secondary_type_desc,
	    pri.name pri_index_name,
	    CAST(ep.value AS NVARCHAR(MAX)) comment
	FROM
	    sys.indexes i
	        LEFT JOIN sys.all_objects o ON o.object_id = i.object_id
	        LEFT JOIN sys.schemas s ON s.schema_id = o.schema_id
	        LEFT JOIN sys.index_columns ic ON ic.object_id = i.object_id AND ic.index_id = i.index_id
	        LEFT JOIN sys.all_columns col ON ic.column_id = col.column_id AND ic.object_id = col.object_id
	        LEFT JOIN sys.xml_indexes xi ON i.object_id = xi.object_id AND i.index_id = xi.index_id
	        LEFT JOIN sys.indexes pri ON xi.object_id = pri.object_id AND xi.using_xml_index_id = pri.index_id
	        LEFT JOIN sys.key_constraints cons ON (cons.parent_object_id = ic.object_id AND cons.unique_index_id = i.index_id)
	        LEFT JOIN sys.extended_properties ep ON (((i.is_primary_key <> 1 AND i.is_unique_constraint <> 1 AND ep.class = 7 AND i.object_id = ep.major_id AND ep.minor_id = i.index_id) OR ((i.is_primary_key = 1 OR i.is_unique_constraint = 1) AND ep.class = 1 AND cons.object_id = ep.major_id AND ep.minor_id = 0)) AND ep.name = 'MS_Description')
	        LEFT JOIN sys.spatial_indexes si ON i.object_id = si.object_id AND i.index_id = si.index_id
	        LEFT JOIN sys.spatial_index_tessellations sit ON i.object_id = sit.object_id AND i.index_id = sit.index_id,
	    sys.stats stat
	        LEFT JOIN sys.all_objects so ON (stat.object_id = so.object_id)
	WHERE (i.object_id = so.object_id OR i.object_id = so.parent_object_id) AND i.name = stat.name AND i.index_id > 0 AND (i.is_primary_key = 0 AND i.is_unique_constraint = 0) AND s.name in (%s) AND o.type IN ('U', 'S', 'V')
	ORDER BY s.name, table_name, i.index_id, ic.key_ordinal, ic.index_column_id
	`
)

func (*Driver) dumpDatabaseTxn(ctx context.Context, txn *sql.Tx, out io.Writer) error {
	schemas, err := getSchemas(txn)
	if err != nil {
		return errors.Wrap(err, "failed to get schemas")
	}

	tableMetaMap, err := dumpTableTxn(ctx, txn, schemas)
	if err != nil {
		return errors.Wrap(err, "failed to dump tables")
	}

	columnMetaMap, err := dumpColumnTxn(ctx, txn, schemas)
	if err != nil {
		return errors.Wrap(err, "failed to dump columns")
	}

	fkMetaMap, err := dumpForeignKeyTxn(ctx, txn, schemas)
	if err != nil {
		return errors.Wrap(err, "failed to dump foreign keys")
	}

	checkMetaMap, err := dumpCheckConstraintTxn(ctx, txn, schemas)
	if err != nil {
		return errors.Wrap(err, "failed to dump check constraints")
	}

	keyMetaMap, err := dumpKeyTxn(ctx, txn, schemas)
	if err != nil {
		return errors.Wrap(err, "failed to dump indexes")
	}

	indexMetaMap, err := dumpIndexTxn(ctx, txn, schemas)
	if err != nil {
		return errors.Wrap(err, "failed to dump indexes")
	}

	return assembleStatement(out, schemas, tableMetaMap, columnMetaMap, fkMetaMap, checkMetaMap, keyMetaMap, indexMetaMap)
}

func assembleStatement(out io.Writer, schemas []string, tableMetaMap map[string][]*tableMeta, columnMetaMap map[string][]*columnMeta, fkMetaMap map[string][]*foreignKeyMeta, checkMetaMap map[string][]*checkConstraintMeta, keyMetaMap map[string][]*keyMeta, indexMetaMap map[string][]*indexMeta) error {
	for _, schema := range schemas {
		if err := assembleSchema(out, schema, tableMetaMap, columnMetaMap, fkMetaMap, checkMetaMap, keyMetaMap, indexMetaMap); err != nil {
			return err
		}
	}
	return nil
}

func assembleSchema(out io.Writer, schema string, tableMetaMap map[string][]*tableMeta, columnMetaMap map[string][]*columnMeta, fkMetaMap map[string][]*foreignKeyMeta, checkMetaMap map[string][]*checkConstraintMeta, keyMetaMap map[string][]*keyMeta, indexMetaMap map[string][]*indexMeta) error {
	if schema != defaultSchema {
		if _, err := fmt.Fprintf(out, "CREATE SCHEMA %s;\nGO\n", schema); err != nil {
			return err
		}
	}

	for _, tableMeta := range tableMetaMap[schema] {
		if _, err := fmt.Fprintf(out, "\n"); err != nil {
			return err
		}
		if err := assembleTable(out, schema, tableMeta, columnMetaMap, fkMetaMap, checkMetaMap, keyMetaMap); err != nil {
			return err
		}
		if err := assembleIndex(out, schema, tableMeta, indexMetaMap); err != nil {
			return err
		}
	}

	// TODO: add other objects like view, procedure, function, etc.
	return nil
}

func assembleIndex(out io.Writer, schema string, tableMeta *tableMeta, indexMetaMap map[string][]*indexMeta) error {
	indexes := mergeIndexMetaMap(indexMetaMap[tableID(schema, tableMeta.name.String)])
	for _, indexMeta := range indexes {
		if len(indexMeta) == 0 {
			continue
		}
		if _, err := fmt.Fprintf(out, "\nCREATE "); err != nil {
			return err
		}
		// CLUSTERED or NONCLUSTERED
		if indexMeta[0].typeDesc.Valid {
			if _, err := fmt.Fprintf(out, "%s", indexMeta[0].typeDesc.String); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintf(out, " INDEX [%s] ON\n[%s].[%s] (\n", indexMeta[0].indexName.String, schema, tableMeta.name.String); err != nil {
			return err
		}
		for i, idx := range indexMeta {
			if i != 0 {
				if _, err := fmt.Fprintf(out, ",\n"); err != nil {
					return err
				}
			}
			if _, err := fmt.Fprintf(out, "    [%s]", idx.columnName.String); err != nil {
				return err
			}
			if idx.isDescendingKey.Valid && idx.isDescendingKey.Bool {
				if _, err := fmt.Fprintf(out, " DESC"); err != nil {
					return err
				}
			} else {
				if _, err := fmt.Fprintf(out, " ASC"); err != nil {
					return err
				}
			}
		}
		if _, err := fmt.Fprintf(out, "\n);\n"); err != nil {
			return err
		}
	}
	return nil
}

func assembleTable(out io.Writer, schema string, tableMeta *tableMeta, columnMetaMap map[string][]*columnMeta, fkMetaMap map[string][]*foreignKeyMeta, checkMetaMap map[string][]*checkConstraintMeta, keyMetaMap map[string][]*keyMeta) error {
	if _, err := fmt.Fprintf(out, "CREATE TABLE [%s].[%s] (\n", schema, tableMeta.name.String); err != nil {
		return err
	}
	for i, columnMeta := range columnMetaMap[tableID(schema, tableMeta.name.String)] {
		if i != 0 {
			if _, err := fmt.Fprintf(out, ",\n"); err != nil {
				return err
			}
		}
		if err := assembleColumn(out, columnMeta); err != nil {
			return err
		}
	}
	keys := mergeKeyMetaMap(keyMetaMap[tableID(schema, tableMeta.name.String)])
	for _, keyMeta := range keys {
		if _, err := fmt.Fprintf(out, ",\n"); err != nil {
			return err
		}
		if err := assembleKey(out, keyMeta); err != nil {
			return err
		}
	}
	fks := mergeFKMetaMap(fkMetaMap[tableID(schema, tableMeta.name.String)])
	for _, fkMeta := range fks {
		if _, err := fmt.Fprintf(out, ",\n"); err != nil {
			return err
		}
		if err := assembleForeignKey(out, fkMeta); err != nil {
			return err
		}
	}
	for _, checkMeta := range checkMetaMap[tableID(schema, tableMeta.name.String)] {
		if _, err := fmt.Fprintf(out, ",\n"); err != nil {
			return err
		}
		if err := assembleCheckConstraint(out, checkMeta); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(out, "\n);\n"); err != nil {
		return err
	}
	return nil
}

func assembleCheckConstraint(out io.Writer, checkMeta *checkConstraintMeta) error {
	if _, err := fmt.Fprintf(out, "    CONSTRAINT [%s] CHECK %s", checkMeta.constraintName.String, checkMeta.definition.String); err != nil {
		return err
	}
	return nil
}

func assembleForeignKey(out io.Writer, fkMeta []*foreignKeyMeta) error {
	if len(fkMeta) == 0 {
		return nil
	}
	if _, err := fmt.Fprintf(out, "    CONSTRAINT [%s] FOREIGN KEY (", fkMeta[0].constraintName.String); err != nil {
		return err
	}
	for i, fk := range fkMeta {
		if i != 0 {
			if _, err := fmt.Fprintf(out, ", "); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintf(out, "[%s]", fk.parentColumn.String); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(out, ") REFERENCES [%s].[%s] (", fkMeta[0].referencedSchema.String, fkMeta[0].referencedTable.String); err != nil {
		return err
	}
	for i, fk := range fkMeta {
		if i != 0 {
			if _, err := fmt.Fprintf(out, ", "); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintf(out, "[%s]", fk.referencedColumn.String); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(out, ")"); err != nil {
		return err
	}
	if fkMeta[0].deleteReferentialAction.Valid {
		if _, err := fmt.Fprintf(out, " ON DELETE %s", referentialAction(int(fkMeta[0].deleteReferentialAction.Int32))); err != nil {
			return err
		}
	}
	if fkMeta[0].updateReferentialAction.Valid {
		if _, err := fmt.Fprintf(out, " ON UPDATE %s", referentialAction(int(fkMeta[0].updateReferentialAction.Int32))); err != nil {
			return err
		}
	}
	return nil
}

func referentialAction(action int) string {
	switch action {
	case 0:
		return "NO ACTION"
	case 1:
		return "CASCADE"
	case 2:
		return "SET NULL"
	case 3:
		return "SET DEFAULT"
	default:
		return "NO ACTION"
	}
}

func mergeFKMetaMap(fkMetaMap []*foreignKeyMeta) [][]*foreignKeyMeta {
	var result [][]*foreignKeyMeta
	lastIdx := 0
	currentKeyID := ""
	for i, fkMeta := range fkMetaMap {
		if !fkMeta.schemaName.Valid || !fkMeta.tableName.Valid || !fkMeta.constraintName.Valid {
			continue
		}
		keyID := keyID(fkMeta.schemaName.String, fkMeta.tableName.String, fkMeta.constraintName.String)
		if keyID != currentKeyID && currentKeyID != "" {
			result = append(result, fkMetaMap[lastIdx:i])
			lastIdx = i
			currentKeyID = keyID
		}
	}
	result = append(result, fkMetaMap[lastIdx:])
	if len(result) == 1 && len(result[0]) == 0 {
		return nil
	}
	return result
}

func mergeKeyMetaMap(keyMetaMap []*keyMeta) [][]*keyMeta {
	var result [][]*keyMeta
	lastIdx := 0
	currentKeyID := ""
	for i, keyMeta := range keyMetaMap {
		if !keyMeta.schemaName.Valid || !keyMeta.tableName.Valid || !keyMeta.indexName.Valid {
			continue
		}
		keyID := keyID(keyMeta.schemaName.String, keyMeta.tableName.String, keyMeta.indexName.String)
		if keyID != currentKeyID && currentKeyID != "" {
			result = append(result, keyMetaMap[lastIdx:i])
			lastIdx = i
			currentKeyID = keyID
		}
	}
	result = append(result, keyMetaMap[lastIdx:])
	if len(result) == 1 && len(result[0]) == 0 {
		return nil
	}
	return result
}

func mergeIndexMetaMap(indexMetaMap []*indexMeta) [][]*indexMeta {
	var result [][]*indexMeta
	lastIdx := 0
	currentKeyID := ""
	for i, indexMeta := range indexMetaMap {
		if !indexMeta.schemaName.Valid || !indexMeta.tableName.Valid || !indexMeta.indexName.Valid {
			continue
		}
		keyID := keyID(indexMeta.schemaName.String, indexMeta.tableName.String, indexMeta.indexName.String)
		if keyID != currentKeyID && currentKeyID != "" {
			result = append(result, indexMetaMap[lastIdx:i])
			lastIdx = i
			currentKeyID = keyID
		}
	}
	result = append(result, indexMetaMap[lastIdx:])
	if len(result) == 1 && len(result[0]) == 0 {
		return nil
	}
	return result
}

func keyID(schemaName, tableName, indexName string) string {
	return fmt.Sprintf("[%s].[%s].[%s]", schemaName, tableName, indexName)
}

func assembleKey(out io.Writer, keyMeta []*keyMeta) error {
	if len(keyMeta) == 0 {
		return nil
	}
	if _, err := fmt.Fprintf(out, "    CONSTRAINT [%s]", keyMeta[0].indexName.String); err != nil {
		return err
	}
	if keyMeta[0].isPrimaryKey.Valid && keyMeta[0].isPrimaryKey.Bool {
		if _, err := fmt.Fprintf(out, " PRIMARY KEY"); err != nil {
			return err
		}
	} else if keyMeta[0].isUniqueConstraint.Valid && keyMeta[0].isUniqueConstraint.Bool {
		if _, err := fmt.Fprintf(out, " UNIQUE"); err != nil {
			return err
		}
	}
	// CLUSTERED or NONCLUSTERED
	if keyMeta[0].typeDesc.Valid {
		if _, err := fmt.Fprintf(out, " %s", keyMeta[0].typeDesc.String); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(out, " ("); err != nil {
		return err
	}
	for i, key := range keyMeta {
		if i != 0 {
			if _, err := fmt.Fprintf(out, ", "); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintf(out, "[%s]", key.columnName.String); err != nil {
			return err
		}
		if key.isDescendingKey.Valid && key.isDescendingKey.Bool {
			if _, err := fmt.Fprintf(out, " DESC"); err != nil {
				return err
			}
		} else {
			if _, err := fmt.Fprintf(out, " ASC"); err != nil {
				return err
			}
		}
	}
	if _, err := fmt.Fprintf(out, ")"); err != nil {
		return err
	}
	// TODO: add key options.
	return nil
}

func assembleColumn(out io.Writer, columnMeta *columnMeta) error {
	if _, err := fmt.Fprintf(out, "    [%s]", columnMeta.columnName.String); err != nil {
		return err
	}

	if err := assembleColumnType(out, columnMeta); err != nil {
		return err
	}

	if err := assembleIdentity(out, columnMeta); err != nil {
		return err
	}

	return assembleDefinitionElement(out, columnMeta)
}

func assembleDefinitionElement(out io.Writer, columnMeta *columnMeta) error {
	if columnMeta.isFilestream.Valid && columnMeta.isFilestream.Bool {
		if _, err := fmt.Fprintf(out, " FILESTREAM"); err != nil {
			return err
		}
	}
	if columnMeta.collationName.Valid {
		if _, err := fmt.Fprintf(out, " COLLATE %s", columnMeta.collationName.String); err != nil {
			return err
		}
	}
	if columnMeta.isSparse.Valid && columnMeta.isSparse.Bool {
		if _, err := fmt.Fprintf(out, " SPARSE"); err != nil {
			return err
		}
	}
	if columnMeta.defaultValue.Valid {
		if _, err := fmt.Fprintf(out, " DEFAULT %s", columnMeta.defaultValue.String); err != nil {
			return err
		}
	}
	if columnMeta.isNullable.Valid {
		if columnMeta.isNullable.Bool {
			if _, err := fmt.Fprintf(out, " NULL"); err != nil {
				return err
			}
		} else {
			if _, err := fmt.Fprintf(out, " NOT NULL"); err != nil {
				return err
			}
		}
	}
	return nil
}

func assembleIdentity(out io.Writer, columnMeta *columnMeta) error {
	if columnMeta.isIdentity.Valid && columnMeta.isIdentity.Bool && columnMeta.seedValue.Valid && columnMeta.incrementValue.Valid {
		if _, err := fmt.Fprintf(out, " IDENTITY(%d,%d)", columnMeta.seedValue.Int64, columnMeta.incrementValue.Int64); err != nil {
			return err
		}
	}
	return nil
}

func assembleColumnType(out io.Writer, columnMeta *columnMeta) error {
	if columnMeta.definition.Valid && columnMeta.isComputed.Valid && columnMeta.isComputed.Bool {
		if _, err := fmt.Fprintf(out, " AS %s", columnMeta.definition.String); err != nil {
			return err
		}
		if columnMeta.isPersisted.Valid && columnMeta.isPersisted.Bool {
			if _, err := fmt.Fprintf(out, " PERSISTED"); err != nil {
				return err
			}
		}
		return nil
	}

	if !columnMeta.typeName.Valid {
		return errors.New("column type name is not valid")
	}

	if _, err := fmt.Fprintf(out, " %s", columnMeta.typeName.String); err != nil {
		return err
	}

	switch columnMeta.typeName.String {
	case "decimal", "numeric":
		if columnMeta.precision.Valid && columnMeta.scale.Valid {
			if _, err := fmt.Fprintf(out, "(%d, %d)", columnMeta.precision.Int64, columnMeta.scale.Int64); err != nil {
				return err
			}
		} else if columnMeta.precision.Valid {
			if _, err := fmt.Fprintf(out, "(%d)", columnMeta.precision.Int64); err != nil {
				return err
			}
		}
	case "float", "real":
		if columnMeta.precision.Valid {
			if _, err := fmt.Fprintf(out, "(%d)", columnMeta.precision.Int64); err != nil {
				return err
			}
		}
	case "dateoffset", "datetime2", "time":
		if columnMeta.scale.Valid {
			if _, err := fmt.Fprintf(out, "(%d)", columnMeta.scale.Int64); err != nil {
				return err
			}
		}
	case "char", "nchar", "varchar", "nvarchar", "binary", "varbinary":
		if columnMeta.maxLength.Valid {
			if columnMeta.maxLength.Int64 == -1 {
				if _, err := fmt.Fprintf(out, "(max)"); err != nil {
					return err
				}
			} else {
				if _, err := fmt.Fprintf(out, "(%d)", columnMeta.maxLength.Int64); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

type indexMeta struct {
	schemaName         sql.NullString
	tableName          sql.NullString
	tableType          sql.NullString
	isSystemObject     sql.NullBool
	objectID           sql.NullInt64
	indexName          sql.NullString
	partitionOrdinal   sql.NullInt64
	typeDesc           sql.NullString
	indexID            sql.NullInt64
	isUnique           sql.NullBool
	isPrimaryKey       sql.NullBool
	isUniqueConstraint sql.NullBool
	fillFactor         sql.NullInt64
	dataSpaceID        sql.NullInt64
	ignoreDupKey       sql.NullBool
	noRecompute        sql.NullBool
	isPadded           sql.NullBool
	isDisabled         sql.NullBool
	isHypothetical     sql.NullBool
	allowRowLocks      sql.NullBool
	allowPageLocks     sql.NullBool
	columnName         sql.NullString
	isDescendingKey    sql.NullBool
	keyOrdinal         sql.NullInt64
	isIncludedColumn   sql.NullBool
	hasFilter          sql.NullBool
	filterDefinition   sql.NullString
	spatialIndexType   sql.NullString
	boundingBoxXmin    sql.NullFloat64
	boundingBoxYmin    sql.NullFloat64
	boundingBoxXmax    sql.NullFloat64
	boundingBoxYmax    sql.NullFloat64
	level1GridDesc     sql.NullString
	level2GridDesc     sql.NullString
	level3GridDesc     sql.NullString
	level4GridDesc     sql.NullString
	cellsPerObject     sql.NullInt64
	secondaryTypeDesc  sql.NullString
	priIndexName       sql.NullString
	comment            sql.NullString
}

func dumpIndexTxn(ctx context.Context, txn *sql.Tx, schemas []string) (map[string][]*indexMeta, error) {
	indexMetaMap := make(map[string][]*indexMeta)
	slog.Debug("running dump index query", slog.String("schemas", fmt.Sprintf("%v", schemas)))
	query := fmt.Sprintf(dumpIndexSQL, quoteList(schemas))
	indexRows, err := txn.QueryContext(ctx, query)
	if err != nil {
		return nil, util.FormatErrorWithQuery(err, query)
	}
	defer indexRows.Close()

	for indexRows.Next() {
		meta := indexMeta{}
		if err := indexRows.Scan(
			&meta.schemaName,
			&meta.tableName,
			&meta.tableType,
			&meta.isSystemObject,
			&meta.objectID,
			&meta.indexName,
			&meta.partitionOrdinal,
			&meta.typeDesc,
			&meta.indexID,
			&meta.isUnique,
			&meta.isPrimaryKey,
			&meta.isUniqueConstraint,
			&meta.fillFactor,
			&meta.dataSpaceID,
			&meta.ignoreDupKey,
			&meta.noRecompute,
			&meta.isPadded,
			&meta.isDisabled,
			&meta.isHypothetical,
			&meta.allowRowLocks,
			&meta.allowPageLocks,
			&meta.columnName,
			&meta.isDescendingKey,
			&meta.keyOrdinal,
			&meta.isIncludedColumn,
			&meta.hasFilter,
			&meta.filterDefinition,
			&meta.spatialIndexType,
			&meta.boundingBoxXmin,
			&meta.boundingBoxYmin,
			&meta.boundingBoxXmax,
			&meta.boundingBoxYmax,
			&meta.level1GridDesc,
			&meta.level2GridDesc,
			&meta.level3GridDesc,
			&meta.level4GridDesc,
			&meta.cellsPerObject,
			&meta.secondaryTypeDesc,
			&meta.priIndexName,
			&meta.comment,
		); err != nil {
			return nil, err
		}
		if !meta.schemaName.Valid || !meta.tableName.Valid || !meta.indexName.Valid {
			continue
		}
		tableID := tableID(meta.schemaName.String, meta.tableName.String)
		if indexMetaMap[tableID] == nil {
			indexMetaMap[tableID] = []*indexMeta{&meta}
		} else {
			indexMetaMap[tableID] = append(indexMetaMap[tableID], &meta)
		}
	}
	if err := indexRows.Err(); err != nil {
		return nil, util.FormatErrorWithQuery(err, query)
	}

	return indexMetaMap, nil
}

type keyMeta struct {
	schemaName         sql.NullString
	tableName          sql.NullString
	indexName          sql.NullString
	constraintType     sql.NullString
	columnName         sql.NullString
	partitionOrdinal   sql.NullInt64
	isDescendingKey    sql.NullBool
	isPrimaryKey       sql.NullBool
	isUniqueConstraint sql.NullBool
	typeDesc           sql.NullString
	isDisabled         sql.NullBool
	isPadded           sql.NullBool
	fillFactor         sql.NullInt64
	ignoreDupKey       sql.NullBool
	isSystemObject     sql.NullBool
	dataSpaceName      sql.NullString
	dataSpaceType      sql.NullString
	comment            sql.NullString
	allowRowLocks      sql.NullBool
	allowPageLocks     sql.NullBool
	noRecompute        sql.NullBool
}

func dumpKeyTxn(ctx context.Context, txn *sql.Tx, schemas []string) (map[string][]*keyMeta, error) {
	keyMetaMap := make(map[string][]*keyMeta)
	slog.Debug("running dump primary key query", slog.String("schemas", fmt.Sprintf("%v", schemas)))
	query := fmt.Sprintf(dumpKeySQL, quoteList(schemas))
	keyRows, err := txn.QueryContext(ctx, query)
	if err != nil {
		return nil, util.FormatErrorWithQuery(err, query)
	}
	defer keyRows.Close()

	for keyRows.Next() {
		meta := keyMeta{}
		if err := keyRows.Scan(
			&meta.schemaName,
			&meta.tableName,
			&meta.indexName,
			&meta.constraintType,
			&meta.columnName,
			&meta.partitionOrdinal,
			&meta.isDescendingKey,
			&meta.isPrimaryKey,
			&meta.isUniqueConstraint,
			&meta.typeDesc,
			&meta.isDisabled,
			&meta.isPadded,
			&meta.fillFactor,
			&meta.ignoreDupKey,
			&meta.isSystemObject,
			&meta.dataSpaceName,
			&meta.dataSpaceType,
			&meta.comment,
			&meta.allowRowLocks,
			&meta.allowPageLocks,
			&meta.noRecompute,
		); err != nil {
			return nil, err
		}
		if !meta.schemaName.Valid || !meta.tableName.Valid || !meta.indexName.Valid {
			continue
		}
		tableID := tableID(meta.schemaName.String, meta.tableName.String)
		if keyMetaMap[tableID] == nil {
			keyMetaMap[tableID] = []*keyMeta{&meta}
		} else {
			keyMetaMap[tableID] = append(keyMetaMap[tableID], &meta)
		}
	}
	if err := keyRows.Err(); err != nil {
		return nil, util.FormatErrorWithQuery(err, query)
	}

	return keyMetaMap, nil
}

type checkConstraintMeta struct {
	schemaName          sql.NullString
	tableName           sql.NullString
	constraintName      sql.NullString
	constraintType      sql.NullString
	tableType           sql.NullString
	objectID            sql.NullInt64
	comment             sql.NullString
	isDisabled          sql.NullBool
	isNotForReplication sql.NullBool
	definition          sql.NullString
}

func dumpCheckConstraintTxn(ctx context.Context, txn *sql.Tx, schemas []string) (map[string][]*checkConstraintMeta, error) {
	checkMetaMap := make(map[string][]*checkConstraintMeta)
	slog.Debug("running dump check constraint query", slog.String("schemas", fmt.Sprintf("%v", schemas)))
	query := fmt.Sprintf(dumpCheckConstraintSQL, quoteList(schemas))
	checkRows, err := txn.QueryContext(ctx, query)
	if err != nil {
		return nil, util.FormatErrorWithQuery(err, query)
	}
	defer checkRows.Close()

	for checkRows.Next() {
		meta := checkConstraintMeta{}
		if err := checkRows.Scan(
			&meta.schemaName,
			&meta.tableName,
			&meta.constraintName,
			&meta.constraintType,
			&meta.tableType,
			&meta.objectID,
			&meta.comment,
			&meta.isDisabled,
			&meta.isNotForReplication,
			&meta.definition,
		); err != nil {
			return nil, err
		}
		if !meta.schemaName.Valid || !meta.tableName.Valid || !meta.constraintName.Valid {
			continue
		}
		tableID := tableID(meta.schemaName.String, meta.tableName.String)
		if checkMetaMap[tableID] == nil {
			checkMetaMap[tableID] = []*checkConstraintMeta{&meta}
		} else {
			checkMetaMap[tableID] = append(checkMetaMap[tableID], &meta)
		}
	}
	if err := checkRows.Err(); err != nil {
		return nil, util.FormatErrorWithQuery(err, query)
	}

	return checkMetaMap, nil
}

type foreignKeyMeta struct {
	schemaName              sql.NullString
	tableName               sql.NullString
	constraintName          sql.NullString
	constraintType          sql.NullString
	parentTable             sql.NullString
	parentTableType         sql.NullString
	referencedSchema        sql.NullString
	referencedTable         sql.NullString
	isDisabled              sql.NullBool
	isNotForReplication     sql.NullBool
	comment                 sql.NullString
	deleteReferentialAction sql.NullInt32
	updateReferentialAction sql.NullInt32
	parentColumn            sql.NullString
	referencedColumn        sql.NullString
}

func dumpForeignKeyTxn(ctx context.Context, txn *sql.Tx, schemas []string) (map[string][]*foreignKeyMeta, error) {
	fkMetaMap := make(map[string][]*foreignKeyMeta)
	slog.Debug("running dump foreign key query", slog.String("schemas", fmt.Sprintf("%v", schemas)))
	query := fmt.Sprintf(dumpForeignKeySQL, quoteList(schemas))
	fkRows, err := txn.QueryContext(ctx, query)
	if err != nil {
		return nil, util.FormatErrorWithQuery(err, query)
	}
	defer fkRows.Close()

	for fkRows.Next() {
		meta := foreignKeyMeta{}
		if err := fkRows.Scan(
			&meta.schemaName,
			&meta.tableName,
			&meta.constraintName,
			&meta.constraintType,
			&meta.parentTable,
			&meta.parentTableType,
			&meta.referencedSchema,
			&meta.referencedTable,
			&meta.isDisabled,
			&meta.isNotForReplication,
			&meta.comment,
			&meta.deleteReferentialAction,
			&meta.updateReferentialAction,
			&meta.parentColumn,
			&meta.referencedColumn,
		); err != nil {
			return nil, err
		}
		if !meta.schemaName.Valid || !meta.tableName.Valid || !meta.constraintName.Valid {
			continue
		}
		tableID := tableID(meta.schemaName.String, meta.tableName.String)
		if fkMetaMap[tableID] == nil {
			fkMetaMap[tableID] = []*foreignKeyMeta{&meta}
		} else {
			fkMetaMap[tableID] = append(fkMetaMap[tableID], &meta)
		}
	}
	if err := fkRows.Err(); err != nil {
		return nil, util.FormatErrorWithQuery(err, query)
	}

	return fkMetaMap, nil
}

type columnMeta struct {
	schemaName                sql.NullString
	tableName                 sql.NullString
	columnName                sql.NullString
	typeName                  sql.NullString
	typeSchema                sql.NullString
	isUserDefined             sql.NullBool
	isComputed                sql.NullBool
	definition                sql.NullString
	isPersisted               sql.NullBool
	maxLength                 sql.NullInt64
	precision                 sql.NullInt64
	scale                     sql.NullInt64
	collationName             sql.NullString
	isNullable                sql.NullBool
	isRowguidcol              sql.NullBool
	isIdentity                sql.NullBool
	isFilestream              sql.NullBool
	defaultSchema             sql.NullString
	defaultName               sql.NullString
	defaultValue              sql.NullString
	isSparse                  sql.NullBool
	isColumnSet               sql.NullBool
	comment                   sql.NullString
	seedValue                 sql.NullInt64
	incrementValue            sql.NullInt64
	identityNotForReplication sql.NullBool
	isDefaultcnst             sql.NullBool
}

func dumpColumnTxn(ctx context.Context, txn *sql.Tx, schemas []string) (map[string][]*columnMeta, error) {
	columnMetaMap := make(map[string][]*columnMeta)
	slog.Debug("running dump column query", slog.String("schemas", fmt.Sprintf("%v", schemas)))
	query := fmt.Sprintf(dumpColumnSQL, quoteList(schemas))
	columnRows, err := txn.QueryContext(ctx, query)
	if err != nil {
		return nil, util.FormatErrorWithQuery(err, query)
	}
	defer columnRows.Close()

	for columnRows.Next() {
		meta := columnMeta{}
		if err := columnRows.Scan(
			&meta.schemaName,
			&meta.tableName,
			&meta.columnName,
			&meta.typeName,
			&meta.typeSchema,
			&meta.isUserDefined,
			&meta.isComputed,
			&meta.definition,
			&meta.isPersisted,
			&meta.maxLength,
			&meta.precision,
			&meta.scale,
			&meta.collationName,
			&meta.isNullable,
			&meta.isRowguidcol,
			&meta.isIdentity,
			&meta.isFilestream,
			&meta.defaultSchema,
			&meta.defaultName,
			&meta.defaultValue,
			&meta.isSparse,
			&meta.isColumnSet,
			&meta.comment,
			&meta.seedValue,
			&meta.incrementValue,
			&meta.identityNotForReplication,
			&meta.isDefaultcnst,
		); err != nil {
			return nil, err
		}
		if !meta.schemaName.Valid || !meta.tableName.Valid || !meta.columnName.Valid {
			continue
		}
		tableID := tableID(meta.schemaName.String, meta.tableName.String)
		if columnMetaMap[tableID] == nil {
			columnMetaMap[tableID] = []*columnMeta{&meta}
		} else {
			columnMetaMap[tableID] = append(columnMetaMap[tableID], &meta)
		}
	}
	if err := columnRows.Err(); err != nil {
		return nil, util.FormatErrorWithQuery(err, query)
	}

	return columnMetaMap, nil
}

type tableMeta struct {
	name                    sql.NullString
	objectID                sql.NullInt64
	typeDesc                sql.NullString
	createDate              sql.NullTime
	modifyDate              sql.NullTime
	lockEscalation          sql.NullString
	isTrackColumnsUpdatedOn sql.NullString
	rowCount                sql.NullInt64
	comment                 sql.NullString
	schemaName              sql.NullString
	currentValue            sql.NullString
}

func quote(s string) string {
	return fmt.Sprintf("N'%s'", s)
}

func quoteList(schemas []string) string {
	var quoted []string
	for _, schema := range schemas {
		quoted = append(quoted, quote(schema))
	}
	return strings.Join(quoted, ",")
}

func dumpTableTxn(ctx context.Context, txn *sql.Tx, schemas []string) (map[string][]*tableMeta, error) {
	tableMap := make(map[string][]*tableMeta)
	slog.Debug("running dump table query", slog.String("schemas", fmt.Sprintf("%v", schemas)))
	query := fmt.Sprintf(dumpTableSQL, quoteList(schemas))
	tableRows, err := txn.QueryContext(ctx, query)
	if err != nil {
		return nil, util.FormatErrorWithQuery(err, query)
	}
	defer tableRows.Close()

	for tableRows.Next() {
		meta := tableMeta{}
		if err := tableRows.Scan(
			&meta.name,
			&meta.objectID,
			&meta.typeDesc,
			&meta.createDate,
			&meta.modifyDate,
			&meta.lockEscalation,
			&meta.isTrackColumnsUpdatedOn,
			&meta.rowCount,
			&meta.comment,
			&meta.schemaName,
			&meta.currentValue,
		); err != nil {
			return nil, err
		}
		if !meta.name.Valid || !meta.schemaName.Valid {
			continue
		}
		if tableMap[meta.schemaName.String] == nil {
			tableMap[meta.schemaName.String] = []*tableMeta{&meta}
		} else {
			tableMap[meta.schemaName.String] = append(tableMap[meta.schemaName.String], &meta)
		}
	}
	if err := tableRows.Err(); err != nil {
		return nil, util.FormatErrorWithQuery(err, query)
	}

	return tableMap, nil
}

func tableID(schemaName, tableName string) string {
	return fmt.Sprintf("[%s].[%s]", schemaName, tableName)
}

// Restore restores a database.
func (*Driver) Restore(_ context.Context, _ io.Reader) (err error) {
	// TODO(d): implement it.
	return nil
}
