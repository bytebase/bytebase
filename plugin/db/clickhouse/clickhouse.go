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

	clickhouse "github.com/ClickHouse/clickhouse-go"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/plugin/db/util"
	"go.uber.org/zap"
)

//go:embed clickhouse_migration_schema.sql
var migrationSchema string

var (
	systemDatabases = map[string]bool{
		"system": true,
	}

	_ db.Driver   = (*Driver)(nil)
	_ util.Driver = (*Driver)(nil)
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
	protocol := "tcp"
	if strings.HasPrefix(config.Host, "/") {
		protocol = "unix"
	}

	params := []string{}

	port := config.Port
	if port == "" {
		port = "9000"
	}
	baseDSN := fmt.Sprintf("%s://%s:%s?", protocol, config.Host, port)

	// Default user name is "default".
	params = append(params, fmt.Sprintf("username=%s", config.Username))
	// Password is constructed later, after loggedDSN is constructed.
	if config.Database != "" {
		params = append(params, fmt.Sprintf("database=%s", config.Database))
	}
	// Set SSL configuration.
	tlsConfig, err := config.TLSConfig.GetSslConfig()
	if err != nil {
		return nil, fmt.Errorf("sql: tls config error: %v", err)
	}
	tlsKey := "db.clickhouse.tls"
	if tlsConfig != nil {
		if err := clickhouse.RegisterTLSConfig(tlsKey, tlsConfig); err != nil {
			return nil, fmt.Errorf("sql: failed to register tls config: %v", err)
		}
		// TLS config is only used during sql.Open, so should be safe to deregister afterwards.
		defer clickhouse.DeregisterTLSConfig(tlsKey)
		params = append(params, fmt.Sprintf("tls_config=%s", tlsKey))
	}

	loggedDSN := fmt.Sprintf("%s%s&password=<<redacted password>>", baseDSN, strings.Join(params, "&"))

	if config.Password != "" {
		params = append(params, fmt.Sprintf("password=%s", config.Password))
	}
	dsn := fmt.Sprintf("%s%s", baseDSN, strings.Join(params, "&"))

	driver.l.Debug("Opening ClickHouse driver",
		zap.String("dsn", loggedDSN),
		zap.String("environment", connCtx.EnvironmentName),
		zap.String("database", connCtx.InstanceName),
	)
	db, err := sql.Open("clickhouse", dsn)
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

// SyncSchema synces the schema.
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
		tableList, ok := columnMap[key]
		if ok {
			columnMap[key] = append(tableList, column)
		} else {
			list := make([]db.Column, 0)
			columnMap[key] = append(list, column)
		}
	}

	// Query table info
	tableWhere := fmt.Sprintf("LOWER(database) NOT IN (%s)", strings.Join(excludedDatabaseList, ", "))
	query = `
			SELECT
				database,
				name,
				engine,
				total_rows,
				total_bytes,
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
		var rowCount, totalBytes sql.NullInt64
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
			if rowCount.Valid {
				table.RowCount = rowCount.Int64
			}
			if totalBytes.Valid {
				table.DataSize = totalBytes.Int64
			}
			table.UpdatedTs = lastUpdatedTime.Unix()
			key := fmt.Sprintf("%s/%s", dbName, name)
			table.ColumnList = columnMap[key]
			tableMap[dbName] = append(tableMap[dbName], table)
		}
	}

	schemaList := make([]*db.Schema, 0)
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
	userList := make([]*db.User, 0)
	userRows, err := driver.db.QueryContext(ctx, query)

	if err != nil {
		return nil, util.FormatErrorWithQuery(err, query)
	}
	defer userRows.Close()

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
func (driver *Driver) Execute(ctx context.Context, statement string, useTransaction bool) error {
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

// QueryString returns the query where clause string for the query parameters.
func (*Driver) QueryString(p *db.QueryParams) string {
	return util.StandardQueryString(p)
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
		if err := driver.Execute(ctx, migrationSchema, true /* useTransaction */); err != nil {
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

func (driver *Driver) RecordPendingMigrationHistory(ctx context.Context, l *zap.Logger, tx *sql.Tx, m *db.MigrationInfo, statement string, sequence int, prevSchema string) (insertedID int64, err error) {
	const (
		maxIDQuery         = "SELECT MAX(id)+1 FROM bytebase.migration_history"
		insertHistoryQuery = `
	INSERT INTO bytebase.migration_history (
		id,
		created_by,
		created_ts,
		updated_by,
		updated_ts,
		release_version,
		namespace,
		sequence,
		engine,
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
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?,  ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`
	)
	rows, err := tx.QueryContext(ctx, maxIDQuery)
	if err != nil {
		return -1, util.FormatErrorWithQuery(err, maxIDQuery)
	}
	defer rows.Close()
	for rows.Next() {
		if err := rows.Scan(
			&insertedID,
		); err != nil {
			return -1, util.FormatErrorWithQuery(err, maxIDQuery)
		}
	}
	if err := rows.Err(); err != nil {
		return -1, util.FormatErrorWithQuery(err, maxIDQuery)
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
		m.Engine,
		m.Type,
		"PENDING",
		m.Version,
		m.Description,
		statement,
		prevSchema,
		prevSchema,
		0,
		m.IssueID,
		m.Payload,
	)
	if err != nil {
		return -1, util.FormatErrorWithQuery(err, insertHistoryQuery)
	}
	return insertedID, nil
}

// ExecuteMigration will execute the migration.
func (driver *Driver) ExecuteMigration(ctx context.Context, m *db.MigrationInfo, statement string) (int64, string, error) {
	insertHistoryQuery := `
	INSERT INTO bytebase.migration_history (
		id,
		created_by,
		created_ts,
		updated_by,
		updated_ts,
		release_version,
		namespace,
		sequence,
		engine,
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
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?,  ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`
	updateHistoryAsDoneQuery := `
		ALTER TABLE
			bytebase.migration_history
		UPDATE
			status = 'DONE',
			execution_duration_ns = ?,
		` + "`schema` = ?" + `
		WHERE id = ?
	`
	updateHistoryAsFailedQuery := `
		ALTER TABLE
			bytebase.migration_history
		UPDATE
			status = 'FAILED',
			execution_duration_ns = ?
		WHERE id = ?
	`

	args := util.MigrationExecutionArgs{
		InsertHistoryQuery:         insertHistoryQuery,
		UpdateHistoryAsDoneQuery:   updateHistoryAsDoneQuery,
		UpdateHistoryAsFailedQuery: updateHistoryAsFailedQuery,
		TablePrefix:                "bytebase.",
	}
	return util.ExecuteMigration(ctx, driver.l, db.ClickHouse, driver, m, statement, args)
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
		engine,
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
	return util.FindMigrationHistoryList(ctx, driver, find, baseQuery)
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
		// Database header.
		header := fmt.Sprintf(databaseHeaderFmt, dbName)
		if _, err := io.WriteString(out, header); err != nil {
			return err
		}
		// Include "USE DATABASE xxx" if dumping multiple databases.
		if len(dumpableDbNames) > 1 {
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
