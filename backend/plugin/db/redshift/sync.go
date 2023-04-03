// Package redshift is the plugin for RedShift driver.
package redshift

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/pkg/errors"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	"github.com/bytebase/bytebase/backend/plugin/parser"
	"github.com/bytebase/bytebase/backend/plugin/parser/ast"
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

	var filteredDatabases []*storepb.DatabaseMetadata
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
func (driver *Driver) SyncDBSchema(ctx context.Context, databaseName string) (*storepb.DatabaseMetadata, error) {
	// Query db info
	databases, err := driver.getDatabases(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get databases")
	}

	var databaseMetadata *storepb.DatabaseMetadata
	for _, database := range databases {
		if database.Name == databaseName {
			databaseMetadata = database
			break
		}
	}
	if databaseMetadata == nil {
		return nil, common.Errorf(common.NotFound, "database %q not found", databaseName)
	}

	sqldb, err := driver.GetDBConnection(ctx, databaseName)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get database connection for %q", databaseName)
	}
	txn, err := sqldb.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer txn.Rollback()

	schemaList, err := getSchemas(txn)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get schemas from database %q", databaseName)
	}
	tableMap, err := getTables(txn)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get tables from database %q", databaseName)
	}
	viewMap, err := getViews(txn)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get views from database %q", databaseName)
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

func getSchemas(txn *sql.Tx) ([]string, error) {
	// For Redshift, we will filter out the schema which owner is 'rdsdb' excluding 'public' or name prefix with 'pg_'.
	query := `
		SELECT
			n.nspname
		FROM
			pg_catalog.pg_namespace AS n
		WHERE
			n.nspname = 'public' OR (n.nspname !~ '^pg_' AND pg_catalog.pg_get_userbyid(n.nspowner) <> 'rdsdb');
	`
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
			pg_catalog.col_description(format('%s.%s', quote_ident(table_schema), quote_ident(table_name))::regclass, cols.ordinal_position::int) as column_comment
		FROM INFORMATION_SCHEMA.COLUMNS AS cols
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

// getViews gets all views of a database.
func getViews(txn *sql.Tx) (map[string][]*storepb.ViewMetadata, error) {
	viewMap := make(map[string][]*storepb.ViewMetadata)

	query := `
		SELECT schemaname, viewname, definition, obj_description(format('%s.%s', quote_ident(schemaname), quote_ident(viewname))::regclass) FROM pg_catalog.pg_views
		WHERE schemaname NOT IN ('pg_catalog', 'information_schema');`
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

	for schemaName, list := range viewMap {
		for _, view := range list {
			dependencies, err := getViewDependencies(txn, schemaName, view.Name)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to get view %q dependencies", view.Name)
			}
			view.DependentColumns = dependencies
		}
	}

	return viewMap, nil
}

// getViewDependencies gets the dependencies of a view.
func getViewDependencies(txn *sql.Tx, schemaName, viewName string) ([]*storepb.DependentColumn, error) {
	var result []*storepb.DependentColumn

	query := fmt.Sprintf(`
		SELECT source_ns.nspname as source_schema,
	  		source_table.relname as source_table,
	  		pg_attribute.attname as column_name
	  	FROM pg_depend 
	  		JOIN pg_rewrite ON pg_depend.objid = pg_rewrite.oid 
	  		JOIN pg_class as dependent_view ON pg_rewrite.ev_class = dependent_view.oid 
	  		JOIN pg_class as source_table ON pg_depend.refobjid = source_table.oid 
	  		JOIN pg_attribute ON pg_depend.refobjid = pg_attribute.attrelid 
	  		    AND pg_depend.refobjsubid = pg_attribute.attnum 
	  		JOIN pg_namespace dependent_ns ON dependent_ns.oid = dependent_view.relnamespace
	  		JOIN pg_namespace source_ns ON source_ns.oid = source_table.relnamespace
	  	WHERE 
	  		dependent_ns.nspname = '%s'
	  		AND dependent_view.relname = '%s'
	  		AND pg_attribute.attnum > 0 
	  	ORDER BY 1,2,3;
	`, schemaName, viewName)

	rows, err := txn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		dependentColumn := &storepb.DependentColumn{}
		if err := rows.Scan(&dependentColumn.Schema, &dependentColumn.Table, &dependentColumn.Column); err != nil {
			return nil, err
		}
		result = append(result, dependentColumn)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

// getIndexes gets all indices of a database.
func getIndexes(txn *sql.Tx) (map[db.TableKey][]*storepb.IndexMetadata, error) {
	indexMap := make(map[db.TableKey][]*storepb.IndexMetadata)

	query := `
		SELECT idx.schemaname, idx.tablename, idx.indexname, idx.indexdef, (SELECT 1
			FROM information_schema.table_constraints
			WHERE constraint_schema = idx.schemaname
			AND constraint_name = idx.indexname
			AND table_schema = idx.schemaname
			AND table_name = idx.tablename
			AND constraint_type = 'PRIMARY KEY') AS primary,
			obj_description(format('%s.%s', quote_ident(idx.schemaname), quote_ident(idx.indexname))::regclass) AS comment
		FROM pg_indexes AS idx WHERE idx.schemaname NOT IN ('pg_catalog', 'information_schema')
		ORDER BY idx.schemaname, idx.tablename, idx.indexname;`
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
func (driver *Driver) getDatabases(ctx context.Context) ([]*storepb.DatabaseMetadata, error) {
	var databases []*storepb.DatabaseMetadata
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
		database := storepb.DatabaseMetadata{}
		if err := rows.Scan(&database.Name, &database.CharacterSet); err != nil {
			return nil, err
		}
		databases = append(databases, &database)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return databases, nil
}
