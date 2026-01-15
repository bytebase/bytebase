package oracle

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	"github.com/bytebase/bytebase/backend/plugin/parser/plsql"
)

const systemSchema = "'ANONYMOUS','APPQOSSYS','AUDSYS','CTXSYS','DBSFWUSER','DBSNMP','DGPDB_INT','DIP','DVF','DVSYS','GGSYS','GSMADMIN_INTERNAL','GSMCATUSER','GSMROOTUSER','GSMUSER','LBACSYS','MDDATA','MDSYS','OPS$ORACLE','ORACLE_OCM','OUTLN','REMOTE_SCHEDULER_AGENT','SYS','SYS$UMF','SYSBACKUP','SYSDG','SYSKM','SYSRAC','SYSTEM','XDB','XS$NULL','XS$$NULL','FLOWS_FILES','HR','MDSYS','EXFSYS','MGMT_VIEW','OLAPSYS','ORDDATA','ORDPLUGINS','ORDSYS','OWBSYS','OWBSYS_AUDIT','SCOTT','SI_INFORMTN_SCHEMA','SPATIAL_CSW_ADMIN_USR','SPATIAL_WFS_ADMIN_USR','SYSMAN','WMSYS','OJVMSYS'"

var (
	semVersionRegex       = regexp.MustCompile(`[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+`)
	canonicalVersionRegex = regexp.MustCompile(`[0-9][0-9][a-z]`)
)

// SyncInstance syncs the instance.
func (d *Driver) SyncInstance(ctx context.Context) (*db.InstanceMetadata, error) {
	var fullVersion string
	queryVersion := "SELECT BANNER FROM v$version WHERE banner LIKE 'Oracle%'"
	if err := d.db.QueryRowContext(ctx, queryVersion).Scan(&fullVersion); err != nil {
		return nil, util.FormatErrorWithQuery(err, queryVersion)
	}
	tokens := strings.Fields(fullVersion)
	var version, canonicalVersion string
	for _, token := range tokens {
		if semVersionRegex.MatchString(token) {
			version = token
			continue
		}
		if canonicalVersionRegex.MatchString(token) {
			canonicalVersion = token
			continue
		}
	}
	if canonicalVersion != "" {
		version = fmt.Sprintf("%s (%s)", version, canonicalVersion)
	}

	txn, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer txn.Rollback()

	schemas, err := getSchemas(txn)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get schemas from database %q", d.databaseName)
	}
	var databases []*storepb.DatabaseSchemaMetadata
	for _, schema := range schemas {
		databases = append(databases, &storepb.DatabaseSchemaMetadata{
			Name:        schema,
			ServiceName: "",
		})
	}

	if err := txn.Commit(); err != nil {
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

	version, err := d.GetVersion()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get version")
	}

	columnMap, err := getTableColumns(txn, d.databaseName, version)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get table columns from database %q", d.databaseName)
	}
	tableTriggerMap, viewTriggerMap, err := getTriggers(txn, d.databaseName)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get triggers from database %q", d.databaseName)
	}
	tableMap, err := getTables(txn, d.databaseName, columnMap, tableTriggerMap)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get tables from database %q", d.databaseName)
	}
	viewMap, err := getViews(txn, d.databaseName, columnMap, viewTriggerMap)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get views from database %q", d.databaseName)
	}
	sequences, err := getSequences(txn, d.databaseName)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get sequences from database %q", d.databaseName)
	}
	dbLinks, err := getDBLinks(txn)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get db links from database %q", d.databaseName)
	}
	functions, procedures, packages, err := getRoutines(txn, d.databaseName)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get routines from database %q", d.databaseName)
	}
	materializedViews, err := getMaterializedViews(txn, d.databaseName, columnMap)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get materialized views from database %q", d.databaseName)
	}

	if err := txn.Commit(); err != nil {
		return nil, err
	}

	databaseMetadata := &storepb.DatabaseSchemaMetadata{
		Name:            d.databaseName,
		ServiceName:     d.serviceName,
		LinkedDatabases: dbLinks,
	}
	databaseMetadata.Schemas = append(databaseMetadata.Schemas, &storepb.SchemaMetadata{
		Name:              "",
		Tables:            tableMap[d.databaseName],
		Views:             viewMap[d.databaseName],
		MaterializedViews: materializedViews,
		Sequences:         sequences,
		Functions:         functions,
		Procedures:        procedures,
		Packages:          packages,
	})
	return databaseMetadata, nil
}

func getDBLinks(txn *sql.Tx) ([]*storepb.LinkedDatabaseMetadata, error) {
	query := `
	SELECT DB_LINK, HOST, USERNAME
	FROM all_db_links
	ORDER BY DB_LINK`
	slog.Debug("running get db link query")
	rows, err := txn.Query(query)
	if err != nil {
		return nil, util.FormatErrorWithQuery(err, query)
	}
	defer rows.Close()

	var result []*storepb.LinkedDatabaseMetadata
	for rows.Next() {
		dbLink := &storepb.LinkedDatabaseMetadata{}
		var name, host, username sql.NullString
		if err := rows.Scan(&name, &host, &username); err != nil {
			return nil, err
		}
		if !name.Valid {
			continue
		}
		dbLink.Name = name.String
		if host.Valid {
			dbLink.Host = host.String
		}
		if username.Valid {
			dbLink.Username = username.String
		}
		result = append(result, dbLink)
	}
	if err := rows.Err(); err != nil {
		return nil, util.FormatErrorWithQuery(err, query)
	}
	if err := rows.Close(); err != nil {
		return nil, errors.Wrapf(err, "failed to close rows")
	}

	return result, nil
}

func getSchemas(txn *sql.Tx) ([]string, error) {
	query := fmt.Sprintf(`
		SELECT username FROM all_users
		WHERE username NOT IN (%s) AND username NOT LIKE 'APEX_%%' ORDER BY username`,
		systemSchema)
	slog.Debug("running get schemas query")
	rows, err := txn.Query(query)
	if err != nil {
		return nil, util.FormatErrorWithQuery(err, query)
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
		return nil, util.FormatErrorWithQuery(err, query)
	}
	if err := rows.Close(); err != nil {
		return nil, errors.Wrapf(err, "failed to close rows")
	}

	return result, nil
}

func getTriggers(txn *sql.Tx, schemaName string) (map[db.TableKey][]*storepb.TriggerMetadata, map[db.TableKey][]*storepb.TriggerMetadata, error) {
	tableTriggerMap := make(map[db.TableKey][]*storepb.TriggerMetadata)
	viewTriggerMap := make(map[db.TableKey][]*storepb.TriggerMetadata)

	// Get trigger comments
	triggerCommentMap, err := getTriggerComments(txn, schemaName)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to get trigger comments")
	}

	query := fmt.Sprintf(`
		SELECT TRIGGER_NAME, TABLE_NAME, DESCRIPTION, TRIGGER_BODY, BASE_OBJECT_TYPE, TRIGGER_TYPE, TRIGGERING_EVENT
		FROM sys.all_triggers
		WHERE OWNER = '%s' AND TABLE_NAME is NOT NULL AND (BASE_OBJECT_TYPE = 'TABLE' OR BASE_OBJECT_TYPE = 'VIEW')
		ORDER BY TRIGGER_NAME`, schemaName)

	slog.Debug("running get triggers query")
	rows, err := txn.Query(query)
	if err != nil {
		return nil, nil, util.FormatErrorWithQuery(err, query)
	}
	defer rows.Close()

	for rows.Next() {
		var triggerName, tableName, description, triggerBody, baseObjectType, triggerType, triggeringEvent sql.NullString
		if err := rows.Scan(&triggerName, &tableName, &description, &triggerBody, &baseObjectType, &triggerType, &triggeringEvent); err != nil {
			return nil, nil, err
		}
		if !triggerName.Valid || !tableName.Valid || !description.Valid || !triggerBody.Valid || !baseObjectType.Valid {
			continue
		}
		key := db.TableKey{Schema: schemaName, Table: tableName.String}
		trigger := &storepb.TriggerMetadata{
			Name: triggerName.String,
			Body: constructTriggerBody(description.String, triggerBody.String),
		}
		if triggerType.Valid {
			trigger.Timing = triggerType.String
		}
		if triggeringEvent.Valid {
			trigger.Event = triggeringEvent.String
		}
		// Add trigger comment if available
		triggerKey := db.TableKey{Schema: schemaName, Table: triggerName.String}
		if comment, ok := triggerCommentMap[triggerKey]; ok {
			trigger.Comment = comment
		}

		if baseObjectType.String == "TABLE" {
			tableTriggerMap[key] = append(tableTriggerMap[key], trigger)
		} else {
			viewTriggerMap[key] = append(viewTriggerMap[key], trigger)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, nil, util.FormatErrorWithQuery(err, query)
	}

	return tableTriggerMap, viewTriggerMap, nil
}

func constructTriggerBody(description, triggerBody string) string {
	var buf strings.Builder
	_, _ = buf.WriteString("CREATE OR REPLACE TRIGGER ")
	_, _ = buf.WriteString(description)
	_, _ = buf.WriteString(triggerBody)
	return buf.String()
}

// getTables gets all tables of a database.
func getTables(txn *sql.Tx, schemaName string, columnMap map[db.TableKey][]*storepb.ColumnMetadata, triggerMap map[db.TableKey][]*storepb.TriggerMetadata) (map[string][]*storepb.TableMetadata, error) {
	indexMap, checkConstraintMap, foreignKeyMap, err := getIndexesAndConstraints(txn, schemaName)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get indices")
	}
	columnCommentMap, err := getTableColumnComments(txn, schemaName)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get table column comments")
	}
	for key, columns := range columnMap {
		for _, column := range columns {
			comment, ok := columnCommentMap[db.ColumnKey{Schema: key.Schema, Table: key.Table, Column: column.Name}]
			if ok {
				column.Comment = comment
			}
		}
	}
	tableCommentMap, err := getTableComments(txn, schemaName)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get table comments")
	}
	tableMap := make(map[string][]*storepb.TableMetadata)

	query := fmt.Sprintf(`
		SELECT OWNER, TABLE_NAME, NUM_ROWS
		FROM all_tables
		WHERE OWNER = '%s'
		AND TABLE_NAME NOT IN (
			SELECT MVIEW_NAME FROM all_mviews WHERE OWNER = '%s'
		)
		ORDER BY TABLE_NAME`, schemaName, schemaName)

	slog.Debug("running get tables query")
	rows, err := txn.Query(query)
	if err != nil {
		return nil, util.FormatErrorWithQuery(err, query)
	}
	defer rows.Close()

	for rows.Next() {
		table := &storepb.TableMetadata{}
		var schemaName string
		// https://github.com/rana/ora/issues/57#issuecomment-179909837
		// NUMBER in Oracle can hold 38 decimal digits, so int64 is not enough with its 19 decimal digits.
		// float64 is a little bit better - not precise enough, but won't overflow.
		var count sql.NullFloat64
		if err := rows.Scan(&schemaName, &table.Name, &count); err != nil {
			return nil, err
		}
		table.RowCount = int64(count.Float64)
		key := db.TableKey{Schema: schemaName, Table: table.Name}
		table.Columns = columnMap[key]
		table.Indexes = indexMap[key]
		table.CheckConstraints = checkConstraintMap[key]
		table.ForeignKeys = foreignKeyMap[key]
		table.Triggers = triggerMap[key]
		if comment, ok := tableCommentMap[db.TableKey{Schema: schemaName, Table: table.Name}]; ok {
			table.Comment = comment
		}

		tableMap[schemaName] = append(tableMap[schemaName], table)
	}
	if err := rows.Err(); err != nil {
		return nil, util.FormatErrorWithQuery(err, query)
	}
	if err := rows.Close(); err != nil {
		return nil, errors.Wrapf(err, "failed to close rows")
	}

	return tableMap, nil
}

func getTableComments(txn *sql.Tx, schemaName string) (map[db.TableKey]string, error) {
	tableCommentMap := make(map[db.TableKey]string)

	query := ""
	if schemaName == "" {
		query = fmt.Sprintf(`
		SELECT OWNER, TABLE_NAME, COMMENTS
		FROM all_tab_comments
		WHERE OWNER NOT IN (%s) AND OWNER NOT LIKE 'APEX_%%' AND COMMENTS IS NOT NULL
		ORDER BY OWNER, TABLE_NAME`, systemSchema)
	} else {
		query = fmt.Sprintf(`
		SELECT OWNER, TABLE_NAME, COMMENTS
		FROM all_tab_comments
		WHERE OWNER = '%s' AND COMMENTS IS NOT NULL
		ORDER BY TABLE_NAME`, schemaName)
	}
	slog.Debug("running get table comments query")
	rows, err := txn.Query(query)
	if err != nil {
		return nil, util.FormatErrorWithQuery(err, query)
	}
	defer rows.Close()

	for rows.Next() {
		var schemaName, tableName, comment string
		if err := rows.Scan(&schemaName, &tableName, &comment); err != nil {
			return nil, err
		}
		key := db.TableKey{Schema: schemaName, Table: tableName}
		tableCommentMap[key] = comment
	}
	if err := rows.Err(); err != nil {
		return nil, util.FormatErrorWithQuery(err, query)
	}
	if err := rows.Close(); err != nil {
		return nil, errors.Wrapf(err, "failed to close rows")
	}

	return tableCommentMap, nil
}

func getTableColumnComments(txn *sql.Tx, schemaName string) (map[db.ColumnKey]string, error) {
	columnCommentsMap := make(map[db.ColumnKey]string)

	query := ""
	if schemaName == "" {
		query = fmt.Sprintf(`
		SELECT OWNER, TABLE_NAME, COLUMN_NAME, COMMENTS
		FROM all_col_comments
		WHERE OWNER NOT IN (%s) AND OWNER NOT LIKE 'APEX_%%' AND COMMENTS IS NOT NULL
		ORDER BY OWNER, TABLE_NAME, COLUMN_NAME`, systemSchema)
	} else {
		query = fmt.Sprintf(`
		SELECT OWNER, TABLE_NAME, COLUMN_NAME, COMMENTS
		FROM all_col_comments
		WHERE OWNER = '%s' AND COMMENTS IS NOT NULL
		ORDER BY TABLE_NAME, COLUMN_NAME`, schemaName)
	}
	slog.Debug("running get table column comments query")
	rows, err := txn.Query(query)
	if err != nil {
		return nil, util.FormatErrorWithQuery(err, query)
	}
	defer rows.Close()
	for rows.Next() {
		var schemaName, tableName, columnName, comment string
		if err := rows.Scan(&schemaName, &tableName, &columnName, &comment); err != nil {
			return nil, err
		}
		key := db.ColumnKey{Schema: schemaName, Table: tableName, Column: columnName}
		columnCommentsMap[key] = comment
	}
	if err := rows.Err(); err != nil {
		return nil, util.FormatErrorWithQuery(err, query)
	}
	if err := rows.Close(); err != nil {
		return nil, errors.Wrapf(err, "failed to close rows")
	}

	return columnCommentsMap, nil
}

// getTableColumns gets the columns of a table.
func getTableColumns(txn *sql.Tx, schemaName string, version *plsql.Version) (map[db.TableKey][]*storepb.ColumnMetadata, error) {
	columnsMap := make(map[db.TableKey][]*storepb.ColumnMetadata)

	// https://github.com/bytebase/bytebase/issues/6663
	// Invisible columns don't have column ID so that we need to filter out them.
	query := ""
	// https://docs.oracle.com/en/database/oracle/oracle-database/12.2/refrn/ALL_TAB_COLS.html#GUID-85036F42-140A-406B-BE11-0AC49A00DBA3
	equalOrHigherThan12c2release := version.First > 12 || (version.First == 12 && version.Second >= 2)
	if equalOrHigherThan12c2release {
		query = fmt.Sprintf(`
		SELECT
			OWNER,
			TABLE_NAME,
			COLUMN_NAME,
			DATA_TYPE,
			DATA_LENGTH,
			DATA_PRECISION,
			DATA_SCALE,
			COLUMN_ID,
			DATA_DEFAULT,
			NULLABLE,
			COLLATION,
			DEFAULT_ON_NULL
		FROM sys.all_tab_columns
		WHERE OWNER = '%s' AND COLUMN_ID IS NOT NULL
		ORDER BY TABLE_NAME, COLUMN_ID`, schemaName)
	} else {
		query = fmt.Sprintf(`
		SELECT
			OWNER,
			TABLE_NAME,
			COLUMN_NAME,
			DATA_TYPE,
			DATA_LENGTH,
			DATA_PRECISION,
			DATA_SCALE,
			COLUMN_ID,
			DATA_DEFAULT,
			NULLABLE,
			NULL,
			NULL
		FROM sys.all_tab_columns
		WHERE OWNER = '%s' AND COLUMN_ID IS NOT NULL
		ORDER BY TABLE_NAME, COLUMN_ID`, schemaName)
	}

	slog.Debug("running get columns query")
	rows, err := txn.Query(query)
	if err != nil {
		return nil, util.FormatErrorWithQuery(err, query)
	}
	defer rows.Close()
	for rows.Next() {
		column := &storepb.ColumnMetadata{}
		var schemaName, tableName, nullable string
		var defaultStr, collation, defaultOnNull sql.NullString
		var dataLength, dataPrecision, dataScale sql.NullInt64
		if err := rows.Scan(
			&schemaName,
			&tableName,
			&column.Name,
			&column.Type,
			&dataLength,
			&dataPrecision,
			&dataScale,
			&column.Position,
			&defaultStr,
			&nullable,
			&collation,
			&defaultOnNull,
		); err != nil {
			return nil, err
		}
		column.Type = getTypeString(column.Type, dataLength, dataPrecision, dataScale)
		if defaultStr.Valid {
			// Clean up the default expression to remove current schema name and fix NEXTVAL syntax
			cleanedDefault := cleanDefaultExpression(defaultStr.String, schemaName)
			// Store in Default field (migration from DefaultExpression to Default)
			column.Default = cleanedDefault
		}
		isNullBool, err := util.ConvertYesNo(nullable)
		if err != nil {
			return nil, err
		}
		column.Nullable = isNullBool
		if collation.Valid {
			column.Collation = collation.String
		}
		if defaultOnNull.Valid && defaultOnNull.String == "YES" {
			column.DefaultOnNull = true
		}

		key := db.TableKey{Schema: schemaName, Table: tableName}
		columnsMap[key] = append(columnsMap[key], column)
	}
	if err := rows.Err(); err != nil {
		return nil, util.FormatErrorWithQuery(err, query)
	}
	if err := rows.Close(); err != nil {
		return nil, errors.Wrapf(err, "failed to close rows")
	}

	return columnsMap, nil
}

func getTypeString(dataType string, dataLength, dataPrecision, dataScale sql.NullInt64) string {
	switch dataType {
	case "VARCHAR2", "CHAR":
		return fmt.Sprintf("%s(%d BYTE)", dataType, dataLength.Int64)
	case "NVARCHAR2", "RAW", "UROWID", "NCHAR":
		return fmt.Sprintf("%s(%d)", dataType, dataLength.Int64)
	case "NUMBER":
		switch {
		case !dataPrecision.Valid || dataPrecision.Int64 == 0:
		// do nothing
		case dataPrecision.Valid && dataPrecision.Int64 > 0 && (!dataScale.Valid || dataScale.Int64 == 0):
			return fmt.Sprintf("%s(%d)", dataType, dataPrecision.Int64)
		case dataPrecision.Valid && dataPrecision.Int64 > 0 && dataScale.Valid && dataScale.Int64 > 0:
			return fmt.Sprintf("%s(%d,%d)", dataType, dataPrecision.Int64, dataScale.Int64)
		default:
			// do nothing
		}
	case "FLOAT":
		switch {
		case !dataPrecision.Valid || dataPrecision.Int64 == 0:
		// do nothing
		case dataPrecision.Valid && dataPrecision.Int64 > 0:
			return fmt.Sprintf("%s(%d)", dataType, dataPrecision.Int64)
		default:
			// do nothing
		}
	default:
		// return dataType as-is for unhandled types
	}
	return dataType
}

func getOuterSchemaRColumns(txn *sql.Tx, outerRTableMap map[db.ConstraintKey]string, outerRColumnMap map[db.ConstraintKey][]string, schemaName, constraintName string) (string, []string, error) {
	queryColumns := fmt.Sprintf(`
		SELECT
			TABLE_NAME,
			CONSTRAINT_NAME,
			COLUMN_NAME
		FROM
			SYS.ALL_CONS_COLUMNS
		WHERE
			OWNER = '%s'
		ORDER BY TABLE_NAME, CONSTRAINT_NAME, POSITION`, schemaName)

	slog.Debug("running get outer schema reference columns query")
	rows, err := txn.Query(queryColumns)
	if err != nil {
		return "", nil, util.FormatErrorWithQuery(err, queryColumns)
	}
	defer rows.Close()

	for rows.Next() {
		var tableName, constraintName string
		var columnName sql.NullString
		if err := rows.Scan(&tableName, &constraintName, &columnName); err != nil {
			return "", nil, err
		}
		if !columnName.Valid {
			continue
		}
		key := db.ConstraintKey{Schema: schemaName, Constraint: constraintName}
		outerRTableMap[key] = tableName
		outerRColumnMap[key] = append(outerRColumnMap[key], columnName.String)
	}
	if err := rows.Err(); err != nil {
		return "", nil, util.FormatErrorWithQuery(err, queryColumns)
	}

	constraintKey := db.ConstraintKey{Schema: schemaName, Constraint: constraintName}
	return outerRTableMap[constraintKey], outerRColumnMap[constraintKey], nil
}

func getConstraints(txn *sql.Tx, schemaName string) (
	map[db.TableKey][]*storepb.IndexMetadata,
	map[db.TableKey][]*storepb.CheckConstraintMetadata,
	map[db.TableKey][]*storepb.ForeignKeyMetadata,
	map[db.IndexKey]bool,
	error,
) {
	queryConstraintColumns := fmt.Sprintf(`
		SELECT
			TABLE_NAME,
			CONSTRAINT_NAME,
			COLUMN_NAME
		FROM SYS.ALL_CONS_COLUMNS
		WHERE OWNER = '%s'
		ORDER BY TABLE_NAME, CONSTRAINT_NAME, POSITION`, schemaName)

	slog.Debug("running get constraint columns query")
	constraintColumnRows, err := txn.Query(queryConstraintColumns)
	if err != nil {
		return nil, nil, nil, nil, util.FormatErrorWithQuery(err, queryConstraintColumns)
	}
	defer constraintColumnRows.Close()
	constraintColumnMap := make(map[db.ConstraintKey][]string)
	constraintTableMap := make(map[db.ConstraintKey]string)
	for constraintColumnRows.Next() {
		var tableName, constraintName string
		var columnName sql.NullString
		if err := constraintColumnRows.Scan(&tableName, &constraintName, &columnName); err != nil {
			return nil, nil, nil, nil, err
		}
		key := db.ConstraintKey{Schema: schemaName, Constraint: constraintName}
		if columnName.Valid {
			constraintColumnMap[key] = append(constraintColumnMap[key], columnName.String)
		}
		constraintTableMap[key] = tableName
	}
	if err := constraintColumnRows.Err(); err != nil {
		return nil, nil, nil, nil, util.FormatErrorWithQuery(err, queryConstraintColumns)
	}

	queryConstraints := fmt.Sprintf(`
		SELECT
			TABLE_NAME,
			CONSTRAINT_NAME,
			CONSTRAINT_TYPE,
			SEARCH_CONDITION,
			R_OWNER,
			R_CONSTRAINT_NAME
		FROM SYS.ALL_CONSTRAINTS
		WHERE OWNER = '%s'
		ORDER BY TABLE_NAME, CONSTRAINT_NAME`, schemaName)

	slog.Debug("running get constraints query")
	constraintRows, err := txn.Query(queryConstraints)
	if err != nil {
		return nil, nil, nil, nil, util.FormatErrorWithQuery(err, queryConstraints)
	}
	defer constraintRows.Close()
	indexMap := make(map[db.TableKey][]*storepb.IndexMetadata)
	checkConstraintMap := make(map[db.TableKey][]*storepb.CheckConstraintMetadata)
	foreignKeyMap := make(map[db.TableKey][]*storepb.ForeignKeyMetadata)
	isConstraint := make(map[db.IndexKey]bool)
	outerRColumnMap := make(map[db.ConstraintKey][]string)
	outerRTableMap := make(map[db.ConstraintKey]string)
	for constraintRows.Next() {
		var tableName, constraintName, constraintType string
		var searchCondition, rOwner, rConstraintName sql.NullString
		if err := constraintRows.Scan(&tableName, &constraintName, &constraintType, &searchCondition, &rOwner, &rConstraintName); err != nil {
			return nil, nil, nil, nil, err
		}
		key := db.TableKey{Schema: schemaName, Table: tableName}
		constraintKey := db.ConstraintKey{Schema: schemaName, Constraint: constraintName}
		switch constraintType {
		case "P":
			index := &storepb.IndexMetadata{
				Name:         constraintName,
				Primary:      true,
				Unique:       true,
				IsConstraint: true,
			}
			if columns, ok := constraintColumnMap[constraintKey]; ok {
				index.Expressions = columns
			}
			indexMap[key] = append(indexMap[key], index)
			isConstraint[db.IndexKey{Schema: schemaName, Table: tableName, Index: constraintName}] = true
		case "U":
			index := &storepb.IndexMetadata{
				Name:         constraintName,
				Unique:       true,
				IsConstraint: true,
			}
			if columns, ok := constraintColumnMap[constraintKey]; ok {
				index.Expressions = columns
			}
			indexMap[key] = append(indexMap[key], index)
			isConstraint[db.IndexKey{Schema: schemaName, Table: tableName, Index: constraintName}] = true
		case "C":
			// Skip system-generated check constraints (e.g., SYS_C*)
			if strings.HasPrefix(constraintName, "SYS_C") {
				continue
			}
			constraint := &storepb.CheckConstraintMetadata{
				Name: constraintName,
			}
			if searchCondition.Valid {
				constraint.Expression = searchCondition.String
			}
			checkConstraintMap[key] = append(checkConstraintMap[key], constraint)
		case "R":
			if rOwner.Valid && rConstraintName.Valid {
				foreignKey := &storepb.ForeignKeyMetadata{
					Name:    constraintName,
					Columns: constraintColumnMap[constraintKey],
				}
				if rOwner.String == schemaName {
					// Same-schema reference - don't set ReferencedSchema for portability
					rConstraintKey := db.ConstraintKey{Schema: rOwner.String, Constraint: rConstraintName.String}
					foreignKey.ReferencedTable = constraintTableMap[rConstraintKey]
					foreignKey.ReferencedColumns = constraintColumnMap[rConstraintKey]
				} else {
					// Cross-schema reference - set ReferencedSchema
					foreignKey.ReferencedSchema = rOwner.String
					foreignKey.ReferencedTable, foreignKey.ReferencedColumns, err = getOuterSchemaRColumns(txn, outerRTableMap, outerRColumnMap, rOwner.String, rConstraintName.String)
					if err != nil {
						return nil, nil, nil, nil, errors.Wrapf(err, "failed to get outer schema reference columns")
					}
				}
				foreignKeyMap[key] = append(foreignKeyMap[key], foreignKey)
			}
		default:
			// Ignore other constraint types
		}
	}
	if err := constraintRows.Err(); err != nil {
		return nil, nil, nil, nil, util.FormatErrorWithQuery(err, queryConstraints)
	}

	return indexMap, checkConstraintMap, foreignKeyMap, isConstraint, nil
}

// getIndexes gets all indices and constraints of a database.
func getIndexesAndConstraints(txn *sql.Tx, schemaName string) (map[db.TableKey][]*storepb.IndexMetadata, map[db.TableKey][]*storepb.CheckConstraintMetadata, map[db.TableKey][]*storepb.ForeignKeyMetadata, error) {
	indexMap, checkConstraintMap, foreignKeyMap, isConstraint, err := getConstraints(txn, schemaName)
	if err != nil {
		return nil, nil, nil, errors.Wrapf(err, "failed to get constraints")
	}

	// Get index comments
	indexCommentMap, err := getIndexComments(txn, schemaName)
	if err != nil {
		return nil, nil, nil, errors.Wrapf(err, "failed to get index comments")
	}

	queryColumn := fmt.Sprintf(`
		SELECT TABLE_OWNER, TABLE_NAME, INDEX_NAME, COLUMN_NAME, DESCEND
		FROM sys.all_ind_columns
		WHERE TABLE_OWNER = '%s'
		ORDER BY TABLE_NAME, INDEX_NAME, COLUMN_POSITION`, schemaName)

	slog.Debug("running get index column query")
	indexExpressionMap := make(map[db.IndexKey][]string)
	indexColumnMap := make(map[db.IndexKey][]string)
	descendingMap := make(map[db.IndexKey][]bool)
	colRows, err := txn.Query(queryColumn)
	if err != nil {
		return nil, nil, nil, util.FormatErrorWithQuery(err, queryColumn)
	}
	defer colRows.Close()
	for colRows.Next() {
		var schemaName, tableName, indexName, columnName string
		var descend sql.NullString
		if err := colRows.Scan(&schemaName, &tableName, &indexName, &columnName, &descend); err != nil {
			return nil, nil, nil, err
		}
		key := db.IndexKey{Schema: schemaName, Table: tableName, Index: indexName}

		// Clean up column name but preserve the original if it's already clean
		cleanColumnName := columnName
		if strings.HasPrefix(columnName, "\"") && strings.HasSuffix(columnName, "\"") {
			cleanColumnName = columnName[1 : len(columnName)-1]
		}

		indexColumnMap[key] = append(indexColumnMap[key], cleanColumnName)
		descendingMap[key] = append(descendingMap[key], descend.String == "DESC")
	}
	if err := colRows.Err(); err != nil {
		return nil, nil, nil, util.FormatErrorWithQuery(err, queryColumn)
	}
	if err := colRows.Close(); err != nil {
		return nil, nil, nil, errors.Wrapf(err, "failed to close rows")
	}

	queryExpression := fmt.Sprintf(`
		SELECT TABLE_OWNER, TABLE_NAME, INDEX_NAME, COLUMN_EXPRESSION
		FROM sys.all_ind_expressions
		WHERE TABLE_OWNER = '%s'
		ORDER BY TABLE_NAME, INDEX_NAME, COLUMN_POSITION`, schemaName)

	slog.Debug("running get index expression query")
	expRows, err := txn.Query(queryExpression)
	if err != nil {
		return nil, nil, nil, util.FormatErrorWithQuery(err, queryExpression)
	}
	defer expRows.Close()
	for expRows.Next() {
		var schemaName, tableName, indexName, columnExpression string
		if err := expRows.Scan(&schemaName, &tableName, &indexName, &columnExpression); err != nil {
			return nil, nil, nil, err
		}
		key := db.IndexKey{Schema: schemaName, Table: tableName, Index: indexName}
		indexExpressionMap[key] = append(indexExpressionMap[key], columnExpression)
	}
	if err := expRows.Err(); err != nil {
		return nil, nil, nil, util.FormatErrorWithQuery(err, queryExpression)
	}
	if err := expRows.Close(); err != nil {
		return nil, nil, nil, errors.Wrapf(err, "failed to close rows")
	}

	query := fmt.Sprintf(`
		SELECT OWNER, TABLE_NAME, INDEX_NAME, UNIQUENESS, INDEX_TYPE, VISIBILITY
		FROM sys.all_indexes
		WHERE OWNER = '%s'
		ORDER BY TABLE_NAME, INDEX_NAME`, schemaName)

	slog.Debug("running get index query")
	rows, err := txn.Query(query)
	if err != nil {
		return nil, nil, nil, util.FormatErrorWithQuery(err, query)
	}
	defer rows.Close()
	for rows.Next() {
		index := &storepb.IndexMetadata{}
		var schemaName, tableName, unique string
		var visibility sql.NullString
		// INDEX_TYPE is NORMAL, or FUNCTION-BASED NORMAL.
		if err := rows.Scan(&schemaName, &tableName, &index.Name, &unique, &index.Type, &visibility); err != nil {
			return nil, nil, nil, err
		}
		if isConstraint[db.IndexKey{Schema: schemaName, Table: tableName, Index: index.Name}] {
			continue
		}

		// Skip system-generated indexes for materialized views and virtual columns
		if strings.HasPrefix(index.Name, "I_SNAP$_") || strings.HasPrefix(index.Name, "SYS_") {
			continue
		}

		index.Unique = unique == "UNIQUE"
		indexKey := db.IndexKey{Schema: schemaName, Table: tableName, Index: index.Name}
		// Handle index expressions - preserve Oracle's classification
		columns := indexColumnMap[indexKey]
		expressions := indexExpressionMap[indexKey]

		// For function-based indexes (including those with DESC columns)
		if len(expressions) > 0 {
			// Combine regular columns with expressions
			// Oracle reports ASC columns in ALL_IND_COLUMNS and DESC columns as expressions
			var combinedExpressions []string
			exprIdx := 0

			for i := 0; i < len(columns); i++ {
				if strings.HasPrefix(columns[i], "SYS_NC") {
					// This is a virtual column for a DESC column, use the expression
					if exprIdx < len(expressions) {
						combinedExpressions = append(combinedExpressions, expressions[exprIdx])
						exprIdx++
					}
				} else {
					// Regular column
					combinedExpressions = append(combinedExpressions, fmt.Sprintf(`"%s"`, columns[i]))
				}
			}

			index.Expressions = combinedExpressions
		} else {
			// Simple column index without expressions
			// Skip indexes that reference only system-generated virtual columns
			hasOnlyVirtualColumns := true
			for _, col := range columns {
				if !strings.HasPrefix(col, "SYS_NC") {
					hasOnlyVirtualColumns = false
					break
				}
			}
			if hasOnlyVirtualColumns {
				continue
			}

			// For column-based indexes, quote the column names to match expected format
			quotedColumns := make([]string, len(columns))
			for i, col := range columns {
				quotedColumns[i] = fmt.Sprintf(`"%s"`, col)
			}
			index.Expressions = quotedColumns
		}
		// Note: Keep index.Type as reported by Oracle (don't modify it)

		// Set descending flags for all indexes (both function-based and normal)
		if desc, ok := descendingMap[indexKey]; ok && len(desc) > 0 {
			// Always set descending array if we have column information,
			// even if all columns are ascending (to maintain consistency)
			index.Descending = desc
		}
		index.Visible = true
		if visibility.Valid && visibility.String == "INVISIBLE" {
			index.Visible = false
		}

		// Add index comment if available
		if comment, ok := indexCommentMap[indexKey]; ok {
			index.Comment = comment
		}

		key := db.TableKey{Schema: schemaName, Table: tableName}
		indexMap[key] = append(indexMap[key], index)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, nil, util.FormatErrorWithQuery(err, query)
	}
	if err := rows.Close(); err != nil {
		return nil, nil, nil, errors.Wrapf(err, "failed to close rows")
	}

	return indexMap, checkConstraintMap, foreignKeyMap, nil
}

// getViews gets all views of a database.
func getViews(txn *sql.Tx, schemaName string, columnMap map[db.TableKey][]*storepb.ColumnMetadata, triggerMap map[db.TableKey][]*storepb.TriggerMetadata) (map[string][]*storepb.ViewMetadata, error) {
	viewMap := make(map[string][]*storepb.ViewMetadata)

	// Get view comments
	viewCommentMap, err := getViewComments(txn, schemaName)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get view comments")
	}

	query := fmt.Sprintf(`
		SELECT OWNER, VIEW_NAME, TEXT
		FROM sys.all_views
		WHERE OWNER = '%s'
		ORDER BY view_name
	`, schemaName)

	slog.Debug("running get view query")
	rows, err := txn.Query(query)
	if err != nil {
		return nil, util.FormatErrorWithQuery(err, query)
	}
	defer rows.Close()
	for rows.Next() {
		view := &storepb.ViewMetadata{}
		var schemaName string
		if err := rows.Scan(&schemaName, &view.Name, &view.Definition); err != nil {
			return nil, err
		}
		key := db.TableKey{Schema: schemaName, Table: view.Name}
		view.Columns = columnMap[key]
		view.Triggers = triggerMap[key]
		if comment, ok := viewCommentMap[key]; ok {
			view.Comment = comment
		}

		viewMap[schemaName] = append(viewMap[schemaName], view)
	}
	if err := rows.Err(); err != nil {
		return nil, util.FormatErrorWithQuery(err, query)
	}
	if err := rows.Close(); err != nil {
		return nil, errors.Wrapf(err, "failed to close rows")
	}

	// Fetch all view dependencies in a single query to avoid N+1 problem
	viewDependenciesMap, err := getAllViewDependencies(txn, schemaName)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get view dependencies")
	}

	// Populate view dependencies from the map
	for schemaName, list := range viewMap {
		for _, view := range list {
			key := db.TableKey{Schema: schemaName, Table: view.Name}
			if dependencies, ok := viewDependenciesMap[key]; ok {
				view.DependencyColumns = dependencies
			}
		}
	}

	return viewMap, nil
}

// getAllViewDependencies gets all view dependencies for a schema in a single query.
func getAllViewDependencies(txn *sql.Tx, schemaName string) (map[db.TableKey][]*storepb.DependencyColumn, error) {
	dependenciesMap := make(map[db.TableKey][]*storepb.DependencyColumn)

	query := fmt.Sprintf(`
		SELECT 
			NAME as view_name,
			REFERENCED_OWNER as source_schema,
			REFERENCED_NAME as source_table,
			'*' as column_name
		FROM ALL_DEPENDENCIES
		WHERE OWNER = '%s'
			AND TYPE = 'VIEW'
			AND REFERENCED_TYPE IN ('TABLE', 'VIEW')
		ORDER BY NAME, REFERENCED_OWNER, REFERENCED_NAME
	`, schemaName)

	rows, err := txn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var viewName string
		dependencyColumn := &storepb.DependencyColumn{}
		if err := rows.Scan(&viewName, &dependencyColumn.Schema, &dependencyColumn.Table, &dependencyColumn.Column); err != nil {
			return nil, err
		}

		key := db.TableKey{Schema: schemaName, Table: viewName}
		dependenciesMap[key] = append(dependenciesMap[key], dependencyColumn)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return dependenciesMap, nil
}

// getAllMaterializedViewDependencies gets all materialized view dependencies for a schema in a single query.
func getAllMaterializedViewDependencies(txn *sql.Tx, schemaName string) (map[db.TableKey][]*storepb.DependencyColumn, error) {
	dependenciesMap := make(map[db.TableKey][]*storepb.DependencyColumn)

	query := fmt.Sprintf(`
		SELECT 
			NAME as mview_name,
			REFERENCED_OWNER as source_schema,
			REFERENCED_NAME as source_table,
			'*' as column_name
		FROM ALL_DEPENDENCIES
		WHERE OWNER = '%s'
			AND TYPE = 'MATERIALIZED VIEW'
			AND REFERENCED_TYPE IN ('TABLE', 'VIEW')
		ORDER BY NAME, REFERENCED_OWNER, REFERENCED_NAME
	`, schemaName)

	rows, err := txn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var mviewName string
		dependencyColumn := &storepb.DependencyColumn{}
		if err := rows.Scan(&mviewName, &dependencyColumn.Schema, &dependencyColumn.Table, &dependencyColumn.Column); err != nil {
			return nil, err
		}

		key := db.TableKey{Schema: schemaName, Table: mviewName}
		dependenciesMap[key] = append(dependenciesMap[key], dependencyColumn)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return dependenciesMap, nil
}

// getMaterializedViews gets all materialized views of a database.
func getMaterializedViews(txn *sql.Tx, schemaName string, _ map[db.TableKey][]*storepb.ColumnMetadata) ([]*storepb.MaterializedViewMetadata, error) {
	var materializedViews []*storepb.MaterializedViewMetadata

	// Get materialized view comments
	materializedViewCommentMap, err := getMaterializedViewComments(txn, schemaName)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get materialized view comments")
	}

	query := fmt.Sprintf(`
		SELECT OWNER, MVIEW_NAME, QUERY
		FROM sys.all_mviews
		WHERE OWNER = '%s'
		ORDER BY MVIEW_NAME
	`, schemaName)

	slog.Debug("running get materialized views query")
	rows, err := txn.Query(query)
	if err != nil {
		return nil, util.FormatErrorWithQuery(err, query)
	}
	defer rows.Close()

	for rows.Next() {
		materializedView := &storepb.MaterializedViewMetadata{}
		var schemaName string
		if err := rows.Scan(&schemaName, &materializedView.Name, &materializedView.Definition); err != nil {
			return nil, err
		}

		// Ensure the definition ends with a newline to match expected format
		if materializedView.Definition != "" && !strings.HasSuffix(materializedView.Definition, "\n") {
			materializedView.Definition += "\n"
		}

		// Add comment if available
		key := db.TableKey{Schema: schemaName, Table: materializedView.Name}
		if comment, ok := materializedViewCommentMap[key]; ok {
			materializedView.Comment = comment
		}

		materializedViews = append(materializedViews, materializedView)
	}
	if err := rows.Err(); err != nil {
		return nil, util.FormatErrorWithQuery(err, query)
	}
	if err := rows.Close(); err != nil {
		return nil, errors.Wrapf(err, "failed to close rows")
	}

	// Fetch all materialized view dependencies in a single query to avoid N+1 problem
	materializedViewDependenciesMap, err := getAllMaterializedViewDependencies(txn, schemaName)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get materialized view dependencies")
	}

	// Populate dependencies from the map
	for _, materializedView := range materializedViews {
		key := db.TableKey{Schema: schemaName, Table: materializedView.Name}
		if dependencies, ok := materializedViewDependenciesMap[key]; ok {
			materializedView.DependencyColumns = dependencies
		}
	}

	return materializedViews, nil
}

// getSequences gets all sequences of a database.
func getSequences(txn *sql.Tx, schemaName string) ([]*storepb.SequenceMetadata, error) {
	var sequences []*storepb.SequenceMetadata

	// Get sequence comments
	sequenceCommentMap, err := getSequenceComments(txn, schemaName)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get sequence comments")
	}

	query := fmt.Sprintf(`
		SELECT SEQUENCE_NAME FROM ALL_SEQUENCES
		WHERE SEQUENCE_OWNER = '%s'
		ORDER BY SEQUENCE_NAME`, schemaName)

	slog.Debug("running get sequences query")
	rows, err := txn.Query(query)
	if err != nil {
		return nil, util.FormatErrorWithQuery(err, query)
	}
	defer rows.Close()

	for rows.Next() {
		seq := &storepb.SequenceMetadata{}
		if err := rows.Scan(&seq.Name); err != nil {
			return nil, err
		}
		// Skip system-generated sequences (e.g., ISEQ$$_* for IDENTITY columns)
		if strings.HasPrefix(seq.Name, "ISEQ$$_") {
			continue
		}
		key := db.TableKey{Schema: schemaName, Table: seq.Name}
		if comment, ok := sequenceCommentMap[key]; ok {
			seq.Comment = comment
		}
		sequences = append(sequences, seq)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}

	return sequences, nil
}

func getRoutines(txn *sql.Tx, schemaName string) ([]*storepb.FunctionMetadata, []*storepb.ProcedureMetadata, []*storepb.PackageMetadata, error) {
	var functions []*storepb.FunctionMetadata
	var procedures []*storepb.ProcedureMetadata
	var packages []*storepb.PackageMetadata

	// Get function comments
	functionCommentMap, err := getFunctionComments(txn, schemaName)
	if err != nil {
		return nil, nil, nil, errors.Wrapf(err, "failed to get function comments")
	}

	query := fmt.Sprintf(`
		SELECT
			NAME,
			TYPE,
			TEXT
		FROM ALL_SOURCE
		WHERE
			TYPE IN ('FUNCTION', 'PROCEDURE', 'PACKAGE')
			AND
			OWNER = '%s'
		ORDER BY NAME, TYPE, LINE`, schemaName)

	slog.Debug("running get routines query")
	rows, err := txn.Query(query)
	if err != nil {
		return nil, nil, nil, util.FormatErrorWithQuery(err, query)
	}
	defer rows.Close()

	var currentName, currentType string
	var defText []string
	for rows.Next() {
		var name, t, def string
		if err := rows.Scan(&name, &t, &def); err != nil {
			return nil, nil, nil, err
		}
		if name == currentName && t == currentType {
			defText = append(defText, def)
		} else {
			switch currentType {
			case "FUNCTION":
				function := &storepb.FunctionMetadata{
					Name:       currentName,
					Definition: strings.Join(defText, ""),
				}
				key := db.TableKey{Schema: schemaName, Table: currentName}
				if comment, ok := functionCommentMap[key]; ok {
					function.Comment = comment
				}
				functions = append(functions, function)
			case "PROCEDURE":
				procedures = append(procedures, &storepb.ProcedureMetadata{
					Name:       currentName,
					Definition: strings.Join(defText, ""),
				})
			case "PACKAGE":
				packages = append(packages, &storepb.PackageMetadata{
					Name:       currentName,
					Definition: strings.Join(defText, ""),
				})
			default:
				// Ignore other types
			}
			currentName = name
			currentType = t
			defText = []string{def}
		}
	}
	switch currentType {
	case "FUNCTION":
		function := &storepb.FunctionMetadata{
			Name:       currentName,
			Definition: strings.Join(defText, ""),
		}
		key := db.TableKey{Schema: schemaName, Table: currentName}
		if comment, ok := functionCommentMap[key]; ok {
			function.Comment = comment
		}
		functions = append(functions, function)
	case "PROCEDURE":
		procedures = append(procedures, &storepb.ProcedureMetadata{
			Name:       currentName,
			Definition: strings.Join(defText, ""),
		})
	case "PACKAGE":
		packages = append(packages, &storepb.PackageMetadata{
			Name:       currentName,
			Definition: strings.Join(defText, ""),
		})
	default:
		// Ignore other types
	}
	if err := rows.Err(); err != nil {
		return nil, nil, nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, nil, nil, err
	}

	return functions, procedures, packages, nil
}

// cleanDefaultExpression cleans up Oracle default expressions to make them more portable
// and fix common syntax issues like schema-qualified sequence NEXTVAL calls
func cleanDefaultExpression(defaultExpr, currentSchema string) string {
	if defaultExpr == "" {
		return defaultExpr
	}

	// Remove leading/trailing whitespace
	cleaned := strings.TrimSpace(defaultExpr)

	// Fix schema-qualified sequence NEXTVAL calls
	// Pattern: "SCHEMA"."SEQUENCE_NAME"."NEXTVAL" -> "SEQUENCE_NAME".NEXTVAL
	// Keep sequence name quoted for compatibility
	if strings.Contains(cleaned, ".\"NEXTVAL\"") {
		// Use regex to match quoted schema and sequence names followed by quoted NEXTVAL
		re := regexp.MustCompile(`"` + regexp.QuoteMeta(currentSchema) + `"\."([^"]+)"\."NEXTVAL"`)
		cleaned = re.ReplaceAllString(cleaned, `"$1".NEXTVAL`)
	}

	// Also handle unquoted NEXTVAL (less common)
	if strings.Contains(cleaned, ".NEXTVAL") {
		re := regexp.MustCompile(`"` + regexp.QuoteMeta(currentSchema) + `"\."([^"]+)"\.NEXTVAL`)
		cleaned = re.ReplaceAllString(cleaned, `"$1".NEXTVAL`)
	}

	return cleaned
}

// getViewComments gets comments for views from Oracle system views
func getViewComments(txn *sql.Tx, schemaName string) (map[db.TableKey]string, error) {
	viewCommentMap := make(map[db.TableKey]string)

	// Oracle stores view comments in ALL_TAB_COMMENTS where TABLE_TYPE = 'VIEW'
	query := ""
	if schemaName == "" {
		query = fmt.Sprintf(`
		SELECT OWNER, TABLE_NAME, COMMENTS
		FROM all_tab_comments
		WHERE OWNER NOT IN (%s) AND OWNER NOT LIKE 'APEX_%%' AND COMMENTS IS NOT NULL
		AND TABLE_TYPE = 'VIEW'
		ORDER BY OWNER, TABLE_NAME`, systemSchema)
	} else {
		query = fmt.Sprintf(`
		SELECT OWNER, TABLE_NAME, COMMENTS
		FROM all_tab_comments
		WHERE OWNER = '%s' AND COMMENTS IS NOT NULL
		AND TABLE_TYPE = 'VIEW'
		ORDER BY TABLE_NAME`, schemaName)
	}

	slog.Debug("running get view comments query")
	rows, err := txn.Query(query)
	if err != nil {
		// If the query fails (e.g., TABLE_TYPE = 'VIEW' not supported), return empty map
		slog.Debug("view comments query failed, returning empty map", slog.String("error", err.Error()))
		return viewCommentMap, nil
	}
	defer rows.Close()

	for rows.Next() {
		var schemaName, viewName, comment string
		if err := rows.Scan(&schemaName, &viewName, &comment); err != nil {
			return nil, err
		}
		key := db.TableKey{Schema: schemaName, Table: viewName}
		viewCommentMap[key] = comment
	}
	if err := rows.Err(); err != nil {
		return nil, util.FormatErrorWithQuery(err, query)
	}
	if err := rows.Close(); err != nil {
		return nil, errors.Wrapf(err, "failed to close rows")
	}

	return viewCommentMap, nil
}

// getSequenceComments gets comments for sequences from Oracle system views
func getSequenceComments(txn *sql.Tx, schemaName string) (map[db.TableKey]string, error) {
	sequenceCommentMap := make(map[db.TableKey]string)

	// Oracle doesn't have a direct sequence comments view, but we can try user_tab_comments
	// where TABLE_TYPE might be 'SEQUENCE' in some Oracle versions
	query := ""
	if schemaName == "" {
		query = fmt.Sprintf(`
		SELECT OWNER, TABLE_NAME, COMMENTS
		FROM all_tab_comments
		WHERE OWNER NOT IN (%s) AND OWNER NOT LIKE 'APEX_%%' AND COMMENTS IS NOT NULL
		AND TABLE_TYPE = 'SEQUENCE'
		ORDER BY OWNER, TABLE_NAME`, systemSchema)
	} else {
		query = fmt.Sprintf(`
		SELECT OWNER, TABLE_NAME, COMMENTS
		FROM all_tab_comments
		WHERE OWNER = '%s' AND COMMENTS IS NOT NULL
		AND TABLE_TYPE = 'SEQUENCE'
		ORDER BY TABLE_NAME`, schemaName)
	}

	slog.Debug("running get sequence comments query")
	rows, err := txn.Query(query)
	if err != nil {
		// If the query fails (e.g., TABLE_TYPE = 'SEQUENCE' not supported), return empty map
		slog.Debug("sequence comments query failed, returning empty map", slog.String("error", err.Error()))
		return sequenceCommentMap, nil
	}
	defer rows.Close()

	for rows.Next() {
		var schemaName, sequenceName, comment string
		if err := rows.Scan(&schemaName, &sequenceName, &comment); err != nil {
			return nil, err
		}
		key := db.TableKey{Schema: schemaName, Table: sequenceName}
		sequenceCommentMap[key] = comment
	}
	if err := rows.Err(); err != nil {
		return nil, util.FormatErrorWithQuery(err, query)
	}
	if err := rows.Close(); err != nil {
		return nil, errors.Wrapf(err, "failed to close rows")
	}

	return sequenceCommentMap, nil
}

// getFunctionComments gets comments for functions and procedures from Oracle system views
func getFunctionComments(txn *sql.Tx, schemaName string) (map[db.TableKey]string, error) {
	functionCommentMap := make(map[db.TableKey]string)

	// Oracle stores function/procedure comments in ALL_OBJECTS or USER_OBJECTS
	// However, Oracle doesn't have a standard system view for function/procedure comments
	// We can try to get them from ALL_TAB_COMMENTS where TABLE_TYPE might be 'FUNCTION' or 'PROCEDURE'
	query := ""
	if schemaName == "" {
		query = fmt.Sprintf(`
		SELECT OWNER, TABLE_NAME, COMMENTS
		FROM all_tab_comments
		WHERE OWNER NOT IN (%s) AND OWNER NOT LIKE 'APEX_%%' AND COMMENTS IS NOT NULL
		AND TABLE_TYPE IN ('FUNCTION', 'PROCEDURE')
		ORDER BY OWNER, TABLE_NAME`, systemSchema)
	} else {
		query = fmt.Sprintf(`
		SELECT OWNER, TABLE_NAME, COMMENTS
		FROM all_tab_comments
		WHERE OWNER = '%s' AND COMMENTS IS NOT NULL
		AND TABLE_TYPE IN ('FUNCTION', 'PROCEDURE')
		ORDER BY TABLE_NAME`, schemaName)
	}

	slog.Debug("running get function comments query")
	rows, err := txn.Query(query)
	if err != nil {
		// If the query fails (e.g., TABLE_TYPE for functions not supported), return empty map
		slog.Debug("function comments query failed, returning empty map", slog.String("error", err.Error()))
		return functionCommentMap, nil
	}
	defer rows.Close()

	for rows.Next() {
		var schemaName, functionName, comment string
		if err := rows.Scan(&schemaName, &functionName, &comment); err != nil {
			return nil, err
		}
		key := db.TableKey{Schema: schemaName, Table: functionName}
		functionCommentMap[key] = comment
	}
	if err := rows.Err(); err != nil {
		return nil, util.FormatErrorWithQuery(err, query)
	}
	if err := rows.Close(); err != nil {
		return nil, errors.Wrapf(err, "failed to close rows")
	}

	return functionCommentMap, nil
}

// getIndexComments gets comments for indexes from Oracle system views
func getIndexComments(_ *sql.Tx, schemaName string) (map[db.IndexKey]string, error) {
	indexCommentMap := make(map[db.IndexKey]string)

	// Oracle doesn't have a dedicated index comments view in all versions
	// We need to join ALL_INDEXES with ALL_TAB_COMMENTS to get index comments
	// However, Oracle stores index comments differently depending on the version
	// For now, return empty map since Oracle index comments are not consistently available
	slog.Debug("Oracle index comments not implemented - returning empty map", "schema", schemaName)
	return indexCommentMap, nil
}

// getMaterializedViewComments gets comments for materialized views from Oracle system views
func getMaterializedViewComments(txn *sql.Tx, schemaName string) (map[db.TableKey]string, error) {
	materializedViewCommentMap := make(map[db.TableKey]string)

	query := ""
	if schemaName == "" {
		query = fmt.Sprintf(`
		SELECT OWNER, MVIEW_NAME, COMMENTS
		FROM all_mview_comments
		WHERE OWNER NOT IN (%s) AND OWNER NOT LIKE 'APEX_%%' AND COMMENTS IS NOT NULL
		ORDER BY OWNER, MVIEW_NAME`, systemSchema)
	} else {
		query = fmt.Sprintf(`
		SELECT OWNER, MVIEW_NAME, COMMENTS
		FROM all_mview_comments
		WHERE OWNER = '%s' AND COMMENTS IS NOT NULL
		ORDER BY MVIEW_NAME`, schemaName)
	}

	slog.Debug("running get materialized view comments query")
	rows, err := txn.Query(query)
	if err != nil {
		// If the view doesn't exist, return empty map
		slog.Debug("materialized view comments query failed, returning empty map", slog.String("error", err.Error()))
		return materializedViewCommentMap, nil
	}
	defer rows.Close()

	for rows.Next() {
		var schemaName, mvName, comment string
		if err := rows.Scan(&schemaName, &mvName, &comment); err != nil {
			return nil, err
		}
		key := db.TableKey{Schema: schemaName, Table: mvName}
		materializedViewCommentMap[key] = comment
	}
	if err := rows.Err(); err != nil {
		return nil, util.FormatErrorWithQuery(err, query)
	}
	if err := rows.Close(); err != nil {
		return nil, errors.Wrapf(err, "failed to close rows")
	}

	return materializedViewCommentMap, nil
}

// getTriggerComments gets comments for triggers from Oracle system views
func getTriggerComments(txn *sql.Tx, schemaName string) (map[db.TableKey]string, error) {
	triggerCommentMap := make(map[db.TableKey]string)

	// Oracle doesn't have a standard trigger comments view
	// Trigger comments might be stored in ALL_TAB_COMMENTS with TABLE_TYPE = 'TRIGGER'
	// or in some Oracle versions, they might not be available at all
	query := ""
	if schemaName == "" {
		query = fmt.Sprintf(`
		SELECT OWNER, TABLE_NAME, COMMENTS
		FROM all_tab_comments
		WHERE OWNER NOT IN (%s) AND OWNER NOT LIKE 'APEX_%%' AND COMMENTS IS NOT NULL
		AND TABLE_TYPE = 'TRIGGER'
		ORDER BY OWNER, TABLE_NAME`, systemSchema)
	} else {
		query = fmt.Sprintf(`
		SELECT OWNER, TABLE_NAME, COMMENTS
		FROM all_tab_comments
		WHERE OWNER = '%s' AND COMMENTS IS NOT NULL
		AND TABLE_TYPE = 'TRIGGER'
		ORDER BY TABLE_NAME`, schemaName)
	}

	slog.Debug("running get trigger comments query")
	rows, err := txn.Query(query)
	if err != nil {
		// If the query fails (e.g., TABLE_TYPE = 'TRIGGER' not supported), return empty map
		slog.Debug("trigger comments query failed, returning empty map", slog.String("error", err.Error()))
		return triggerCommentMap, nil
	}
	defer rows.Close()

	for rows.Next() {
		var schemaName, triggerName, comment string
		if err := rows.Scan(&schemaName, &triggerName, &comment); err != nil {
			return nil, err
		}
		key := db.TableKey{Schema: schemaName, Table: triggerName}
		triggerCommentMap[key] = comment
	}
	if err := rows.Err(); err != nil {
		return nil, util.FormatErrorWithQuery(err, query)
	}
	if err := rows.Close(); err != nil {
		return nil, errors.Wrapf(err, "failed to close rows")
	}

	return triggerCommentMap, nil
}
