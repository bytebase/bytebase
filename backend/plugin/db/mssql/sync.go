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

	// Get table comments separately
	tableCommentsMap := make(map[db.TableKey]string)

	// First get object IDs with schema info
	objectSchemaMap := make(map[int]struct{ Schema, Table string })
	objectQuery := `
		SELECT o.object_id, s.name AS schema_name, o.name AS table_name
		FROM sys.all_objects o
		INNER JOIN sys.schemas s ON s.schema_id = o.schema_id
		WHERE o.type = 'U'`
	objectRows, err := txn.Query(objectQuery)
	if err == nil {
		defer objectRows.Close()
		for objectRows.Next() {
			var objectID int
			var schemaName, tableName string
			if err := objectRows.Scan(&objectID, &schemaName, &tableName); err != nil {
				continue
			}
			objectSchemaMap[objectID] = struct{ Schema, Table string }{schemaName, tableName}
		}
		if err := objectRows.Err(); err != nil {
			return nil, errors.Wrap(err, "failed to iterate object schema mapping")
		}
	}

	// Then get extended properties
	commentsQuery := `
		SELECT major_id, value
		FROM sys.extended_properties
		WHERE name = 'MS_Description' AND minor_id = 0 AND class = 1`
	commentsRows, err := txn.Query(commentsQuery)
	if err == nil {
		defer commentsRows.Close()
		for commentsRows.Next() {
			var objectID int
			var comment string
			if err := commentsRows.Scan(&objectID, &comment); err != nil {
				continue
			}
			// Join in Go
			if obj, ok := objectSchemaMap[objectID]; ok {
				key := db.TableKey{Schema: obj.Schema, Table: obj.Table}
				tableCommentsMap[key] = comment
			}
		}
		if err := commentsRows.Err(); err != nil {
			// Log error but continue - comments are not critical
			return nil, errors.Wrap(err, "failed to fetch table comments")
		}
	}

	// Get table row counts separately
	rowCountMap := make(map[int]int64)
	rowCountQuery := `
		SELECT object_id, SUM(row_count) AS total_rows
		FROM sys.dm_db_partition_stats
		WHERE index_id < 2
		GROUP BY object_id`
	rowCountRows, err := txn.Query(rowCountQuery)
	if err == nil {
		defer rowCountRows.Close()
		for rowCountRows.Next() {
			var objectID int
			var rowCount int64
			if err := rowCountRows.Scan(&objectID, &rowCount); err != nil {
				continue
			}
			rowCountMap[objectID] = rowCount
		}
		if err := rowCountRows.Err(); err != nil {
			return nil, errors.Wrap(err, "failed to iterate row counts")
		}
	}

	tableMap := make(map[string][]*storepb.TableMetadata)
	query := `
		SELECT
			t.object_id,
			SCHEMA_NAME(t.schema_id),
			t.name
		FROM sys.tables t
		ORDER BY SCHEMA_NAME(t.schema_id), t.name ASC`
	rows, err := txn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		table := &storepb.TableMetadata{}
		var objectID int
		var schemaName string
		if err := rows.Scan(&objectID, &schemaName, &table.Name); err != nil {
			return nil, err
		}
		// Join with row count in Go
		if rowCount, ok := rowCountMap[objectID]; ok {
			table.RowCount = rowCount
		}
		key := db.TableKey{Schema: schemaName, Table: table.Name}
		table.Columns = columnMap[key]
		table.Indexes = indexMap[key]
		table.ForeignKeys = fkMap[key]
		table.CheckConstraints = checkMap[key]

		// Join with comments in Go
		if comment, ok := tableCommentsMap[key]; ok {
			table.Comment = comment
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

	// Get check constraint comments separately
	checkCommentsMap := make(map[int]string)
	commentsQuery := `
	SELECT 
		p.major_id AS object_id,
		CAST(p.[value] AS nvarchar(4000)) AS comment
	FROM sys.extended_properties p
	WHERE p.minor_id = 0 AND p.name = 'MS_Description'
		AND p.major_id IN (SELECT object_id FROM sys.check_constraints)`
	commentsRows, err := txn.Query(commentsQuery)
	if err == nil {
		defer commentsRows.Close()
		for commentsRows.Next() {
			var objectID int
			var comment sql.NullString
			if err := commentsRows.Scan(&objectID, &comment); err != nil {
				continue
			}
			if comment.Valid {
				checkCommentsMap[objectID] = comment.String
			}
		}
		if err := commentsRows.Err(); err != nil {
			// Log error but continue - comments are not critical
			return nil, errors.Wrap(err, "failed to fetch check constraint comments")
		}
	}

	// Get object schema mapping first
	objectSchemaMap := make(map[int]struct{ Schema, Table string })
	objectQuery := fmt.Sprintf(`
	SELECT o.object_id, s.name AS schema_name, o.name AS table_name
	FROM sys.objects o
	INNER JOIN sys.schemas s ON s.schema_id = o.schema_id
	WHERE s.name in (%s) AND o.type IN ('U')
	`, quoteList(schemas))
	objectRows, err := txn.Query(objectQuery)
	if err != nil {
		return nil, util.FormatErrorWithQuery(err, objectQuery)
	}
	defer objectRows.Close()
	for objectRows.Next() {
		var objectID int
		var schemaName, tableName string
		if err := objectRows.Scan(&objectID, &schemaName, &tableName); err != nil {
			continue
		}
		objectSchemaMap[objectID] = struct{ Schema, Table string }{schemaName, tableName}
	}
	if err := objectRows.Err(); err != nil {
		return nil, errors.Wrap(err, "failed to fetch object schema mapping")
	}

	dumpCheckConstraintSQL := `
	SELECT
		ch.parent_object_id,
		ch.name,
		ch.object_id,
		ch.definition
	FROM sys.check_constraints ch
	ORDER BY ch.parent_object_id, ch.object_id`

	rows, err := txn.Query(dumpCheckConstraintSQL)
	if err != nil {
		return nil, util.FormatErrorWithQuery(err, dumpCheckConstraintSQL)
	}
	defer rows.Close()
	for rows.Next() {
		var parentObjectID int
		var checkName, definition sql.NullString
		var objectID sql.NullInt32
		if err := rows.Scan(&parentObjectID, &checkName, &objectID, &definition); err != nil {
			return nil, err
		}
		if !checkName.Valid || !definition.Valid {
			continue
		}
		// Join with schema mapping in Go
		obj, ok := objectSchemaMap[parentObjectID]
		if !ok {
			continue
		}
		key := db.TableKey{Schema: obj.Schema, Table: obj.Table}
		check := &storepb.CheckConstraintMetadata{
			Name:       checkName.String,
			Expression: definition.String,
		}
		// TODO: Join with comments when CheckConstraintMetadata supports it
		// Currently the proto doesn't support comments for check constraints
		// When support is added, use:
		// if objectID.Valid {
		//     if comment, ok := checkCommentsMap[int(objectID.Int32)]; ok {
		//         check.Comment = comment
		//     }
		// }
		checkMap[key] = append(checkMap[key], check)
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

	// Get foreign key comments separately
	fkCommentsMap := make(map[int]string)
	commentsQuery := `
	SELECT 
		p.major_id AS object_id,
		CAST(p.[value] AS nvarchar(4000)) AS comment
	FROM sys.extended_properties p
	WHERE p.minor_id = 0 AND p.name = 'MS_Description'
		AND p.major_id IN (SELECT object_id FROM sys.foreign_keys)`
	commentsRows, err := txn.Query(commentsQuery)
	if err == nil {
		defer commentsRows.Close()
		for commentsRows.Next() {
			var objectID int
			var comment sql.NullString
			if err := commentsRows.Scan(&objectID, &comment); err != nil {
				continue
			}
			if comment.Valid {
				fkCommentsMap[objectID] = comment.String
			}
		}
		if err := commentsRows.Err(); err != nil {
			// Log error but continue - comments are not critical
			return nil, errors.Wrap(err, "failed to fetch foreign key comments")
		}
	}

	dumpForeignKeySQL := fmt.Sprintf(`
	SELECT
		s.name AS schema_name,
		o.name AS table_name,
		fk.object_id,
		fk.name,
		OBJECT_SCHEMA_NAME(fk.referenced_object_id) AS referenced_schema,
		OBJECT_NAME(fk.referenced_object_id) AS referenced_table,
		fk.delete_referential_action,
		fk.update_referential_action,
		pc.name AS parent_column,
		rc.name AS referenced_column
	FROM sys.foreign_keys fk
	INNER JOIN sys.foreign_key_columns fkc ON fkc.constraint_object_id = fk.object_id
	INNER JOIN sys.objects o ON o.object_id = fk.parent_object_id
	INNER JOIN sys.schemas s ON s.schema_id = o.schema_id
	INNER JOIN sys.all_columns pc ON pc.object_id = fkc.parent_object_id AND pc.column_id = fkc.parent_column_id
	INNER JOIN sys.all_columns rc ON rc.object_id = fkc.referenced_object_id AND rc.column_id = fkc.referenced_column_id
	WHERE s.name in (%s)
	ORDER BY s.name ASC, o.object_id ASC, fk.object_id ASC, fkc.constraint_column_id ASC
	`, quoteList(schemas))

	rows, err := txn.Query(dumpForeignKeySQL)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var schemaName, tableName, fkName, referencedSchemaName, referencedTableName, parentColumnName, referencedColumnName sql.NullString
		var objectID, onDelete, onUpdate sql.NullInt32
		if err := rows.Scan(&schemaName, &tableName, &objectID, &fkName, &referencedSchemaName, &referencedTableName, &onDelete, &onUpdate, &parentColumnName, &referencedColumnName); err != nil {
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
			// TODO: Join with comments when ForeignKeyMetadata supports it
			// Currently the proto doesn't support comments for foreign keys
			// When support is added, use:
			// if objectID.Valid {
			//     if comment, ok := fkCommentsMap[int(objectID.Int32)]; ok {
			//         fk.Comment = comment
			//     }
			// }

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

	// Get column comments separately
	columnCommentsMap := make(map[struct{ ObjectID, ColumnID int }]string)
	commentsQuery := `
	SELECT
		p.major_id AS object_id,
		p.minor_id AS column_id,
		CAST(p.[value] AS nvarchar(4000)) AS comment
	FROM sys.extended_properties p
	WHERE p.class = 1 AND p.name = 'MS_Description' AND p.minor_id > 0`
	commentsRows, err := txn.Query(commentsQuery)
	if err == nil {
		defer commentsRows.Close()
		for commentsRows.Next() {
			var objectID, columnID int
			var comment sql.NullString
			if err := commentsRows.Scan(&objectID, &columnID, &comment); err != nil {
				continue
			}
			if comment.Valid {
				columnCommentsMap[struct{ ObjectID, ColumnID int }{objectID, columnID}] = comment.String
			}
		}
		if err := commentsRows.Err(); err != nil {
			// Log error but continue - comments are not critical
			return nil, errors.Wrap(err, "failed to fetch column comments")
		}
	}

	// Get default constraints separately
	defaultsMap := make(map[int]struct{ Definition, Name string })
	defaultsQuery := `
	SELECT 
		so.object_id,
		dc.definition,
		so.name AS default_name
	FROM sys.objects so
	INNER JOIN sys.default_constraints dc ON dc.object_id = so.object_id
	WHERE so.type = 'D'`
	defaultsRows, err := txn.Query(defaultsQuery)
	if err == nil {
		defer defaultsRows.Close()
		for defaultsRows.Next() {
			var objectID int
			var definition, name sql.NullString
			if err := defaultsRows.Scan(&objectID, &definition, &name); err != nil {
				continue
			}
			if definition.Valid && name.Valid {
				defaultsMap[objectID] = struct{ Definition, Name string }{definition.String, name.String}
			}
		}
		if err := defaultsRows.Err(); err != nil {
			// Log error but continue - defaults are not critical
			return nil, errors.Wrap(err, "failed to fetch default constraints")
		}
	}

	getColumnSQL := fmt.Sprintf(`
	SELECT
		s.name AS schema_name,
		OBJECT_NAME(c.object_id) AS table_name,
		c.object_id,
		c.column_id,
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
		c.default_object_id,
		id.seed_value AS seed_value,
		id.increment_value AS increment_value
	FROM sys.columns c
		LEFT JOIN sys.computed_columns cc ON cc.object_id = c.object_id AND cc.column_id = c.column_id
		LEFT JOIN sys.types t ON c.user_type_id = t.user_type_id
		LEFT JOIN sys.objects o ON o.object_id = c.object_id
		LEFT JOIN sys.schemas s ON s.schema_id = o.schema_id
		LEFT JOIN sys.identity_columns id ON c.object_id = id.object_id AND c.column_id = id.column_id
	WHERE s.name in (%s)
	ORDER BY s.name ASC, c.object_id ASC, c.column_id ASC
	`, quoteList(schemas))

	rows, err := txn.Query(getColumnSQL)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Track position counter per table to compute ordinal positions
	// Since results are ordered by object_id and column_id, we can calculate sequential positions
	positionMap := make(map[db.TableKey]int32)

	for rows.Next() {
		var schemaName, tableName, columnName, typeName, definition, collationName sql.NullString
		var isComputed, isPersisted, isNullable, isIdentity sql.NullBool
		var objectID, columnID, defaultObjectID sql.NullInt32
		var maxLength, precision, scale, seedValue, incrementValue sql.NullInt64
		if err := rows.Scan(
			&schemaName,
			&tableName,
			&objectID,
			&columnID,
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
			&defaultObjectID,
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

		// Calculate ordinal position (1-based sequential position per table)
		// Results are ordered by column_id, so we increment position for each column in the table
		key := db.TableKey{Schema: schemaName.String, Table: tableName.String}
		positionMap[key]++
		column.Position = positionMap[key]
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
		// Join with defaults in Go
		if defaultObjectID.Valid {
			if def, ok := defaultsMap[int(defaultObjectID.Int32)]; ok {
				column.Default = def.Definition
				column.DefaultConstraintName = def.Name
			}
		}
		column.Nullable = true
		if isNullable.Valid && !isNullable.Bool {
			column.Nullable = false
		}
		// Join with comments in Go
		if objectID.Valid && columnID.Valid {
			if comment, ok := columnCommentsMap[struct{ ObjectID, ColumnID int }{int(objectID.Int32), int(columnID.Int32)}]; ok {
				column.Comment = comment
			}
		}
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

	// Get index comments separately
	indexCommentsMap := make(map[struct{ ObjectID, IndexID int }]string)
	commentsQuery := `
	SELECT 
		ep.major_id AS object_id,
		ep.minor_id AS index_id,
		CAST(ep.value AS NVARCHAR(MAX)) AS comment
	FROM sys.extended_properties ep
	WHERE ep.class = 7 AND ep.name = 'MS_Description' AND ep.minor_id > 0`
	commentsRows, err := txn.Query(commentsQuery)
	if err == nil {
		defer commentsRows.Close()
		for commentsRows.Next() {
			var objectID, indexID int
			var comment sql.NullString
			if err := commentsRows.Scan(&objectID, &indexID, &comment); err != nil {
				continue
			}
			if comment.Valid {
				indexCommentsMap[struct{ ObjectID, IndexID int }{objectID, indexID}] = comment.String
			}
		}
		if err := commentsRows.Err(); err != nil {
			// Log error but continue - comments are not critical
			return nil, errors.Wrap(err, "failed to fetch index comments")
		}
	}

	query := fmt.Sprintf(`
	SELECT
		s.name AS schema_name,
		o.name AS table_name,
		i.object_id,
		i.index_id,
		i.name,
		i.type_desc,
		col.name AS column_name,
		ic.is_descending_key
	FROM sys.indexes i
	INNER JOIN sys.all_objects o ON o.object_id = i.object_id
	INNER JOIN sys.schemas s ON s.schema_id = o.schema_id
	INNER JOIN sys.index_columns ic ON ic.object_id = i.object_id AND ic.index_id = i.index_id
	INNER JOIN sys.all_columns col ON ic.column_id = col.column_id AND ic.object_id = col.object_id
	WHERE i.index_id > 0 AND i.is_primary_key = 0 AND i.is_unique_constraint = 0 
		AND s.name in (%s) AND o.type IN ('U', 'S', 'V')
	ORDER BY s.name, o.name, i.index_id, ic.key_ordinal
	`, quoteList(schemas))
	rows, err := txn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var schemaName, tableName, indexName, typeDesc, colName sql.NullString
		var objectID, indexID sql.NullInt32
		var isDescending sql.NullBool
		if err := rows.Scan(&schemaName, &tableName, &objectID, &indexID, &indexName, &typeDesc, &colName, &isDescending); err != nil {
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
			// Join with comments in Go
			if objectID.Valid && indexID.Valid {
				if comment, ok := indexCommentsMap[struct{ ObjectID, IndexID int }{int(objectID.Int32), int(indexID.Int32)}]; ok {
					index.Comment = comment
				}
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

	// Get spatial index comments separately
	spatialCommentsMap := make(map[struct{ ObjectID, IndexID int }]string)
	commentsQuery := `
	SELECT 
		ep.major_id AS object_id,
		ep.minor_id AS index_id,
		CAST(ep.value AS NVARCHAR(MAX)) AS comment
	FROM sys.extended_properties ep
	WHERE ep.class = 7 AND ep.name = 'MS_Description' AND ep.minor_id > 0
		AND ep.major_id IN (SELECT object_id FROM sys.spatial_indexes)`
	commentsRows, err := txn.Query(commentsQuery)
	if err == nil {
		defer commentsRows.Close()
		for commentsRows.Next() {
			var objectID, indexID int
			var comment sql.NullString
			if err := commentsRows.Scan(&objectID, &indexID, &comment); err != nil {
				continue
			}
			if comment.Valid {
				spatialCommentsMap[struct{ ObjectID, IndexID int }{objectID, indexID}] = comment.String
			}
		}
		if err := commentsRows.Err(); err != nil {
			return nil, errors.Wrap(err, "failed to fetch spatial index comments")
		}
	}

	// Get tessellation data separately
	tessellationMap := make(map[struct{ ObjectID, IndexID int }]struct {
		BoundingBox    [4]float64
		GridLevels     [4]string
		CellsPerObject int32
	})
	tessellationQuery := `
	SELECT
		object_id,
		index_id,
		COALESCE(bounding_box_xmin, 0),
		COALESCE(bounding_box_ymin, 0),
		COALESCE(bounding_box_xmax, 0),
		COALESCE(bounding_box_ymax, 0),
		COALESCE(level_1_grid_desc, ''),
		COALESCE(level_2_grid_desc, ''),
		COALESCE(level_3_grid_desc, ''),
		COALESCE(level_4_grid_desc, ''),
		COALESCE(cells_per_object, 0)
	FROM sys.spatial_index_tessellations`
	tessRows, err := txn.Query(tessellationQuery)
	if err == nil {
		defer tessRows.Close()
		for tessRows.Next() {
			var objectID, indexID int
			var xmin, ymin, xmax, ymax float64
			var level1, level2, level3, level4 string
			var cellsPerObject int32
			if err := tessRows.Scan(&objectID, &indexID, &xmin, &ymin, &xmax, &ymax,
				&level1, &level2, &level3, &level4, &cellsPerObject); err != nil {
				continue
			}
			key := struct{ ObjectID, IndexID int }{objectID, indexID}
			tessellationMap[key] = struct {
				BoundingBox    [4]float64
				GridLevels     [4]string
				CellsPerObject int32
			}{
				BoundingBox:    [4]float64{xmin, ymin, xmax, ymax},
				GridLevels:     [4]string{level1, level2, level3, level4},
				CellsPerObject: cellsPerObject,
			}
		}
		if err := tessRows.Err(); err != nil {
			return nil, errors.Wrap(err, "failed to iterate tessellation data")
		}
	}

	// Main query for spatial indexes
	query := fmt.Sprintf(`
	SELECT
		s.name AS schema_name,
		o.name AS table_name,
		si.object_id,
		si.index_id,
		i.name AS index_name,
		ic.key_ordinal,
		col.name AS column_name,
		si.spatial_index_type,
		si.spatial_index_type_desc,
		si.tessellation_scheme,
		col.system_type_id,
		i.filter_definition,
		i.fill_factor,
		CASE WHEN i.is_padded = 1 THEN 1 ELSE 0 END AS is_padded,
		CASE WHEN i.allow_row_locks = 1 THEN 1 ELSE 0 END AS allow_row_locks,
		CASE WHEN i.allow_page_locks = 1 THEN 1 ELSE 0 END AS allow_page_locks
	FROM sys.spatial_indexes si
	INNER JOIN sys.indexes i ON si.object_id = i.object_id AND si.index_id = i.index_id
	INNER JOIN sys.objects o ON i.object_id = o.object_id
	INNER JOIN sys.schemas s ON o.schema_id = s.schema_id
	INNER JOIN sys.index_columns ic ON i.object_id = ic.object_id AND i.index_id = ic.index_id
	INNER JOIN sys.columns col ON ic.object_id = col.object_id AND ic.column_id = col.column_id
	WHERE s.name IN (%s) AND o.type IN ('U', 'S', 'V')
	ORDER BY s.name, o.name, i.name, ic.key_ordinal
	`, quoteList(schemas))

	rows, err := txn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var schemaName, tableName, indexName, columnName sql.NullString
		var objectID, indexID, keyOrdinal sql.NullInt32
		var spatialIndexType sql.NullInt32
		var spatialIndexTypeDesc, tessellationScheme sql.NullString
		var systemTypeID sql.NullInt32
		var filterDefinition sql.NullString
		var fillFactor sql.NullInt32
		var isPadded, allowRowLocks, allowPageLocks sql.NullInt32

		if err := rows.Scan(&schemaName, &tableName, &objectID, &indexID, &indexName, &keyOrdinal, &columnName,
			&spatialIndexType, &spatialIndexTypeDesc, &tessellationScheme,
			&systemTypeID, &filterDefinition, &fillFactor, &isPadded,
			&allowRowLocks, &allowPageLocks); err != nil {
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

			// Join with comments in Go
			if objectID.Valid && indexID.Valid {
				if comment, ok := spatialCommentsMap[struct{ ObjectID, IndexID int }{int(objectID.Int32), int(indexID.Int32)}]; ok {
					index.Comment = comment
				}
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

			// Join with tessellation data in Go
			if objectID.Valid && indexID.Valid {
				tessKey := struct{ ObjectID, IndexID int }{int(objectID.Int32), int(indexID.Int32)}
				if tessData, ok := tessellationMap[tessKey]; ok {
					// Add bounding box (for GEOMETRY indexes or when explicitly provided)
					if tessData.BoundingBox[0] != 0 || tessData.BoundingBox[1] != 0 ||
						tessData.BoundingBox[2] != 0 || tessData.BoundingBox[3] != 0 {
						index.SpatialConfig.Tessellation.BoundingBox = &storepb.BoundingBox{
							Xmin: tessData.BoundingBox[0],
							Ymin: tessData.BoundingBox[1],
							Xmax: tessData.BoundingBox[2],
							Ymax: tessData.BoundingBox[3],
						}
					}

					// Add grid levels with proper descriptions
					gridLevels := []*storepb.GridLevel{}
					for i, level := range tessData.GridLevels {
						if level != "" {
							gridLevels = append(gridLevels, &storepb.GridLevel{Level: int32(i + 1), Density: level})
						}
					}
					index.SpatialConfig.Tessellation.GridLevels = gridLevels

					// Add cells per object
					if tessData.CellsPerObject > 0 {
						index.SpatialConfig.Tessellation.CellsPerObject = tessData.CellsPerObject
					}
				}
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
		// Try to enhance spatial indexes with additional properties
		if err := enhanceSpatialIndexesWithXMLPlan(txn, tableIndexes); err != nil {
			return nil, errors.Wrap(err, "failed to enhance spatial indexes")
		}
	}

	return tableIndexes, nil
}

// enhanceSpatialIndexesWithXMLPlan attempts to extract additional spatial index options
// that are not available in system tables but might be visible in execution plans
func enhanceSpatialIndexesWithXMLPlan(txn *sql.Tx, spatialIndexes map[db.TableKey][]*storepb.IndexMetadata) error {
	// This is a best-effort function, so we ignore errors
	// Collect all spatial indexes that need enhancement
	type indexKey struct {
		schema string
		table  string
		index  string
	}
	indexMap := make(map[indexKey]*storepb.IndexMetadata)

	for tableKey, indexes := range spatialIndexes {
		for _, index := range indexes {
			if index.Type != "SPATIAL" || index.SpatialConfig == nil {
				continue
			}
			key := indexKey{
				schema: tableKey.Schema,
				table:  tableKey.Table,
				index:  index.Name,
			}
			indexMap[key] = index
		}
	}

	if len(indexMap) == 0 {
		return nil
	}

	// Batch query all spatial index properties in a single roundtrip
	query := `
	SELECT 
		s.name AS schema_name,
		o.name AS table_name,
		i.name AS index_name,
		OBJECTPROPERTY(i.object_id, 'ExecIsAnsiNullsOn') AS ansi_nulls,
		OBJECTPROPERTY(i.object_id, 'ExecIsQuotedIdentOn') AS quoted_ident
	FROM sys.indexes i
	INNER JOIN sys.objects o ON i.object_id = o.object_id
	INNER JOIN sys.schemas s ON o.schema_id = s.schema_id
	INNER JOIN sys.spatial_indexes si ON si.object_id = i.object_id AND si.index_id = i.index_id
	WHERE i.type = 4 -- SPATIAL index type
	`

	rows, err := txn.Query(query)
	if err != nil {
		return errors.Wrap(err, "failed to query spatial index properties")
	}
	defer rows.Close()

	for rows.Next() {
		var schemaName, tableName, indexName sql.NullString
		var ansiNulls, quotedIdent sql.NullInt32

		if err := rows.Scan(&schemaName, &tableName, &indexName, &ansiNulls, &quotedIdent); err != nil {
			continue
		}

		if !schemaName.Valid || !tableName.Valid || !indexName.Valid {
			continue
		}

		key := indexKey{
			schema: schemaName.String,
			table:  tableName.String,
			index:  indexName.String,
		}

		if index, ok := indexMap[key]; ok {
			// If we don't have storage config yet, create a basic one
			// These properties give us hints about the index creation context
			if (ansiNulls.Valid || quotedIdent.Valid) && index.SpatialConfig.Storage == nil {
				index.SpatialConfig.Storage = &storepb.StorageConfig{}
				// These are indirect indicators but better than nothing
				index.SpatialConfig.Storage.AllowRowLocks = true
				index.SpatialConfig.Storage.AllowPageLocks = true
			}
		}
	}
	if err := rows.Err(); err != nil {
		return errors.Wrap(err, "error iterating spatial index properties")
	}
	return nil
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

	// Get view comments separately
	viewCommentsMap := make(map[int]string)
	commentsQuery := `
	SELECT 
		ep.major_id AS object_id,
		CAST(ep.value AS NVARCHAR(MAX)) AS comment
	FROM sys.extended_properties ep
	WHERE ep.minor_id = 0 AND ep.class = 1 AND ep.name = 'MS_Description'
		AND ep.major_id IN (SELECT object_id FROM sys.views)`
	commentsRows, err := txn.Query(commentsQuery)
	if err == nil {
		defer commentsRows.Close()
		for commentsRows.Next() {
			var objectID int
			var comment sql.NullString
			if err := commentsRows.Scan(&objectID, &comment); err != nil {
				continue
			}
			if comment.Valid {
				viewCommentsMap[objectID] = comment.String
			}
		}
		if err := commentsRows.Err(); err != nil {
			// Log error but continue - comments are not critical
			return nil, errors.Wrap(err, "failed to fetch view comments")
		}
	}

	query := `
		SELECT
			SCHEMA_NAME(v.schema_id) AS schema_name,
			v.name AS view_name,
			v.object_id,
			m.definition
		FROM sys.views v
		INNER JOIN sys.sql_modules m ON v.object_id = m.object_id
		ORDER BY schema_name, view_name;`
	rows, err := txn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		view := &storepb.ViewMetadata{}
		var schemaName string
		var objectID sql.NullInt32
		var definition sql.NullString
		if err := rows.Scan(&schemaName, &view.Name, &objectID, &definition); err != nil {
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

		// Join with comments in Go
		if objectID.Valid {
			if comment, ok := viewCommentsMap[int(objectID.Int32)]; ok {
				view.Comment = comment
			}
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
	// Get trigger comments separately
	triggerCommentsMap := make(map[int]string)
	commentsQuery := `
	SELECT 
		ep.major_id AS object_id,
		CAST(ep.value AS NVARCHAR(MAX)) AS comment
	FROM sys.extended_properties ep
	WHERE ep.minor_id = 0 AND ep.class = 1 AND ep.name = 'MS_Description'
		AND ep.major_id IN (SELECT object_id FROM sys.triggers WHERE is_disabled = 0 AND is_ms_shipped = 0)`
	commentsRows, err := txn.Query(commentsQuery)
	if err == nil {
		defer commentsRows.Close()
		for commentsRows.Next() {
			var objectID int
			var comment sql.NullString
			if err := commentsRows.Scan(&objectID, &comment); err != nil {
				continue
			}
			if comment.Valid {
				triggerCommentsMap[objectID] = comment.String
			}
		}
		if err := commentsRows.Err(); err != nil {
			// Log error but continue - comments are not critical
			return nil, nil, errors.Wrap(err, "failed to fetch trigger comments")
		}
	}

	query := `
SELECT
    st.object_id,
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
so.type AS objectType
FROM sys.triggers AS st
JOIN sys.sql_modules AS ssm ON st.object_id = ssm.object_id
JOIN sys.objects AS so ON st.parent_id = so.object_id
JOIN sys.schemas AS ss ON so.schema_id = ss.schema_id
WHERE st.is_disabled = 0 AND st.is_ms_shipped = 0 AND st.parent_id <> 0 AND so.type IN ('U', 'V')
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
		var objectID sql.NullInt32
		var name, events, timing, parentName, schemaName, parentType string
		var body sql.NullString
		if err := rows.Scan(&objectID, &name, &events, &timing, &body, &parentName, &schemaName, &parentType); err != nil {
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
		// Join with comments in Go
		if objectID.Valid {
			if comment, ok := triggerCommentsMap[int(objectID.Int32)]; ok {
				trigger.Comment = comment
			}
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
