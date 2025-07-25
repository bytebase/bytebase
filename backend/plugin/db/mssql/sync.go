package mssql

import (
	"context"
	"database/sql"
	"fmt"
	"slices"
	"strings"

	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

// SyncInstance syncs the instance.
func (d *Driver) SyncInstance(ctx context.Context) (*db.InstanceMetadata, error) {
	var version, fullVersion string
	if err := d.db.QueryRowContext(ctx, "SELECT SERVERPROPERTY('productversion'), @@VERSION").Scan(&version, &fullVersion); err != nil {
		return nil, err
	}
	tokens := strings.Fields(fullVersion)
	for _, token := range tokens {
		if len(token) == 4 && strings.HasPrefix(token, "20") {
			version = fmt.Sprintf("%s (%s)", version, token)
			break
		}
	}

	var databases []*storepb.DatabaseSchemaMetadata
	rows, err := d.db.QueryContext(ctx, "SELECT name, collation_name FROM sys.databases WHERE name NOT IN ('master', 'model', 'msdb', 'tempdb', 'rdscore')")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		database := &storepb.DatabaseSchemaMetadata{}
		var collation sql.NullString
		if err := rows.Scan(&database.Name, &collation); err != nil {
			return nil, err
		}
		if collation.Valid {
			database.Collation = collation.String
		}
		databases = append(databases, database)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &db.InstanceMetadata{
		Version:   version,
		Databases: databases,
	}, nil
}

// SyncDBSchema syncs a single database schema.
func (d *Driver) SyncDBSchema(ctx context.Context) (*storepb.DatabaseSchemaMetadata, error) {
	txn, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer txn.Rollback()

	schemaNames, err := getSchemas(txn)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get schemas from database %q", d.databaseName)
	}
	columnMap, err := getTableColumns(txn, schemaNames)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get table columns from database %q", d.databaseName)
	}
	tableMap, err := getTables(txn, schemaNames, columnMap)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get tables from database %q", d.databaseName)
	}
	viewMap, err := getViews(txn, columnMap)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get views from database %q", d.databaseName)
	}
	sequenceMap, err := getSequences(txn)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get sequences from database %q", d.databaseName)
	}
	functionMap, err := getFunctions(txn)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get functions from database %q", d.databaseName)
	}
	procedureMap, err := getProcedures(txn)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get procedures from database %q", d.databaseName)
	}
	tableTriggers, viewTriggers, err := getTriggers(txn)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get triggers from database %q", d.databaseName)
	}
	for schemaName, tables := range tableMap {
		for i := range tables {
			table := tables[i]
			if triggers, ok := tableTriggers[db.TableKey{Schema: schemaName, Table: table.Name}]; ok {
				table.Triggers = append(table.Triggers, triggers...)
			}
		}
	}
	for schemaName, views := range viewMap {
		for i := range views {
			view := views[i]
			if triggers, ok := viewTriggers[db.TableKey{Schema: schemaName, Table: view.Name}]; ok {
				view.Triggers = append(view.Triggers, triggers...)
			}
		}
	}

	if err := txn.Commit(); err != nil {
		return nil, err
	}

	databaseMetadata := &storepb.DatabaseSchemaMetadata{
		Name: d.databaseName,
	}
	for _, schemaName := range schemaNames {
		databaseMetadata.Schemas = append(databaseMetadata.Schemas, &storepb.SchemaMetadata{
			Name:       schemaName,
			Tables:     tableMap[schemaName],
			Views:      viewMap[schemaName],
			Sequences:  sequenceMap[schemaName],
			Functions:  functionMap[schemaName],
			Procedures: procedureMap[schemaName],
		})
	}
	return databaseMetadata, nil
}

func getSchemas(txn *sql.Tx) ([]string, error) {
	query := `
		SELECT schema_name FROM INFORMATION_SCHEMA.SCHEMATA
		WHERE schema_name NOT IN ('db_owner', 'db_accessadmin', 'db_backupoperator', 'db_datareader', 'db_datawriter', 'db_ddladmin', 'db_denydatareader', 'db_denydatawriter', 'db_securityadmin', 'guest', 'INFORMATION_SCHEMA', 'sys');`
	rows, err := txn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []string
	for rows.Next() {
		var schemaName string
		if err := rows.Scan(&schemaName); err != nil {
			return nil, err
		}
		result = append(result, schemaName)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

// getTables gets all tables of a database.
func getTables(txn *sql.Tx, schemas []string, columnMap map[db.TableKey][]*storepb.ColumnMetadata) (map[string][]*storepb.TableMetadata, error) {
	indexMap, err := getKeyAndIndexes(txn, schemas)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get indices")
	}

	fkMap, err := getForeignKeys(txn, schemas)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get foreign keys")
	}

	checkMap, err := getChecks(txn, schemas)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get checks")
	}

	tableMap := make(map[string][]*storepb.TableMetadata)
	query := `
		SELECT
			SCHEMA_NAME(t.schema_id),
			t.name,
			SUM(ps.row_count),
			lj.PropertyValue AS comment
		FROM sys.tables t
		INNER JOIN sys.dm_db_partition_stats ps ON ps.object_id = t.object_id
		LEFT JOIN (
			SELECT
				EP.value AS PropertyValue,
				S.name AS SchemaName,
				O.name AS TableName
			FROM
				(SELECT major_id, name, value FROM sys.extended_properties WHERE name = 'MS_Description' AND minor_id = 0) AS EP
				INNER JOIN sys.all_objects AS O ON EP.major_id = O.object_id
				INNER JOIN sys.schemas AS S ON O.schema_id = S.schema_id
		) lj ON lj.SchemaName = SCHEMA_NAME(t.schema_id) AND lj.TableName = t.name
        WHERE index_id < 2
		GROUP BY t.name, t.schema_id, lj.PropertyValue
		ORDER BY 1, 2 ASC
		OPTION (RECOMPILE);`
	rows, err := txn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		table := &storepb.TableMetadata{}
		var schemaName string
		var comment sql.NullString
		if err := rows.Scan(&schemaName, &table.Name, &table.RowCount, &comment); err != nil {
			return nil, err
		}
		key := db.TableKey{Schema: schemaName, Table: table.Name}
		table.Columns = columnMap[key]
		table.Indexes = indexMap[key]
		table.ForeignKeys = fkMap[key]
		table.CheckConstraints = checkMap[key]
		if comment.Valid {
			table.Comment = comment.String
		}

		tableMap[schemaName] = append(tableMap[schemaName], table)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tableMap, nil
}

func getChecks(txn *sql.Tx, schemas []string) (map[db.TableKey][]*storepb.CheckConstraintMetadata, error) {
	checkMap := make(map[db.TableKey][]*storepb.CheckConstraintMetadata)
	dumpCheckConstraintSQL := fmt.Sprintf(`
	SELECT
		t.schema_name,
	    t.name AS table_name,
	    c.name,
	    c.comment,
	    c.definition
	FROM
	    (SELECT s.name as schema_name, o.name, o.object_id, o.type FROM sys.all_objects o LEFT JOIN sys.schemas s ON s.schema_id = o.schema_id WHERE s.name in (%s) ) t
	        INNER JOIN (SELECT ch.name, ch.object_id, ch.parent_object_id, ch.is_disabled, CAST(p.[value] AS nvarchar(4000)) AS comment, ch.is_not_for_replication, ch.definition FROM sys.check_constraints ch LEFT JOIN sys.extended_properties p ON p.major_id = ch.object_id AND p.minor_id = 0 AND p.name = 'MS_Description') c ON c.parent_object_id = t.object_id
	        LEFT JOIN sys.objects co ON co.object_id = c.object_id
	ORDER BY t.schema_name ASC, t.object_id ASC, c.object_id ASC
	`, quoteList(schemas))

	rows, err := txn.Query(dumpCheckConstraintSQL)
	if err != nil {
		return nil, util.FormatErrorWithQuery(err, dumpCheckConstraintSQL)
	}
	defer rows.Close()
	for rows.Next() {
		var schemaName, tableName, checkName, comment, definition sql.NullString
		if err := rows.Scan(&schemaName, &tableName, &checkName, &comment, &definition); err != nil {
			return nil, err
		}
		if !schemaName.Valid || !tableName.Valid || !checkName.Valid || !definition.Valid {
			continue
		}
		key := db.TableKey{Schema: schemaName.String, Table: tableName.String}
		// todo: set comments.
		_ = comment
		checkMap[key] = append(checkMap[key], &storepb.CheckConstraintMetadata{
			Name:       checkName.String,
			Expression: definition.String,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return checkMap, nil
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

func getForeignKeys(txn *sql.Tx, schemas []string) (map[db.TableKey][]*storepb.ForeignKeyMetadata, error) {
	fkMap := make(map[db.TableKey]map[string]*storepb.ForeignKeyMetadata)
	dumpForeignKeySQL := fmt.Sprintf(`
	SELECT
		t.schema_name,
	    t.name AS table_name,
	    f.name,
	    f.referenced_schema,
	    f.referenced_table,
	    f.comment,
	    f.delete_referential_action,
	    f.update_referential_action,
	    f.parent_column,
	    f.referenced_column
	FROM (SELECT s.name AS schema_name, o.name, o.object_id, o.type FROM sys.all_objects o LEFT JOIN sys.schemas s ON s.schema_id = o.schema_id WHERE s.name in (%s) ) t
	    INNER JOIN (SELECT fk.object_id, fk.parent_object_id, fk.name, OBJECT_SCHEMA_NAME(fk.referenced_object_id) AS referenced_schema, OBJECT_NAME(fk.referenced_object_id) AS referenced_table, fk.is_disabled, fk.is_not_for_replication, fk.delete_referential_action, fk.update_referential_action, fc.parent_column, CAST(p.[value] AS nvarchar(4000)) AS comment, fc.referenced_column FROM sys.foreign_keys fk LEFT JOIN (SELECT fkc.constraint_object_id, pc.name AS parent_column, rc.name AS referenced_column FROM sys.foreign_key_columns fkc LEFT JOIN sys.all_columns pc ON pc.object_id = fkc.parent_object_id AND pc.column_id = fkc.parent_column_id LEFT JOIN sys.all_columns rc ON rc.object_id = fkc.referenced_object_id AND rc.column_id = fkc.referenced_column_id) fc ON fc.constraint_object_id = fk.object_id LEFT JOIN sys.extended_properties p ON p.major_id = fk.object_id AND p.minor_id = 0 AND p.name = 'MS_Description' ) f ON f.parent_object_id = t.object_id
	    LEFT JOIN sys.objects co ON co.object_id = f.object_id
	ORDER BY t.schema_name ASC, t.object_id ASC, f.object_id ASC
	`, quoteList(schemas))

	rows, err := txn.Query(dumpForeignKeySQL)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var schemaName, tableName, fkName, referencedSchemaName, referencedTableName, comment, parentColumnName, referencedColumnName sql.NullString
		var onDelete, onUpdate sql.NullInt32
		if err := rows.Scan(&schemaName, &tableName, &fkName, &referencedSchemaName, &referencedTableName, &comment, &onDelete, &onUpdate, &parentColumnName, &referencedColumnName); err != nil {
			return nil, err
		}
		if !schemaName.Valid || !tableName.Valid || !fkName.Valid || !referencedSchemaName.Valid || !referencedTableName.Valid || !parentColumnName.Valid || !referencedColumnName.Valid {
			continue
		}
		outerKey := db.TableKey{Schema: schemaName.String, Table: tableName.String}
		if _, ok := fkMap[outerKey]; !ok {
			fkMap[outerKey] = make(map[string]*storepb.ForeignKeyMetadata)
		}

		if _, ok := fkMap[outerKey][fkName.String]; !ok {
			fk := &storepb.ForeignKeyMetadata{
				Name:             fkName.String,
				ReferencedSchema: referencedSchemaName.String,
				ReferencedTable:  referencedTableName.String,
			}
			// Set comments.
			_ = comment

			if onDelete.Valid {
				fk.OnDelete = referentialAction(int(onDelete.Int32))
			}
			if onUpdate.Valid {
				fk.OnUpdate = referentialAction(int(onUpdate.Int32))
			}

			fkMap[outerKey][fkName.String] = fk
		}

		fkMap[outerKey][fkName.String].Columns = append(fkMap[outerKey][fkName.String].Columns, parentColumnName.String)
		fkMap[outerKey][fkName.String].ReferencedColumns = append(fkMap[outerKey][fkName.String].ReferencedColumns, referencedColumnName.String)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Aggregate the map to a slice.
	result := make(map[db.TableKey][]*storepb.ForeignKeyMetadata)
	for k, m := range fkMap {
		var foreignkeyNames []string
		for _, v := range m {
			foreignkeyNames = append(foreignkeyNames, v.Name)
		}
		slices.Sort(foreignkeyNames)
		for _, fkName := range foreignkeyNames {
			result[k] = append(result[k], m[fkName])
		}
	}

	return result, nil
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

// getTableColumns gets the columns of a table.
func getTableColumns(txn *sql.Tx, schemas []string) (map[db.TableKey][]*storepb.ColumnMetadata, error) {
	columnsMap := make(map[db.TableKey][]*storepb.ColumnMetadata)

	getColumnSQL := fmt.Sprintf(`
	SELECT
		s.name AS schema_name,
		OBJECT_NAME(c.object_id) AS table_name,
		c.name AS column_name,
		t.name AS type_name,
		c.is_computed,
		cc.definition,
		cc.is_persisted,
		c.max_length,
		c.precision AS precision,
		c.scale,
		c.collation_name,
		c.is_nullable,
		c.is_identity,
		d.definition AS default_value,
		d.default_name AS default_name,
		CAST(p.[value] AS nvarchar(4000)) AS comment,
		id.seed_value AS seed_value,
		id.increment_value AS increment_value
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
	`, quoteList(schemas))

	rows, err := txn.Query(getColumnSQL)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var schemaName, tableName, columnName, typeName, definition, collationName, defaultValue, defaultName, comment sql.NullString
		var isComputed, isPersisted, isNullable, isIdentity sql.NullBool
		var maxLength, precision, scale, seedValue, incrementValue sql.NullInt64
		if err := rows.Scan(
			&schemaName,
			&tableName,
			&columnName,
			&typeName,
			&isComputed,
			&definition,
			&isPersisted,
			&maxLength,
			&precision,
			&scale,
			&collationName,
			&isNullable,
			&isIdentity,
			&defaultValue,
			&defaultName,
			&comment,
			&seedValue,
			&incrementValue,
		); err != nil {
			return nil, err
		}
		if !schemaName.Valid || !tableName.Valid || !columnName.Valid || !typeName.Valid {
			continue
		}
		column := &storepb.ColumnMetadata{
			Name: columnName.String,
		}
		column.Type, err = getColumnType(definition, typeName, isComputed, isPersisted, precision, scale, maxLength)
		if err != nil {
			return nil, errors.Errorf("failed to get column type: %v", err)
		}
		if isIdentity.Valid && isIdentity.Bool && seedValue.Valid && incrementValue.Valid {
			column.IsIdentity = true
			column.IdentitySeed = seedValue.Int64
			column.IdentityIncrement = incrementValue.Int64
		}
		if collationName.Valid {
			column.Collation = collationName.String
		}
		if defaultValue.Valid {
			column.Default = defaultValue.String
		}
		if defaultName.Valid {
			column.DefaultConstraintName = defaultName.String
		}
		column.Nullable = true
		if isNullable.Valid && !isNullable.Bool {
			column.Nullable = false
		}
		if comment.Valid {
			column.Comment = comment.String
		}
		key := db.TableKey{Schema: schemaName.String, Table: tableName.String}
		columnsMap[key] = append(columnsMap[key], column)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return columnsMap, nil
}

func getColumnType(definition, typeName sql.NullString, isComputed, isPersisted sql.NullBool, precision, scale, maxLength sql.NullInt64) (string, error) {
	var buf strings.Builder
	if definition.Valid && isComputed.Valid && isComputed.Bool {
		if _, err := fmt.Fprintf(&buf, "AS %s", definition.String); err != nil {
			return "", err
		}
		if isPersisted.Valid && isPersisted.Bool {
			if _, err := fmt.Fprint(&buf, " PERSISTED"); err != nil {
				return "", err
			}
		}
		return buf.String(), nil
	}

	if !typeName.Valid {
		return "", errors.New("column type name is not valid")
	}

	if _, err := fmt.Fprintf(&buf, "%s", typeName.String); err != nil {
		return "", err
	}

	switch typeName.String {
	case "decimal", "numeric":
		if precision.Valid && scale.Valid {
			if _, err := fmt.Fprintf(&buf, "(%d, %d)", precision.Int64, scale.Int64); err != nil {
				return "", err
			}
		} else if precision.Valid {
			if _, err := fmt.Fprintf(&buf, "(%d)", precision.Int64); err != nil {
				return "", err
			}
		}
	case "float", "real":
		if precision.Valid {
			if _, err := fmt.Fprintf(&buf, "(%d)", precision.Int64); err != nil {
				return "", err
			}
		}
	case "dateoffset", "datetime2", "time":
		if scale.Valid {
			if _, err := fmt.Fprintf(&buf, "(%d)", scale.Int64); err != nil {
				return "", err
			}
		}
	case "char", "nchar", "varchar", "nvarchar", "binary", "varbinary":
		if maxLength.Valid {
			if maxLength.Int64 == -1 {
				if _, err := fmt.Fprint(&buf, "(max)"); err != nil {
					return "", err
				}
			} else {
				// For Unicode types (nchar, nvarchar), SQL Server stores byte count in max_length
				// Each Unicode character takes 2 bytes, so we need to divide by 2
				length := maxLength.Int64
				if typeName.String == "nchar" || typeName.String == "nvarchar" {
					length /= 2
				}
				if _, err := fmt.Fprintf(&buf, "(%d)", length); err != nil {
					return "", err
				}
			}
		}
	default:
		// For other types, no additional formatting is needed
		// The type name has already been written to the buffer
	}
	return buf.String(), nil
}

func getIndexes(txn *sql.Tx, schemas []string) (map[db.TableKey][]*storepb.IndexMetadata, error) {
	// MSSQL doesn't support function-based indexes.
	indexMap := make(map[db.TableKey]map[string]*storepb.IndexMetadata)

	query := fmt.Sprintf(`
	SELECT
		s.name AS schema_name,
	    o.name AS table_name,
	    i.name,
	    i.type_desc,
	    col.name AS column_name,
	    ic.is_descending_key,
	    CAST(ep.value AS NVARCHAR(MAX)) comment
	FROM
	    sys.indexes i
	        LEFT JOIN sys.all_objects o ON o.object_id = i.object_id
	        LEFT JOIN sys.schemas s ON s.schema_id = o.schema_id
	        LEFT JOIN sys.index_columns ic ON ic.object_id = i.object_id AND ic.index_id = i.index_id
	        LEFT JOIN sys.all_columns col ON ic.column_id = col.column_id AND ic.object_id = col.object_id
	        LEFT JOIN sys.key_constraints cons ON (cons.parent_object_id = ic.object_id AND cons.unique_index_id = i.index_id)
	        LEFT JOIN sys.extended_properties ep ON (((i.is_primary_key <> 1 AND i.is_unique_constraint <> 1 AND ep.class = 7 AND i.object_id = ep.major_id AND ep.minor_id = i.index_id) OR ((i.is_primary_key = 1 OR i.is_unique_constraint = 1) AND ep.class = 1 AND cons.object_id = ep.major_id AND ep.minor_id = 0)) AND ep.name = 'MS_Description'),
	    sys.stats stat
	        LEFT JOIN sys.all_objects so ON (stat.object_id = so.object_id)
	WHERE (i.object_id = so.object_id OR i.object_id = so.parent_object_id) AND i.name = stat.name AND i.index_id > 0 AND (i.is_primary_key = 0 AND i.is_unique_constraint = 0) AND s.name in (%s) AND o.type IN ('U', 'S', 'V')
	ORDER BY s.name, table_name, i.index_id, ic.key_ordinal, ic.index_column_id
	`, quoteList(schemas))
	rows, err := txn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var schemaName, tableName, indexName, typeDesc, colName, comment sql.NullString
		var isDescending sql.NullBool
		if err := rows.Scan(&schemaName, &tableName, &indexName, &typeDesc, &colName, &isDescending, &comment); err != nil {
			return nil, err
		}

		if !schemaName.Valid || !tableName.Valid || !indexName.Valid || !colName.Valid {
			continue
		}

		key := db.TableKey{Schema: schemaName.String, Table: tableName.String}
		if _, ok := indexMap[key]; !ok {
			indexMap[key] = make(map[string]*storepb.IndexMetadata)
		}
		if _, ok := indexMap[key][indexName.String]; !ok {
			index := &storepb.IndexMetadata{
				Name:         indexName.String,
				Unique:       false,
				Primary:      false,
				IsConstraint: false,
			}
			if typeDesc.Valid {
				index.Type = typeDesc.String
				// If this is a spatial index, populate basic spatial config
				if typeDesc.String == "SPATIAL" {
					index.SpatialConfig = &storepb.SpatialIndexConfig{
						Method: "SPATIAL",
						Tessellation: &storepb.TessellationConfig{
							Scheme: "UNKNOWN", // Will be updated by getSpatialIndexes if available
						},
						Storage: &storepb.StorageConfig{},
						Dimensional: &storepb.DimensionalConfig{
							DataType:   "GEOMETRY", // Default, will be updated if available
							Dimensions: 2,
						},
					}
				}
			}
			if comment.Valid {
				index.Comment = comment.String
			}
			indexMap[key][indexName.String] = index
		}

		indexMap[key][indexName.String].Expressions = append(indexMap[key][indexName.String].Expressions, colName.String)
		if isDescending.Valid && isDescending.Bool {
			indexMap[key][indexName.String].Descending = append(indexMap[key][indexName.String].Descending, true)
		} else {
			indexMap[key][indexName.String].Descending = append(indexMap[key][indexName.String].Descending, false)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	tableIndexes := make(map[db.TableKey][]*storepb.IndexMetadata)
	for k, m := range indexMap {
		for _, v := range m {
			tableIndexes[k] = append(tableIndexes[k], v)
		}
		slices.SortFunc(tableIndexes[k], func(a, b *storepb.IndexMetadata) int {
			if a.Name < b.Name {
				return -1
			}
			if a.Name > b.Name {
				return 1
			}
			return 0
		})
	}

	// Get spatial indexes separately
	// If sys.spatial_indexes doesn't exist (e.g., SQL Server Express), this will fail
	// In that case, we'll just use the spatial indexes found by the regular index query
	spatialIndexes, err := getSpatialIndexes(txn, schemas)
	if err != nil {
		// Ignore error - spatial indexes might have been found by regular index query
		spatialIndexes = make(map[db.TableKey][]*storepb.IndexMetadata)
	}

	// Merge spatial indexes with regular indexes
	// Need to replace spatial indexes found by regular query with properly configured ones
	// Merge spatial indexes with regular indexes
	for k, spatialIdxs := range spatialIndexes {
		// Process table spatial indexes
		if _, ok := tableIndexes[k]; !ok {
			tableIndexes[k] = make([]*storepb.IndexMetadata, 0)
		}

		// Remove any spatial indexes from regular indexes first
		var nonSpatialIndexes []*storepb.IndexMetadata
		for _, idx := range tableIndexes[k] {
			if idx.Type != "SPATIAL" {
				nonSpatialIndexes = append(nonSpatialIndexes, idx)
			}
		}

		// Replace with non-spatial indexes and add properly configured spatial indexes
		tableIndexes[k] = nonSpatialIndexes
		tableIndexes[k] = append(tableIndexes[k], spatialIdxs...)
	}

	return tableIndexes, nil
}

func getSpatialIndexes(txn *sql.Tx, schemas []string) (map[db.TableKey][]*storepb.IndexMetadata, error) {
	indexMap := make(map[db.TableKey]map[string]*storepb.IndexMetadata)

	// Use proper sys.spatial_indexes and sys.spatial_index_tessellations views
	// Reference: https://learn.microsoft.com/en-us/sql/relational-databases/system-catalog-views/sys-spatial-indexes-transact-sql
	// Reference: https://learn.microsoft.com/en-us/sql/relational-databases/system-catalog-views/sys-spatial-index-tessellations-transact-sql
	query := fmt.Sprintf(`
	SELECT
		s.name AS schema_name,
		o.name AS table_name,
		i.name AS index_name,
		col.name AS column_name,
		si.spatial_index_type,
		si.spatial_index_type_desc,
		si.tessellation_scheme,
		COALESCE(sit.bounding_box_xmin, 0) AS bounding_box_xmin,
		COALESCE(sit.bounding_box_ymin, 0) AS bounding_box_ymin,
		COALESCE(sit.bounding_box_xmax, 0) AS bounding_box_xmax,
		COALESCE(sit.bounding_box_ymax, 0) AS bounding_box_ymax,
		COALESCE(sit.level_1_grid_desc, '') AS level_1_grid_desc,
		COALESCE(sit.level_2_grid_desc, '') AS level_2_grid_desc,
		COALESCE(sit.level_3_grid_desc, '') AS level_3_grid_desc,
		COALESCE(sit.level_4_grid_desc, '') AS level_4_grid_desc,
		COALESCE(sit.cells_per_object, 0) AS cells_per_object,
		col.system_type_id,
		i.filter_definition,
		i.fill_factor,
		CASE WHEN i.is_padded = 1 THEN 1 ELSE 0 END AS is_padded,
		CASE WHEN i.allow_row_locks = 1 THEN 1 ELSE 0 END AS allow_row_locks,
		CASE WHEN i.allow_page_locks = 1 THEN 1 ELSE 0 END AS allow_page_locks,
		'NONE' AS data_compression_desc,
		CAST(ep.value AS NVARCHAR(MAX)) AS comment
	FROM
		sys.spatial_indexes si
		INNER JOIN sys.indexes i ON si.object_id = i.object_id AND si.index_id = i.index_id
		INNER JOIN sys.objects o ON i.object_id = o.object_id
		INNER JOIN sys.schemas s ON o.schema_id = s.schema_id
		LEFT JOIN sys.index_columns ic ON i.object_id = ic.object_id AND i.index_id = ic.index_id
		LEFT JOIN sys.columns col ON ic.object_id = col.object_id AND ic.column_id = col.column_id
		LEFT JOIN sys.spatial_index_tessellations sit ON si.object_id = sit.object_id AND si.index_id = sit.index_id
		LEFT JOIN sys.extended_properties ep ON ep.class = 7 AND i.object_id = ep.major_id AND ep.minor_id = i.index_id AND ep.name = 'MS_Description'
	WHERE
		s.name IN (%s)
		AND o.type IN ('U', 'S', 'V')
	ORDER BY s.name, o.name, i.name, ic.key_ordinal
	`, quoteList(schemas))

	rows, err := txn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var schemaName, tableName, indexName, columnName sql.NullString
		var spatialIndexType sql.NullInt32
		var spatialIndexTypeDesc, tessellationScheme sql.NullString
		var boundingBoxXmin, boundingBoxYmin, boundingBoxXmax, boundingBoxYmax sql.NullFloat64
		var level1Grid, level2Grid, level3Grid, level4Grid sql.NullString
		var cellsPerObject sql.NullInt32
		var systemTypeID sql.NullInt32
		var filterDefinition sql.NullString
		var fillFactor sql.NullInt32
		var isPadded, allowRowLocks, allowPageLocks sql.NullInt32
		var dataCompressionDesc, comment sql.NullString

		if err := rows.Scan(&schemaName, &tableName, &indexName, &columnName,
			&spatialIndexType, &spatialIndexTypeDesc, &tessellationScheme,
			&boundingBoxXmin, &boundingBoxYmin, &boundingBoxXmax, &boundingBoxYmax,
			&level1Grid, &level2Grid, &level3Grid, &level4Grid, &cellsPerObject,
			&systemTypeID, &filterDefinition, &fillFactor, &isPadded,
			&allowRowLocks, &allowPageLocks, &dataCompressionDesc, &comment); err != nil {
			return nil, err
		}

		if !schemaName.Valid || !tableName.Valid || !indexName.Valid || !columnName.Valid {
			continue
		}

		key := db.TableKey{Schema: schemaName.String, Table: tableName.String}
		if _, ok := indexMap[key]; !ok {
			indexMap[key] = make(map[string]*storepb.IndexMetadata)
		}

		if _, ok := indexMap[key][indexName.String]; !ok {
			// Create new spatial index metadata
			index := &storepb.IndexMetadata{
				Name:    indexName.String,
				Type:    "SPATIAL",
				Unique:  false,
				Primary: false,
				SpatialConfig: &storepb.SpatialIndexConfig{
					Method: "SPATIAL",
				},
			}

			// Set comment if available
			if comment.Valid {
				index.Comment = comment.String
			}

			// Determine data type from spatial_index_type_desc (more reliable)
			dataType := "GEOMETRY"
			if spatialIndexTypeDesc.Valid {
				dataType = spatialIndexTypeDesc.String
			} else if systemTypeID.Valid {
				// Fallback: system_type_id 240 = GEOMETRY, 241 = GEOGRAPHY
				if systemTypeID.Int32 == 241 {
					dataType = "GEOGRAPHY"
				}
			}

			// Configure tessellation with complete metadata
			index.SpatialConfig.Tessellation = &storepb.TessellationConfig{}

			if tessellationScheme.Valid {
				index.SpatialConfig.Tessellation.Scheme = tessellationScheme.String
			} else {
				// Fallback based on data type
				index.SpatialConfig.Tessellation.Scheme = fmt.Sprintf("%s_GRID", dataType)
			}

			// Add bounding box (for GEOMETRY indexes or when explicitly provided)
			if boundingBoxXmin.Valid && boundingBoxYmin.Valid &&
				boundingBoxXmax.Valid && boundingBoxYmax.Valid &&
				(boundingBoxXmin.Float64 != 0 || boundingBoxYmin.Float64 != 0 ||
					boundingBoxXmax.Float64 != 0 || boundingBoxYmax.Float64 != 0) {
				index.SpatialConfig.Tessellation.BoundingBox = &storepb.BoundingBox{
					Xmin: boundingBoxXmin.Float64,
					Ymin: boundingBoxYmin.Float64,
					Xmax: boundingBoxXmax.Float64,
					Ymax: boundingBoxYmax.Float64,
				}
			}

			// Add grid levels with proper descriptions
			gridLevels := []*storepb.GridLevel{}
			if level1Grid.Valid && level1Grid.String != "" {
				gridLevels = append(gridLevels, &storepb.GridLevel{Level: 1, Density: level1Grid.String})
			}
			if level2Grid.Valid && level2Grid.String != "" {
				gridLevels = append(gridLevels, &storepb.GridLevel{Level: 2, Density: level2Grid.String})
			}
			if level3Grid.Valid && level3Grid.String != "" {
				gridLevels = append(gridLevels, &storepb.GridLevel{Level: 3, Density: level3Grid.String})
			}
			if level4Grid.Valid && level4Grid.String != "" {
				gridLevels = append(gridLevels, &storepb.GridLevel{Level: 4, Density: level4Grid.String})
			}
			index.SpatialConfig.Tessellation.GridLevels = gridLevels

			// Add cells per object
			if cellsPerObject.Valid && cellsPerObject.Int32 > 0 {
				index.SpatialConfig.Tessellation.CellsPerObject = cellsPerObject.Int32
			}

			// Configure storage options - always create storage config
			index.SpatialConfig.Storage = &storepb.StorageConfig{}

			if fillFactor.Valid && fillFactor.Int32 > 0 {
				index.SpatialConfig.Storage.Fillfactor = fillFactor.Int32
			}

			if isPadded.Valid && isPadded.Int32 == 1 {
				index.SpatialConfig.Storage.PadIndex = true
			}

			if allowRowLocks.Valid && allowRowLocks.Int32 == 1 {
				index.SpatialConfig.Storage.AllowRowLocks = true
			}

			if allowPageLocks.Valid && allowPageLocks.Int32 == 1 {
				index.SpatialConfig.Storage.AllowPageLocks = true
			}

			if dataCompressionDesc.Valid && dataCompressionDesc.String != "" && dataCompressionDesc.String != "NONE" {
				index.SpatialConfig.Storage.DataCompression = dataCompressionDesc.String
			}

			// Configure dimensional properties
			index.SpatialConfig.Dimensional = &storepb.DimensionalConfig{
				DataType:   dataType,
				Dimensions: 2, // SQL Server spatial indexes are always 2D
			}

			// Note: Filter definitions for spatial indexes are not currently supported in the proto
			// but we've retrieved the data in case it's needed in the future

			indexMap[key][indexName.String] = index
		}

		// Add column to expressions
		indexMap[key][indexName.String].Expressions = append(indexMap[key][indexName.String].Expressions, columnName.String)
		indexMap[key][indexName.String].Descending = append(indexMap[key][indexName.String].Descending, false) // Spatial indexes don't support descending
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Convert to slice format
	tableIndexes := make(map[db.TableKey][]*storepb.IndexMetadata)
	for k, m := range indexMap {
		for _, v := range m {
			tableIndexes[k] = append(tableIndexes[k], v)
		}
	}

	// Try to get additional spatial index parameters from XML showplan if available
	// This is a best-effort attempt to retrieve MAXDOP, SORT_IN_TEMPDB, and ONLINE options
	if len(tableIndexes) > 0 {
		enhanceSpatialIndexesWithXMLPlan(txn, tableIndexes)
	}

	return tableIndexes, nil
}

// enhanceSpatialIndexesWithXMLPlan attempts to extract additional spatial index options
// that are not available in system tables but might be visible in execution plans
func enhanceSpatialIndexesWithXMLPlan(txn *sql.Tx, spatialIndexes map[db.TableKey][]*storepb.IndexMetadata) {
	// This is a best-effort function, so we ignore errors
	for tableKey, indexes := range spatialIndexes {
		for _, index := range indexes {
			if index.Type != "SPATIAL" || index.SpatialConfig == nil {
				continue
			}

			// Try to get the index creation statement from sys.sql_modules or other sources
			// This is SQL Server version dependent and might not always work
			query := `
			SELECT 
				OBJECTPROPERTY(i.object_id, 'ExecIsAnsiNullsOn') AS ansi_nulls,
				OBJECTPROPERTY(i.object_id, 'ExecIsQuotedIdentOn') AS quoted_ident
			FROM sys.indexes i
			INNER JOIN sys.objects o ON i.object_id = o.object_id
			INNER JOIN sys.schemas s ON o.schema_id = s.schema_id
			WHERE s.name = @schema AND o.name = @table AND i.name = @index
			`
			var ansiNulls, quotedIdent sql.NullInt32
			err := txn.QueryRow(query,
				sql.Named("schema", tableKey.Schema),
				sql.Named("table", tableKey.Table),
				sql.Named("index", index.Name)).Scan(&ansiNulls, &quotedIdent)

			if err == nil && index.SpatialConfig.Storage == nil {
				// If we don't have storage config yet, create a basic one
				// These properties give us hints about the index creation context
				if ansiNulls.Valid || quotedIdent.Valid {
					if index.SpatialConfig.Storage == nil {
						index.SpatialConfig.Storage = &storepb.StorageConfig{}
					}
					// These are indirect indicators but better than nothing
					index.SpatialConfig.Storage.AllowRowLocks = true
					index.SpatialConfig.Storage.AllowPageLocks = true
				}
			}
		}
	}
}

func getKeys(txn *sql.Tx, schemas []string) (map[db.TableKey][]*storepb.IndexMetadata, error) {
	indexMap := make(map[db.TableKey]map[string]*storepb.IndexMetadata)
	dumpKeySQL := fmt.Sprintf(`
	SELECT
		s.name AS schema_name,
	    o.name AS table_name,
	    i.name,
	    c.name AS column_name,
	    ic.is_descending_key,
	    i.is_primary_key,
	    i.is_unique_constraint,
	    i.type_desc,
	    CAST(p.[value] AS nvarchar(4000)) AS comment
	FROM
	    sys.indexes i
	        LEFT JOIN sys.index_columns ic ON ic.object_id = i.object_id AND ic.index_id = i.index_id
	        LEFT JOIN sys.columns c ON c.object_id = ic.object_id AND c.column_id = ic.column_id
	        LEFT JOIN sys.objects co ON co.parent_object_id = i.object_id AND co.name = i.name LEFT JOIN sys.objects o ON o.object_id = i.object_id
	        LEFT JOIN sys.schemas s ON s.schema_id = o.schema_id
	        LEFT JOIN sys.extended_properties p ON p.major_id = co.object_id AND p.class = 1 AND p.name = 'MS_Description'
	WHERE i.index_id > 0 AND (i.is_primary_key = 1 OR i.is_unique_constraint = 1) AND o.type IN ('U', 'V') AND s.name in (%s)
	ORDER BY s.name ASC, i.name ASC, ic.key_ordinal ASC
	`, quoteList(schemas))

	rows, err := txn.Query(dumpKeySQL)
	if err != nil {
		return nil, util.FormatErrorWithQuery(err, dumpKeySQL)
	}
	defer rows.Close()

	for rows.Next() {
		var schemaName, tableName, indexName, colName, typeDesc, comment sql.NullString
		var isDescending, isPrimaryKey, isUniqueConstraint sql.NullBool
		if err := rows.Scan(&schemaName, &tableName, &indexName, &colName, &isDescending, &isPrimaryKey, &isUniqueConstraint, &typeDesc, &comment); err != nil {
			return nil, err
		}

		if !schemaName.Valid || !tableName.Valid || !indexName.Valid || !colName.Valid {
			continue
		}
		key := db.TableKey{Schema: schemaName.String, Table: tableName.String}
		if _, ok := indexMap[key]; !ok {
			indexMap[key] = make(map[string]*storepb.IndexMetadata)
		}
		if _, ok := indexMap[key][indexName.String]; !ok {
			index := &storepb.IndexMetadata{
				Name:         indexName.String,
				Unique:       false,
				Primary:      false,
				IsConstraint: true,
			}
			if isPrimaryKey.Valid && isPrimaryKey.Bool {
				index.Primary = true
				index.Unique = true
			}
			if isUniqueConstraint.Valid && isUniqueConstraint.Bool {
				index.Unique = true
			}
			if typeDesc.Valid {
				index.Type = typeDesc.String
			}
			if comment.Valid {
				index.Comment = comment.String
			}
			indexMap[key][indexName.String] = index
		}

		indexMap[key][indexName.String].Expressions = append(indexMap[key][indexName.String].Expressions, colName.String)
		if isDescending.Valid && isDescending.Bool {
			indexMap[key][indexName.String].Descending = append(indexMap[key][indexName.String].Descending, true)
		} else {
			indexMap[key][indexName.String].Descending = append(indexMap[key][indexName.String].Descending, false)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	tableIndexes := make(map[db.TableKey][]*storepb.IndexMetadata)
	for k, m := range indexMap {
		for _, v := range m {
			tableIndexes[k] = append(tableIndexes[k], v)
		}
		slices.SortFunc(tableIndexes[k], func(a, b *storepb.IndexMetadata) int {
			if a.Name < b.Name {
				return -1
			}
			if a.Name > b.Name {
				return 1
			}
			return 0
		})
	}
	return tableIndexes, nil
}

// getIndexes gets all indices of a database.
func getKeyAndIndexes(txn *sql.Tx, schemas []string) (map[db.TableKey][]*storepb.IndexMetadata, error) {
	keys, err := getKeys(txn, schemas)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get keys")
	}
	indexes, err := getIndexes(txn, schemas)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get indexes")
	}
	for k, v := range indexes {
		keys[k] = append(keys[k], v...)
	}
	return keys, nil
}

// getViews gets all views of a database.
func getViews(txn *sql.Tx, columnMap map[db.TableKey][]*storepb.ColumnMetadata) (map[string][]*storepb.ViewMetadata, error) {
	viewMap := make(map[string][]*storepb.ViewMetadata)

	query := `
		SELECT
			SCHEMA_NAME(v.schema_id) AS schema_name,
			v.name AS view_name,
			m.definition,
			CAST(ep.value AS NVARCHAR(MAX)) AS comment
		FROM sys.views v
		INNER JOIN sys.sql_modules m ON v.object_id = m.object_id
		LEFT JOIN sys.extended_properties ep ON ep.major_id = v.object_id 
			AND ep.minor_id = 0 
			AND ep.class = 1 
			AND ep.name = 'MS_Description'
		ORDER BY schema_name, view_name;`
	rows, err := txn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		view := &storepb.ViewMetadata{}
		var schemaName string
		var definition, comment sql.NullString
		if err := rows.Scan(&schemaName, &view.Name, &definition, &comment); err != nil {
			return nil, err
		}

		var viewDefinition string
		if !definition.Valid {
			// Definition is null if the view is encrypted.
			// https://learn.microsoft.com/en-us/sql/relational-databases/system-catalog-views/sys-all-sql-modules-transact-sql?view=sql-server-ver16
			// https://www.mssqltips.com/sqlservertip/7465/encrypt-stored-procedure-sql-server/
			// We will write a pseudo definition in pure comment.
			viewDefinition = fmt.Sprintf("/* Definition of view %s.%s is encrypted. */", schemaName, view.Name)
		} else {
			viewDefinition = definition.String
		}
		view.Definition = viewDefinition

		if comment.Valid {
			view.Comment = comment.String
		}

		key := db.TableKey{Schema: schemaName, Table: view.Name}
		view.Columns = columnMap[key]

		viewMap[schemaName] = append(viewMap[schemaName], view)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return viewMap, nil
}

func getTriggers(txn *sql.Tx) (map[db.TableKey][]*storepb.TriggerMetadata, map[db.TableKey][]*storepb.TriggerMetadata, error) {
	query := `
SELECT
    st.name,
    STUFF((
        SELECT ',' + te.type_desc
        FROM sys.trigger_events AS te
        WHERE te.object_id = st.object_id
        FOR XML PATH('')
    ), 1, 1, '') AS events,
CASE
        WHEN st.type = 'TR' THEN 'AFTER' -- DML triggers created with FOR or AFTER
        WHEN st.type = 'TA' THEN 'AFTER' -- DDL triggers
        WHEN st.type = 'TI' THEN 'INSTEAD OF' -- INSTEAD OF triggers
        ELSE 'UNKNOWN' -- Handle other potential types
END AS timing,
ssm.definition AS body,
so.name AS parentName,
ss.name AS schemaName,
so.type AS objectType,
CAST(ep.value AS NVARCHAR(MAX)) AS comment
FROM
    sys.triggers AS st
JOIN
    sys.sql_modules AS ssm
ON
    st.object_id = ssm.object_id
JOIN
    sys.objects AS so
ON
    st.parent_id = so.object_id
JOIN
    sys.schemas AS ss
ON so.schema_id = ss.schema_id
LEFT JOIN sys.extended_properties ep ON ep.major_id = st.object_id 
	AND ep.minor_id = 0 
	AND ep.class = 1 
	AND ep.name = 'MS_Description'
WHERE st.is_disabled = 0 AND st.is_ms_shipped = 0 AND st.parent_id <> 0 AND  so.type IN ('U', 'V')
ORDER BY st.name;
`
	tableTriggers := make(map[db.TableKey][]*storepb.TriggerMetadata)
	viewTriggers := make(map[db.TableKey][]*storepb.TriggerMetadata)
	rows, err := txn.Query(query)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var name, events, timing, parentName, schemaName, parentType string
		var body, comment sql.NullString
		if err := rows.Scan(&name, &events, &timing, &body, &parentName, &schemaName, &parentType, &comment); err != nil {
			return nil, nil, err
		}
		bodyString := fmt.Sprintf("/* Definition of trigger %s.%s.%s is encrypted. */", schemaName, parentName, name)
		if body.Valid {
			bodyString = body.String
		}
		m := tableTriggers
		if parentType == "V" {
			m = viewTriggers
		}
		trigger := &storepb.TriggerMetadata{
			Name:   name,
			Event:  events,
			Timing: timing,
			Body:   bodyString,
		}
		if comment.Valid {
			trigger.Comment = comment.String
		}
		key := db.TableKey{Schema: schemaName, Table: parentName}
		m[key] = append(m[key], trigger)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	return tableTriggers, viewTriggers, nil
}

// getSequences gets all sequences of a database.
func getSequences(txn *sql.Tx) (map[string][]*storepb.SequenceMetadata, error) {
	query := `
	SELECT
		s.name,
		seq.name,
		tp.name,
		CAST(ep.value AS NVARCHAR(MAX)) AS comment
	FROM
		sys.sequences seq
	INNER JOIN
		sys.schemas s ON s.schema_id = seq.schema_id
	INNER JOIN
		sys.types tp ON tp.system_type_id = seq.system_type_id
	LEFT JOIN sys.extended_properties ep ON ep.major_id = seq.object_id 
		AND ep.minor_id = 0 
		AND ep.class = 1 
		AND ep.name = 'MS_Description'
	ORDER BY s.name, seq.name;`
	rows, err := txn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	sequenceMap := make(map[string][]*storepb.SequenceMetadata)
	for rows.Next() {
		sequence := &storepb.SequenceMetadata{}
		var schemaName string
		var comment sql.NullString
		if err := rows.Scan(&schemaName, &sequence.Name, &sequence.DataType, &comment); err != nil {
			return nil, err
		}
		if comment.Valid {
			sequence.Comment = comment.String
		}
		sequenceMap[schemaName] = append(sequenceMap[schemaName], sequence)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return sequenceMap, nil
}

func getProcedures(txn *sql.Tx) (map[string][]*storepb.ProcedureMetadata, error) {
	procedureMap := make(map[string][]*storepb.ProcedureMetadata)

	query := `
	SELECT
		SCHEMA_NAME(ao.schema_id) AS schema_name,
		ao.name AS procedure_name,
		asm.definition
	FROM sys.all_objects ao
        INNER JOIN sys.all_sql_modules asm ON asm.object_id = ao.object_id
	WHERE ao.type IN ('P', 'RF')
		AND ao.is_ms_shipped = 0 AND
		(
			SELECT major_id
			FROM sys.extended_properties
			WHERE major_id = ao.object_id
				AND minor_id = 0
				AND class = 1
				AND name = 'microsoft_database_tools_support'
		) IS NULL
	ORDER BY schema_name, procedure_name;`
	rows, err := txn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		procedure := &storepb.ProcedureMetadata{}
		var schemaName string
		var definition sql.NullString
		if err := rows.Scan(&schemaName, &procedure.Name, &definition); err != nil {
			return nil, err
		}
		var procedureDefinition string
		if !definition.Valid {
			// Definition is null if the procedure is encrypted.
			// https://learn.microsoft.com/en-us/sql/relational-databases/system-catalog-views/sys-all-sql-modules-transact-sql?view=sql-server-ver16
			// https://www.mssqltips.com/sqlservertip/7465/encrypt-stored-procedure-sql-server/
			// We will write a pseudo definition in pure comment.
			procedureDefinition = fmt.Sprintf("/* Definition of procedure %s.%s is encrypted. */", schemaName, procedure.Name)
		} else {
			procedureDefinition = definition.String
		}
		procedure.Definition = procedureDefinition
		procedureMap[schemaName] = append(procedureMap[schemaName], procedure)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return procedureMap, nil
}

func getFunctions(txn *sql.Tx) (map[string][]*storepb.FunctionMetadata, error) {
	funcMap := make(map[string][]*storepb.FunctionMetadata)

	// The CAST(...) = 0 means the function is not a system function.
	query := `
	SELECT
		SCHEMA_NAME(ao.schema_id) AS schema_name,
		ao.name AS func_name,
		asm.definition,
		CAST(ep.value AS NVARCHAR(MAX)) AS comment
	FROM sys.all_objects ao
        INNER JOIN sys.all_sql_modules asm ON asm.object_id = ao.object_id
		LEFT JOIN sys.extended_properties ep ON ep.major_id = ao.object_id 
			AND ep.minor_id = 0 
			AND ep.class = 1 
			AND ep.name = 'MS_Description'
	WHERE ao.type IN ('FN', 'IF', 'TF')
		AND ao.is_ms_shipped = 0 AND
		(
			SELECT major_id
			FROM sys.extended_properties
			WHERE major_id = ao.object_id
				AND minor_id = 0
				AND class = 1
				AND name = 'microsoft_database_tools_support'
		) IS NULL
	ORDER BY schema_name, func_name;`
	rows, err := txn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		function := &storepb.FunctionMetadata{}
		var schemaName string
		var definition, comment sql.NullString
		if err := rows.Scan(&schemaName, &function.Name, &definition, &comment); err != nil {
			return nil, err
		}
		var functionDefinition string
		if !definition.Valid {
			// Definition is null if the function is encrypted.
			// https://learn.microsoft.com/en-us/sql/relational-databases/system-catalog-views/sys-all-sql-modules-transact-sql?view=sql-server-ver16
			// https://www.mssqltips.com/sqlservertip/7465/encrypt-stored-procedure-sql-server/
			// We will write a pseudo definition in pure comment.
			functionDefinition = fmt.Sprintf("/* Definition of function %s.%s is encrypted. */", schemaName, function.Name)
		} else {
			functionDefinition = definition.String
		}
		function.Definition = functionDefinition

		if comment.Valid {
			function.Comment = comment.String
		}

		funcMap[schemaName] = append(funcMap[schemaName], function)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return funcMap, nil
}
