package risingwave

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	parser "github.com/bytebase/bytebase/backend/plugin/parser/sql"
	"github.com/bytebase/bytebase/backend/plugin/parser/sql/ast"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

const systemSchemas = "'information_schema', 'pg_catalog', 'rw_catalog', 'pg_toast', '_timescaledb_cache', '_timescaledb_catalog', '_timescaledb_internal', '_timescaledb_config', 'timescaledb_information', 'timescaledb_experimental'"

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

	// Query db info
	databases, err := driver.getDatabases(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get databases")
	}

	var filteredDatabases []*storepb.DatabaseSchemaMetadata
	for _, database := range databases {
		// Skip all system databases
		if _, ok := ExcludedDatabaseList[database.Name]; ok {
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

	txn, err := driver.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer txn.Rollback()

	schemaList, err := getSchemas(txn)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get schemas from database %q", driver.databaseName)
	}
	tableMap, err := getTables(txn)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get tables from database %q", driver.databaseName)
	}
	viewMap, err := getViews(txn)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get views from database %q", driver.databaseName)
	}
	functionMap, err := getFunctions(txn)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get functions from database %q", driver.databaseName)
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
		var functions []*storepb.FunctionMetadata
		var exists bool
		if tables, exists = tableMap[schemaName]; !exists {
			tables = []*storepb.TableMetadata{}
		}
		if views, exists = viewMap[schemaName]; !exists {
			views = []*storepb.ViewMetadata{}
		}
		if functions, exists = functionMap[schemaName]; !exists {
			functions = []*storepb.FunctionMetadata{}
		}
		databaseMetadata.Schemas = append(databaseMetadata.Schemas, &storepb.SchemaMetadata{
			Name:      schemaName,
			Tables:    tables,
			Views:     views,
			Functions: functions,
		})
	}
	// No extensions in RisingWave.
	databaseMetadata.Extensions = make([]*storepb.ExtensionMetadata, 0)

	return databaseMetadata, err
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

var listSchemaQuery = fmt.Sprintf(`
SELECT nspname
FROM pg_catalog.pg_namespace
WHERE nspname NOT IN (%s);
`, systemSchemas)

func getSchemas(txn *sql.Tx) ([]string, error) {
	rows, err := txn.Query(listSchemaQuery)
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

var listTableQuery = `
SELECT
	T.name as table_name,
	S.name AS schema_name
FROM rw_catalog.rw_tables AS T
	JOIN rw_catalog.rw_schemas AS S
	ON T.schema_id = S.id;`

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

	tableMap := make(map[string][]*storepb.TableMetadata)
	rows, err := txn.Query(listTableQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		table := &storepb.TableMetadata{}
		// var tbl tableSchema
		var schemaName string
		if err := rows.Scan(&table.Name, &schemaName); err != nil {
			return nil, err
		}
		// TODO: get table statistics of RisingWave later.
		table.DataSize = 0
		table.IndexSize = 0
		table.RowCount = 0
		// Comment is not supported in RisingWave.
		table.Comment = ""
		key := db.TableKey{Schema: schemaName, Table: table.Name}
		table.Columns = columnMap[key]
		table.Indexes = indexMap[key]
		// No foreign keys in RisingWave.
		table.ForeignKeys = make([]*storepb.ForeignKeyMetadata, 0)

		tableMap[schemaName] = append(tableMap[schemaName], table)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tableMap, nil
}

var listColumnQuery = `
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
FROM INFORMATION_SCHEMA.COLUMNS AS cols` + fmt.Sprintf(`
WHERE cols.table_schema NOT IN (%s)
ORDER BY cols.table_schema, cols.table_name, cols.ordinal_position;`, systemSchemas)

// getTableColumns gets the columns of a table.
func getTableColumns(txn *sql.Tx) (map[db.TableKey][]*storepb.ColumnMetadata, error) {
	columnsMap := make(map[db.TableKey][]*storepb.ColumnMetadata)
	rows, err := txn.Query(listColumnQuery)
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

var listViewQuery = `
SELECT schemaname, viewname, definition, obj_description(format('%s.%s', quote_ident(schemaname), quote_ident(viewname))::regclass) FROM pg_catalog.pg_views` + fmt.Sprintf(`
WHERE schemaname NOT IN (%s);`, systemSchemas)

// getViews gets all views of a database.
func getViews(txn *sql.Tx) (map[string][]*storepb.ViewMetadata, error) {
	viewMap := make(map[string][]*storepb.ViewMetadata)

	rows, err := txn.Query(listViewQuery)
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

var listIndexQuery = `
SELECT
	I.name AS idx_name,
	T.name AS table_name,
	S.name AS schema_name,
	I.definition
FROM rw_indexes AS I
	JOIN rw_tables AS T
	ON I.primary_table_id = T.id
	JOIN rw_schemas AS S
	ON I.schema_id = S.id;
`

// getIndexes gets all indices of a database.
func getIndexes(txn *sql.Tx) (map[db.TableKey][]*storepb.IndexMetadata, error) {
	indexMap := make(map[db.TableKey][]*storepb.IndexMetadata)

	rows, err := txn.Query(listIndexQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		index := &storepb.IndexMetadata{}
		var schemaName, tableName, statement string
		if err := rows.Scan(&index.Name, &tableName, &schemaName, &statement); err != nil {
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
		// No primary key index in RisingWave.
		index.Primary = false
		// Comment is unsupported in RisingWave in v1.0.
		index.Comment = ""

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

var listFunctionQuery = `SELECT
	F.name AS function_name,
	S.name AS schema_name
FROM rw_catalog.rw_functions AS F
	JOIN rw_schemas AS S
	ON F.schema_id = S.id;`

// getFunctions gets all functions of a database.
func getFunctions(txn *sql.Tx) (map[string][]*storepb.FunctionMetadata, error) {
	functionMap := make(map[string][]*storepb.FunctionMetadata)

	rows, err := txn.Query(listFunctionQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		function := &storepb.FunctionMetadata{}
		var schemaName string
		if err := rows.Scan(&function.Name, &schemaName); err != nil {
			return nil, err
		}
		// Omit the definition.
		function.Definition = "..."

		functionMap[schemaName] = append(functionMap[schemaName], function)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return functionMap, nil
}

// SyncSlowQuery syncs the slow query.
func (driver *Driver) SyncSlowQuery(ctx context.Context, _ time.Time) (map[string]*storepb.SlowQueryStatistics, error) {
	var now time.Time
	getNow := `SELECT NOW();`
	nowRows, err := driver.db.QueryContext(ctx, getNow)
	if err != nil {
		return nil, util.FormatErrorWithQuery(err, getNow)
	}
	defer nowRows.Close()
	for nowRows.Next() {
		if err := nowRows.Scan(&now); err != nil {
			return nil, util.FormatErrorWithQuery(err, getNow)
		}
	}
	if err := nowRows.Err(); err != nil {
		return nil, util.FormatErrorWithQuery(err, getNow)
	}

	result := make(map[string]*storepb.SlowQueryStatistics)
	version, err := driver.getPGStatStatementsVersion(ctx)
	if err != nil {
		return nil, err
	}
	var query string
	// pg_stat_statements version 1.8 changed the column names of pg_stat_statements.
	// version is a string in the form of "major.minor".
	// We need to check if the major version is greater than or equal to 1 and the minor version is greater than or equal to 8.
	versions := strings.Split(version, ".")
	if len(versions) == 2 && ((versions[0] == "1" && versions[1] >= "8") || versions[0] > "1") {
		query = `
		SELECT
			pg_database.datname,
			query,
			calls,
			total_exec_time,
			max_exec_time,
			rows
		FROM
			pg_stat_statements
			JOIN pg_database ON pg_database.oid = pg_stat_statements.dbid
		WHERE max_exec_time >= 1000;
	`
	} else {
		query = `
		SELECT
			pg_database.datname,
			query,
			calls,
			total_time,
			max_time,
			rows
		FROM
			pg_stat_statements
			JOIN pg_database ON pg_database.oid = pg_stat_statements.dbid
		WHERE max_time >= 1000;
		`
	}

	slowQueryStatisticsRows, err := driver.db.QueryContext(ctx, query)
	if err != nil {
		return nil, util.FormatErrorWithQuery(err, query)
	}
	defer slowQueryStatisticsRows.Close()
	for slowQueryStatisticsRows.Next() {
		var database string
		var fingerprint string
		var calls int64
		var totalExecTime float64
		var maxExecTime float64
		var rows int64
		if err := slowQueryStatisticsRows.Scan(&database, &fingerprint, &calls, &totalExecTime, &maxExecTime, &rows); err != nil {
			return nil, err
		}
		if len(fingerprint) > db.SlowQueryMaxLen {
			fingerprint = fingerprint[:db.SlowQueryMaxLen]
		}
		item := storepb.SlowQueryStatisticsItem{
			SqlFingerprint:   fingerprint,
			Count:            calls,
			LatestLogTime:    timestamppb.New(now.UTC()),
			TotalQueryTime:   durationpb.New(time.Duration(totalExecTime * float64(time.Millisecond))),
			MaximumQueryTime: durationpb.New(time.Duration(maxExecTime * float64(time.Millisecond))),
			TotalRowsSent:    rows,
		}
		if statistics, exists := result[database]; exists {
			statistics.Items = append(statistics.Items, &item)
		} else {
			result[database] = &storepb.SlowQueryStatistics{
				Items: []*storepb.SlowQueryStatisticsItem{&item},
			}
		}
	}
	if err := slowQueryStatisticsRows.Err(); err != nil {
		return nil, err
	}

	reset := `SELECT pg_stat_statements_reset();`
	if _, err := driver.db.ExecContext(ctx, reset); err != nil {
		return nil, util.FormatErrorWithQuery(err, reset)
	}
	return result, nil
}

// CheckSlowQueryLogEnabled checks if slow query log is enabled.
func (driver *Driver) CheckSlowQueryLogEnabled(ctx context.Context) error {
	showSharedPreloadLibraries := `SELECT setting FROM pg_settings WHERE name = 'shared_preload_libraries';`

	sharedPreloadLibrariesRows, err := driver.db.QueryContext(ctx, showSharedPreloadLibraries)
	if err != nil {
		return util.FormatErrorWithQuery(err, showSharedPreloadLibraries)
	}
	defer sharedPreloadLibrariesRows.Close()
	for sharedPreloadLibrariesRows.Next() {
		var sharedPreloadLibraries string
		if err := sharedPreloadLibrariesRows.Scan(&sharedPreloadLibraries); err != nil {
			return err
		}
		if !strings.Contains(sharedPreloadLibraries, "pg_stat_statements") {
			return errors.New("pg_stat_statements is not loaded")
		}
	}
	if err := sharedPreloadLibrariesRows.Err(); err != nil {
		return util.FormatErrorWithQuery(err, showSharedPreloadLibraries)
	}

	checkPGStatStatements := `SELECT count(*) FROM pg_stat_statements limit 10;`

	pgStatStatementsInfoRows, err := driver.db.QueryContext(ctx, checkPGStatStatements)
	if err != nil {
		return util.FormatErrorWithQuery(err, checkPGStatStatements)
	}
	defer pgStatStatementsInfoRows.Close()
	// no need to scan rows, just check if there is any row
	if !pgStatStatementsInfoRows.Next() {
		return errors.New("pg_stat_statements is empty")
	}
	if err := pgStatStatementsInfoRows.Err(); err != nil {
		return util.FormatErrorWithQuery(err, checkPGStatStatements)
	}

	return nil
}
