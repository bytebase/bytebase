package mysql

import (
	"bufio"
	"context"
	"database/sql"
	"fmt"
	"io"
	"regexp"
	"strings"

	// embed will embeds the migration schema.
	_ "embed"

	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/plugin/db/util"
	"github.com/go-sql-driver/mysql"
	"go.uber.org/zap"
)

//go:embed mysql_migration_schema.sql
var migrationSchema string

var (
	systemDatabases = map[string]bool{
		"information_schema": true,
		// TiDB only
		"metrics_schema":     true,
		"mysql":              true,
		"performance_schema": true,
		"sys":                true,
	}
	baseTableType        = "BASE TABLE"
	viewTableType        = "VIEW"
	excludeAutoIncrement = regexp.MustCompile(`AUTO_INCREMENT=\d+ `)

	_ db.Driver              = (*Driver)(nil)
	_ util.MigrationExecutor = (*Driver)(nil)
)

func init() {
	db.Register(db.MySQL, newDriver)
	db.Register(db.TiDB, newDriver)
}

// Driver is the MySQL driver.
type Driver struct {
	l             *zap.Logger
	connectionCtx db.ConnectionContext
	dbType        db.Type

	db *sql.DB
}

func newDriver(config db.DriverConfig) db.Driver {
	return &Driver{
		l: config.Logger,
	}
}

// Open opens a MySQL driver.
func (driver *Driver) Open(ctx context.Context, dbType db.Type, config db.ConnectionConfig, connCtx db.ConnectionContext) (db.Driver, error) {
	protocol := "tcp"
	if strings.HasPrefix(config.Host, "/") {
		protocol = "unix"
	}

	params := []string{"multiStatements=true"}

	port := config.Port
	if port == "" {
		port = "3306"
		if dbType == db.TiDB {
			port = "4000"
		}
	}

	tlsConfig, err := config.TLSConfig.GetSslConfig()

	if err != nil {
		return nil, fmt.Errorf("sql: tls config error: %v", err)
	}

	loggedDSN := fmt.Sprintf("%s:<<redacted password>>@%s(%s:%s)/%s?%s", config.Username, protocol, config.Host, port, config.Database, strings.Join(params, "&"))
	dsn := fmt.Sprintf("%s@%s(%s:%s)/%s?%s", config.Username, protocol, config.Host, port, config.Database, strings.Join(params, "&"))
	if config.Password != "" {
		dsn = fmt.Sprintf("%s:%s@%s(%s:%s)/%s?%s", config.Username, config.Password, protocol, config.Host, port, config.Database, strings.Join(params, "&"))
	}
	tlsKey := "db.mysql.tls"
	if tlsConfig != nil {
		if err := mysql.RegisterTLSConfig(tlsKey, tlsConfig); err != nil {
			return nil, fmt.Errorf("sql: failed to register tls config: %v", err)
		}
		// TLS config is only used during sql.Open, so should be safe to deregister afterwards.
		defer mysql.DeregisterTLSConfig(tlsKey)
		dsn += fmt.Sprintf("?tls=%s", tlsKey)
	}
	driver.l.Debug("Opening MySQL driver",
		zap.String("dsn", loggedDSN),
		zap.String("environment", connCtx.EnvironmentName),
		zap.String("database", connCtx.InstanceName),
	)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		panic(err)
	}
	driver.dbType = dbType
	driver.db = db
	driver.connectionCtx = connCtx

	return driver, nil
}

// Close closes the driver.
func (driver *Driver) Close(ctx context.Context) error {
	return driver.db.Close()
}

// Ping pings the database.
func (driver *Driver) Ping(ctx context.Context) error {
	return driver.db.PingContext(ctx)
}

// GetDbConnection gets a database connection.
func (driver *Driver) GetDbConnection(ctx context.Context, database string) (*sql.DB, error) {
	return driver.db, nil
}

// GetVersion gets the version.
func (driver *Driver) GetVersion(ctx context.Context) (string, error) {
	query := "SELECT VERSION()"
	versionRow, err := driver.db.QueryContext(ctx, query)
	if err != nil {
		return "", util.FormatErrorWithQuery(err, query)
	}
	defer versionRow.Close()

	var version string
	versionRow.Next()
	if err := versionRow.Scan(&version); err != nil {
		return "", err
	}
	return version, nil
}

// SyncSchema syncs the schema.
func (driver *Driver) SyncSchema(ctx context.Context) ([]*db.User, []*db.Schema, error) {
	// Query MySQL version
	version, err := driver.GetVersion(ctx)
	if err != nil {
		return nil, nil, err
	}
	isMySQL8 := strings.HasPrefix(version, "8.0")

	excludedDatabaseList := []string{
		// Skip our internal "bytebase" database
		"'bytebase'",
	}

	// Skip all system databases
	for k := range systemDatabases {
		excludedDatabaseList = append(excludedDatabaseList, fmt.Sprintf("'%s'", k))
	}

	// Query user info
	userList, err := driver.getUserList(ctx)
	if err != nil {
		return nil, nil, err
	}

	// Query index info
	indexWhere := fmt.Sprintf("LOWER(TABLE_SCHEMA) NOT IN (%s)", strings.Join(excludedDatabaseList, ", "))
	query := `
			SELECT
				TABLE_SCHEMA,
				TABLE_NAME,
				INDEX_NAME,
				COLUMN_NAME,
				'',
				SEQ_IN_INDEX,
				INDEX_TYPE,
				CASE NON_UNIQUE WHEN 0 THEN 1 ELSE 0 END AS IS_UNIQUE,
				1,
				INDEX_COMMENT
			FROM information_schema.STATISTICS
			WHERE ` + indexWhere
	if isMySQL8 {
		query = `
			SELECT
				TABLE_SCHEMA,
				TABLE_NAME,
				INDEX_NAME,
				COLUMN_NAME,
				EXPRESSION,
				SEQ_IN_INDEX,
				INDEX_TYPE,
				CASE NON_UNIQUE WHEN 0 THEN 1 ELSE 0 END AS IS_UNIQUE,
				CASE IS_VISIBLE WHEN 'YES' THEN 1 ELSE 0 END,
				INDEX_COMMENT
			FROM information_schema.STATISTICS
			WHERE ` + indexWhere
	}
	indexRows, err := driver.db.QueryContext(ctx, query)
	if err != nil {
		return nil, nil, util.FormatErrorWithQuery(err, query)
	}
	defer indexRows.Close()

	// dbName/tableName -> indexList map
	indexMap := make(map[string][]db.Index)
	for indexRows.Next() {
		var dbName string
		var tableName string
		var columnName sql.NullString
		var expression sql.NullString
		var index db.Index
		if err := indexRows.Scan(
			&dbName,
			&tableName,
			&index.Name,
			&columnName,
			&expression,
			&index.Position,
			&index.Type,
			&index.Unique,
			&index.Visible,
			&index.Comment,
		); err != nil {
			return nil, nil, err
		}

		if columnName.Valid {
			index.Expression = columnName.String
		} else if expression.Valid {
			index.Expression = expression.String
		}

		key := fmt.Sprintf("%s/%s", dbName, tableName)
		if indexList, ok := indexMap[key]; ok {
			indexMap[key] = append(indexList, index)
		} else {
			indexMap[key] = []db.Index{index}
		}
	}

	// Query column info
	columnWhere := fmt.Sprintf("LOWER(TABLE_SCHEMA) NOT IN (%s)", strings.Join(excludedDatabaseList, ", "))
	query = `
			SELECT
				TABLE_SCHEMA,
				TABLE_NAME,
				IFNULL(COLUMN_NAME, ''),
				ORDINAL_POSITION,
				COLUMN_DEFAULT,
				IS_NULLABLE,
				COLUMN_TYPE,
				IFNULL(CHARACTER_SET_NAME, ''),
				IFNULL(COLLATION_NAME, ''),
				COLUMN_COMMENT
			FROM information_schema.COLUMNS
			WHERE ` + columnWhere
	columnRows, err := driver.db.QueryContext(ctx, query)
	if err != nil {
		return nil, nil, util.FormatErrorWithQuery(err, query)
	}
	defer columnRows.Close()

	// dbName/tableName -> columnList map
	columnMap := make(map[string][]db.Column)
	for columnRows.Next() {
		var dbName string
		var tableName string
		var nullable string
		var defaultStr sql.NullString
		var column db.Column
		if err := columnRows.Scan(
			&dbName,
			&tableName,
			&column.Name,
			&column.Position,
			&defaultStr,
			&nullable,
			&column.Type,
			&column.CharacterSet,
			&column.Collation,
			&column.Comment,
		); err != nil {
			return nil, nil, err
		}

		if defaultStr.Valid {
			column.Default = &defaultStr.String
		}

		key := fmt.Sprintf("%s/%s", dbName, tableName)
		if tableList, ok := columnMap[key]; ok {
			columnMap[key] = append(tableList, column)
		} else {
			columnMap[key] = []db.Column{column}
		}
	}

	// Query table info
	tableWhere := fmt.Sprintf("LOWER(TABLE_SCHEMA) NOT IN (%s)", strings.Join(excludedDatabaseList, ", "))
	query = `
			SELECT
				TABLE_SCHEMA,
				TABLE_NAME,
				IFNULL(UNIX_TIMESTAMP(CREATE_TIME), 0),
				IFNULL(UNIX_TIMESTAMP(UPDATE_TIME), 0),
				TABLE_TYPE,
				IFNULL(ENGINE, ''),
				IFNULL(TABLE_COLLATION, ''),
				IFNULL(TABLE_ROWS, 0),
				IFNULL(DATA_LENGTH, 0),
				IFNULL(INDEX_LENGTH, 0),
				IFNULL(DATA_FREE, 0),
				IFNULL(CREATE_OPTIONS, ''),
				IFNULL(TABLE_COMMENT, '')
			FROM information_schema.TABLES
			WHERE ` + tableWhere
	tableRows, err := driver.db.QueryContext(ctx, query)
	if err != nil {
		return nil, nil, util.FormatErrorWithQuery(err, query)
	}
	defer tableRows.Close()

	// dbName -> tableList map
	tableMap := make(map[string][]db.Table)
	type ViewInfo struct {
		createdTs int64
		updatedTs int64
		comment   string
	}
	// dbName/viewName -> ViewInfo
	viewInfoMap := make(map[string]ViewInfo)
	for tableRows.Next() {
		var dbName string
		// Workaround TiDB bug https://github.com/pingcap/tidb/issues/27970
		var tableCollation sql.NullString
		var table db.Table
		if err := tableRows.Scan(
			&dbName,
			&table.Name,
			&table.CreatedTs,
			&table.UpdatedTs,
			&table.Type,
			&table.Engine,
			&tableCollation,
			&table.RowCount,
			&table.DataSize,
			&table.IndexSize,
			&table.DataFree,
			&table.CreateOptions,
			&table.Comment,
		); err != nil {
			return nil, nil, err
		}

		if table.Type == baseTableType {
			if tableCollation.Valid {
				table.Collation = tableCollation.String
			}

			key := fmt.Sprintf("%s/%s", dbName, table.Name)
			table.ColumnList = columnMap[key]
			table.IndexList = indexMap[key]

			if tableList, ok := tableMap[dbName]; ok {
				tableMap[dbName] = append(tableList, table)
			} else {
				tableMap[dbName] = []db.Table{table}
			}
		} else if table.Type == viewTableType {
			viewInfoMap[fmt.Sprintf("%s/%s", dbName, table.Name)] = ViewInfo{
				createdTs: table.CreatedTs,
				updatedTs: table.UpdatedTs,
				comment:   table.Comment,
			}
		}
	}

	// Query view info
	viewWhere := fmt.Sprintf("LOWER(TABLE_SCHEMA) NOT IN (%s)", strings.Join(excludedDatabaseList, ", "))
	query = `
			SELECT
				TABLE_SCHEMA,
				TABLE_NAME,
				VIEW_DEFINITION
			FROM information_schema.VIEWS
			WHERE ` + viewWhere
	viewRows, err := driver.db.QueryContext(ctx, query)
	if err != nil {
		return nil, nil, util.FormatErrorWithQuery(err, query)
	}
	defer viewRows.Close()

	// dbName -> viewList map
	viewMap := make(map[string][]db.View)
	for viewRows.Next() {
		var dbName string
		var view db.View
		if err := viewRows.Scan(
			&dbName,
			&view.Name,
			&view.Definition,
		); err != nil {
			return nil, nil, err
		}

		info := viewInfoMap[fmt.Sprintf("%s/%s", dbName, view.Name)]
		view.CreatedTs = info.createdTs
		view.UpdatedTs = info.updatedTs
		view.Comment = info.comment

		if viewList, ok := viewMap[dbName]; ok {
			viewMap[dbName] = append(viewList, view)
		} else {
			viewMap[dbName] = []db.View{view}
		}
	}

	// Query db info
	where := fmt.Sprintf("LOWER(SCHEMA_NAME) NOT IN (%s)", strings.Join(excludedDatabaseList, ", "))
	query = `
			SELECT
		    SCHEMA_NAME,
			DEFAULT_CHARACTER_SET_NAME,
			DEFAULT_COLLATION_NAME
		FROM information_schema.SCHEMATA
		WHERE ` + where
	rows, err := driver.db.QueryContext(ctx, query)
	if err != nil {
		return nil, nil, util.FormatErrorWithQuery(err, query)
	}
	defer rows.Close()

	var schemaList []*db.Schema
	for rows.Next() {
		var schema db.Schema
		if err := rows.Scan(
			&schema.Name,
			&schema.CharacterSet,
			&schema.Collation,
		); err != nil {
			return nil, nil, err
		}

		schema.TableList = tableMap[schema.Name]
		schema.ViewList = viewMap[schema.Name]

		schemaList = append(schemaList, &schema)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	return userList, schemaList, err
}

func (driver *Driver) getUserList(ctx context.Context) ([]*db.User, error) {
	// Query user info
	query := `
	  SELECT
			user,
			host
		FROM mysql.user
		WHERE user NOT LIKE 'mysql.%'
	`
	var userList []*db.User
	userRows, err := driver.db.QueryContext(ctx, query)

	if err != nil {
		return nil, util.FormatErrorWithQuery(err, query)
	}
	defer userRows.Close()

	for userRows.Next() {
		var user string
		var host string
		if err := userRows.Scan(
			&user,
			&host,
		); err != nil {
			return nil, err
		}

		// Uses single quote instead of backtick to escape because this is a string
		// instead of table (which should use backtick instead). MySQL actually works
		// in both ways. On the other hand, some other MySQL compatible engines might not (OceanBase in this case).
		name := fmt.Sprintf("'%s'@'%s'", user, host)
		query = fmt.Sprintf("SHOW GRANTS FOR %s", name)
		grantRows, err := driver.db.QueryContext(ctx,
			query,
		)
		if err != nil {
			return nil, util.FormatErrorWithQuery(err, query)
		}
		defer grantRows.Close()

		grantList := []string{}
		for grantRows.Next() {
			var grant string
			if err := grantRows.Scan(&grant); err != nil {
				return nil, err
			}
			grantList = append(grantList, grant)
		}

		userList = append(userList, &db.User{
			Name:  name,
			Grant: strings.Join(grantList, "\n"),
		})
	}
	return userList, nil
}

// Execute executes a SQL statement.
func (driver *Driver) Execute(ctx context.Context, statement string) error {
	tx, err := driver.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, statement)

	if err == nil {
		if err := tx.Commit(); err != nil {
			return err
		}
	}

	return err
}

// Query queries a SQL statement.
func (driver *Driver) Query(ctx context.Context, statement string, limit int) ([]interface{}, error) {
	return util.Query(ctx, driver.l, driver.db, statement, limit)
}

// NeedsSetupMigration returns whether it needs to setup migration.
func (driver *Driver) NeedsSetupMigration(ctx context.Context) (bool, error) {
	const query = `
		SELECT
		    1
		FROM information_schema.TABLES
		WHERE TABLE_SCHEMA = 'bytebase' AND TABLE_NAME = 'migration_history'
		`
	return util.NeedsSetupMigrationSchema(ctx, driver.db, query)
}

// SetupMigrationIfNeeded sets up migration if needed.
func (driver *Driver) SetupMigrationIfNeeded(ctx context.Context) error {
	setup, err := driver.NeedsSetupMigration(ctx)
	if err != nil {
		return nil
	}

	if setup {
		driver.l.Info("Bytebase migration schema not found, creating schema...",
			zap.String("environment", driver.connectionCtx.EnvironmentName),
			zap.String("database", driver.connectionCtx.InstanceName),
		)
		// Do not wrap it in a single transaction here because:
		// 1. For MySQL, each DDL is in its own transaction. See https://dev.mysql.com/doc/refman/8.0/en/implicit-commit.html
		// 2. For TiDB, the created database/table is not visible to the followup statements from the same transaction.
		if _, err := driver.db.ExecContext(ctx, migrationSchema); err != nil {
			driver.l.Error("Failed to initialize migration schema.",
				zap.Error(err),
				zap.String("environment", driver.connectionCtx.EnvironmentName),
				zap.String("database", driver.connectionCtx.InstanceName),
			)
			return util.FormatErrorWithQuery(err, migrationSchema)
		}
		driver.l.Info("Successfully created migration schema.",
			zap.String("environment", driver.connectionCtx.EnvironmentName),
			zap.String("database", driver.connectionCtx.InstanceName),
		)
	}

	return nil
}

// GetLargestVersionSinceLastBaseline will return the largest version since the last baseline.
func (Driver) GetLargestVersionSinceLastBaseline(ctx context.Context, tx *sql.Tx, namespace string, minSequence int) (*string, error) {
	const getLargestVersionSinceLastBaselineQuery = `
		SELECT MAX(version) FROM bytebase.migration_history
		WHERE namespace = ? AND sequence >= ?
	`
	row, err := tx.QueryContext(ctx, getLargestVersionSinceLastBaselineQuery,
		namespace, minSequence,
	)
	if err != nil {
		return nil, util.FormatErrorWithQuery(err, getLargestVersionSinceLastBaselineQuery)
	}
	defer row.Close()

	var version sql.NullString
	row.Next()
	if err := row.Scan(&version); err != nil {
		return nil, err
	}

	if version.Valid {
		return &version.String, nil
	}

	return nil, nil
}

// FindLargestSequence will return the largest sequence number.
func (Driver) FindLargestSequence(ctx context.Context, tx *sql.Tx, namespace string, baseline bool) (int, error) {
	findLargestSequenceQuery := `
		SELECT MAX(sequence) FROM bytebase.migration_history
		WHERE namespace = ?`
	if baseline {
		findLargestSequenceQuery = fmt.Sprintf("%s AND (type = '%s' OR type = '%s')", findLargestSequenceQuery, db.Baseline, db.Branch)
	}
	row, err := tx.QueryContext(ctx, findLargestSequenceQuery,
		namespace,
	)
	if err != nil {
		return -1, util.FormatErrorWithQuery(err, findLargestSequenceQuery)
	}
	defer row.Close()

	var sequence sql.NullInt32
	row.Next()
	if err := row.Scan(&sequence); err != nil {
		return -1, err
	}

	if !sequence.Valid {
		// Returns 0 if we haven't applied any migration for this namespace.
		return 0, nil
	}

	return int(sequence.Int32), nil
}

// InsertPendingHistory will insert the migration record with pending status and return the inserted ID.
func (Driver) InsertPendingHistory(ctx context.Context, tx *sql.Tx, sequence int, prevSchema string, m *db.MigrationInfo, statement string) (int64, error) {
	const insertHistoryQuery = `
		INSERT INTO bytebase.migration_history (
			created_by,
			created_ts,
			updated_by,
			updated_ts,
			release_version,
			namespace,
			sequence,
			source,
			type,
			status,
			version,
			description,
			statement,
			` + "`schema`," + `
			schema_prev,
			execution_duration_ns,
			issue_id,
			payload
		)
		VALUES (?, unix_timestamp(), ?, unix_timestamp(), ?, ?, ?, ?,  ?, 'PENDING', ?, ?, ?, ?, ?, 0, ?, ?)
		`
	res, err := tx.ExecContext(ctx, insertHistoryQuery,
		m.Creator,
		m.Creator,
		m.ReleaseVersion,
		m.Namespace,
		sequence,
		m.Source,
		m.Type,
		m.Version,
		m.Description,
		statement,
		prevSchema,
		prevSchema,
		m.IssueID,
		m.Payload,
	)
	if err != nil {
		return int64(0), util.FormatErrorWithQuery(err, insertHistoryQuery)
	}

	insertedID, err := res.LastInsertId()
	if err != nil {
		return int64(0), util.FormatErrorWithQuery(err, insertHistoryQuery)
	}
	return insertedID, nil
}

// UpdateHistoryAsDone will update the migration record as done.
func (Driver) UpdateHistoryAsDone(ctx context.Context, tx *sql.Tx, migrationDurationNs int64, updatedSchema string, insertedID int64) error {
	const updateHistoryAsDoneQuery = `
		UPDATE
			bytebase.migration_history
		SET
			status = 'DONE',
			execution_duration_ns = ?,
		` + "`schema` = ?" + `
		WHERE id = ?
		`
	_, err := tx.ExecContext(ctx, updateHistoryAsDoneQuery, migrationDurationNs, updatedSchema, insertedID)
	return err
}

// UpdateHistoryAsFailed will update the migration record as failed.
func (Driver) UpdateHistoryAsFailed(ctx context.Context, tx *sql.Tx, migrationDurationNs int64, insertedID int64) error {
	const updateHistoryAsFailedQuery = `
		UPDATE
			bytebase.migration_history
		SET
			status = 'FAILED',
			execution_duration_ns = ?
		WHERE id = ?
		`
	_, err := tx.ExecContext(ctx, updateHistoryAsFailedQuery, migrationDurationNs, insertedID)
	return err
}

// ExecuteMigration will execute the migration.
func (driver *Driver) ExecuteMigration(ctx context.Context, m *db.MigrationInfo, statement string) (int64, string, error) {
	return util.ExecuteMigration(ctx, driver.l, driver, m, statement)
}

// FindMigrationHistoryList finds the migration history.
func (driver *Driver) FindMigrationHistoryList(ctx context.Context, find *db.MigrationHistoryFind) ([]*db.MigrationHistory, error) {
	baseQuery := `
	SELECT
		id,
		created_by,
		created_ts,
		updated_by,
		updated_ts,
		release_version,
		namespace,
		sequence,
		source,
		type,
		status,
		version,
		description,
		statement,
		` + "`schema`," + `
		schema_prev,
		execution_duration_ns,
		issue_id,
		payload
		FROM bytebase.migration_history `
	paramNames, params := []string{}, []interface{}{}
	if v := find.ID; v != nil {
		paramNames, params = append(paramNames, "id"), append(params, *v)
	}
	if v := find.Database; v != nil {
		paramNames, params = append(paramNames, "namespace"), append(params, *v)
	}
	if v := find.Version; v != nil {
		paramNames, params = append(paramNames, "version"), append(params, *v)
	}
	if v := find.Source; v != nil {
		paramNames, params = append(paramNames, "source"), append(params, *v)
	}
	var query = baseQuery +
		db.FormatParamNameInQuestionMark(paramNames) +
		`ORDER BY created_ts DESC`
	if v := find.Limit; v != nil {
		query += fmt.Sprintf(" LIMIT %d", *v)
	}
	return util.FindMigrationHistoryList(ctx, query, params, driver, find, baseQuery)
}

// Dump and restore
const (
	databaseHeaderFmt = "" +
		"--\n" +
		"-- MySQL database structure for `%s`\n" +
		"--\n"
	useDatabaseFmt = "USE `%s`;\n\n"
	settingsStmt   = "" +
		"SET character_set_client  = %s;\n" +
		"SET character_set_results = %s;\n" +
		"SET collation_connection  = %s;\n" +
		"SET sql_mode              = '%s';\n"
	tableStmtFmt = "" +
		"--\n" +
		"-- Table structure for `%s`\n" +
		"--\n" +
		"%s;\n"
	viewStmtFmt = "" +
		"--\n" +
		"-- View structure for `%s`\n" +
		"--\n" +
		"%s;\n"
	routineStmtFmt = "" +
		"--\n" +
		"-- %s structure for `%s`\n" +
		"--\n" +
		settingsStmt +
		"DELIMITER ;;\n" +
		"%s ;;\n" +
		"DELIMITER ;\n"
	eventStmtFmt = "" +
		"--\n" +
		"-- Event structure for `%s`\n" +
		"--\n" +
		settingsStmt +
		"SET time_zone = '%s';\n" +
		"DELIMITER ;;\n" +
		"%s ;;\n" +
		"DELIMITER ;\n"
	triggerStmtFmt = "" +
		"--\n" +
		"-- Trigger structure for `%s`\n" +
		"--\n" +
		settingsStmt +
		"DELIMITER ;;\n" +
		"%s ;;\n" +
		"DELIMITER ;\n"
)

// Dump dumps the database.
func (driver *Driver) Dump(ctx context.Context, database string, out io.Writer, schemaOnly bool) error {
	// mysqldump -u root --databases dbName --no-data --routines --events --triggers --compact

	options := sql.TxOptions{}
	// TiDB does not support readonly, so we only set for MySQL.
	if driver.dbType == "MYSQL" {
		options.ReadOnly = true
	}
	txn, err := driver.db.BeginTx(ctx, &options)
	if err != nil {
		return err
	}
	defer txn.Rollback()

	if err := dumpTxn(ctx, txn, database, out, schemaOnly); err != nil {
		return err
	}

	if err := txn.Commit(); err != nil {
		return err
	}

	return nil
}

// Restore restores a database.
func (driver *Driver) Restore(ctx context.Context, sc *bufio.Scanner) (err error) {
	txn, err := driver.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer txn.Rollback()

	f := func(stmt string) error {
		if _, err := txn.Exec(stmt); err != nil {
			return err
		}
		return nil
	}

	if err := util.ApplyMultiStatements(sc, f); err != nil {
		return err
	}

	if err := txn.Commit(); err != nil {
		return err
	}

	return nil
}

func dumpTxn(ctx context.Context, txn *sql.Tx, database string, out io.Writer, schemaOnly bool) error {
	// Find all dumpable databases
	dbNames, err := getDatabases(txn)
	if err != nil {
		return fmt.Errorf("failed to get databases: %s", err)
	}

	var dumpableDbNames []string
	if database != "" {
		exist := false
		for _, n := range dbNames {
			if n == database {
				exist = true
				break
			}
		}
		if !exist {
			return common.Errorf(common.NotFound, fmt.Errorf("database %s not found", database))
		}
		dumpableDbNames = []string{database}
	} else {
		for _, dbName := range dbNames {
			if systemDatabases[dbName] {
				continue
			}
			dumpableDbNames = append(dumpableDbNames, dbName)
		}
	}

	for _, dbName := range dumpableDbNames {
		// Include "USE DATABASE xxx" if dumping multiple databases.
		if len(dumpableDbNames) > 1 {
			// Database header.
			header := fmt.Sprintf(databaseHeaderFmt, dbName)
			if _, err := io.WriteString(out, header); err != nil {
				return err
			}
			dbStmt, err := getDatabaseStmt(txn, dbName)
			if err != nil {
				return fmt.Errorf("failed to get database %q: %s", dbName, err)
			}
			if _, err := io.WriteString(out, dbStmt); err != nil {
				return err
			}
			// Use database statement.
			useStmt := fmt.Sprintf(useDatabaseFmt, dbName)
			if _, err := io.WriteString(out, useStmt); err != nil {
				return err
			}
		}

		// Table and view statement.
		tables, err := getTables(txn, dbName)
		if err != nil {
			return fmt.Errorf("failed to get tables of database %q: %s", dbName, err)
		}
		for _, tbl := range tables {
			if schemaOnly && tbl.tableType == baseTableType {
				tbl.statement = excludeSchemaAutoIncrementValue(tbl.statement)
			}
			if _, err := io.WriteString(out, fmt.Sprintf("%s\n", tbl.statement)); err != nil {
				return err
			}
			if !schemaOnly && tbl.tableType == baseTableType {
				// Include db prefix if dumping multiple databases.
				includeDbPrefix := len(dumpableDbNames) > 1
				if err := exportTableData(txn, dbName, tbl.name, includeDbPrefix, out); err != nil {
					return err
				}
			}
		}

		// Procedure and function (routine) statements.
		routines, err := getRoutines(txn, dbName)
		if err != nil {
			return fmt.Errorf("failed to get routines of database %q: %s", dbName, err)
		}
		for _, rt := range routines {
			if _, err := io.WriteString(out, fmt.Sprintf("%s\n", rt.statement)); err != nil {
				return err
			}
		}

		// Event statements.
		events, err := getEvents(txn, dbName)
		if err != nil {
			return fmt.Errorf("failed to get events of database %q: %s", dbName, err)
		}
		for _, et := range events {
			if _, err := io.WriteString(out, fmt.Sprintf("%s\n", et.statement)); err != nil {
				return err
			}
		}

		// Trigger statements.
		triggers, err := getTriggers(txn, dbName)
		if err != nil {
			return fmt.Errorf("failed to get triggers of database %q: %s", dbName, err)
		}
		for _, tr := range triggers {
			if _, err := io.WriteString(out, fmt.Sprintf("%s\n", tr.statement)); err != nil {
				return err
			}
		}
	}

	return nil
}

// excludeSchemaAutoIncrementValue excludes the starting value of AUTO_INCREMENT if it's a schema only dump.
// https://github.com/bytebase/bytebase/issues/123
func excludeSchemaAutoIncrementValue(s string) string {
	return excludeAutoIncrement.ReplaceAllString(s, ``)
}

// getDatabases gets all databases of an instance.
func getDatabases(txn *sql.Tx) ([]string, error) {
	var dbNames []string
	rows, err := txn.Query("SHOW DATABASES;")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		dbNames = append(dbNames, name)
	}
	return dbNames, nil
}

// getDatabaseStmt gets the create statement of a database.
func getDatabaseStmt(txn *sql.Tx, dbName string) (string, error) {
	query := fmt.Sprintf("SHOW CREATE DATABASE IF NOT EXISTS %s;", dbName)
	rows, err := txn.Query(query)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	if rows.Next() {
		var stmt, unused string
		if err := rows.Scan(&unused, &stmt); err != nil {
			return "", err
		}
		return fmt.Sprintf("%s;\n", stmt), nil
	}
	return "", fmt.Errorf("query %q returned empty row", query)
}

// tableSchema describes the schema of a table or view.
type tableSchema struct {
	name      string
	tableType string
	statement string
}

// routineSchema describes the schema of a function or procedure (routine).
type routineSchema struct {
	name        string
	routineType string
	statement   string
}

// eventSchema describes the schema of an event.
type eventSchema struct {
	name      string
	statement string
}

// triggerSchema describes the schema of a trigger.
type triggerSchema struct {
	name      string
	statement string
}

// getTables gets all tables of a database.
func getTables(txn *sql.Tx, dbName string) ([]*tableSchema, error) {
	var tables []*tableSchema
	query := fmt.Sprintf("SHOW FULL TABLES FROM `%s`;", dbName)
	rows, err := txn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var tbl tableSchema
		if err := rows.Scan(&tbl.name, &tbl.tableType); err != nil {
			return nil, err
		}
		tables = append(tables, &tbl)
	}
	for _, tbl := range tables {
		stmt, err := getTableStmt(txn, dbName, tbl.name, tbl.tableType)
		if err != nil {
			return nil, fmt.Errorf("getTableStmt(%q, %q, %q) got error: %s", dbName, tbl.name, tbl.tableType, err)
		}
		tbl.statement = stmt
	}
	return tables, nil
}

// getTableStmt gets the create statement of a table.
func getTableStmt(txn *sql.Tx, dbName, tblName, tblType string) (string, error) {
	switch tblType {
	case baseTableType:
		query := fmt.Sprintf("SHOW CREATE TABLE `%s`.`%s`;", dbName, tblName)
		rows, err := txn.Query(query)
		if err != nil {
			return "", err
		}
		defer rows.Close()

		if rows.Next() {
			var stmt, unused string
			if err := rows.Scan(&unused, &stmt); err != nil {
				return "", err
			}
			return fmt.Sprintf(tableStmtFmt, tblName, stmt), nil
		}
		return "", fmt.Errorf("query %q returned invalid rows", query)
	case viewTableType:
		// This differs from mysqldump as it includes.
		query := fmt.Sprintf("SHOW CREATE VIEW `%s`.`%s`;", dbName, tblName)
		rows, err := txn.Query(query)
		if err != nil {
			return "", err
		}
		defer rows.Close()

		if rows.Next() {
			var createStmt, unused string
			if err := rows.Scan(&unused, &createStmt, &unused, &unused); err != nil {
				return "", err
			}
			return fmt.Sprintf(viewStmtFmt, tblName, createStmt), nil
		}
		return "", fmt.Errorf("query %q returned invalid rows", query)
	default:
		return "", fmt.Errorf("unrecognized table type %q for database %q table %q", tblType, dbName, tblName)
	}

}

// exportTableData gets the data of a table.
func exportTableData(txn *sql.Tx, dbName, tblName string, includeDbPrefix bool, out io.Writer) error {
	query := fmt.Sprintf("SELECT * FROM `%s`.`%s`;", dbName, tblName)
	rows, err := txn.Query(query)
	if err != nil {
		return err
	}
	defer rows.Close()

	cols, err := rows.ColumnTypes()
	if err != nil {
		return err
	}
	if len(cols) <= 0 {
		return nil
	}
	values := make([]*sql.NullString, len(cols))
	refs := make([]interface{}, len(cols))
	for i := 0; i < len(cols); i++ {
		refs[i] = &values[i]
	}
	for rows.Next() {
		if err := rows.Scan(refs...); err != nil {
			return err
		}
		tokens := make([]string, len(cols))
		for i, v := range values {
			switch {
			case v == nil || !v.Valid:
				tokens[i] = "NULL"
			case isNumeric(cols[i].ScanType().Name()):
				tokens[i] = v.String
			default:
				tokens[i] = fmt.Sprintf("'%s'", v.String)
			}
		}
		dbPrefix := ""
		if includeDbPrefix {
			dbPrefix = fmt.Sprintf("`%s`.", dbName)
		}
		stmt := fmt.Sprintf("INSERT INTO %s`%s` VALUES (%s);\n", dbPrefix, tblName, strings.Join(tokens, ", "))
		if _, err := io.WriteString(out, stmt); err != nil {
			return err
		}
	}
	if _, err := io.WriteString(out, "\n"); err != nil {
		return err
	}
	return nil
}

// isNumeric determines whether the value needs quotes.
// Even if the function returns incorrect result, the data dump will still work.
func isNumeric(t string) bool {
	return strings.Contains(t, "int") || strings.Contains(t, "bool") || strings.Contains(t, "float") || strings.Contains(t, "byte")
}

// getRoutines gets all routines of a database.
func getRoutines(txn *sql.Tx, dbName string) ([]*routineSchema, error) {
	var routines []*routineSchema
	for _, routineType := range []string{"FUNCTION", "PROCEDURE"} {
		query := fmt.Sprintf("SHOW %s STATUS WHERE Db = ?;", routineType)
		rows, err := txn.Query(query, dbName)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		cols, err := rows.Columns()
		if err != nil {
			return nil, err
		}
		var values []interface{}
		for i := 0; i < len(cols); i++ {
			values = append(values, new(interface{}))
		}
		for rows.Next() {
			var r routineSchema
			if err := rows.Scan(values...); err != nil {
				return nil, err
			}
			r.name = fmt.Sprintf("%s", *values[1].(*interface{}))
			r.routineType = fmt.Sprintf("%s", *values[2].(*interface{}))

			routines = append(routines, &r)
		}
	}

	for _, r := range routines {
		stmt, err := getRoutineStmt(txn, dbName, r.name, r.routineType)
		if err != nil {
			return nil, fmt.Errorf("getRoutineStmt(%q, %q, %q) got error: %s", dbName, r.name, r.routineType, err)
		}
		r.statement = stmt
	}
	return routines, nil
}

// getRoutineStmt gets the create statement of a routine.
func getRoutineStmt(txn *sql.Tx, dbName, routineName, routineType string) (string, error) {
	query := fmt.Sprintf("SHOW CREATE %s `%s`.`%s`;", routineType, dbName, routineName)
	rows, err := txn.Query(query)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	if rows.Next() {
		var sqlmode, stmt, charset, collation, unused string
		if err := rows.Scan(&unused, &sqlmode, &stmt, &charset, &collation, &unused); err != nil {
			return "", err
		}
		return fmt.Sprintf(routineStmtFmt, getReadableRoutineType(routineType), routineName, charset, charset, collation, sqlmode, stmt), nil
	}
	return "", fmt.Errorf("query %q returned invalid rows", query)

}

// getReadableRoutineType gets the printable routine type.
func getReadableRoutineType(s string) string {
	switch s {
	case "FUNCTION":
		return "Function"
	case "PROCEDURE":
		return "Procedure"
	default:
		return s
	}
}

// getEvents gets all events of a database.
func getEvents(txn *sql.Tx, dbName string) ([]*eventSchema, error) {
	var events []*eventSchema
	rows, err := txn.Query(fmt.Sprintf("SHOW EVENTS FROM `%s`;", dbName))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	var values []interface{}
	for i := 0; i < len(cols); i++ {
		values = append(values, new(interface{}))
	}
	for rows.Next() {
		var r eventSchema
		if err := rows.Scan(values...); err != nil {
			return nil, err
		}
		r.name = fmt.Sprintf("%s", *values[1].(*interface{}))
		events = append(events, &r)
	}

	for _, r := range events {
		stmt, err := getEventStmt(txn, dbName, r.name)
		if err != nil {
			return nil, fmt.Errorf("getEventStmt(%q, %q) got error: %s", dbName, r.name, err)
		}
		r.statement = stmt
	}
	return events, nil
}

// getEventStmt gets the create statement of an event.
func getEventStmt(txn *sql.Tx, dbName, eventName string) (string, error) {
	query := fmt.Sprintf("SHOW CREATE EVENT `%s`.`%s`;", dbName, eventName)
	rows, err := txn.Query(query)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	if rows.Next() {
		var sqlmode, timezone, stmt, charset, collation, unused string
		if err := rows.Scan(&unused, &sqlmode, &timezone, &stmt, &charset, &collation, &unused); err != nil {
			return "", err
		}
		return fmt.Sprintf(eventStmtFmt, eventName, charset, charset, collation, sqlmode, timezone, stmt), nil
	}
	return "", fmt.Errorf("query %q returned invalid rows", query)
}

// getTriggers gets all triggers of a database.
func getTriggers(txn *sql.Tx, dbName string) ([]*triggerSchema, error) {
	var triggers []*triggerSchema
	rows, err := txn.Query(fmt.Sprintf("SHOW TRIGGERS FROM `%s`;", dbName))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	var values []interface{}
	for i := 0; i < len(cols); i++ {
		values = append(values, new(interface{}))
	}
	for rows.Next() {
		var tr triggerSchema
		if err := rows.Scan(values...); err != nil {
			return nil, err
		}
		tr.name = fmt.Sprintf("%s", *values[0].(*interface{}))
		triggers = append(triggers, &tr)
	}
	for _, tr := range triggers {
		stmt, err := getTriggerStmt(txn, dbName, tr.name)
		if err != nil {
			return nil, fmt.Errorf("getTriggerStmt(%q, %q) got error: %s", dbName, tr.name, err)
		}
		tr.statement = stmt
	}
	return triggers, nil
}

// getTriggerStmt gets the create statement of a trigger.
func getTriggerStmt(txn *sql.Tx, dbName, triggerName string) (string, error) {
	query := fmt.Sprintf("SHOW CREATE TRIGGER `%s`.`%s`;", dbName, triggerName)
	rows, err := txn.Query(query)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	if rows.Next() {
		var sqlmode, stmt, charset, collation, unused string
		if err := rows.Scan(&unused, &sqlmode, &stmt, &charset, &collation, &unused, &unused); err != nil {
			return "", err
		}
		return fmt.Sprintf(triggerStmtFmt, triggerName, charset, charset, collation, sqlmode, stmt), nil
	}
	return "", fmt.Errorf("query %q returned invalid rows", query)
}
