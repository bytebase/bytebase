package oracle

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

const systemSchema = "'ANONYMOUS','APPQOSSYS','AUDSYS','CTXSYS','DBSFWUSER','DBSNMP','DGPDB_INT','DIP','DVF','DVSYS','GGSYS','GSMADMIN_INTERNAL','GSMCATUSER','GSMROOTUSER','GSMUSER','LBACSYS','MDDATA','MDSYS','OPS$ORACLE','ORACLE_OCM','OUTLN','REMOTE_SCHEDULER_AGENT','SYS','SYS$UMF','SYSBACKUP','SYSDG','SYSKM','SYSRAC','SYSTEM','XDB','XS$NULL','XS$$NULL','FLOWS_FILES','HR','MDSYS','EXFSYS','MGMT_VIEW','OLAPSYS','ORDDATA','ORDPLUGINS','ORDSYS','OWBSYS','OWBSYS_AUDIT','SCOTT','SI_INFORMTN_SCHEMA','SPATIAL_CSW_ADMIN_USR','SPATIAL_WFS_ADMIN_USR','SYSMAN','WMSYS','OJVMSYS'"

var (
	semVersionRegex       = regexp.MustCompile(`[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+`)
	canonicalVersionRegex = regexp.MustCompile(`[0-9][0-9][a-z]`)
)

// SyncInstance syncs the instance.
func (driver *Driver) SyncInstance(ctx context.Context) (*db.InstanceMetadata, error) {
	var fullVersion string
	queryVersion := "SELECT BANNER FROM v$version WHERE banner LIKE 'Oracle%'"
	if err := driver.db.QueryRowContext(ctx, queryVersion).Scan(&fullVersion); err != nil {
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

	txn, err := driver.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer txn.Rollback()

	schemas, err := getSchemas(txn)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get schemas from database %q", driver.databaseName)
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
func (driver *Driver) SyncDBSchema(ctx context.Context) (*storepb.DatabaseSchemaMetadata, error) {
	txn, err := driver.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer txn.Rollback()

	columnMap, err := getTableColumns(txn, driver.databaseName)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get table columns from database %q", driver.databaseName)
	}
	tableMap, err := getTables(txn, driver.databaseName, columnMap)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get tables from database %q", driver.databaseName)
	}
	viewMap, err := getViews(txn, driver.databaseName, columnMap)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get views from database %q", driver.databaseName)
	}
	sequences, err := getSequences(txn, driver.databaseName)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get sequences from database %q", driver.databaseName)
	}
	dbLinks, err := getDBLinks(txn)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get db links from database %q", driver.databaseName)
	}
	functions, procedures, packages, err := getRoutines(txn, driver.databaseName)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get routines from database %q", driver.databaseName)
	}

	if err := txn.Commit(); err != nil {
		return nil, err
	}

	databaseMetadata := &storepb.DatabaseSchemaMetadata{
		Name:            driver.databaseName,
		ServiceName:     driver.serviceName,
		LinkedDatabases: dbLinks,
	}
	databaseMetadata.Schemas = append(databaseMetadata.Schemas, &storepb.SchemaMetadata{
		Name:       "",
		Tables:     tableMap[driver.databaseName],
		Views:      viewMap[driver.databaseName],
		Sequences:  sequences,
		Functions:  functions,
		Procedures: procedures,
		Packages:   packages,
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

// getTables gets all tables of a database.
func getTables(txn *sql.Tx, schemaName string, columnMap map[db.TableKey][]*storepb.ColumnMetadata) (map[string][]*storepb.TableMetadata, error) {
	indexMap, err := getIndexes(txn, schemaName)
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
	// TODO(d): foreign keys.
	tableMap := make(map[string][]*storepb.TableMetadata)
	query := ""
	if schemaName == "" {
		query = fmt.Sprintf(`
		SELECT OWNER, TABLE_NAME, NUM_ROWS
		FROM all_tables
		WHERE OWNER NOT IN (%s) AND OWNER NOT LIKE 'APEX_%%'
		ORDER BY OWNER, TABLE_NAME`, systemSchema)
	} else {
		query = fmt.Sprintf(`
		SELECT OWNER, TABLE_NAME, NUM_ROWS
		FROM all_tables
		WHERE OWNER = '%s'
		ORDER BY TABLE_NAME`, schemaName)
	}

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
func getTableColumns(txn *sql.Tx, schemaName string) (map[db.TableKey][]*storepb.ColumnMetadata, error) {
	columnsMap := make(map[db.TableKey][]*storepb.ColumnMetadata)

	// https://github.com/bytebase/bytebase/issues/6663
	// Invisible columns don't have column ID so that we need to filter out them.
	query := ""
	if schemaName == "" {
		query = fmt.Sprintf(`
		SELECT
			OWNER,
			TABLE_NAME,
			COLUMN_NAME,
			DATA_TYPE,
			COLUMN_ID,
			DATA_DEFAULT,
			NULLABLE
		FROM sys.all_tab_columns
		WHERE OWNER NOT IN (%s) AND OWNER NOT LIKE 'APEX_%%' AND COLUMN_ID IS NOT NULL
		ORDER BY OWNER, TABLE_NAME, COLUMN_ID`, systemSchema)
	} else {
		query = fmt.Sprintf(`
		SELECT
			OWNER,
			TABLE_NAME,
			COLUMN_NAME,
			DATA_TYPE,
			COLUMN_ID,
			DATA_DEFAULT,
			NULLABLE
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
		var defaultStr sql.NullString
		if err := rows.Scan(&schemaName, &tableName, &column.Name, &column.Type, &column.Position, &defaultStr, &nullable); err != nil {
			return nil, err
		}
		if defaultStr.Valid {
			// TODO: use correct default type
			column.DefaultValue = &storepb.ColumnMetadata_DefaultExpression{DefaultExpression: defaultStr.String}
		}
		isNullBool, err := util.ConvertYesNo(nullable)
		if err != nil {
			return nil, err
		}
		column.Nullable = isNullBool
		// TODO(d): add collation.

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

// getIndexes gets all indices of a database.
func getIndexes(txn *sql.Tx, schemaName string) (map[db.TableKey][]*storepb.IndexMetadata, error) {
	indexMap := make(map[db.TableKey][]*storepb.IndexMetadata)

	expressionsMap := make(map[db.IndexKey][]string)
	queryColumn := ""
	if schemaName == "" {
		queryColumn = fmt.Sprintf(`
		SELECT TABLE_OWNER, TABLE_NAME, INDEX_NAME, COLUMN_NAME
		FROM sys.all_ind_columns
		WHERE TABLE_OWNER NOT IN (%s) AND TABLE_OWNER NOT LIKE 'APEX_%%'
		ORDER BY TABLE_OWNER, TABLE_NAME, INDEX_NAME, COLUMN_POSITION`, systemSchema)
	} else {
		queryColumn = fmt.Sprintf(`
		SELECT TABLE_OWNER, TABLE_NAME, INDEX_NAME, COLUMN_NAME
		FROM sys.all_ind_columns
		WHERE TABLE_OWNER = '%s'
		ORDER BY TABLE_NAME, INDEX_NAME, COLUMN_POSITION`, schemaName)
	}
	slog.Debug("running get index column query")
	colRows, err := txn.Query(queryColumn)
	if err != nil {
		return nil, util.FormatErrorWithQuery(err, queryColumn)
	}
	defer colRows.Close()
	for colRows.Next() {
		var schemaName, tableName, indexName, columnName string
		if err := colRows.Scan(&schemaName, &tableName, &indexName, &columnName); err != nil {
			return nil, err
		}
		key := db.IndexKey{Schema: schemaName, Table: tableName, Index: indexName}
		expressionsMap[key] = append(expressionsMap[key], columnName)
	}
	if err := colRows.Err(); err != nil {
		return nil, util.FormatErrorWithQuery(err, queryColumn)
	}
	if err := colRows.Close(); err != nil {
		return nil, errors.Wrapf(err, "failed to close rows")
	}

	queryExpression := ""
	if schemaName == "" {
		queryExpression = fmt.Sprintf(`
		SELECT TABLE_OWNER, TABLE_NAME, INDEX_NAME, COLUMN_EXPRESSION, COLUMN_POSITION
		FROM sys.all_ind_expressions
		WHERE TABLE_OWNER NOT IN (%s) AND TABLE_OWNER NOT LIKE 'APEX_%%'
		ORDER BY TABLE_OWNER, TABLE_NAME, INDEX_NAME, COLUMN_POSITION`, systemSchema)
	} else {
		queryExpression = fmt.Sprintf(`
		SELECT TABLE_OWNER, TABLE_NAME, INDEX_NAME, COLUMN_EXPRESSION, COLUMN_POSITION
		FROM sys.all_ind_expressions
		WHERE TABLE_OWNER = '%s'
		ORDER BY TABLE_NAME, INDEX_NAME, COLUMN_POSITION`, schemaName)
	}
	slog.Debug("running get index expression query")
	expRows, err := txn.Query(queryExpression)
	if err != nil {
		return nil, util.FormatErrorWithQuery(err, queryExpression)
	}
	defer expRows.Close()
	for expRows.Next() {
		var schemaName, tableName, indexName, columnExpression string
		var position int
		if err := expRows.Scan(&schemaName, &tableName, &indexName, &columnExpression, &position); err != nil {
			return nil, err
		}
		key := db.IndexKey{Schema: schemaName, Table: tableName, Index: indexName}
		// Position starts from 1.
		expIndex := position - 1
		if expIndex >= len(expressionsMap[key]) {
			return nil, errors.Errorf("expression %q position %v out of range for index %q.%q.%q", columnExpression, position, schemaName, tableName, indexName)
		}
		expressionsMap[key][expIndex] = columnExpression
	}
	if err := expRows.Err(); err != nil {
		return nil, util.FormatErrorWithQuery(err, queryExpression)
	}
	if err := expRows.Close(); err != nil {
		return nil, errors.Wrapf(err, "failed to close rows")
	}

	pkMap := make(map[db.TableKey]string)
	queryPK := ""
	if schemaName == "" {
		queryPK = fmt.Sprintf(`
		SELECT OWNER, TABLE_NAME, CONSTRAINT_NAME
		FROM sys.all_constraints
		WHERE OWNER NOT IN (%s) AND OWNER NOT LIKE 'APEX_%%' AND CONSTRAINT_TYPE = 'P'
		ORDER BY OWNER, TABLE_NAME, CONSTRAINT_NAME`, systemSchema)
	} else {
		queryPK = fmt.Sprintf(`
		SELECT OWNER, TABLE_NAME, CONSTRAINT_NAME
		FROM sys.all_constraints
		WHERE OWNER = '%s' AND CONSTRAINT_TYPE = 'P'
		ORDER BY TABLE_NAME, CONSTRAINT_NAME`, schemaName)
	}
	slog.Debug("running get primary key query")
	pkRows, err := txn.Query(queryPK)
	if err != nil {
		return nil, util.FormatErrorWithQuery(err, queryPK)
	}
	defer pkRows.Close()
	for pkRows.Next() {
		var schemaName, tableName, pkName string
		if err := pkRows.Scan(&schemaName, &tableName, &pkName); err != nil {
			return nil, err
		}
		key := db.TableKey{Schema: schemaName, Table: tableName}
		pkMap[key] = pkName
	}
	if err := pkRows.Err(); err != nil {
		return nil, util.FormatErrorWithQuery(err, queryPK)
	}

	query := ""
	if schemaName == "" {
		query = fmt.Sprintf(`
		SELECT OWNER, TABLE_NAME, INDEX_NAME, UNIQUENESS, INDEX_TYPE
		FROM sys.all_indexes
		WHERE OWNER NOT IN (%s) AND OWNER NOT LIKE 'APEX_%%'
		ORDER BY OWNER, TABLE_NAME, INDEX_NAME`, systemSchema)
	} else {
		query = fmt.Sprintf(`
		SELECT OWNER, TABLE_NAME, INDEX_NAME, UNIQUENESS, INDEX_TYPE
		FROM sys.all_indexes
		WHERE OWNER = '%s'
		ORDER BY TABLE_NAME, INDEX_NAME`, schemaName)
	}
	slog.Debug("running get index query")
	rows, err := txn.Query(query)
	if err != nil {
		return nil, util.FormatErrorWithQuery(err, query)
	}
	defer rows.Close()
	for rows.Next() {
		index := &storepb.IndexMetadata{}
		var schemaName, tableName, unique string
		// INDEX_TYPE is NORMAL, or FUNCTION-BASED NORMAL.
		if err := rows.Scan(&schemaName, &tableName, &index.Name, &unique, &index.Type); err != nil {
			return nil, err
		}

		index.Unique = unique == "UNIQUE"
		indexKey := db.IndexKey{Schema: schemaName, Table: tableName, Index: index.Name}
		index.Expressions = expressionsMap[indexKey]

		if pkName, ok := pkMap[db.TableKey{Schema: schemaName, Table: tableName}]; ok && pkName == index.Name {
			index.Primary = true
		}

		key := db.TableKey{Schema: schemaName, Table: tableName}
		indexMap[key] = append(indexMap[key], index)
	}
	if err := rows.Err(); err != nil {
		return nil, util.FormatErrorWithQuery(err, query)
	}
	if err := rows.Close(); err != nil {
		return nil, errors.Wrapf(err, "failed to close rows")
	}

	return indexMap, nil
}

// getViews gets all views of a database.
func getViews(txn *sql.Tx, schemaName string, columnMap map[db.TableKey][]*storepb.ColumnMetadata) (map[string][]*storepb.ViewMetadata, error) {
	viewMap := make(map[string][]*storepb.ViewMetadata)

	query := ""
	if schemaName == "" {
		query = fmt.Sprintf(`
		SELECT OWNER, VIEW_NAME, TEXT
		FROM sys.all_views
		WHERE OWNER NOT IN (%s) AND OWNER NOT LIKE 'APEX_%%'
		ORDER BY owner, view_name
	`, systemSchema)
	} else {
		query = fmt.Sprintf(`
		SELECT OWNER, VIEW_NAME, TEXT
		FROM sys.all_views
		WHERE OWNER = '%s'
		ORDER BY view_name
	`, schemaName)
	}

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

		viewMap[schemaName] = append(viewMap[schemaName], view)
	}
	if err := rows.Err(); err != nil {
		return nil, util.FormatErrorWithQuery(err, query)
	}
	if err := rows.Close(); err != nil {
		return nil, errors.Wrapf(err, "failed to close rows")
	}

	return viewMap, nil
}

// getSequences gets all sequences of a database.
func getSequences(txn *sql.Tx, schemaName string) ([]*storepb.SequenceMetadata, error) {
	var sequences []*storepb.SequenceMetadata
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
				functions = append(functions, &storepb.FunctionMetadata{
					Name:       currentName,
					Definition: strings.Join(defText, ""),
				})
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
			}
			currentName = name
			currentType = t
			defText = []string{def}
		}
	}
	switch currentType {
	case "FUNCTION":
		functions = append(functions, &storepb.FunctionMetadata{
			Name:       currentName,
			Definition: strings.Join(defText, ""),
		})
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
	}
	if err := rows.Err(); err != nil {
		return nil, nil, nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, nil, nil, err
	}

	return functions, procedures, packages, nil
}
