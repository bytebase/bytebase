package clickhouse

import (
	"bufio"
	"context"
	"database/sql"
	"fmt"
	"io"
	"strings"
	"time"

	// embed will embeds the migration schema.
	_ "embed"

	clickhouse "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/plugin/db/util"
	"go.uber.org/zap"
)

//go:embed clickhouse_migration_schema.sql
var migrationSchema string

var (
	systemDatabases = map[string]bool{
		"system":             true,
		"information_schema": true,
		"INFORMATION_SCHEMA": true,
	}

	_ db.Driver              = (*Driver)(nil)
	_ util.MigrationExecutor = (*Driver)(nil)
)

func init() {
	db.Register(db.ClickHouse, newDriver)
}

// Driver is the ClickHouse driver.
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

// Open opens a ClickHouse driver.
func (driver *Driver) Open(ctx context.Context, dbType db.Type, config db.ConnectionConfig, connCtx db.ConnectionContext) (db.Driver, error) {
	port := config.Port
	if port == "" {
		port = "9000"
	}
	addr := fmt.Sprintf("%s:%s", config.Host, port)
	// Set SSL configuration.
	tlsConfig, err := config.TLSConfig.GetSslConfig()
	if err != nil {
		return nil, fmt.Errorf("sql: tls config error: %v", err)
	}
	// Default user name is "default".
	conn := clickhouse.OpenDB(&clickhouse.Options{
		Addr: []string{addr},
		Auth: clickhouse.Auth{
			Database: config.Database,
			Username: config.Username,
			Password: config.Password,
		},
		TLS: tlsConfig,
		Settings: clickhouse.Settings{
			"max_execution_time": 60, // 60 seconds.
		},
		DialTimeout: 10 * time.Second,
	})

	driver.l.Debug("Opening ClickHouse driver",
		zap.String("addr", addr),
		zap.String("environment", connCtx.EnvironmentName),
		zap.String("database", connCtx.InstanceName),
	)

	driver.dbType = dbType
	driver.db = conn
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

	// Query column info
	columnWhere := fmt.Sprintf("LOWER(database) NOT IN (%s)", strings.Join(excludedDatabaseList, ", "))
	query := `
			SELECT
				database,
				table,
				name,
				position,
				default_expression,
				type,
				comment
			FROM system.columns
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
		var column db.Column
		if err := columnRows.Scan(
			&dbName,
			&tableName,
			&column.Name,
			&column.Position,
			&column.Default,
			&column.Type,
			&column.Comment,
		); err != nil {
			return nil, nil, err
		}

		key := fmt.Sprintf("%s/%s", dbName, tableName)
		if tableList, ok := columnMap[key]; ok {
			columnMap[key] = append(tableList, column)
		} else {
			columnMap[key] = append([]db.Column(nil), column)
		}
	}

	// Query table info
	tableWhere := fmt.Sprintf("LOWER(database) NOT IN (%s)", strings.Join(excludedDatabaseList, ", "))
	query = `
			SELECT
				database,
				name,
				engine,
				IFNULL(total_rows, 0),
				IFNULL(total_bytes, 0),
				metadata_modification_time,
				create_table_query,
				comment
			FROM system.tables
			WHERE ` + tableWhere
	tableRows, err := driver.db.QueryContext(ctx, query)
	if err != nil {
		return nil, nil, util.FormatErrorWithQuery(err, query)
	}
	defer tableRows.Close()

	// dbName -> tableList map
	tableMap := make(map[string][]db.Table)
	// dbName -> viewList map
	viewMap := make(map[string][]db.View)

	for tableRows.Next() {
		var dbName, name, engine, definition, comment string
		var rowCount, totalBytes int64
		var lastUpdatedTime time.Time
		if err := tableRows.Scan(
			&dbName,
			&name,
			&engine,
			&rowCount,
			&totalBytes,
			&lastUpdatedTime,
			&definition,
			&comment,
		); err != nil {
			return nil, nil, err
		}

		if engine == "View" {
			var view db.View
			view.Name = name
			view.UpdatedTs = lastUpdatedTime.Unix()
			view.Definition = definition
			view.Comment = comment
			viewMap[dbName] = append(viewMap[dbName], view)
		} else {
			var table db.Table
			table.Type = "BASE TABLE"
			table.Name = name
			table.Engine = engine
			table.Comment = comment
			table.RowCount = rowCount
			table.DataSize = totalBytes
			table.UpdatedTs = lastUpdatedTime.Unix()
			key := fmt.Sprintf("%s/%s", dbName, name)
			table.ColumnList = columnMap[key]
			tableMap[dbName] = append(tableMap[dbName], table)
		}
	}

	var schemaList []*db.Schema
	// Query db info
	where := fmt.Sprintf("name NOT IN (%s)", strings.Join(excludedDatabaseList, ", "))
	query = `
		SELECT
			name
		FROM system.databases
		WHERE ` + where
	rows, err := driver.db.QueryContext(ctx, query)
	if err != nil {
		return nil, nil, util.FormatErrorWithQuery(err, query)
	}
	defer rows.Close()
	for rows.Next() {
		var schema db.Schema
		if err := rows.Scan(
			&schema.Name,
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

	return userList, schemaList, nil
}

func (driver *Driver) getUserList(ctx context.Context) ([]*db.User, error) {
	// Query user info
	// host_ip isn't used for user identifier.
	query := `
	  SELECT
			name
		FROM system.users
	`
	userRows, err := driver.db.QueryContext(ctx, query)

	if err != nil {
		return nil, util.FormatErrorWithQuery(err, query)
	}
	defer userRows.Close()

	var userList []*db.User
	for userRows.Next() {
		var user string
		if err := userRows.Scan(
			&user,
		); err != nil {
			return nil, err
		}

		// Uses single quote instead of backtick to escape because this is a string
		// instead of table (which should use backtick instead). MySQL actually works
		// in both ways. On the other hand, some other MySQL compatible engines might not (OceanBase in this case).
		query = fmt.Sprintf("SHOW GRANTS FOR %s", user)
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
			Name:  user,
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

	f := func(stmt string) error {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return err
		}
		return nil
	}
	sc := bufio.NewScanner(strings.NewReader(statement))
	if err := util.ApplyMultiStatements(sc, f); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
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
		FROM system.tables
		WHERE database = 'bytebase' AND name = 'migration_history'
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
		if err := driver.Execute(ctx, migrationSchema); err != nil {
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

// FindLargestVersionSinceBaseline will find the largest version since last baseline or branch.
func (driver Driver) FindLargestVersionSinceBaseline(ctx context.Context, tx *sql.Tx, namespace string) (*string, error) {
	largestBaselineSequence, err := driver.FindLargestSequence(ctx, tx, namespace, true /* baseline */)
	if err != nil {
		return nil, err
	}
	const getLargestVersionSinceLastBaselineQuery = `
		SELECT MAX(version) FROM bytebase.migration_history
		WHERE namespace = $1 AND sequence >= $2
	`
	row, err := tx.QueryContext(ctx, getLargestVersionSinceLastBaselineQuery,
		namespace, largestBaselineSequence,
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
		WHERE namespace = $1`
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
func (Driver) InsertPendingHistory(ctx context.Context, tx *sql.Tx, sequence int, prevSchema string, m *db.MigrationInfo, storedVersion, statement string) (int64, error) {
	const insertHistoryQuery = `
	INSERT INTO bytebase.migration_history (
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
	)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
	`
	var insertedID int64
	maxIDQuery := "SELECT MAX(id)+1 FROM bytebase.migration_history"
	rows, err := tx.QueryContext(ctx, maxIDQuery)
	if err != nil {
		return int64(0), util.FormatErrorWithQuery(err, maxIDQuery)
	}
	defer rows.Close()
	for rows.Next() {
		if err := rows.Scan(
			&insertedID,
		); err != nil {
			return int64(0), util.FormatErrorWithQuery(err, maxIDQuery)
		}
	}
	if err := rows.Err(); err != nil {
		return int64(0), util.FormatErrorWithQuery(err, maxIDQuery)
	}
	// Clickhouse sql driver doesn't support taking now() as prepared value.
	now := time.Now().Unix()
	_, err = tx.ExecContext(ctx, insertHistoryQuery,
		insertedID,
		m.Creator,
		now,
		m.Creator,
		now,
		m.ReleaseVersion,
		m.Namespace,
		sequence,
		m.Source,
		m.Type,
		"PENDING",
		storedVersion,
		m.Description,
		statement,
		prevSchema,
		prevSchema,
		0,
		m.IssueID,
		m.Payload,
	)
	if err != nil {
		return int64(0), util.FormatErrorWithQuery(err, insertHistoryQuery)
	}

	return insertedID, nil
}

// UpdateHistoryAsDone will update the migration record as done.
func (Driver) UpdateHistoryAsDone(ctx context.Context, tx *sql.Tx, migrationDurationNs int64, updatedSchema string, insertedID int64) error {
	const updateHistoryAsDoneQuery = `
		ALTER TABLE
			bytebase.migration_history
		UPDATE
			status = 'DONE',
			execution_duration_ns = $1,
		` + "`schema` = $2" + `
		WHERE id = $3
	`
	_, err := tx.ExecContext(ctx, updateHistoryAsDoneQuery, migrationDurationNs, updatedSchema, insertedID)
	return err
}

// UpdateHistoryAsFailed will update the migration record as failed.
func (Driver) UpdateHistoryAsFailed(ctx context.Context, tx *sql.Tx, migrationDurationNs int64, insertedID int64) error {
	const updateHistoryAsFailedQuery = `
		ALTER TABLE
			bytebase.migration_history
		UPDATE
			status = 'FAILED',
			execution_duration_ns = $1
		WHERE id = $2
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
		// TODO(d): support semantic versioning.
		storedVersion, err := util.ToStoredVersion(false, *v, "")
		if err != nil {
			return nil, err
		}
		paramNames, params = append(paramNames, "version"), append(params, storedVersion)
	}
	if v := find.Source; v != nil {
		paramNames, params = append(paramNames, "source"), append(params, *v)
	}
	var query = baseQuery +
		db.FormatParamNameInNumberedPosition(paramNames) +
		`ORDER BY created_ts DESC`
	if v := find.Limit; v != nil {
		query += fmt.Sprintf(" LIMIT %d", *v)
	}
	history, err := util.FindMigrationHistoryList(ctx, query, params, driver, find, baseQuery)
	// TODO(d): remove this block once all existing customers all migrated to semantic versioning.
	if err != nil {
		if !strings.Contains(err.Error(), "invalid stored version") {
			return nil, err
		}
		if err := driver.updateMigrationHistoryStorageVersion(ctx); err != nil {
			return nil, err
		}
		return util.FindMigrationHistoryList(ctx, query, params, driver, find, baseQuery)
	}
	return history, err
}

func (driver *Driver) updateMigrationHistoryStorageVersion(ctx context.Context) error {
	sqldb, err := driver.GetDbConnection(ctx, "bytebase")
	if err != nil {
		return err
	}
	query := `SELECT id, version FROM bytebase.migration_history`
	rows, err := sqldb.Query(query)
	if err != nil {
		return err
	}
	type ver struct {
		id      int
		version string
	}
	var vers []ver
	for rows.Next() {
		var v ver
		if err := rows.Scan(&v.id, &v.version); err != nil {
			return err
		}
		vers = append(vers, v)
	}
	if err := rows.Close(); err != nil {
		return err
	}

	updateQuery := `
		ALTER TABLE
			bytebase.migration_history
		UPDATE
			version = $1
		WHERE id = $2 AND version = $3
	`
	for _, v := range vers {
		if strings.HasPrefix(v.version, util.NonSemanticPrefix) {
			continue
		}
		newVersion := fmt.Sprintf("%s%s", util.NonSemanticPrefix, v.version)
		if _, err := sqldb.Exec(updateQuery, newVersion, v.id, v.version); err != nil {
			return err
		}
	}
	return nil
}

// Dump and restore
const (
	databaseHeaderFmt = "" +
		"--\n" +
		"-- ClickHouse database structure for `%s`\n" +
		"--\n"
	useDatabaseFmt = "USE `%s`;\n\n"
	tableStmtFmt   = "" +
		"--\n" +
		"-- Table structure for `%s`\n" +
		"--\n" +
		"%s;\n"
	viewStmtFmt = "" +
		"--\n" +
		"-- View structure for `%s`\n" +
		"--\n" +
		"%s;\n"
)

// Dump dumps the database.
func (driver *Driver) Dump(ctx context.Context, database string, out io.Writer, schemaOnly bool) error {
	txn, err := driver.db.BeginTx(ctx, &sql.TxOptions{})
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

// dumpTxn will dump the input database. schemaOnly isn't supported yet and true by default.
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
			if _, err := io.WriteString(out, fmt.Sprintf("%s\n", tbl.statement)); err != nil {
				return err
			}
		}
	}

	return nil
}

// getDatabases gets all databases of an instance.
func getDatabases(txn *sql.Tx) ([]string, error) {
	var dbNames []string
	rows, err := txn.Query("SELECT name FROM system.databases")
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

// getTables gets all tables of a database.
func getTables(txn *sql.Tx, dbName string) ([]*tableSchema, error) {
	var tables []*tableSchema
	query := fmt.Sprintf("SELECT name, engine, create_table_query FROM system.tables WHERE database='%s';", dbName)
	rows, err := txn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var tbl tableSchema
		if err := rows.Scan(&tbl.name, &tbl.tableType, &tbl.statement); err != nil {
			return nil, err
		}
		tables = append(tables, &tbl)
	}

	for _, tbl := range tables {
		// Remove the database prefix from statement.
		tbl.statement = strings.ReplaceAll(tbl.statement, fmt.Sprintf(" %s.%s ", dbName, tbl.name), fmt.Sprintf(" %s ", tbl.name))
		if tbl.tableType == "View" {
			tbl.statement = fmt.Sprintf(viewStmtFmt, tbl.name, tbl.statement)
		} else {
			tbl.statement = fmt.Sprintf(tableStmtFmt, tbl.name, tbl.statement)
		}
	}
	return tables, nil
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
