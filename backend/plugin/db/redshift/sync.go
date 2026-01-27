// Package redshift is the plugin for RedShift driver.
package redshift

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

// SyncInstance syncs the instance.
func (d *Driver) SyncInstance(ctx context.Context) (*db.InstanceMetadata, error) {
	version, err := d.getVersion(ctx)
	if err != nil {
		return nil, err
	}
	instanceRoles, err := d.getInstanceRoles(ctx)
	if err != nil {
		return nil, err
	}

	databases, err := d.getDatabases(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get databases")
	}

	var filteredDatabases []*storepb.DatabaseSchemaMetadata
	for _, database := range databases {
		// Skip all system databases
		if _, ok := excludedDatabaseList[database.Name]; ok {
			continue
		}
		filteredDatabases = append(filteredDatabases, database)
	}

	return &db.InstanceMetadata{
		Version:   version,
		Databases: filteredDatabases,
		Metadata: &storepb.Instance{
			Roles: instanceRoles,
		},
	}, nil
}

// SyncDBSchema syncs a single database schema.
func (d *Driver) SyncDBSchema(ctx context.Context) (*storepb.DatabaseSchemaMetadata, error) {
	// Query db info
	databases, err := d.getDatabases(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get databases")
	}

	var databaseMetadata *storepb.DatabaseSchemaMetadata
	for _, database := range databases {
		if database.Name == d.databaseName {
			databaseMetadata = database
			break
		}
	}
	if databaseMetadata == nil {
		return nil, common.Errorf(common.NotFound, "database %q not found", d.databaseName)
	}

	txn, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer txn.Rollback()

	schemaList, err := d.getSchemas(txn)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get schemas from database %q", d.databaseName)
	}
	var tableMap map[string][]*storepb.TableMetadata
	var viewMap map[string][]*storepb.ViewMetadata
	if d.datashare {
		tableMap, err = d.getDatashareTables(txn)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get tables from datashare database %q", d.databaseName)
		}
	} else {
		columnMap, err := getTableColumns(txn)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get table columns")
		}
		tableMap, err = getTables(txn, columnMap)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get tables from database %q", d.databaseName)
		}
		viewMap, err = getViews(txn, columnMap)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get views from database %q", d.databaseName)
		}
	}
	if err := txn.Commit(); err != nil {
		return nil, err
	}
	for _, schemaName := range schemaList {
		databaseMetadata.Schemas = append(databaseMetadata.Schemas, &storepb.SchemaMetadata{
			Name:   schemaName,
			Tables: tableMap[schemaName],
			Views:  viewMap[schemaName],
		})
	}

	return databaseMetadata, err
}

func getForeignKeys(txn *sql.Tx) (map[db.TableKey][]*storepb.ForeignKeyMetadata, error) {
	query := `
	SELECT
		n.nspname AS fk_schema,
		quote_ident(fk_nsp.nspname) || '.' || quote_ident(fk_cls.relname) AS fk_table,
		conname AS fk_name,
		ref_nsp.nspname AS fk_ref_schema,
		quote_ident(ref_nsp.nspname) || '.' || quote_ident(ref_cls.relname) AS fk_ref_table,
		confdeltype AS delete_option,
		confupdtype AS update_option,
		confmatchtype AS match_option,
		pg_get_constraintdef(c.oid) AS fk_def
	FROM
		pg_constraint c
		JOIN pg_namespace n ON n.oid = c.connamespace
		JOIN pg_class fk_cls ON fk_cls.oid = c.conrelid
		JOIN pg_namespace fk_nsp ON fk_nsp.oid = fk_cls.relnamespace
		JOIN pg_class ref_cls ON ref_cls.oid = c.confrelid
		JOIN pg_namespace ref_nsp ON ref_nsp.oid = ref_cls.relnamespace
	WHERE
		n.nspname NOT IN('pg_catalog', 'information_schema')
		AND c.contype = 'f'
	ORDER BY fk_schema, fk_table, fk_name;
	`
	foreignKeysMap := make(map[db.TableKey][]*storepb.ForeignKeyMetadata)
	rows, err := txn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var fkMetadata storepb.ForeignKeyMetadata
		var fkSchema, fkTable, fkDefinition string
		if err := rows.Scan(
			&fkSchema,
			&fkTable,
			&fkMetadata.Name,
			&fkMetadata.ReferencedSchema,
			&fkMetadata.ReferencedTable,
			&fkMetadata.OnDelete,
			&fkMetadata.OnUpdate,
			&fkMetadata.MatchType,
			&fkDefinition,
		); err != nil {
			return nil, err
		}

		fkTable = formatTableNameFromRegclass(fkTable)
		fkMetadata.ReferencedTable = formatTableNameFromRegclass(fkMetadata.ReferencedTable)
		fkMetadata.OnDelete = convertForeignKeyActionCode(fkMetadata.OnDelete)
		fkMetadata.OnUpdate = convertForeignKeyActionCode(fkMetadata.OnUpdate)
		fkMetadata.MatchType = convertForeignKeyMatchType(fkMetadata.MatchType)

		if fkMetadata.Columns, fkMetadata.ReferencedColumns, err = getForeignKeyColumnsAndReferencedColumns(fkDefinition); err != nil {
			return nil, err
		}
		key := db.TableKey{Schema: fkSchema, Table: fkTable}
		foreignKeysMap[key] = append(foreignKeysMap[key], &fkMetadata)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return foreignKeysMap, nil
}

func convertForeignKeyMatchType(in string) string {
	switch in {
	case "f":
		return "FULL"
	case "p":
		return "PARTIAL"
	case "s":
		return "SIMPLE"
	default:
		return in
	}
}

func convertForeignKeyActionCode(in string) string {
	switch in {
	case "a":
		return "NO ACTION"
	case "r":
		return "RESTRICT"
	case "c":
		return "CASCADE"
	case "n":
		return "SET NULL"
	case "d":
		return "SET DEFAULT"
	default:
		return in
	}
}

func getForeignKeyColumnsAndReferencedColumns(definition string) ([]string, []string, error) {
	columnsRegexp := regexp.MustCompile(`FOREIGN KEY \((.*)\) REFERENCES (.*)\((.*)\)`)
	matches := columnsRegexp.FindStringSubmatch(definition)
	if len(matches) != 4 {
		return nil, nil, errors.Errorf("invalid foreign key definition: %q", definition)
	}
	columnList, err := getColumnList(matches[1])
	if err != nil {
		return nil, nil, errors.Wrapf(err, "invalid foreign key definition: %q", definition)
	}
	referencedColumnList, err := getColumnList(matches[3])
	if err != nil {
		return nil, nil, errors.Wrapf(err, "invalid foreign key definition: %q", definition)
	}

	return columnList, referencedColumnList, nil
}

func getColumnList(definition string) ([]string, error) {
	list := strings.Split(definition, ",")
	if len(list) == 0 {
		return nil, errors.Errorf("invalid column list definition: %q", definition)
	}
	var result []string
	for _, name := range list {
		name = strings.TrimSpace(name)
		name = strings.Trim(name, `"`)
		result = append(result, name)
	}
	return result, nil
}

func formatTableNameFromRegclass(name string) string {
	if strings.Contains(name, ".") {
		name = name[1+strings.Index(name, "."):]
	}
	return strings.Trim(name, `"`)
}

func (d *Driver) getSchemas(txn *sql.Tx) ([]string, error) {
	query := `
		SELECT
			schema_name
		FROM
			SVV_ALL_SCHEMAS
		WHERE
			database_name = $1
		ORDER BY
			schema_name;
	`
	rows, err := txn.Query(query, d.databaseName)
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
func getTables(txn *sql.Tx, columnMap map[db.TableKey][]*storepb.ColumnMetadata) (map[string][]*storepb.TableMetadata, error) {
	indexMap, err := getIndexes(txn)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get indices")
	}
	foreignKeysMap, err := getForeignKeys(txn)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get foreign keys")
	}

	tableMap := make(map[string][]*storepb.TableMetadata)
	query := `
	SELECT
		ptbl.schemaname,
		ptbl.tablename,
		0, -- data size
		0, -- index size
		GREATEST(pc.reltuples::bigint, 0::bigint) AS estimate,
		obj_description(pc.oid) AS comment
	FROM pg_catalog.pg_tables AS ptbl
	JOIN pg_namespace AS pns ON pns.nspname = ptbl.schemaname
	LEFT JOIN pg_class AS pc ON pc.relname = ptbl.tablename AND pns.oid = pc.relnamespace
	WHERE ptbl.schemaname NOT IN ('pg_catalog', 'information_schema');`
	rows, err := txn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		table := &storepb.TableMetadata{}
		var schemaName string
		var comment sql.NullString
		if err := rows.Scan(&schemaName, &table.Name, &table.DataSize, &table.IndexSize, &table.RowCount, &comment); err != nil {
			return nil, err
		}
		if comment.Valid {
			table.Comment = comment.String
		}
		key := db.TableKey{Schema: schemaName, Table: table.Name}
		table.Columns = columnMap[key]
		table.Indexes = indexMap[key]
		table.ForeignKeys = foreignKeysMap[key]

		tableMap[schemaName] = append(tableMap[schemaName], table)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tableMap, nil
}

// getTableColumns gets the columns of a table.
func getTableColumns(txn *sql.Tx) (map[db.TableKey][]*storepb.ColumnMetadata, error) {
	columnsMap := make(map[db.TableKey][]*storepb.ColumnMetadata)

	query := `
	SELECT
		cols.table_schema,
		cols.table_name,
		cols.column_name,
		cols.data_type,
		cols.character_maximum_length,
		cols.ordinal_position,
		cols.column_default,
		cols.is_nullable,
		cols.collation_name,
		cols.udt_schema,
		cols.udt_name,
		pg_catalog.col_description(pc.oid, cols.ordinal_position::int) as column_comment
	FROM 
		INFORMATION_SCHEMA.COLUMNS AS cols
		JOIN pg_namespace AS pns ON pns.nspname = cols.table_schema
		LEFT JOIN pg_class AS pc ON pc.relname = cols.table_name AND pns.oid = pc.relnamespace
	WHERE cols.table_schema NOT IN ('pg_catalog', 'information_schema')
	ORDER BY cols.table_schema, cols.table_name, cols.ordinal_position;`
	rows, err := txn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		column := &storepb.ColumnMetadata{}
		var schemaName, tableName, nullable string
		var characterMaxLength, defaultStr, collation, udtSchema, udtName, comment sql.NullString
		if err := rows.Scan(&schemaName, &tableName, &column.Name, &column.Type, &characterMaxLength, &column.Position, &defaultStr, &nullable, &collation, &udtSchema, &udtName, &comment); err != nil {
			return nil, err
		}
		if defaultStr.Valid {
			// Store in Default field (migration from DefaultExpression to Default)
			column.Default = defaultStr.String
		}
		isNullBool, err := util.ConvertYesNo(nullable)
		if err != nil {
			return nil, err
		}
		column.Nullable = isNullBool
		switch column.Type {
		case "USER-DEFINED":
			column.Type = fmt.Sprintf("%s.%s", udtSchema.String, udtName.String)
		case "ARRAY":
			column.Type = udtName.String
		case "character", "character varying", "bit", "bit varying":
			if characterMaxLength.Valid {
				// For character varying(n), the character maximum length is n.
				// For character without length specifier, key character_maximum_length is null,
				// we don't need to append the length.
				// https://www.postgresql.org/docs/current/infoschema-columns.html.
				column.Type = fmt.Sprintf("%s(%s)", column.Type, characterMaxLength.String)
			}
		default:
			// No special handling needed for other types
		}
		column.Collation = collation.String
		column.Comment = comment.String

		key := db.TableKey{Schema: schemaName, Table: tableName}
		columnsMap[key] = append(columnsMap[key], column)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return columnsMap, nil
}

// getDatashareTables gets all tables of a datashare database.
func (d *Driver) getDatashareTables(txn *sql.Tx) (map[string][]*storepb.TableMetadata, error) {
	columnMap, err := d.getDatashareTableColumns(txn)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get table columns")
	}

	tableMap := make(map[string][]*storepb.TableMetadata)

	// table_type
	query := `
	SELECT
		schema_name,
		table_name
	FROM SVV_ALL_TABLES
	WHERE database_name = $1;`
	rows, err := txn.Query(query, d.databaseName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		table := &storepb.TableMetadata{}
		var schemaName string
		if err := rows.Scan(&schemaName, &table.Name); err != nil {
			return nil, err
		}
		key := db.TableKey{Schema: schemaName, Table: table.Name}
		table.Columns = columnMap[key]

		tableMap[schemaName] = append(tableMap[schemaName], table)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tableMap, nil
}

// getDatashareTableColumns gets the columns of tables in datashare database.
func (d *Driver) getDatashareTableColumns(txn *sql.Tx) (map[db.TableKey][]*storepb.ColumnMetadata, error) {
	columnsMap := make(map[db.TableKey][]*storepb.ColumnMetadata)

	query := `
	SELECT
		schema_name,
		table_name,
		column_name,
		data_type,
		ordinal_position,
		column_default,
		is_nullable,
		character_maximum_length
	FROM SVV_ALL_COLUMNS
	WHERE database_name = $1;`
	rows, err := txn.Query(query, d.databaseName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		column := &storepb.ColumnMetadata{}
		var schemaName, tableName, nullable string
		var defaultStr sql.NullString
		var varcharMaxLength sql.NullInt32
		if err := rows.Scan(&schemaName, &tableName, &column.Name, &column.Type, &column.Position, &defaultStr, &nullable, &varcharMaxLength); err != nil {
			return nil, err
		}
		if defaultStr.Valid {
			// Store in Default field (migration from DefaultExpression to Default)
			column.Default = defaultStr.String
		}
		isNullBool, err := util.ConvertYesNo(nullable)
		if err != nil {
			return nil, err
		}
		column.Nullable = isNullBool
		if column.Type == "character varying" && varcharMaxLength.Valid {
			column.Type = fmt.Sprintf("varchar(%d)", varcharMaxLength.Int32)
		}

		key := db.TableKey{Schema: schemaName, Table: tableName}
		columnsMap[key] = append(columnsMap[key], column)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return columnsMap, nil
}

// getViews gets all views of a database.
func getViews(txn *sql.Tx, columnMap map[db.TableKey][]*storepb.ColumnMetadata) (map[string][]*storepb.ViewMetadata, error) {
	viewMap := make(map[string][]*storepb.ViewMetadata)

	query := `
SELECT
	pgv.schemaname,
	pgv.viewname,
	pgv.definition,
	obj_description(pc.oid) AS comment
FROM pg_catalog.pg_views AS pgv
JOIN pg_namespace AS pns ON pns.nspname = pgv.schemaname
JOIN pg_class AS pc ON pc.relname = pgv.viewname AND pns.oid = pc.relnamespace
WHERE pgv.schemaname NOT IN ('pg_catalog', 'information_schema')
ORDER BY pgv.schemaname, pgv.viewname;`
	rows, err := txn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		view := &storepb.ViewMetadata{}
		var schemaName string
		var def, comment sql.NullString
		if err := rows.Scan(&schemaName, &view.Name, &def, &comment); err != nil {
			return nil, err
		}
		// Return error on NULL view definition.
		// https://github.com/bytebase/bytebase/issues/343
		if !def.Valid {
			return nil, errors.Errorf("schema %q view %q has empty definition; please check whether proper privileges have been granted to Bytebase", schemaName, view.Name)
		}
		view.Definition = def.String
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

// getIndexes gets all indices of a database.
func getIndexes(txn *sql.Tx) (map[db.TableKey][]*storepb.IndexMetadata, error) {
	indexMap := make(map[db.TableKey][]*storepb.IndexMetadata)

	// Use generate_series to mimic PostgreSQL's generate_subscripts for getting column expressions
	// Combined with pg_get_indexdef to get accurate column expressions
	query := `
	WITH index_positions AS (
		SELECT generate_series(0, 31) as pos
	)
	SELECT 
		pgidx.schemaname,
		pgidx.tablename, 
		pgidx.indexname,
		pgidx.indexdef,
		ix.indisunique,
		ix.indisprimary,
		p.pos + 1 as position,
		pg_get_indexdef(pc.oid, p.pos + 1, true) as column_expression,
		obj_description(pc.oid) AS comment
	FROM
		pg_indexes AS pgidx 
		JOIN pg_namespace AS pns ON pns.nspname = pgidx.schemaname
		JOIN pg_class AS pc ON pc.relname = pgidx.indexname AND pns.oid = pc.relnamespace
		JOIN pg_index AS ix ON ix.indexrelid = pc.oid
		CROSS JOIN index_positions p
	WHERE 
		pgidx.schemaname NOT IN ('pg_catalog', 'information_schema')
		AND ix.indkey[p.pos] > 0  -- Only process positions that have actual columns
	ORDER BY
		pgidx.schemaname, pgidx.tablename, pgidx.indexname, p.pos;`

	rows, err := txn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Track index data as we process rows
	type indexKey struct {
		schema string
		table  string
		index  string
	}
	indexData := make(map[indexKey]*storepb.IndexMetadata)
	indexColumns := make(map[indexKey][]string)

	for rows.Next() {
		var schemaName, tableName, indexName, indexDef string
		var isUnique, isPrimary bool
		var position int
		var columnExpression sql.NullString
		var comment sql.NullString

		if err := rows.Scan(&schemaName, &tableName, &indexName, &indexDef,
			&isUnique, &isPrimary, &position, &columnExpression, &comment); err != nil {
			return nil, err
		}

		key := indexKey{schema: schemaName, table: tableName, index: indexName}

		// Create index metadata on first encounter
		if _, exists := indexData[key]; !exists {
			indexData[key] = &storepb.IndexMetadata{
				Name:       indexName,
				Definition: indexDef,
				Type:       getIndexMethodType(indexDef),
				Unique:     isUnique,
				Primary:    isPrimary,
			}
			if comment.Valid {
				indexData[key].Comment = comment.String
			}
		}

		// Collect column expressions
		if columnExpression.Valid {
			indexColumns[key] = append(indexColumns[key], columnExpression.String)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Build final index map with expressions
	for key, index := range indexData {
		if columns, ok := indexColumns[key]; ok {
			index.Expressions = columns
		}
		tableKey := db.TableKey{Schema: key.schema, Table: key.table}
		indexMap[tableKey] = append(indexMap[tableKey], index)
	}

	return indexMap, nil
}

func getIndexMethodType(stmt string) string {
	re := regexp.MustCompile(`USING (\w+)`)
	matches := re.FindStringSubmatch(stmt)
	if len(matches) > 1 {
		return matches[1]
	}
	return "btree" // Default for Redshift
}

func (d *Driver) getVersion(ctx context.Context) (string, error) {
	// Redshift doesn't support SHOW server_version to retrieve the clean version number.
	// We can parse the output of `SELECT version()` to get the PostgreSQL version and the
	// Redshift version because Redshift is based on PostgreSQL.
	// For example, the output of `SELECT version()` is:
	// PostgreSQL 8.0.2 on i686-pc-linux-gnu, compiled by GCC gcc (GCC) 3.4.2 20041017 (Red Hat 3.4.2-6.fc3), Redshift 1.0.48042
	// We will return the 'Redshift 1.0.48042 based on PostgreSQL 8.0.2'.
	rows, err := d.db.QueryContext(ctx, "SELECT version()")
	if err != nil {
		return "", err
	}
	defer rows.Close()
	var version string
	for rows.Next() {
		if err := rows.Scan(&version); err != nil {
			return "", err
		}
	}
	if err := rows.Err(); err != nil {
		return "", err
	}
	// We try to parse the version string to get the PostgreSQL version and the Redshift version, but it's not a big deal if we fail.
	// We will just return the version string as is.
	pgVersion, redshiftVersion, err := getPgVersionAndRedshiftVersion(version)
	if err != nil {
		slog.Debug("Failed to parse version string", slog.String("version", version))
		// nolint
		return version, nil
	}
	return buildRedshiftVersionString(redshiftVersion, pgVersion), nil
}

// parseVersionRegex is a regex to parse the output from Redshift's `SELECT version()`, captures the PostgreSQL version and the Redshift version.
var parseVersionRegex = regexp.MustCompile(`(?i)^PostgreSQL (?P<pgVersion>\d+\.\d+\.\d+) on .*, Redshift (?P<redshiftVersion>\d+\.\d+\.\d+)`)

// getPgVersionAndRedshiftVersion parses the output from Redshift's `SELECT version()` to get the PostgreSQL version and the Redshift version.
func getPgVersionAndRedshiftVersion(version string) (string, string, error) {
	matches := parseVersionRegex.FindStringSubmatch(version)
	if len(matches) == 0 {
		return "", "", errors.Errorf("unable to parse version string: %s", version)
	}

	pgVersion := ""
	redshiftVersion := ""
	for i, name := range parseVersionRegex.SubexpNames() {
		if i != 0 && name != "" {
			switch name {
			case "pgVersion":
				pgVersion = matches[i]
			case "redshiftVersion":
				redshiftVersion = matches[i]
			default:
				// Ignore other named groups
			}
		}
	}

	return pgVersion, redshiftVersion, nil
}

// buildRedshiftVersionString builds the Redshift version string, format is "Redshift <redshiftVersion> based on PostgreSQL <postgresVersion>".
func buildRedshiftVersionString(redshiftVersion, postgresVersion string) string {
	return "Redshift " + redshiftVersion + " based on PostgreSQL " + postgresVersion
}

// getDatabases gets all databases of an instance.
func (d *Driver) getDatabases(ctx context.Context) ([]*storepb.DatabaseSchemaMetadata, error) {
	consumerDatabases := make(map[string]bool)
	dsRows, err := d.db.QueryContext(ctx, `
		SELECT consumer_database FROM SVV_DATASHARES WHERE share_type = 'INBOUND';
	`)
	if err != nil {
		return nil, err
	}
	defer dsRows.Close()

	for dsRows.Next() {
		var v string
		if err := dsRows.Scan(&v); err != nil {
			return nil, err
		}
		consumerDatabases[v] = true
	}
	if err := dsRows.Err(); err != nil {
		return nil, err
	}

	var databases []*storepb.DatabaseSchemaMetadata
	rows, err := d.db.QueryContext(ctx, `
		SELECT datname,
		pg_encoding_to_char(encoding)
		FROM pg_database;
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		database := storepb.DatabaseSchemaMetadata{}
		if err := rows.Scan(&database.Name, &database.CharacterSet); err != nil {
			return nil, err
		}
		database.Datashare = consumerDatabases[database.Name]
		databases = append(databases, &database)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return databases, nil
}
