package pg

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/blang/semver/v4"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	pgparser "github.com/bytebase/bytebase/backend/plugin/parser/pg"
	"github.com/bytebase/bytebase/backend/plugin/parser/sql/ast"
	pgrawparser "github.com/bytebase/bytebase/backend/plugin/parser/sql/engine/pg"
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

	// Query db info
	databases, err := driver.getDatabases(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get databases")
	}

	var filteredDatabases []*storepb.DatabaseSchemaMetadata
	for _, database := range databases {
		// Skip all system databases
		if pgparser.IsSystemDatabase(database.Name) {
			continue
		}
		filteredDatabases = append(filteredDatabases, database)
	}

	return &db.InstanceMetadata{
		Version:   version,
		Databases: filteredDatabases,
		Metadata: &storepb.InstanceMetadata{
			Roles: instanceRoles,
		},
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
	isAtLeastPG10 := isAtLeastPG10(driver.connectionCtx.EngineVersion)

	txn, err := driver.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer txn.Rollback()

	schemas, schemaOwners, err := getSchemas(txn)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get schemas from database %q", driver.databaseName)
	}
	columnMap, err := getTableColumns(txn)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get columns from database %q", driver.databaseName)
	}
	tableMap, externalTableMap, err := getTables(txn, isAtLeastPG10, columnMap)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get tables from database %q", driver.databaseName)
	}
	var tablePartitionMap map[db.TableKey][]*storepb.TablePartitionMetadata
	if isAtLeastPG10 {
		tablePartitionMap, err = getTablePartitions(txn)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get table partitions from database %q", driver.databaseName)
		}
	}
	viewMap, err := getViews(txn, columnMap)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get views from database %q", driver.databaseName)
	}
	materializedViewMap, err := getMaterializedViews(txn)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get materialized views from database %q", driver.databaseName)
	}
	functionMap, err := getFunctions(txn)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get functions from database %q", driver.databaseName)
	}
	sequenceMap, err := getSequences(txn)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get sequences from database %q", driver.databaseName)
	}

	extensions, err := getExtensions(txn)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get extensions from database %q", driver.databaseName)
	}

	if err := txn.Commit(); err != nil {
		return nil, err
	}

	for i, schemaName := range schemas {
		tables := tableMap[schemaName]
		for _, table := range tables {
			if isAtLeastPG10 {
				table.Partitions = warpTablePartitions(tablePartitionMap, schemaName, table.Name)
			}
		}
		databaseMetadata.Schemas = append(databaseMetadata.Schemas, &storepb.SchemaMetadata{
			Name:              schemaName,
			Tables:            tables,
			ExternalTables:    externalTableMap[schemaName],
			Views:             viewMap[schemaName],
			Functions:         functionMap[schemaName],
			Sequences:         sequenceMap[schemaName],
			MaterializedViews: materializedViewMap[schemaName],
			Owner:             schemaOwners[i],
		})
	}
	databaseMetadata.Extensions = extensions

	return databaseMetadata, err
}

func warpTablePartitions(m map[db.TableKey][]*storepb.TablePartitionMetadata, schemaName, tableName string) []*storepb.TablePartitionMetadata {
	key := db.TableKey{Schema: schemaName, Table: tableName}
	if partitions, exists := m[key]; exists {
		defer delete(m, key)
		for _, partition := range partitions {
			partition.Subpartitions = warpTablePartitions(m, schemaName, partition.Name)
		}
		return partitions
	}
	return []*storepb.TablePartitionMetadata{}
}

var listForeignKeyQuery = `
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
	JOIN pg_namespace n ON n.oid = c.connamespace` + fmt.Sprintf(`
WHERE
	n.nspname NOT IN(%s)
	AND c.contype = 'f'
ORDER BY fk_schema, fk_table, fk_name;`, pgparser.SystemSchemaWhereClause)

func getForeignKeys(txn *sql.Tx) (map[db.TableKey][]*storepb.ForeignKeyMetadata, error) {
	foreignKeysMap := make(map[db.TableKey][]*storepb.ForeignKeyMetadata)
	rows, err := txn.Query(listForeignKeyQuery)
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

var listSchemaQuery = fmt.Sprintf(`
SELECT nspname, pg_catalog.pg_get_userbyid(nspowner) as schema_owner
FROM pg_catalog.pg_namespace
WHERE nspname NOT IN (%s)
ORDER BY nspname;
`, pgparser.SystemSchemaWhereClause)

func getSchemas(txn *sql.Tx) ([]string, []string, error) {
	rows, err := txn.Query(listSchemaQuery)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var schemaNames, schemaOwners []string
	for rows.Next() {
		var schemaName, schemaOwner string
		if err := rows.Scan(&schemaName, &schemaOwner); err != nil {
			return nil, nil, err
		}
		if pgparser.IsSystemSchema(schemaName) {
			continue
		}
		schemaNames = append(schemaNames, schemaName)
		schemaOwners = append(schemaOwners, schemaOwner)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	return schemaNames, schemaOwners, nil
}

func getListForeignTableQuery() string {
	return `SELECT
		foreign_table.foreign_table_schema,
		foreign_table.foreign_table_name,
		foreign_table.foreign_server_catalog,
		foreign_table.foreign_server_name
	FROM information_schema.foreign_tables AS foreign_table;`
}
func getListTableQuery(isAtLeastPG10 bool) string {
	relisPartition := ""
	if isAtLeastPG10 {
		relisPartition = " AND pc.relispartition IS FALSE"
	}
	return `
	SELECT tbl.schemaname, tbl.tablename,
		pg_table_size(format('%s.%s', quote_ident(tbl.schemaname), quote_ident(tbl.tablename))::regclass),
		pg_indexes_size(format('%s.%s', quote_ident(tbl.schemaname), quote_ident(tbl.tablename))::regclass),
		GREATEST(pc.reltuples::bigint, 0::BIGINT) AS estimate,
		obj_description(format('%s.%s', quote_ident(tbl.schemaname), quote_ident(tbl.tablename))::regclass) AS comment,
		tbl.tableowner
	FROM pg_catalog.pg_tables tbl
	LEFT JOIN pg_class as pc ON pc.oid = format('%s.%s', quote_ident(tbl.schemaname), quote_ident(tbl.tablename))::regclass` + fmt.Sprintf(`
	WHERE tbl.schemaname NOT IN (%s)%s
	ORDER BY tbl.schemaname, tbl.tablename;`, pgparser.SystemSchemaWhereClause, relisPartition)
}

// getTables gets all tables of a database.
func getTables(txn *sql.Tx, isAtLeastPG10 bool, columnMap map[db.TableKey][]*storepb.ColumnMetadata) (map[string][]*storepb.TableMetadata, map[string][]*storepb.ExternalTableMetadata, error) {
	indexMap, err := getIndexes(txn)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to get indexes")
	}
	foreignKeysMap, err := getForeignKeys(txn)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to get foreign keys")
	}
	foreignTablesMap, err := getForeignTables(txn, columnMap)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to get foreign tables")
	}

	tableMap := make(map[string][]*storepb.TableMetadata)
	query := getListTableQuery(isAtLeastPG10)
	rows, err := txn.Query(query)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	for rows.Next() {
		table := &storepb.TableMetadata{}
		var schemaName string
		var comment sql.NullString
		if err := rows.Scan(&schemaName, &table.Name, &table.DataSize, &table.IndexSize, &table.RowCount, &comment, &table.Owner); err != nil {
			return nil, nil, err
		}
		if pgparser.IsSystemTable(table.Name) {
			continue
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
		return nil, nil, err
	}

	return tableMap, foreignTablesMap, nil
}

func getForeignTables(txn *sql.Tx, columnMap map[db.TableKey][]*storepb.ColumnMetadata) (map[string][]*storepb.ExternalTableMetadata, error) {
	query := getListForeignTableQuery()
	rows, err := txn.Query(query)
	if err != nil {
		// Experimental feature, log error and return.
		slog.Error("failed to query foreign table: %v", log.BBError(err))
		return nil, nil
	}
	defer rows.Close()

	foreignTablesMap := make(map[string][]*storepb.ExternalTableMetadata)

	for rows.Next() {
		var schemaName, tableName, foreignServerCatalog, foreignServerName string
		if err := rows.Scan(&schemaName, &tableName, &foreignServerCatalog, &foreignServerName); err != nil {
			slog.Error("failed to scan foreign table: %v", log.BBError(err))
			return nil, nil
		}
		externalTable := &storepb.ExternalTableMetadata{
			Name:                 tableName,
			ExternalServerName:   foreignServerName,
			ExternalDatabaseName: foreignServerCatalog,
		}
		key := db.TableKey{Schema: schemaName, Table: externalTable.Name}
		externalTable.Columns = columnMap[key]

		foreignTablesMap[schemaName] = append(foreignTablesMap[schemaName], externalTable)
	}

	if err := rows.Err(); err != nil {
		slog.Error("failed to scan foreign table: %v", log.BBError(err))
		return nil, nil
	}

	return foreignTablesMap, nil
}

var listTablePartitionQuery = `
SELECT
	n.nspname AS schema_name,
	c.relname AS table_name,
	i2.nspname AS inh_schema_name,
	i2.relname AS inh_table_name,
	i2.partstrat AS partition_type,
	pg_get_expr(c.relpartbound, c.oid) AS rel_part_bound
FROM
	pg_catalog.pg_class c
	LEFT JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
    LEFT JOIN (
		pg_inherits i 
		INNER JOIN pg_class c2 ON i.inhparent = c2.oid 
		LEFT JOIN pg_namespace n2 ON n2.oid = c2.relnamespace
		LEFT JOIN pg_partitioned_table p ON p.partrelid = c2.oid
	) i2 ON i2.inhrelid = c.oid 
WHERE
	((c.relkind = 'r'::"char") OR (c.relkind = 'f'::"char") OR (c.relkind = 'p'::"char"))
	AND c.relispartition IS TRUE ` + fmt.Sprintf(`
	AND n.nspname NOT IN (%s)
ORDER BY c.oid;`, pgparser.SystemSchemaWhereClause)

func getTablePartitions(txn *sql.Tx) (map[db.TableKey][]*storepb.TablePartitionMetadata, error) {
	result := make(map[db.TableKey][]*storepb.TablePartitionMetadata)
	rows, err := txn.Query(listTablePartitionQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var schemaName, tableName, inhSchemaName, inhTableName, partitionType, relPartBound string
		if err := rows.Scan(&schemaName, &tableName, &inhSchemaName, &inhTableName, &partitionType, &relPartBound); err != nil {
			return nil, err
		}
		if pgparser.IsSystemTable(tableName) || pgparser.IsSystemTable(inhTableName) {
			continue
		}
		key := db.TableKey{Schema: inhSchemaName, Table: inhTableName}
		metadata := &storepb.TablePartitionMetadata{
			Name:       tableName,
			Expression: relPartBound,
		}
		switch strings.ToLower(partitionType) {
		case "l":
			metadata.Type = storepb.TablePartitionMetadata_LIST
		case "r":
			metadata.Type = storepb.TablePartitionMetadata_RANGE
		case "h":
			metadata.Type = storepb.TablePartitionMetadata_HASH
		default:
			return nil, errors.Errorf("invalid partition type %q", partitionType)
		}
		result[key] = append(result[key], metadata)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

var listColumnQuery = `
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
	pg_catalog.col_description(format('%s.%s', quote_ident(table_schema), quote_ident(table_name))::regclass, cols.ordinal_position::int) as column_comment
FROM INFORMATION_SCHEMA.COLUMNS AS cols` + fmt.Sprintf(`
WHERE cols.table_schema NOT IN (%s)
ORDER BY cols.table_schema, cols.table_name, cols.ordinal_position;`, pgparser.SystemSchemaWhereClause)

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
		var characterMaxLength, defaultStr, collation, udtSchema, udtName, comment sql.NullString
		if err := rows.Scan(&schemaName, &tableName, &column.Name, &column.Type, &characterMaxLength, &column.Position, &defaultStr, &nullable, &collation, &udtSchema, &udtName, &comment); err != nil {
			return nil, err
		}
		if defaultStr.Valid {
			column.DefaultValue = &storepb.ColumnMetadata_DefaultExpression{
				DefaultExpression: defaultStr.String,
			}
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

var listMaterializedViewQuery = `
SELECT schemaname, matviewname, definition, obj_description(format('%s.%s', quote_ident(schemaname), quote_ident(matviewname))::regclass) FROM pg_catalog.pg_matviews` + fmt.Sprintf(`
WHERE schemaname NOT IN (%s)
ORDER BY schemaname, matviewname;`, pgparser.SystemSchemaWhereClause)

func getMaterializedViews(txn *sql.Tx) (map[string][]*storepb.MaterializedViewMetadata, error) {
	matviewMap := make(map[string][]*storepb.MaterializedViewMetadata)

	rows, err := txn.Query(listMaterializedViewQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		matview := &storepb.MaterializedViewMetadata{}
		var schemaName string
		var def, comment sql.NullString
		if err := rows.Scan(&schemaName, &matview.Name, &def, &comment); err != nil {
			return nil, err
		}
		// Skip system views.
		if pgparser.IsSystemView(matview.Name) {
			continue
		}

		// Return error on NULL view definition.
		if !def.Valid {
			return nil, errors.Errorf("schema %q materialized view %q has empty definition; please check whether proper privileges have been granted to Bytebase", schemaName, matview.Name)
		}
		matview.Definition = def.String
		if comment.Valid {
			matview.Comment = comment.String
		}

		matviewMap[schemaName] = append(matviewMap[schemaName], matview)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for schemaName, list := range matviewMap {
		for _, matview := range list {
			dependencies, err := getViewDependencies(txn, schemaName, matview.Name)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to get materialized view %q dependencies", matview.Name)
			}
			matview.DependentColumns = dependencies
		}
	}

	return matviewMap, nil
}

var listViewQuery = `
SELECT schemaname, viewname, definition, obj_description(format('%s.%s', quote_ident(schemaname), quote_ident(viewname))::regclass) FROM pg_catalog.pg_views` + fmt.Sprintf(`
WHERE schemaname NOT IN (%s)
ORDER BY schemaname, viewname;`, pgparser.SystemSchemaWhereClause)

// getViews gets all views of a database.
func getViews(txn *sql.Tx, columnMap map[db.TableKey][]*storepb.ColumnMetadata) (map[string][]*storepb.ViewMetadata, error) {
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
		// Skip system views.
		if pgparser.IsSystemView(view.Name) {
			continue
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

// getExtensions gets all extensions of a database.
func getExtensions(txn *sql.Tx) ([]*storepb.ExtensionMetadata, error) {
	var extensions []*storepb.ExtensionMetadata

	query := `
		SELECT e.extname, e.extversion, n.nspname, c.description
		FROM pg_catalog.pg_extension e
		LEFT JOIN pg_catalog.pg_namespace n ON n.oid = e.extnamespace
		LEFT JOIN pg_catalog.pg_description c ON c.objoid = e.oid AND c.classoid = 'pg_catalog.pg_extension'::pg_catalog.regclass
		WHERE n.nspname != 'pg_catalog'
		ORDER BY e.extname;`
	rows, err := txn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		e := &storepb.ExtensionMetadata{}
		var description sql.NullString
		if err := rows.Scan(&e.Name, &e.Version, &e.Schema, &description); err != nil {
			return nil, err
		}
		if description.Valid {
			e.Description = description.String
		}
		extensions = append(extensions, e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return extensions, nil
}

// getSequences gets all sequences of a database.
func getSequences(txn *sql.Tx) (map[string][]*storepb.SequenceMetadata, error) {
	query := `
	SELECT
		schemaname,
		sequencename,
		data_type,
		start_value,
		min_value,
		max_value,
		increment_by,
		cycle,
		cache_size,
		last_value
	FROM pg_sequences
	ORDER BY schemaname, sequencename;`
	rows, err := txn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	sequenceMap := make(map[string][]*storepb.SequenceMetadata)
	for rows.Next() {
		var schemaName, sequenceName, dataType string
		var startValue, minValue, maxValue, incrementBy, cacheSize int64
		var cycle bool
		var lastValue sql.NullInt64
		if err := rows.Scan(&schemaName, &sequenceName, &dataType, &startValue, &minValue, &maxValue, &incrementBy, &cycle, &cacheSize, &lastValue); err != nil {
			return nil, err
		}
		lastValueStr := ""
		if lastValue.Valid {
			lastValueStr = strconv.FormatInt(lastValue.Int64, 10)
		}
		sequence := &storepb.SequenceMetadata{
			Name:      sequenceName,
			DataType:  dataType,
			Start:     strconv.FormatInt(startValue, 10),
			MinValue:  strconv.FormatInt(minValue, 10),
			MaxValue:  strconv.FormatInt(maxValue, 10),
			Increment: strconv.FormatInt(incrementBy, 10),
			Cycle:     cycle,
			CacheSize: strconv.FormatInt(cacheSize, 10),
			LastValue: lastValueStr,
		}
		sequenceMap[schemaName] = append(sequenceMap[schemaName], sequence)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return sequenceMap, nil
}

var listIndexQuery = `
SELECT idx.schemaname, idx.tablename, idx.indexname, idx.indexdef, (SELECT 1
	FROM information_schema.table_constraints
	WHERE constraint_schema = idx.schemaname
	AND constraint_name = idx.indexname
	AND table_schema = idx.schemaname
	AND table_name = idx.tablename
	AND constraint_type = 'PRIMARY KEY') AS primary,
	obj_description(format('%s.%s', quote_ident(idx.schemaname), quote_ident(idx.indexname))::regclass) AS comment` + fmt.Sprintf(`
FROM pg_indexes AS idx WHERE idx.schemaname NOT IN (%s)
ORDER BY idx.schemaname, idx.tablename, idx.indexname;`, pgparser.SystemSchemaWhereClause)

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
		var primary sql.NullInt32
		var comment sql.NullString
		if err := rows.Scan(&schemaName, &tableName, &index.Name, &statement, &primary, &comment); err != nil {
			return nil, err
		}

		nodes, err := pgrawparser.Parse(pgrawparser.ParseContext{}, statement)
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
		deparsed, err := pgrawparser.Deparse(pgrawparser.DeparseContext{}, node)
		if err != nil {
			return nil, err
		}
		// Instead of using indexdef, we use deparsed format so that the definition has quoted identifiers.
		index.Definition = deparsed

		index.Type = getIndexMethodType(statement)
		index.Unique = node.Index.Unique
		index.Expressions = node.Index.GetKeyNameList()
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

var listFunctionQuery = `
select n.nspname as function_schema,
	p.proname as function_name,
	pg_catalog.pg_get_function_identity_arguments(p.oid) as arguments,
	case when l.lanname = 'internal' then p.prosrc
			else pg_get_functiondef(p.oid)
			end as definition
from pg_proc p
left join pg_namespace n on p.pronamespace = n.oid
left join pg_language l on p.prolang = l.oid
left join pg_type t on t.oid = p.prorettype ` + fmt.Sprintf(`
where n.nspname not in (%s)
order by function_schema, function_name;`, pgparser.SystemSchemaWhereClause)

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
		var schemaName, arguments string
		if err := rows.Scan(&schemaName, &function.Name, &arguments, &function.Definition); err != nil {
			return nil, err
		}
		// Skip internal functions.
		if pgparser.IsSystemFunction(function.Name, function.Definition) {
			continue
		}

		function.Signature = fmt.Sprintf("%s(%s)", function.Name, arguments)
		functionMap[schemaName] = append(functionMap[schemaName], function)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return functionMap, nil
}

var statPluginVersion = semver.MustParse("1.8.0")

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
	sv, err := semver.ParseTolerant(version)
	if err != nil {
		return nil, err
	}
	if sv.GTE(statPluginVersion) {
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
		var calls int32
		var totalExecTime float64
		var maxExecTime float64
		var rows int32
		if err := slowQueryStatisticsRows.Scan(&database, &fingerprint, &calls, &totalExecTime, &maxExecTime, &rows); err != nil {
			return nil, err
		}
		if len(fingerprint) > db.SlowQueryMaxLen {
			fingerprint, _ = common.TruncateString(fingerprint, db.SlowQueryMaxLen)
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

func isAtLeastPG10(version string) bool {
	v, err := semver.ParseTolerant(version)
	if err != nil {
		slog.Error("invalid postgres version", slog.String("version", version))
		// Assume the version is at least 10.0 for any error.
		return true
	}
	return v.Major >= 10
}
