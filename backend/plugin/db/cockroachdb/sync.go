package cockroachdb

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"github.com/blang/semver/v4"
	"github.com/pkg/errors"

	crrawparser "github.com/cockroachdb/cockroachdb-parser/pkg/sql/parser"
	crrawparsertree "github.com/cockroachdb/cockroachdb-parser/pkg/sql/sem/tree"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	crparser "github.com/bytebase/bytebase/backend/plugin/parser/cockroachdb"
	pgparser "github.com/bytebase/bytebase/backend/plugin/parser/pg"
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

	// Query db info
	databases, err := d.getDatabases(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get databases")
	}

	var filteredDatabases []*storepb.DatabaseSchemaMetadata
	for _, database := range databases {
		// Skip all system databases
		if crparser.IsSystemDatabase(database.Name) {
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
	isAtLeastPG10 := isAtLeastPG10(d.connectionCtx.EngineVersion)

	txn, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer txn.Rollback()

	schemas, err := getSchemas(txn)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get schemas from database %q", d.databaseName)
	}
	columnMap, err := getTableColumns(txn)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get columns from database %q", d.databaseName)
	}
	tableMap, externalTableMap, err := getTables(txn, isAtLeastPG10, columnMap)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get tables from database %q", d.databaseName)
	}
	var tablePartitionMap map[db.TableKey][]*storepb.TablePartitionMetadata
	if isAtLeastPG10 {
		tablePartitionMap, err = getTablePartitions(txn)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get table partitions from database %q", d.databaseName)
		}
	}
	viewMap, err := getViews(txn, columnMap)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get views from database %q", d.databaseName)
	}
	materializedViewMap, err := getMaterializedViews(txn)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get materialized views from database %q", d.databaseName)
	}
	functionMap, err := getFunctions(txn)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get functions from database %q", d.databaseName)
	}
	sequenceMap, err := getSequences(txn)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get sequences from database %q", d.databaseName)
	}

	extensions, err := getExtensions(txn)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get extensions from database %q", d.databaseName)
	}

	if err := txn.Commit(); err != nil {
		return nil, err
	}

	for _, schemaName := range schemas {
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
ORDER BY fk_schema, fk_table, fk_name;`, crparser.SystemSchemaWhereClause)

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
SELECT nspname
FROM pg_catalog.pg_namespace
WHERE nspname NOT IN (%s)
ORDER BY nspname;
`, crparser.SystemSchemaWhereClause)

func getSchemas(txn *sql.Tx) ([]string, error) {
	rows, err := txn.Query(listSchemaQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var schemaNames []string
	for rows.Next() {
		var schemaName string
		if err := rows.Scan(&schemaName); err != nil {
			return nil, err
		}
		if crparser.IsSystemSchema(schemaName) {
			continue
		}
		schemaNames = append(schemaNames, schemaName)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return schemaNames, nil
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
		relisPartition = " AND (pc.relispartition IS NULL OR pc.relispartition IS FALSE)"
	}
	// CockroachDB does not support pg_table_size and pg_indexes_size now.
	return `
	SELECT tbl.schemaname, tbl.tablename,
		GREATEST(pc.reltuples::bigint, 0::BIGINT) AS estimate,
		obj_description(format('%s.%s', quote_ident(tbl.schemaname), quote_ident(tbl.tablename))::regclass) AS comment
	FROM pg_catalog.pg_tables tbl
	LEFT JOIN pg_class as pc ON pc.oid = format('%s.%s', quote_ident(tbl.schemaname), quote_ident(tbl.tablename))::regclass` + fmt.Sprintf(`
	WHERE tbl.schemaname NOT IN (%s)%s
	ORDER BY tbl.schemaname, tbl.tablename;`, crparser.SystemSchemaWhereClause, relisPartition)
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
		if err := rows.Scan(&schemaName, &table.Name, &table.RowCount, &comment); err != nil {
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
ORDER BY c.oid;`, crparser.SystemSchemaWhereClause)

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
ORDER BY cols.table_schema, cols.table_name, cols.ordinal_position;`, crparser.SystemSchemaWhereClause)

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
			// Keep the type as is
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
ORDER BY schemaname, matviewname;`, crparser.SystemSchemaWhereClause)

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
			matview.DependencyColumns = dependencies
		}
	}

	return matviewMap, nil
}

var listViewQuery = `
SELECT schemaname, viewname, definition, obj_description(format('%s.%s', quote_ident(schemaname), quote_ident(viewname))::regclass) FROM pg_catalog.pg_views` + fmt.Sprintf(`
WHERE schemaname NOT IN (%s)
ORDER BY schemaname, viewname;`, crparser.SystemSchemaWhereClause)

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
			view.DependencyColumns = dependencies
		}
	}

	return viewMap, nil
}

// getViewDependencies gets the dependencies of a view.
func getViewDependencies(txn *sql.Tx, schemaName, viewName string) ([]*storepb.DependencyColumn, error) {
	var result []*storepb.DependencyColumn

	query := fmt.Sprintf(`
		SELECT source_ns.nspname as source_schema,
	  		source_table.relname as source_table,
	  		pg_attribute.attname as column_name
	  	FROM pg_depend 
	  		JOIN pg_rewrite ON pg_depend.objid = pg_rewrite.oid 
	  		JOIN pg_class as dependency_view ON pg_rewrite.ev_class = dependency_view.oid 
	  		JOIN pg_class as source_table ON pg_depend.refobjid = source_table.oid 
	  		JOIN pg_attribute ON pg_depend.refobjid = pg_attribute.attrelid 
	  		    AND pg_depend.refobjsubid = pg_attribute.attnum 
	  		JOIN pg_namespace dependency_ns ON dependency_ns.oid = dependency_view.relnamespace
	  		JOIN pg_namespace source_ns ON source_ns.oid = source_table.relnamespace
	  	WHERE 
	  		dependency_ns.nspname = '%s'
	  		AND dependency_view.relname = '%s'
	  		AND pg_attribute.attnum > 0
	  		-- Only consider SELECT rules (view definitions), not INSERT/UPDATE/DELETE rules
	  		AND pg_rewrite.ev_type = '1'
	  	ORDER BY 1,2,3;
	`, schemaName, viewName)

	rows, err := txn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		dependencyColumn := &storepb.DependencyColumn{}
		if err := rows.Scan(&dependencyColumn.Schema, &dependencyColumn.Table, &dependencyColumn.Column); err != nil {
			return nil, err
		}
		result = append(result, dependencyColumn)
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
	query := `SELECT sequence_schema, sequence_name, data_type FROM information_schema.sequences;`
	rows, err := txn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	sequenceMap := make(map[string][]*storepb.SequenceMetadata)
	for rows.Next() {
		sequence := &storepb.SequenceMetadata{}
		var schemaName string
		if err := rows.Scan(&schemaName, &sequence.Name, &sequence.DataType); err != nil {
			return nil, err
		}
		sequenceMap[schemaName] = append(sequenceMap[schemaName], sequence)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return sequenceMap, nil
}

var listIndexQuery = `
SELECT idx.schemaname, idx.tablename, idx.indexname, idx.indexdef, (SELECT constraint_type
	FROM information_schema.table_constraints
	WHERE constraint_schema = idx.schemaname
	AND constraint_name = idx.indexname
	AND table_schema = idx.schemaname
	AND table_name = idx.tablename
	AND constraint_type = 'PRIMARY KEY') AS constraint_type,
	obj_description(format('%s.%s', quote_ident(idx.schemaname), quote_ident(idx.indexname))::regclass) AS comment` + fmt.Sprintf(`
FROM pg_indexes AS idx WHERE idx.schemaname NOT IN (%s)
ORDER BY idx.schemaname, idx.tablename, idx.indexname;`, crparser.SystemSchemaWhereClause)

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
		var constraintType sql.NullString
		var comment sql.NullString
		if err := rows.Scan(&schemaName, &tableName, &index.Name, &statement, &constraintType, &comment); err != nil {
			return nil, err
		}

		nodes, err := crrawparser.Parse(statement)
		if err != nil {
			return nil, err
		}
		if len(nodes) != 1 {
			return nil, errors.Errorf("invalid number of statement %v, expecting one", len(nodes))
		}
		node, ok := nodes[0].AST.(*crrawparsertree.CreateIndex)
		if !ok {
			return nil, errors.Errorf("invalid statement type %T, expecting CreateIndex", nodes[0].AST)
		}
		for _, indexElem := range node.Columns {
			if indexElem.Column != "" {
				index.Expressions = append(index.Expressions, indexElem.Column.String())
				continue
			}
			if indexElem.Expr != nil {
				index.Expressions = append(index.Expressions, indexElem.Expr.String())
				continue
			}
			if indexElem.OpClass != "" {
				index.Expressions = append(index.Expressions, indexElem.OpClass.String())
				continue
			}
		}

		index.Definition = statement

		// FIXME(zp): Get expression.
		index.Type = getIndexMethodType(statement)
		if constraintType.Valid {
			switch constraintType.String {
			case "PRIMARY KEY":
				index.Primary = true
				index.Unique = true
			case "UNIQUE":
				index.Unique = true
			default:
				// Other constraint types
			}
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
	case when l.lanname = 'internal' then p.prosrc
			else pg_get_functiondef(p.oid)
			end as definition
from pg_proc p
left join pg_namespace n on p.pronamespace = n.oid
left join pg_language l on p.prolang = l.oid
left join pg_type t on t.oid = p.prorettype ` + fmt.Sprintf(`
where n.nspname not in (%s)
order by function_schema, function_name;`, crparser.SystemSchemaWhereClause)

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
		if err := rows.Scan(&schemaName, &function.Name, &function.Definition); err != nil {
			return nil, err
		}
		// Skip internal functions.
		if pgparser.IsSystemFunction(function.Name, function.Definition) {
			continue
		}

		functionMap[schemaName] = append(functionMap[schemaName], function)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return functionMap, nil
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
