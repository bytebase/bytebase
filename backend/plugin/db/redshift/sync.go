// Package redshift is the plugin for RedShift driver.
package redshift

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	parser "github.com/bytebase/bytebase/backend/plugin/parser/sql"
	"github.com/bytebase/bytebase/backend/plugin/parser/sql/ast"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// SyncInstance syncs the instance.
func (driver *Driver) SyncInstance(ctx context.Context) (*db.InstanceMetadata, error) {
	version, err := driver.getVersion(ctx)
	if err != nil {
		return nil, err
	}
	instanceRoles, err := driver.getInstanceRoles(ctx)
	if err != nil {
		return nil, err
	}

	databases, err := driver.getDatabases(ctx)
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
		Version:       version,
		InstanceRoles: instanceRoles,
		Databases:     filteredDatabases,
	}, nil
}

func (driver *Driver) getInstanceRoles(ctx context.Context) ([]*storepb.InstanceRoleMetadata, error) {
	// Reference: https://sourcegraph.com/github.com/postgres/postgres@REL_14_0/-/blob/src/bin/psql/describe.c?L3792
	query := `
	SELECT
		u.usename AS rolename,
		u.usesuper AS rolsuper,
		true AS rolinherit,
		false AS rolcreaterole,
		u.usecreatedb AS rolcreatedb,
		true AS rolcanlogin,
		-1 AS rolconnlimit,
		u.valuntil as rolvaliduntil
	FROM pg_catalog.pg_user u
	ORDER BY 1;
	`
	var instanceRoles []*storepb.InstanceRoleMetadata
	rows, err := driver.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var role string
		var super, inherit, createrole, createdb, canLogin bool
		var connectionLimit int32
		var validUntil sql.NullString
		if err := rows.Scan(
			&role,
			&super,
			&inherit,
			&createrole,
			&createdb,
			&canLogin,
			&connectionLimit,
			&validUntil,
		); err != nil {
			return nil, err
		}

		var attributes []string
		if super {
			attributes = append(attributes, "Superuser")
		}
		if !inherit {
			attributes = append(attributes, "No inheritance")
		}
		if createrole {
			attributes = append(attributes, "Create role")
		}
		if createdb {
			attributes = append(attributes, "Create DB")
		}
		if !canLogin {
			attributes = append(attributes, "Cannot login")
		}
		if connectionLimit >= 0 {
			if connectionLimit == 0 {
				attributes = append(attributes, "No connections")
			} else if connectionLimit == 1 {
				attributes = append(attributes, "1 connection")
			} else {
				attributes = append(attributes, fmt.Sprintf("%d connections", connectionLimit))
			}
		}
		if validUntil.Valid {
			attributes = append(attributes, fmt.Sprintf("Password valid until %s", validUntil.String))
		}
		instanceRoles = append(instanceRoles, &storepb.InstanceRoleMetadata{
			Name:  role,
			Grant: strings.Join(attributes, ", "),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return instanceRoles, nil
}

// SyncDBSchema syncs a single database schema.
func (driver *Driver) SyncDBSchema(ctx context.Context) (*storepb.DatabaseSchemaMetadata, error) {
	// Query db info
	databases, err := driver.getDatabases(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get databases")
	}

	var databaseMetadata *storepb.DatabaseSchemaMetadata
	for _, database := range databases {
		if database.Name == driver.databaseName {
			databaseMetadata = database
			break
		}
	}
	if databaseMetadata == nil {
		return nil, common.Errorf(common.NotFound, "database %q not found", driver.databaseName)
	}

	if err != nil {
		return nil, errors.Wrapf(err, "failed to get database connection for %q", driver.databaseName)
	}
	txn, err := driver.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer txn.Rollback()

	schemaList, err := driver.getSchemas(txn)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get schemas from database %q", driver.databaseName)
	}
	var tableMap map[string][]*storepb.TableMetadata
	var viewMap map[string][]*storepb.ViewMetadata
	if driver.datashare {
		tableMap, err = driver.getDatashareTables(txn)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get tables from datashare database %q", driver.databaseName)
		}
	} else {
		tableMap, err = getTables(txn)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get tables from database %q", driver.databaseName)
		}
		viewMap, err = getViews(txn)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get views from database %q", driver.databaseName)
		}
	}

	if err := txn.Commit(); err != nil {
		return nil, err
	}

	schemaNameMap := make(map[string]bool)
	for _, schemaName := range schemaList {
		schemaNameMap[schemaName] = true
	}
	for schemaName := range tableMap {
		schemaNameMap[schemaName] = true
	}
	for schemaName := range viewMap {
		schemaNameMap[schemaName] = true
	}
	var schemaNames []string
	for schemaName := range schemaNameMap {
		schemaNames = append(schemaNames, schemaName)
	}
	sort.Strings(schemaNames)
	for _, schemaName := range schemaNames {
		var tables []*storepb.TableMetadata
		var views []*storepb.ViewMetadata
		var exists bool
		if tables, exists = tableMap[schemaName]; !exists {
			tables = []*storepb.TableMetadata{}
		}
		if views, exists = viewMap[schemaName]; !exists {
			views = []*storepb.ViewMetadata{}
		}
		databaseMetadata.Schemas = append(databaseMetadata.Schemas, &storepb.SchemaMetadata{
			Name:   schemaName,
			Tables: tables,
			Views:  views,
		})
	}

	return databaseMetadata, err
}

func getForeignKeys(txn *sql.Tx) (map[db.TableKey][]*storepb.ForeignKeyMetadata, error) {
	query := `
	SELECT
		n.nspname AS fk_schema,
		conrelid::regclass AS fk_table,
		conname AS fk_name,
		(SELECT nspname FROM pg_namespace JOIN pg_class ON pg_namespace.oid = pg_class.relnamespace WHERE c.confrelid = pg_class.oid) AS fk_ref_schema,
		confrelid::regclass AS fk_ref_table,
		confdeltype AS delete_option,
		confupdtype AS update_option,
		confmatchtype AS match_option,
		pg_get_constraintdef(c.oid) AS fk_def
	FROM
		pg_constraint c
		JOIN pg_namespace n ON n.oid = c.connamespace
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

func (driver *Driver) getSchemas(txn *sql.Tx) ([]string, error) {
	query := `
		SELECT
			schema_name
		FROM
			SVV_ALL_SCHEMAS
		WHERE
			database_name = $1;
	`
	rows, err := txn.Query(query, driver.databaseName)
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
func getTables(txn *sql.Tx) (map[string][]*storepb.TableMetadata, error) {
	columnMap, err := getTableColumns(txn)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get table columns")
	}
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
		var defaultStr, collation, udtSchema, udtName, comment sql.NullString
		if err := rows.Scan(&schemaName, &tableName, &column.Name, &column.Type, &column.Position, &defaultStr, &nullable, &collation, &udtSchema, &udtName, &comment); err != nil {
			return nil, err
		}
		if defaultStr.Valid {
			column.Default = &wrapperspb.StringValue{Value: defaultStr.String}
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
func (driver *Driver) getDatashareTables(txn *sql.Tx) (map[string][]*storepb.TableMetadata, error) {
	columnMap, err := driver.getDatashareTableColumns(txn)
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
	rows, err := txn.Query(query, driver.databaseName)
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
func (driver *Driver) getDatashareTableColumns(txn *sql.Tx) (map[db.TableKey][]*storepb.ColumnMetadata, error) {
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
	rows, err := txn.Query(query, driver.databaseName)
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
			column.Default = &wrapperspb.StringValue{Value: defaultStr.String}
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
func getViews(txn *sql.Tx) (map[string][]*storepb.ViewMetadata, error) {
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
	WHERE pgv.schemaname NOT IN ('pg_catalog', 'information_schema');
	`
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

	query := `
	SELECT 
		pgidx.schemaname, 
		pgidx.tablename, 
		pgidx.indexname, 
		pgidx.indexdef, 
		(
		SELECT 1
			FROM information_schema.table_constraints
			WHERE 	constraint_schema = pgidx.schemaname
					AND constraint_name = pgidx.indexname
					AND table_schema = pgidx.schemaname
					AND table_name = pgidx.tablename
					AND constraint_type = 'PRIMARY KEY'
		) AS primary,
		obj_description(pc.oid) AS comment
	FROM
		pg_indexes AS pgidx 
		JOIN pg_namespace AS pns ON pns.nspname = pgidx.schemaname
		JOIN pg_class AS pc ON pc.relname = pgidx.indexname AND pns.oid = pc.relnamespace
	WHERE 
		pgidx.schemaname NOT IN ('pg_catalog', 'information_schema')
	ORDER BY
		pgidx.schemaname, pgidx.tablename, pgidx.indexname;`
	rows, err := txn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		index := &storepb.IndexMetadata{}
		var schemaName, tableName, statement string
		var primary sql.NullInt32
		var comment sql.NullString
		if err := rows.Scan(&schemaName, &tableName, &index.Name, &statement, &primary, &comment); err != nil {
			return nil, err
		}

		nodes, err := parser.Parse(parser.Postgres, parser.ParseContext{}, statement)
		if err != nil {
			return nil, err
		}
		if len(nodes) != 1 {
			return nil, errors.Errorf("invalid number of statements %v, expecting one", len(nodes))
		}
		node, ok := nodes[0].(*ast.CreateIndexStmt)
		if !ok {
			return nil, errors.Errorf("statement %q is not index statement", statement)
		}

		index.Type = getIndexMethodType(statement)
		index.Unique = node.Index.Unique
		index.Expressions = node.Index.GetKeyNameList()
		if err != nil {
			return nil, err
		}
		if primary.Valid && primary.Int32 == 1 {
			index.Primary = true
		}
		if comment.Valid {
			index.Comment = comment.String
		}

		key := db.TableKey{Schema: schemaName, Table: tableName}
		indexMap[key] = append(indexMap[key], index)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return indexMap, nil
}

func getIndexMethodType(stmt string) string {
	re := regexp.MustCompile(`USING (\w+) `)
	matches := re.FindStringSubmatch(stmt)
	if len(matches) == 0 {
		return ""
	}
	return matches[1]
}

func (driver *Driver) getVersion(ctx context.Context) (string, error) {
	// Redshift doesn't support SHOW server_version to retrieve the clean version number.
	// We can parse the output of `SELECT version()` to get the PostgreSQL version and the
	// Redshift version because Redshift is based on PostgreSQL.
	// For example, the output of `SELECT version()` is:
	// PostgreSQL 8.0.2 on i686-pc-linux-gnu, compiled by GCC gcc (GCC) 3.4.2 20041017 (Red Hat 3.4.2-6.fc3), Redshift 1.0.48042
	// We will return the 'Redshift 1.0.48042 based on PostgreSQL 8.0.2'.
	rows, err := driver.db.QueryContext(ctx, "SELECT version()")
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
		log.Debug("Failed to parse version string", zap.String("version", version))
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
func (driver *Driver) getDatabases(ctx context.Context) ([]*storepb.DatabaseSchemaMetadata, error) {
	consumerDatabases := make(map[string]bool)
	dsRows, err := driver.db.QueryContext(ctx, `
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
	rows, err := driver.db.QueryContext(ctx, `
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

// SyncSlowQuery syncs the slow query.
func (*Driver) SyncSlowQuery(_ context.Context, _ time.Time) (map[string]*storepb.SlowQueryStatistics, error) {
	return nil, errors.Errorf("not implemented")
}

// CheckSlowQueryLogEnabled checks if slow query log is enabled.
func (*Driver) CheckSlowQueryLogEnabled(_ context.Context) error {
	return errors.Errorf("not implemented")
}
