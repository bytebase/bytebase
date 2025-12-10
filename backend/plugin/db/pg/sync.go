package pg

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"regexp"
	"strconv"
	"strings"

	"github.com/blang/semver/v4"
	"github.com/lib/pq"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
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
		if pgparser.IsSystemDatabase(database.Name) {
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
	searchPath, err := d.GetSearchPath(ctx)
	if err != nil {
		return nil, common.Errorf(common.Internal, "failed to get search path for database %q: %v", d.databaseName, err)
	}
	databaseMetadata.SearchPath = searchPath

	txn, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer txn.Rollback()

	// We set the search path to empty before the column sync.
	// The reason is that we can get the expression with default schema name.
	if err := setTxSearchPath(txn, ""); err != nil {
		return nil, errors.Wrapf(err, "failed to set search path")
	}

	extensionDepend, err := getExtensionDepend(txn)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get extension dependencies from database %q", d.databaseName)
	}

	// Pre-filter accessible schemas to avoid permission errors on Supabase and similar restricted environments.
	accessibleSchemas, skippedSchemas, err := filterAccessibleSchemas(txn)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to filter accessible schemas from database %q", d.databaseName)
	}

	// Log schema filtering results for debugging
	if len(skippedSchemas) > 0 {
		slog.Info("Schema sync: some schemas skipped due to insufficient permissions",
			slog.String("database", d.databaseName),
			slog.Int("accessible", len(accessibleSchemas)),
			slog.Int("skipped", len(skippedSchemas)),
			slog.Any("skipped_schemas", skippedSchemas))
	}

	// Handle edge case: no accessible schemas
	if len(accessibleSchemas) == 0 {
		slog.Warn("No accessible schemas found",
			slog.String("database", d.databaseName),
			slog.Any("skipped_schemas", skippedSchemas))

		if err := txn.Commit(); err != nil {
			return nil, err
		}

		databaseMetadata.SkippedSchemas = skippedSchemas
		return databaseMetadata, nil
	}

	schemas, schemaOwners, schemaComments, skipDumps, err := getSchemas(txn, extensionDepend, accessibleSchemas)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get schemas from database %q", d.databaseName)
	}
	columnMap, err := getTableColumns(txn, accessibleSchemas)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get columns from database %q", d.databaseName)
	}
	var indexInheritanceMap map[db.IndexKey]*db.IndexKey
	if isAtLeastPG10 {
		indexInheritanceMap, err = getIndexInheritance(txn)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get index inheritance from database %q", d.databaseName)
		}
	}
	indexMap, err := getIndexes(txn, indexInheritanceMap, accessibleSchemas)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get indexes from database %q", d.databaseName)
	}
	triggerMap, err := getTriggers(txn, extensionDepend, accessibleSchemas)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get triggers from database %q", d.databaseName)
	}
	checksMap, err := getChecks(txn, accessibleSchemas)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get checks from database %q", d.databaseName)
	}
	excludesMap, err := getExcludeConstraints(txn, accessibleSchemas)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get exclude constraints from database %q", d.databaseName)
	}
	tableMap, externalTableMap, tableOidMap, err := getTables(txn, isAtLeastPG10, columnMap, indexMap, triggerMap, checksMap, excludesMap, extensionDepend, accessibleSchemas)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get tables from database %q", d.databaseName)
	}
	var tablePartitionMap map[db.TableKey][]*storepb.TablePartitionMetadata
	if isAtLeastPG10 {
		tablePartitionMap, err = getTablePartitions(txn, indexMap, checksMap, excludesMap, accessibleSchemas)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get table partitions from database %q", d.databaseName)
		}
	}
	viewMap, viewOidMap, err := getViews(txn, columnMap, triggerMap, extensionDepend, accessibleSchemas)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get views from database %q", d.databaseName)
	}
	ruleMap, err := getRules(txn, accessibleSchemas)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get rules from database %q", d.databaseName)
	}
	materializedViewMap, materializedViewOidMap, err := getMaterializedViews(txn, indexMap, triggerMap, extensionDepend, accessibleSchemas)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get materialized views from database %q", d.databaseName)
	}
	functionDependencyTables, err := getFunctionDependencyTables(txn, accessibleSchemas)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get function dependency tables from database %q", d.databaseName)
	}
	functionMap, err := getFunctions(txn, functionDependencyTables, tableOidMap, viewOidMap, materializedViewOidMap, extensionDepend, accessibleSchemas)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get functions from database %q", d.databaseName)
	}
	sequenceMap, err := getSequences(txn, tableOidMap, extensionDepend, accessibleSchemas)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get sequences from database %q", d.databaseName)
	}

	extensions, err := getExtensions(txn)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get extensions from database %q", d.databaseName)
	}

	enumTypes, err := getEnumTypes(txn, extensionDepend, accessibleSchemas)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get enum types from database %q", d.databaseName)
	}

	eventTriggers, err := getEventTriggers(txn, extensionDepend)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get event triggers from database %q", d.databaseName)
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
			// Add rules to table
			key := db.TableKey{Schema: schemaName, Table: table.Name}
			table.Rules = ruleMap[key]
		}
		// Add rules to views
		views := viewMap[schemaName]
		for _, view := range views {
			key := db.TableKey{Schema: schemaName, Table: view.Name}
			view.Rules = ruleMap[key]
		}
		databaseMetadata.Schemas = append(databaseMetadata.Schemas, &storepb.SchemaMetadata{
			Name:              schemaName,
			Tables:            tables,
			ExternalTables:    externalTableMap[schemaName],
			Views:             views,
			Functions:         functionMap[schemaName],
			Sequences:         sequenceMap[schemaName],
			MaterializedViews: materializedViewMap[schemaName],
			Owner:             schemaOwners[i],
			Comment:           schemaComments[i],
			EnumTypes:         enumTypes[schemaName],
			SkipDump:          skipDumps[i],
		})
	}
	databaseMetadata.Extensions = extensions
	databaseMetadata.EventTriggers = eventTriggers
	databaseMetadata.SkippedSchemas = skippedSchemas

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

var listExtensionDependQuery = `
SELECT
	objid
FROM
	pg_depend
WHERE
	deptype = 'e'
`

func getExtensionDepend(txn *sql.Tx) (map[int]bool, error) {
	extensionDepend := make(map[int]bool)
	rows, err := txn.Query(listExtensionDependQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var objid int
		if err := rows.Scan(&objid); err != nil {
			return nil, err
		}
		extensionDepend[objid] = true
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return extensionDepend, nil
}

func getListCheckQuery(schemaFilter string) string {
	return fmt.Sprintf(`
SELECT nsp.nspname, rel.relname, con.conname, pg_get_constraintdef(con.oid, true)
    FROM pg_catalog.pg_constraint con
        INNER JOIN pg_catalog.pg_class rel ON rel.oid = con.conrelid
        INNER JOIN pg_catalog.pg_namespace nsp ON nsp.oid = connamespace
        WHERE contype = 'c' AND %s
        ORDER BY nsp.nspname, rel.relname, con.conname`, schemaFilter)
}

func getChecks(txn *sql.Tx, accessibleSchemas []string) (map[db.TableKey][]*storepb.CheckConstraintMetadata, error) {
	checksMap := make(map[db.TableKey][]*storepb.CheckConstraintMetadata)
	query := getListCheckQuery(buildSchemaFilter("nsp.nspname", accessibleSchemas))
	rows, err := txn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var checkMetadata storepb.CheckConstraintMetadata
		var schemaName, tableName, checkDefinition string
		if err := rows.Scan(&schemaName, &tableName, &checkMetadata.Name, &checkDefinition); err != nil {
			return nil, err
		}
		checkMetadata.Expression = strings.TrimPrefix(checkDefinition, "CHECK ")

		key := db.TableKey{Schema: schemaName, Table: tableName}
		checksMap[key] = append(checksMap[key], &checkMetadata)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return checksMap, nil
}

func getListExcludeConstraintsQuery(schemaFilter string) string {
	return fmt.Sprintf(`
SELECT nsp.nspname, rel.relname, con.conname, pg_get_constraintdef(con.oid, true)
    FROM pg_catalog.pg_constraint con
        INNER JOIN pg_catalog.pg_class rel ON rel.oid = con.conrelid
        INNER JOIN pg_catalog.pg_namespace nsp ON nsp.oid = connamespace
        WHERE contype = 'x' AND %s`, schemaFilter)
}

func getExcludeConstraints(txn *sql.Tx, accessibleSchemas []string) (map[db.TableKey][]*storepb.ExcludeConstraintMetadata, error) {
	excludesMap := make(map[db.TableKey][]*storepb.ExcludeConstraintMetadata)
	query := getListExcludeConstraintsQuery(buildSchemaFilter("nsp.nspname", accessibleSchemas))
	rows, err := txn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var excludeMetadata storepb.ExcludeConstraintMetadata
		var schemaName, tableName string
		if err := rows.Scan(&schemaName, &tableName, &excludeMetadata.Name, &excludeMetadata.Expression); err != nil {
			return nil, err
		}
		// Expression keeps full definition including "EXCLUDE" keyword

		key := db.TableKey{Schema: schemaName, Table: tableName}
		excludesMap[key] = append(excludesMap[key], &excludeMetadata)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return excludesMap, nil
}

func getListForeignKeyQuery(schemaFilter string) string {
	return fmt.Sprintf(`
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
	%s
	AND c.contype = 'f'
	AND c.conparentid = 0
ORDER BY fk_schema, fk_table, fk_name;`, schemaFilter)
}

func getForeignKeys(txn *sql.Tx, accessibleSchemas []string) (map[db.TableKey][]*storepb.ForeignKeyMetadata, error) {
	foreignKeysMap := make(map[db.TableKey][]*storepb.ForeignKeyMetadata)
	query := getListForeignKeyQuery(buildSchemaFilter("n.nspname", accessibleSchemas))
	rows, err := txn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var fkMetadata storepb.ForeignKeyMetadata
		var fkSchema, fkTable, fkDefinition string
		var fkRefSchema sql.NullString
		if err := rows.Scan(
			&fkSchema,
			&fkTable,
			&fkMetadata.Name,
			&fkRefSchema,
			&fkMetadata.ReferencedTable,
			&fkMetadata.OnDelete,
			&fkMetadata.OnUpdate,
			&fkMetadata.MatchType,
			&fkDefinition,
		); err != nil {
			return nil, err
		}

		if !fkRefSchema.Valid {
			continue
		}
		fkMetadata.ReferencedSchema = fkRefSchema.String

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

// filterAccessibleSchemas returns two lists: schemas the user can access,
// and schemas that were skipped due to insufficient permissions.
func filterAccessibleSchemas(txn *sql.Tx) (accessible []string, skipped []string, err error) {
	query := fmt.Sprintf(`
		SELECT
			nspname,
			has_schema_privilege(oid, 'USAGE') as has_usage
		FROM pg_catalog.pg_namespace
		WHERE nspname NOT IN (%s)
		  AND nspname NOT LIKE 'pg_temp%%'
		  AND nspname NOT LIKE 'pg_toast%%'
		ORDER BY nspname
	`, pgparser.SystemSchemaWhereClause)

	rows, err := txn.Query(query)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var schemaName string
		var hasUsage bool
		if err := rows.Scan(&schemaName, &hasUsage); err != nil {
			return nil, nil, err
		}
		if pgparser.IsSystemSchema(schemaName) {
			continue
		}
		if hasUsage {
			accessible = append(accessible, schemaName)
		} else {
			skipped = append(skipped, schemaName)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	return accessible, skipped, nil
}

// buildSchemaFilter returns a SQL clause for filtering schemas.
// Uses pq.QuoteLiteral for proper escaping to prevent SQL injection.
func buildSchemaFilter(columnName string, accessibleSchemas []string) string {
	if len(accessibleSchemas) == 0 {
		// Return a condition that matches nothing
		return fmt.Sprintf("%s IN (NULL)", columnName)
	}
	quoted := make([]string, len(accessibleSchemas))
	for i, s := range accessibleSchemas {
		quoted[i] = pq.QuoteLiteral(s)
	}
	return fmt.Sprintf("%s IN (%s)", columnName, strings.Join(quoted, ","))
}

func getSchemas(txn *sql.Tx, extensionDepend map[int]bool, accessibleSchemas []string) ([]string, []string, []string, []bool, error) {
	query := fmt.Sprintf(`
SELECT oid, nspname, pg_catalog.pg_get_userbyid(nspowner) as schema_owner,
       obj_description(oid, 'pg_namespace') as schema_comment
FROM pg_catalog.pg_namespace
WHERE %s
ORDER BY nspname;
`, buildSchemaFilter("nspname", accessibleSchemas))
	rows, err := txn.Query(query)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	defer rows.Close()

	var schemaNames, schemaOwners, schemaComments []string
	var skipDump []bool
	for rows.Next() {
		var oid int
		var schemaName, schemaOwner string
		var comment sql.NullString
		if err := rows.Scan(&oid, &schemaName, &schemaOwner, &comment); err != nil {
			return nil, nil, nil, nil, err
		}
		if pgparser.IsSystemSchema(schemaName) {
			continue
		}
		skipDump = append(skipDump, extensionDepend[oid])
		schemaNames = append(schemaNames, schemaName)
		schemaOwners = append(schemaOwners, schemaOwner)
		if comment.Valid {
			schemaComments = append(schemaComments, comment.String)
		} else {
			schemaComments = append(schemaComments, "")
		}
	}
	if err := rows.Err(); err != nil {
		return nil, nil, nil, nil, err
	}
	return schemaNames, schemaOwners, schemaComments, skipDump, nil
}

func getListForeignTableQuery() string {
	return `SELECT
		foreign_table.foreign_table_schema,
		foreign_table.foreign_table_name,
		foreign_table.foreign_server_catalog,
		foreign_table.foreign_server_name
	FROM information_schema.foreign_tables AS foreign_table;`
}
func getListTableQuery(isAtLeastPG10 bool, schemaFilter string) string {
	relisPartition := ""
	if isAtLeastPG10 {
		relisPartition = " AND pc.relispartition IS FALSE"
	}
	return fmt.Sprintf(`
	SELECT pc.oid, tbl.schemaname, tbl.tablename,
		pg_table_size(format('%%s.%%s', quote_ident(tbl.schemaname), quote_ident(tbl.tablename))::regclass),
		pg_indexes_size(format('%%s.%%s', quote_ident(tbl.schemaname), quote_ident(tbl.tablename))::regclass),
		GREATEST(pc.reltuples::bigint, 0::BIGINT) AS estimate,
		obj_description(format('%%s.%%s', quote_ident(tbl.schemaname), quote_ident(tbl.tablename))::regclass) AS comment,
		tbl.tableowner
	FROM pg_catalog.pg_tables tbl
	LEFT JOIN pg_class as pc ON pc.oid = format('%%s.%%s', quote_ident(tbl.schemaname), quote_ident(tbl.tablename))::regclass
	WHERE %s%s
	ORDER BY tbl.schemaname, tbl.tablename;`, schemaFilter, relisPartition)
}

// getTables gets all tables of a database.
func getTables(
	txn *sql.Tx,
	isAtLeastPG10 bool,
	columnMap map[db.TableKey][]*storepb.ColumnMetadata,
	indexMap map[db.TableKey][]*storepb.IndexMetadata,
	triggerMap map[db.TableKey][]*storepb.TriggerMetadata,
	checksMap map[db.TableKey][]*storepb.CheckConstraintMetadata,
	excludesMap map[db.TableKey][]*storepb.ExcludeConstraintMetadata,
	extensionDepend map[int]bool,
	accessibleSchemas []string,
) (map[string][]*storepb.TableMetadata, map[string][]*storepb.ExternalTableMetadata, map[int]*db.TableKeyWithColumns, error) {
	foreignKeysMap, err := getForeignKeys(txn, accessibleSchemas)
	if err != nil {
		return nil, nil, nil, errors.Wrapf(err, "failed to get foreign keys")
	}
	foreignTablesMap, err := getForeignTables(txn, columnMap)
	if err != nil {
		return nil, nil, nil, errors.Wrapf(err, "failed to get foreign tables")
	}

	tableMap := make(map[string][]*storepb.TableMetadata)
	tableOidMap := make(map[int]*db.TableKeyWithColumns)
	query := getListTableQuery(isAtLeastPG10, buildSchemaFilter("tbl.schemaname", accessibleSchemas))
	rows, err := txn.Query(query)
	if err != nil {
		return nil, nil, nil, err
	}
	defer rows.Close()

	for rows.Next() {
		table := &storepb.TableMetadata{}
		var oid int
		var schemaName string
		var comment sql.NullString
		if err := rows.Scan(&oid, &schemaName, &table.Name, &table.DataSize, &table.IndexSize, &table.RowCount, &comment, &table.Owner); err != nil {
			return nil, nil, nil, err
		}
		if pgparser.IsSystemTable(table.Name) {
			continue
		}
		if extensionDepend[oid] {
			table.SkipDump = true
		}
		if comment.Valid {
			table.Comment = comment.String
		}
		key := db.TableKey{Schema: schemaName, Table: table.Name}
		table.Columns = columnMap[key]
		table.Indexes = indexMap[key]
		table.ForeignKeys = foreignKeysMap[key]
		table.Triggers = triggerMap[key]
		table.CheckConstraints = checksMap[key]
		table.ExcludeConstraints = excludesMap[key]

		tableMap[schemaName] = append(tableMap[schemaName], table)
		tableOidMap[oid] = &db.TableKeyWithColumns{Schema: schemaName, Table: table.Name, Columns: table.Columns}
	}
	if err := rows.Err(); err != nil {
		return nil, nil, nil, err
	}

	return tableMap, foreignTablesMap, tableOidMap, nil
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

func getListTablePartitionQuery(schemaFilter string) string {
	return fmt.Sprintf(`
SELECT
	n.nspname AS schema_name,
	c.relname AS table_name,
	i2.nspname AS inh_schema_name,
	i2.relname AS inh_table_name,
	i2.partstrat AS partition_type,
	pg_get_expr(c.relpartbound, c.oid) AS rel_part_bound,
	pg_get_partkeydef(i2.inhparent) AS part_key_def
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
	AND c.relispartition IS TRUE
	AND %s
ORDER BY c.oid;`, schemaFilter)
}

func getTablePartitions(txn *sql.Tx, indexMap map[db.TableKey][]*storepb.IndexMetadata, checksMap map[db.TableKey][]*storepb.CheckConstraintMetadata, excludesMap map[db.TableKey][]*storepb.ExcludeConstraintMetadata, accessibleSchemas []string) (map[db.TableKey][]*storepb.TablePartitionMetadata, error) {
	result := make(map[db.TableKey][]*storepb.TablePartitionMetadata)
	query := getListTablePartitionQuery(buildSchemaFilter("n.nspname", accessibleSchemas))
	rows, err := txn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var schemaName, tableName, inhSchemaName, inhTableName, partitionType, relPartBound, partKeyDef string
		if err := rows.Scan(&schemaName, &tableName, &inhSchemaName, &inhTableName, &partitionType, &relPartBound, &partKeyDef); err != nil {
			return nil, err
		}
		if pgparser.IsSystemTable(tableName) || pgparser.IsSystemTable(inhTableName) {
			continue
		}
		key := db.TableKey{Schema: schemaName, Table: tableName}
		inhKey := db.TableKey{Schema: inhSchemaName, Table: inhTableName}
		metadata := &storepb.TablePartitionMetadata{
			Name:               tableName,
			Expression:         partKeyDef,
			Value:              relPartBound,
			Indexes:            indexMap[key],
			CheckConstraints:   checksMap[key],
			ExcludeConstraints: excludesMap[key],
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
		result[inhKey] = append(result[inhKey], metadata)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

var listIndexInheritanceQuery = `
SELECT
  sc.nspname,
  cc.relname,
  sp.nspname,
  cp.relname
FROM
  pg_catalog.pg_inherits i
  left JOIN pg_catalog.pg_class cp ON cp.oid = i.inhparent
  left join pg_catalog.pg_class cc ON cc.oid = i.inhrelid
  left join pg_catalog.pg_namespace sp on cp.relnamespace = sp.oid
  left join pg_catalog.pg_namespace sc on cc.relnamespace = sc.oid
WHERE (cp.relkind = 'i' or cp.relkind = 'I') and (cc.relkind = 'i' or cc.relkind = 'I')
`

func getIndexInheritance(txn *sql.Tx) (map[db.IndexKey]*db.IndexKey, error) {
	result := make(map[db.IndexKey]*db.IndexKey)
	rows, err := txn.Query(listIndexInheritanceQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var schemaName, indexName, parentSchemaName, parentIndexName string
		if err := rows.Scan(&schemaName, &indexName, &parentSchemaName, &parentIndexName); err != nil {
			return nil, err
		}
		key := db.IndexKey{Schema: schemaName, Index: indexName}
		parentKey := db.IndexKey{Schema: parentSchemaName, Index: parentIndexName}
		result[key] = &parentKey
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

func setTxSearchPath(txn *sql.Tx, searchPath string) error {
	// The new value of the search_path will only apply during the current transaction.
	const setSearchPathQuery = `SELECT pg_catalog.set_config('search_path', $1, true);`
	if _, err := txn.Exec(setSearchPathQuery, searchPath); err != nil {
		return err
	}
	return nil
}

func getListColumnQuery(schemaFilter string) string {
	return fmt.Sprintf(`
SELECT
	cols.table_schema,
	cols.table_name,
	cols.column_name,
	cols.data_type,
	cols.character_maximum_length,
	cols.numeric_precision,
	cols.numeric_scale,
	cols.datetime_precision,
	cols.ordinal_position,
	cols.column_default,
	cols.is_nullable,
	cols.collation_name,
	cols.udt_schema,
	cols.udt_name,
	cols.identity_generation,
	pg_catalog.col_description(format('%%s.%%s', quote_ident(table_schema), quote_ident(table_name))::regclass, cols.ordinal_position::int) as column_comment
FROM INFORMATION_SCHEMA.COLUMNS AS cols
WHERE %s
ORDER BY cols.table_schema, cols.table_name, cols.ordinal_position;`, schemaFilter)
}

// getTableColumns gets the columns of a table.
func getTableColumns(txn *sql.Tx, accessibleSchemas []string) (map[db.TableKey][]*storepb.ColumnMetadata, error) {
	columnsMap := make(map[db.TableKey][]*storepb.ColumnMetadata)
	query := getListColumnQuery(buildSchemaFilter("cols.table_schema", accessibleSchemas))
	rows, err := txn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		column := &storepb.ColumnMetadata{}
		var schemaName, tableName, nullable string
		var characterMaxLength, numericPrecision, numericScale, datetimePrecision, defaultStr, collation, udtSchema, udtName, identityGeneration, comment sql.NullString
		if err := rows.Scan(&schemaName, &tableName, &column.Name, &column.Type, &characterMaxLength, &numericPrecision, &numericScale, &datetimePrecision, &column.Position, &defaultStr, &nullable, &collation, &udtSchema, &udtName, &identityGeneration, &comment); err != nil {
			return nil, err
		}
		// Store schema-qualified default in the Default field for Step 4 of column default migration
		if defaultStr.Valid {
			column.Default = defaultStr.String
		} else {
			column.Default = "" // Handle NULL case (no default or DEFAULT NULL)
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
		case "numeric", "decimal":
			// Handle numeric/decimal precision and scale
			if numericPrecision.Valid && numericScale.Valid {
				// If scale is 0, only show precision (NUMERIC(8) not NUMERIC(8,0))
				if numericScale.String == "0" {
					column.Type = fmt.Sprintf("%s(%s)", column.Type, numericPrecision.String)
				} else {
					column.Type = fmt.Sprintf("%s(%s,%s)", column.Type, numericPrecision.String, numericScale.String)
				}
			} else if numericPrecision.Valid {
				column.Type = fmt.Sprintf("%s(%s)", column.Type, numericPrecision.String)
			}
		case "time", "time without time zone", "time with time zone",
			"timestamp", "timestamp without time zone", "timestamp with time zone":
			// Handle time/timestamp precision
			if datetimePrecision.Valid {
				// For time types, add precision before "without time zone" part
				if strings.Contains(column.Type, "without time zone") {
					baseType := strings.Replace(column.Type, " without time zone", "", 1)
					column.Type = fmt.Sprintf("%s(%s) without time zone", baseType, datetimePrecision.String)
				} else if strings.Contains(column.Type, "with time zone") {
					baseType := strings.Replace(column.Type, " with time zone", "", 1)
					column.Type = fmt.Sprintf("%s(%s) with time zone", baseType, datetimePrecision.String)
				} else {
					// For plain "time" or "timestamp"
					column.Type = fmt.Sprintf("%s(%s)", column.Type, datetimePrecision.String)
				}
			}
		default:
			// Keep the type as is
		}
		column.Collation = collation.String
		column.Comment = comment.String
		if identityGeneration.Valid {
			switch strings.ToUpper(identityGeneration.String) {
			case "ALWAYS":
				column.IdentityGeneration = storepb.ColumnMetadata_ALWAYS
			case "BY DEFAULT":
				column.IdentityGeneration = storepb.ColumnMetadata_BY_DEFAULT
			default:
				// Keep the default value
			}
		}

		key := db.TableKey{Schema: schemaName, Table: tableName}
		columnsMap[key] = append(columnsMap[key], column)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return columnsMap, nil
}

func getListMaterializedViewQuery(schemaFilter string) string {
	return fmt.Sprintf(`
SELECT pc.oid, schemaname, matviewname, definition, obj_description(format('%%s.%%s', quote_ident(schemaname), quote_ident(matviewname))::regclass)
FROM pg_catalog.pg_matviews
	LEFT JOIN pg_class as pc ON pc.oid = format('%%s.%%s', quote_ident(schemaname), quote_ident(matviewname))::regclass
WHERE %s
ORDER BY schemaname, matviewname;`, schemaFilter)
}

func getMaterializedViews(txn *sql.Tx, indexMap map[db.TableKey][]*storepb.IndexMetadata, triggerMap map[db.TableKey][]*storepb.TriggerMetadata, extensionDepend map[int]bool, accessibleSchemas []string) (map[string][]*storepb.MaterializedViewMetadata, map[int]*db.TableKey, error) {
	matviewMap := make(map[string][]*storepb.MaterializedViewMetadata)
	materializedViewOidMap := make(map[int]*db.TableKey)

	query := getListMaterializedViewQuery(buildSchemaFilter("schemaname", accessibleSchemas))
	rows, err := txn.Query(query)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	for rows.Next() {
		matview := &storepb.MaterializedViewMetadata{}
		var oid int
		var schemaName string
		var def, comment sql.NullString
		if err := rows.Scan(&oid, &schemaName, &matview.Name, &def, &comment); err != nil {
			return nil, nil, err
		}
		// Skip system views.
		if pgparser.IsSystemView(matview.Name) {
			continue
		}
		if extensionDepend[oid] {
			matview.SkipDump = true
		}

		// Return error on NULL view definition.
		if !def.Valid {
			return nil, nil, errors.Errorf("schema %q materialized view %q has empty definition; please check whether proper privileges have been granted to Bytebase", schemaName, matview.Name)
		}
		matview.Definition = def.String
		if comment.Valid {
			matview.Comment = comment.String
		}
		viewKey := db.TableKey{Schema: schemaName, Table: matview.Name}
		matview.Indexes = indexMap[viewKey]
		matview.Triggers = triggerMap[viewKey]

		matviewMap[schemaName] = append(matviewMap[schemaName], matview)
		materializedViewOidMap[oid] = &viewKey
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	for schemaName, list := range matviewMap {
		for _, matview := range list {
			dependencies, err := getViewDependencies(txn, schemaName, matview.Name)
			if err != nil {
				return nil, nil, errors.Wrapf(err, "failed to get materialized view %q dependencies", matview.Name)
			}
			matview.DependencyColumns = dependencies
		}
	}

	return matviewMap, materializedViewOidMap, nil
}

func getListViewQuery(schemaFilter string) string {
	return fmt.Sprintf(`
SELECT pc.oid, schemaname, viewname, definition, obj_description(format('%%s.%%s', quote_ident(schemaname), quote_ident(viewname))::regclass)
FROM pg_catalog.pg_views
	LEFT JOIN pg_class as pc ON pc.oid = format('%%s.%%s', quote_ident(schemaname), quote_ident(viewname))::regclass
WHERE %s
ORDER BY schemaname, viewname;`, schemaFilter)
}

// getViews gets all views of a database.
func getViews(txn *sql.Tx, columnMap map[db.TableKey][]*storepb.ColumnMetadata, triggerMap map[db.TableKey][]*storepb.TriggerMetadata, extensionDepend map[int]bool, accessibleSchemas []string) (map[string][]*storepb.ViewMetadata, map[int]*db.TableKey, error) {
	viewMap := make(map[string][]*storepb.ViewMetadata)
	viewOidMap := make(map[int]*db.TableKey)

	query := getListViewQuery(buildSchemaFilter("schemaname", accessibleSchemas))
	rows, err := txn.Query(query)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	for rows.Next() {
		view := &storepb.ViewMetadata{}
		var oid int
		var schemaName string
		var def, comment sql.NullString
		if err := rows.Scan(&oid, &schemaName, &view.Name, &def, &comment); err != nil {
			return nil, nil, err
		}
		// Skip system views.
		if pgparser.IsSystemView(view.Name) {
			continue
		}
		if extensionDepend[oid] {
			view.SkipDump = true
		}

		// Return error on NULL view definition.
		// https://github.com/bytebase/bytebase/issues/343
		if !def.Valid {
			return nil, nil, errors.Errorf("schema %q view %q has empty definition; please check whether proper privileges have been granted to Bytebase", schemaName, view.Name)
		}
		view.Definition = def.String
		if comment.Valid {
			view.Comment = comment.String
		}

		key := db.TableKey{Schema: schemaName, Table: view.Name}
		view.Columns = columnMap[key]
		view.Triggers = triggerMap[key]

		viewMap[schemaName] = append(viewMap[schemaName], view)
		viewOidMap[oid] = &key
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	for schemaName, list := range viewMap {
		for _, view := range list {
			dependencies, err := getViewDependencies(txn, schemaName, view.Name)
			if err != nil {
				return nil, nil, errors.Wrapf(err, "failed to get view %q dependencies", view.Name)
			}
			view.DependencyColumns = dependencies
		}
	}

	return viewMap, viewOidMap, nil
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

// getRules gets all rules for tables and views in a database.
func getRules(txn *sql.Tx, accessibleSchemas []string) (map[db.TableKey][]*storepb.RuleMetadata, error) {
	ruleMap := make(map[db.TableKey][]*storepb.RuleMetadata)

	query := fmt.Sprintf(`
		SELECT
			n.nspname AS schema_name,
			c.relname AS table_name,
			r.rulename AS rule_name,
			CASE r.ev_type
				WHEN '1' THEN 'SELECT'
				WHEN '2' THEN 'UPDATE'
				WHEN '3' THEN 'INSERT'
				WHEN '4' THEN 'DELETE'
			END AS event,
			r.is_instead,
			r.ev_enabled != 'D' AS is_enabled,
			pg_get_expr(r.ev_qual, r.ev_class, true) AS condition,
			pg_get_ruledef(r.oid, true) AS definition
		FROM pg_rewrite r
		JOIN pg_class c ON c.oid = r.ev_class
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE r.rulename NOT IN ('_RETURN', '_NOTHING')
			AND %s
		ORDER BY n.nspname, c.relname, r.rulename;`, buildSchemaFilter("n.nspname", accessibleSchemas))

	rows, err := txn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		rule := &storepb.RuleMetadata{}
		var schemaName, tableName string
		var condition sql.NullString

		if err := rows.Scan(
			&schemaName,
			&tableName,
			&rule.Name,
			&rule.Event,
			&rule.IsInstead,
			&rule.IsEnabled,
			&condition,
			&rule.Definition,
		); err != nil {
			return nil, err
		}

		if condition.Valid {
			rule.Condition = condition.String
		}

		// Extract the action from the definition
		// The definition looks like: CREATE RULE rule_name AS ON event TO table WHERE condition DO action
		// We'll store the full definition and parse the action if needed
		rule.Action = rule.Definition

		key := db.TableKey{Schema: schemaName, Table: tableName}
		ruleMap[key] = append(ruleMap[key], rule)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return ruleMap, nil
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

func getEnumTypes(txn *sql.Tx, extensionDepend map[int]bool, accessibleSchemas []string) (map[string][]*storepb.EnumTypeMetadata, error) {
	query := fmt.Sprintf(`
	SELECT
		pt.oid,
		pn.nspname as schema_name,
		pt.typname as enum_name,
		pe.enumlabel as enum_value,
		pg_catalog.obj_description(pt.oid) as enum_comment
	FROM pg_enum as pe
		LEFT JOIN pg_type as pt ON pe.enumtypid = pt.oid
		LEFT JOIN pg_namespace as pn ON pt.typnamespace = pn.oid
	WHERE %s
	ORDER BY pn.nspname, pt.typname, pe.enumsortorder;`, buildSchemaFilter("pn.nspname", accessibleSchemas))
	rows, err := txn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	enumTypes := make(map[string][]*storepb.EnumTypeMetadata)
	currentEnumSchema := ""
	currentEnumNmae := ""
	currentEnumComment := ""
	currentSkipDump := false
	var currentEnumValues []string
	for rows.Next() {
		var oid int
		var schemaName, enumName, enumValue string
		var comment sql.NullString
		if err := rows.Scan(&oid, &schemaName, &enumName, &enumValue, &comment); err != nil {
			return nil, err
		}

		if currentEnumSchema != schemaName || currentEnumNmae != enumName {
			if currentEnumSchema != "" {
				enumTypes[currentEnumSchema] = append(enumTypes[currentEnumSchema], &storepb.EnumTypeMetadata{
					Name:     currentEnumNmae,
					Values:   currentEnumValues,
					Comment:  currentEnumComment,
					SkipDump: currentSkipDump,
				})
			}
			currentEnumSchema = schemaName
			currentEnumNmae = enumName
			currentEnumValues = []string{}
			if comment.Valid {
				currentEnumComment = comment.String
			} else {
				currentEnumComment = ""
			}
			currentSkipDump = extensionDepend[oid]
		}
		currentEnumValues = append(currentEnumValues, enumValue)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if currentEnumSchema != "" {
		enumTypes[currentEnumSchema] = append(enumTypes[currentEnumSchema], &storepb.EnumTypeMetadata{
			Name:     currentEnumNmae,
			Values:   currentEnumValues,
			Comment:  currentEnumComment,
			SkipDump: currentSkipDump,
		})
	}

	return enumTypes, nil
}

// getSequences gets all sequences of a database.
func getSequences(txn *sql.Tx, tableOidMap map[int]*db.TableKeyWithColumns, extensionDepend map[int]bool, accessibleSchemas []string) (map[string][]*storepb.SequenceMetadata, error) {
	sequenceOwnerMap, err := getSequenceOwners(txn)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get sequence owners")
	}

	query := fmt.Sprintf(`
	SELECT
		pc.oid,
		schemaname,
		sequencename,
		data_type,
		start_value,
		min_value,
		max_value,
		increment_by,
		cycle,
		cache_size,
		last_value,
		pg_catalog.obj_description(pc.oid) as sequence_comment
	FROM pg_sequences
		LEFT JOIN pg_class as pc ON pc.oid = format('%%s.%%s', quote_ident(schemaname), quote_ident(sequencename))::regclass
	WHERE %s
	ORDER BY schemaname, sequencename;`, buildSchemaFilter("schemaname", accessibleSchemas))
	rows, err := txn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	sequenceMap := make(map[string][]*storepb.SequenceMetadata)
	for rows.Next() {
		var oid int
		var schemaName, sequenceName, dataType string
		var startValue, minValue, maxValue, incrementBy, cacheSize int64
		var comment sql.NullString
		var cycle bool
		var lastValue sql.NullInt64
		if err := rows.Scan(&oid, &schemaName, &sequenceName, &dataType, &startValue, &minValue, &maxValue, &incrementBy, &cycle, &cacheSize, &lastValue, &comment); err != nil {
			return nil, err
		}
		skipDump := false
		if extensionDepend[oid] {
			skipDump = true
		}
		lastValueStr := ""
		if lastValue.Valid {
			lastValueStr = strconv.FormatInt(lastValue.Int64, 10)
		}
		sequenceComment := ""
		if comment.Valid {
			sequenceComment = comment.String
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
			Comment:   sequenceComment,
			SkipDump:  skipDump,
		}
		if columnOidKey, ok := sequenceOwnerMap[oid]; ok {
			if tableKey, ok := tableOidMap[columnOidKey.TableOid]; ok {
				sequence.OwnerTable = tableKey.Table
				// PostgreSQL column ID is 1-based.
				if len(tableKey.Columns) > columnOidKey.ColumnID-1 {
					sequence.OwnerColumn = tableKey.Columns[columnOidKey.ColumnID-1].Name
				}
			}
		}

		sequenceMap[schemaName] = append(sequenceMap[schemaName], sequence)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return sequenceMap, nil
}

type ColumnOidKey struct {
	TableOid int
	ColumnID int
}

func getSequenceOwners(txn *sql.Tx) (map[int]ColumnOidKey, error) {
	query := `
	SELECT
		c.oid,
		refobjid AS owning_tab,
		refobjsubid AS owning_col
	FROM pg_class c
  		LEFT JOIN pg_depend d ON
  			(c.relkind =  'S' AND
                d.classid = 'pg_class'::regclass AND d.objid = c.oid AND
                d.objsubid = 0 AND
                d.refclassid = 'pg_class'::regclass AND d.deptype IN ('a', 'i'))
	WHERE refobjid is NOT NULL and refobjsubid is NOT NULL;`

	rows, err := txn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	sequenceOwnerMap := make(map[int]ColumnOidKey)
	for rows.Next() {
		var oid, tableOid, columnID int
		if err := rows.Scan(&oid, &tableOid, &columnID); err != nil {
			return nil, err
		}
		sequenceOwnerMap[oid] = ColumnOidKey{TableOid: tableOid, ColumnID: columnID}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return sequenceOwnerMap, nil
}

func getTriggers(txn *sql.Tx, extensionDepend map[int]bool, accessibleSchemas []string) (map[db.TableKey][]*storepb.TriggerMetadata, error) {
	query := fmt.Sprintf(`
	SELECT
		pt.oid,
		pn.nspname as schema_name,
		pc.relname as table_name,
		pt.tgname as trigger_name,
		pg_get_triggerdef(pt.oid) as trigger_def,
		obj_description(pt.oid) as trigger_comment
	FROM pg_trigger as pt
		LEFT JOIN pg_class as pc ON pc.oid = pt.tgrelid
		LEFT JOIN pg_namespace as pn ON pn.oid = pc.relnamespace
	WHERE %s AND pt.tgisinternal = false;`, buildSchemaFilter("pn.nspname", accessibleSchemas))
	rows, err := txn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	triggersMap := make(map[db.TableKey][]*storepb.TriggerMetadata)
	for rows.Next() {
		trigger := &storepb.TriggerMetadata{}
		var oid int
		var schemaName, tableName string
		var comment sql.NullString
		if err := rows.Scan(&oid, &schemaName, &tableName, &trigger.Name, &trigger.Body, &comment); err != nil {
			return nil, err
		}
		if extensionDepend[oid] {
			trigger.SkipDump = true
		}
		if comment.Valid {
			trigger.Comment = comment.String
		}
		tableKey := db.TableKey{Schema: schemaName, Table: tableName}
		triggersMap[tableKey] = append(triggersMap[tableKey], trigger)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return triggersMap, nil
}

func getEventTriggers(txn *sql.Tx, extensionDepend map[int]bool) ([]*storepb.EventTriggerMetadata, error) {
	query := `
	SELECT
		et.oid,
		et.evtname AS trigger_name,
		et.evtevent AS event_type,
		et.evttags AS tags,
		n.nspname AS function_schema,
		p.proname AS function_name,
		et.evtenabled AS enabled,
		obj_description(et.oid, 'pg_event_trigger') AS comment
	FROM pg_event_trigger et
	JOIN pg_proc p ON et.evtfoid = p.oid
	JOIN pg_namespace n ON p.pronamespace = n.oid
	ORDER BY et.evtname;`

	rows, err := txn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var eventTriggers []*storepb.EventTriggerMetadata
	for rows.Next() {
		var oid int
		var name, eventType, functionSchema, functionName, enabled string
		var tags pq.StringArray
		var comment sql.NullString

		if err := rows.Scan(&oid, &name, &eventType, &tags, &functionSchema,
			&functionName, &enabled, &comment); err != nil {
			return nil, err
		}

		eventTrigger := &storepb.EventTriggerMetadata{
			Name:           name,
			Event:          eventType,
			Tags:           []string(tags),
			FunctionSchema: functionSchema,
			FunctionName:   functionName,
			Enabled:        enabled != "D", // D = disabled
		}

		// Build the CREATE EVENT TRIGGER definition manually
		// PostgreSQL doesn't have pg_get_event_trigger_def() function
		eventTrigger.Definition = buildEventTriggerDefinition(eventTrigger)

		if comment.Valid {
			eventTrigger.Comment = comment.String
		}
		if extensionDepend[oid] {
			eventTrigger.SkipDump = true
		}

		eventTriggers = append(eventTriggers, eventTrigger)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return eventTriggers, nil
}

// buildEventTriggerDefinition constructs the CREATE EVENT TRIGGER statement from metadata.
func buildEventTriggerDefinition(et *storepb.EventTriggerMetadata) string {
	var buf strings.Builder
	buf.WriteString("CREATE EVENT TRIGGER ")
	buf.WriteString(fmt.Sprintf("%q", et.Name))
	buf.WriteString(" ON ")
	buf.WriteString(et.Event)

	if len(et.Tags) > 0 {
		buf.WriteString("\n  WHEN TAG IN (")
		for i, tag := range et.Tags {
			if i > 0 {
				buf.WriteString(", ")
			}
			buf.WriteString("'")
			buf.WriteString(tag)
			buf.WriteString("'")
		}
		buf.WriteString(")")
	}

	buf.WriteString("\n  EXECUTE FUNCTION ")
	if et.FunctionSchema != "" {
		buf.WriteString(fmt.Sprintf("%q", et.FunctionSchema))
		buf.WriteString(".")
	}
	buf.WriteString(fmt.Sprintf("%q", et.FunctionName))
	buf.WriteString("()")

	return buf.String()
}

// getUniqueConstraints is no longer needed - we get constraint info directly from pg_constraint in the main query

func getListIndexQuery(schemaFilter string) string {
	return fmt.Sprintf(`
SELECT
    n.nspname as schema_name,
    t.relname as table_name,
    i.relname as index_name,
    am.amname as index_method,
    ix.indisunique as is_unique,
    ix.indisprimary as is_primary,
    pg_get_indexdef(i.oid) as index_definition,
    -- Get array of key expressions
    (SELECT array_agg(pg_get_indexdef(i.oid, k + 1, true) ORDER BY k)
     FROM generate_subscripts(ix.indkey, 1) as k
     WHERE k < ix.indnkeyatts  -- Only key columns, not included columns
    ) as key_expressions,
    (SELECT array_agg(op.opcname ORDER BY k)
     FROM generate_subscripts(ix.indkey, 1) as k
     JOIN pg_opclass op ON ix.indclass[k] = op.oid
     WHERE k < ix.indnkeyatts  -- Only key columns, not included columns
    ) as key_opclass_names,
    (SELECT array_agg(op.opcdefault ORDER BY k)
     FROM generate_subscripts(ix.indkey, 1) as k
     JOIN pg_opclass op ON ix.indclass[k] = op.oid
     WHERE k < ix.indnkeyatts  -- Only key columns, not included columns
    ) as key_opclass_defaults,
    -- Check if it's a constraint (primary key, unique, or exclude constraint)
    CASE
        WHEN ix.indisprimary THEN true
        WHEN EXISTS (
            SELECT 1 FROM pg_constraint c
            WHERE c.conindid = i.oid AND c.contype IN ('u', 'p', 'x')
        ) THEN true
        ELSE false
    END as is_constraint,
    obj_description(i.oid) as comment,
    -- Extract indoption array for sort order information (ASC/DESC, NULLS FIRST/LAST)
    ix.indoption as index_options
FROM pg_class i
JOIN pg_index ix ON i.oid = ix.indexrelid
JOIN pg_class t ON ix.indrelid = t.oid
JOIN pg_namespace n ON t.relnamespace = n.oid
JOIN pg_am am ON i.relam = am.oid
WHERE %s
ORDER BY n.nspname, t.relname, i.relname;`, schemaFilter)
}

// parseIndexOptions parses PostgreSQL indoption int2vector to extract sort order information
// indoption is a space-separated string of integers where each integer is a bitmask:
// - Bit 0: DESC (1 = DESC, 0 = ASC)
// - Bit 1: NULLS FIRST (1 = NULLS FIRST, 0 = NULLS LAST)
func parseIndexOptions(optionsStr string, keyCount int) ([]bool, []bool) {
	descending := make([]bool, keyCount)
	nullsFirst := make([]bool, keyCount)

	if optionsStr == "" {
		return descending, nullsFirst
	}

	// Parse space-separated int2vector
	optionStrs := strings.Fields(optionsStr)

	for i := 0; i < keyCount && i < len(optionStrs); i++ {
		option, err := strconv.ParseInt(optionStrs[i], 10, 16)
		if err != nil {
			continue // Skip invalid options, keep defaults (false, false)
		}

		// Bit 0: DESC flag
		descending[i] = (option & 1) != 0
		// Bit 1: NULLS FIRST flag
		nullsFirst[i] = (option & 2) != 0
	}

	return descending, nullsFirst
}

// getIndexes gets all indices of a database.
func getIndexes(txn *sql.Tx, indexInheritanceMap map[db.IndexKey]*db.IndexKey, accessibleSchemas []string) (map[db.TableKey][]*storepb.IndexMetadata, error) {
	indexMap := make(map[db.TableKey][]*storepb.IndexMetadata)

	query := getListIndexQuery(buildSchemaFilter("n.nspname", accessibleSchemas))
	rows, err := txn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		index := &storepb.IndexMetadata{}
		var schemaName, tableName, indexMethod, indexDefinition string
		var isUnique, isPrimary, isConstraint bool
		var keyExpressions pq.StringArray
		var opclassNames pq.StringArray
		var opclassDefaults pq.BoolArray
		var indexOptions string // indoption as string (int2vector)
		var comment sql.NullString

		if err := rows.Scan(
			&schemaName,
			&tableName,
			&index.Name,
			&indexMethod,
			&isUnique,
			&isPrimary,
			&indexDefinition,
			&keyExpressions,
			&opclassNames,
			&opclassDefaults,
			&isConstraint,
			&comment,
			&indexOptions, // scan indoption string
		); err != nil {
			return nil, err
		}

		// Set index properties from query results
		index.Type = indexMethod
		index.Unique = isUnique
		index.Primary = isPrimary
		index.IsConstraint = isConstraint
		index.Expressions = []string(keyExpressions)
		index.OpclassNames = opclassNames
		index.OpclassDefaults = opclassDefaults

		// Parse sort order information from indoption string
		keyCount := len(keyExpressions)
		if keyCount > 0 {
			descending, _ := parseIndexOptions(indexOptions, keyCount)
			index.Descending = descending
			// Note: nullsFirst would be used when we add the proto field for NULLS handling
		}

		// Ensure definition ends with semicolon
		if !strings.HasSuffix(indexDefinition, ";") {
			index.Definition = indexDefinition + ";"
		} else {
			index.Definition = indexDefinition
		}

		if comment.Valid {
			index.Comment = comment.String
		}

		// Handle index inheritance
		if parentKey, ok := indexInheritanceMap[db.IndexKey{Schema: schemaName, Index: index.Name}]; ok && parentKey != nil {
			index.ParentIndexSchema = parentKey.Schema
			index.ParentIndexName = parentKey.Index
		}

		key := db.TableKey{Schema: schemaName, Table: tableName}
		indexMap[key] = append(indexMap[key], index)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return indexMap, nil
}

func getListFunctionDependencyTablesQuery(schemaFilter string) string {
	return fmt.Sprintf(`
select
	p.oid as function_oid,
	pt.typrelid as table_oid
from pg_proc p
	left join pg_depend d on p.oid = d.objid
	left join pg_type pt on d.refobjid = pt.oid
	left join pg_namespace n on p.pronamespace = n.oid
where %s
  AND pt.typrelid IS NOT NULL
`, schemaFilter)
}

func getFunctionDependencyTables(txn *sql.Tx, accessibleSchemas []string) (map[int][]int, error) {
	dependencyTableMap := make(map[int][]int)

	query := getListFunctionDependencyTablesQuery(buildSchemaFilter("n.nspname", accessibleSchemas))
	rows, err := txn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var functionOid, tableOid int
		if err := rows.Scan(&functionOid, &tableOid); err != nil {
			return nil, err
		}
		dependencyTableMap[functionOid] = append(dependencyTableMap[functionOid], tableOid)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return dependencyTableMap, nil
}

func getListFunctionQuery(schemaFilter string) string {
	return fmt.Sprintf(`
select p.oid, n.nspname as function_schema,
	p.proname as function_name,
	pg_catalog.pg_get_function_identity_arguments(p.oid) as arguments,
	case when l.lanname = 'internal' then p.prosrc
			else pg_get_functiondef(p.oid)
			end as definition,
	pg_catalog.obj_description(p.oid) as comment
from pg_proc p
left join pg_namespace n on p.pronamespace = n.oid
left join pg_language l on p.prolang = l.oid
left join pg_type t on t.oid = p.prorettype
where %s
order by function_schema, function_name;`, schemaFilter)
}

// getFunctions gets all functions of a database.
func getFunctions(
	txn *sql.Tx,
	functionDependencyTables map[int][]int,
	tableOidMap map[int]*db.TableKeyWithColumns,
	viewOidMap, materializedViewOidMap map[int]*db.TableKey,
	extensionDepend map[int]bool,
	accessibleSchemas []string,
) (map[string][]*storepb.FunctionMetadata, error) {
	functionMap := make(map[string][]*storepb.FunctionMetadata)

	query := getListFunctionQuery(buildSchemaFilter("n.nspname", accessibleSchemas))
	rows, err := txn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		function := &storepb.FunctionMetadata{}
		var oid int
		var schemaName, arguments string
		var comment sql.NullString
		if err := rows.Scan(&oid, &schemaName, &function.Name, &arguments, &function.Definition, &comment); err != nil {
			return nil, err
		}
		// Skip internal functions.
		if pgparser.IsSystemFunction(function.Name, function.Definition) {
			continue
		}
		if extensionDepend[oid] {
			function.SkipDump = true
		}
		if comment.Valid {
			function.Comment = comment.String
		}

		function.Signature = fmt.Sprintf("%s(%s)", function.Name, arguments)
		for _, tableOid := range functionDependencyTables[oid] {
			if table, ok := tableOidMap[tableOid]; ok {
				function.DependencyTables = append(function.DependencyTables, &storepb.DependencyTable{
					Schema: table.Schema,
					Table:  table.Table,
				})
			} else if view, ok := viewOidMap[tableOid]; ok {
				function.DependencyTables = append(function.DependencyTables, &storepb.DependencyTable{
					Schema: view.Schema,
					Table:  view.Table,
				})
			} else if matview, ok := materializedViewOidMap[tableOid]; ok {
				function.DependencyTables = append(function.DependencyTables, &storepb.DependencyTable{
					Schema: matview.Schema,
					Table:  matview.Table,
				})
			}
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
