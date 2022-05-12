package pg

import (
	"bufio"
	"context"
	"database/sql"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"github.com/blang/semver/v4"
	"github.com/bytebase/bytebase/common"

	// embed will embeds the migration schema.
	_ "embed"

	// Import pg driver.
	// init() in pgx/v4/stdlib will register it's pgx driver
	_ "github.com/jackc/pgx/v4/stdlib"

	"github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/plugin/db/util"
	"go.uber.org/zap"
)

//go:embed pg_migration_schema.sql
var migrationSchema string

var (
	systemDatabases = map[string]bool{
		"template0": true,
		"template1": true,
	}
	ident             = regexp.MustCompile(`(?i)^[a-z_][a-z0-9_$]*$`)
	databaseHeaderFmt = "" +
		"--\n" +
		"-- PostgreSQL database structure for %s\n" +
		"--\n"
	useDatabaseFmt             = "\\connect %s;\n\n"
	createBytebaseDatabaseStmt = "CREATE DATABASE bytebase;"

	// driverName is the driver name that our driver dependence register, now is "pgx".
	driverName = "pgx"

	_ db.Driver              = (*Driver)(nil)
	_ util.MigrationExecutor = (*Driver)(nil)
)

func init() {
	db.Register(db.Postgres, newDriver)
}

// Driver is the Postgres driver.
type Driver struct {
	l             *zap.Logger
	connectionCtx db.ConnectionContext

	db      *sql.DB
	baseDSN string

	// strictDatabase should be used only if the user gives only a database instead of a whole instance to access.
	strictDatabase string
}

func newDriver(config db.DriverConfig) db.Driver {
	return &Driver{
		l: config.Logger,
	}
}

// Open opens a Postgres driver.
func (driver *Driver) Open(ctx context.Context, dbType db.Type, config db.ConnectionConfig, connCtx db.ConnectionContext) (db.Driver, error) {
	if (config.TLSConfig.SslCert == "" && config.TLSConfig.SslKey != "") ||
		(config.TLSConfig.SslCert != "" && config.TLSConfig.SslKey == "") {
		return nil, fmt.Errorf("ssl-cert and ssl-key must be both set or unset")
	}

	dsn, err := guessDSN(
		config.Username,
		config.Password,
		config.Host,
		config.Port,
		config.Database,
		config.TLSConfig.SslCA,
		config.TLSConfig.SslCert,
		config.TLSConfig.SslKey,
	)
	if err != nil {
		return nil, err
	}
	if config.ReadOnly {
		dsn = fmt.Sprintf("%s default_transaction_read_only=true", dsn)
	}
	driver.baseDSN = dsn
	driver.connectionCtx = connCtx
	if config.StrictUseDb {
		driver.strictDatabase = config.Database
	}

	db, err := sql.Open(driverName, dsn)
	if err != nil {
		return nil, err
	}
	driver.db = db
	return driver, nil
}

// guessDSN will guess the dsn of a valid DB connection.
func guessDSN(username, password, hostname, port, database, sslCA, sslCert, sslKey string) (string, error) {
	// dbname is guessed if not specified.
	m := map[string]string{
		"host":     hostname,
		"port":     port,
		"user":     username,
		"password": password,
	}

	if sslCA == "" {
		// We should use the default connection dsn without setting sslmode.
		// Some provider might still perform default SSL check at the server side so we
		// shouldn't disable sslmode at the client side.
		// m["sslmode"] = "disable"
	} else {
		m["sslmode"] = "verify-ca"
		m["sslrootcert"] = sslCA
		if sslCert != "" && sslKey != "" {
			m["sslcert"] = sslCert
			m["sslkey"] = sslKey
		}
	}
	var tokens []string
	for k, v := range m {
		if v != "" {
			tokens = append(tokens, fmt.Sprintf("%s=%s", k, v))
		}
	}
	dsn := strings.Join(tokens, " ")

	var guesses []string
	if database != "" {
		guesses = append(guesses, dsn+" dbname="+database)
	} else {
		// Guess default database postgres, template1.
		guesses = append(guesses, dsn)
		guesses = append(guesses, dsn+" dbname=bytebase")
		guesses = append(guesses, dsn+" dbname=postgres")
		guesses = append(guesses, dsn+" dbname=template1")
	}

	for _, dsn := range guesses {
		db, err := sql.Open(driverName, dsn)
		if err != nil {
			continue
		}
		defer db.Close()

		if err = db.Ping(); err != nil {
			continue
		}
		return dsn, nil
	}

	if database != "" {
		return "", fmt.Errorf("cannot connecting %q, make sure the connection info is correct and the database exists", database)
	}
	return "", fmt.Errorf("cannot connecting instance, make sure the connection info is correct")
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
	if err := driver.switchDatabase(database); err != nil {
		return nil, err
	}
	return driver.db, nil
}

// GetVersion gets the version of Postgres server.
func (driver *Driver) GetVersion(ctx context.Context) (string, error) {
	query := "SHOW server_version"
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
	excludedDatabaseList := map[string]bool{
		// Skip our internal "bytebase" database
		"bytebase": true,
		// Skip internal databases from cloud service providers
		// see https://github.com/bytebase/bytebase/issues/30
		// aws
		"rdsadmin": true,
		// gcp
		"cloudsql": true,
	}
	// Skip all system databases
	for k := range systemDatabases {
		excludedDatabaseList[k] = true
	}

	// Query user info
	userList, err := driver.getUserList(ctx)
	if err != nil {
		return nil, nil, err
	}

	// Query db info
	databases, err := driver.getDatabases()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get databases: %s", err)
	}

	var schemaList []*db.Schema
	for _, database := range databases {
		dbName := database.name
		if _, ok := excludedDatabaseList[dbName]; ok {
			continue
		}

		var schema db.Schema
		schema.Name = dbName
		schema.CharacterSet = database.encoding
		schema.Collation = database.collate

		sqldb, err := driver.GetDbConnection(ctx, dbName)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get database connection for %q: %s", dbName, err)
		}
		txn, err := sqldb.BeginTx(ctx, nil)
		if err != nil {
			return nil, nil, err
		}
		defer txn.Rollback()

		// Index statements.
		indicesMap := make(map[string][]*indexSchema)
		indices, err := getIndices(txn)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get indices from database %q: %s", dbName, err)
		}
		for _, idx := range indices {
			key := fmt.Sprintf("%s.%s", idx.schemaName, idx.tableName)
			indicesMap[key] = append(indicesMap[key], idx)
		}

		// Table statements.
		tables, err := getPgTables(txn)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get tables from database %q: %s", dbName, err)
		}
		for _, tbl := range tables {
			var dbTable db.Table
			dbTable.Name = fmt.Sprintf("%s.%s", tbl.schemaName, tbl.name)
			dbTable.Type = "BASE TABLE"
			dbTable.Comment = tbl.comment
			dbTable.RowCount = tbl.rowCount
			dbTable.DataSize = tbl.tableSizeByte
			dbTable.IndexSize = tbl.indexSizeByte
			for _, col := range tbl.columns {
				var dbColumn db.Column
				dbColumn.Name = col.columnName
				dbColumn.Position = col.ordinalPosition
				dbColumn.Default = &col.columnDefault
				dbColumn.Type = col.dataType
				dbColumn.Nullable = col.isNullable
				dbColumn.Collation = col.collationName
				dbColumn.Comment = col.comment
				dbTable.ColumnList = append(dbTable.ColumnList, dbColumn)
			}
			indices := indicesMap[dbTable.Name]
			for _, idx := range indices {
				for i, colExp := range idx.columnExpressions {
					var dbIndex db.Index
					dbIndex.Name = idx.name
					dbIndex.Expression = colExp
					dbIndex.Position = i + 1
					dbIndex.Type = idx.methodType
					dbIndex.Unique = idx.unique
					dbIndex.Comment = idx.comment
					dbTable.IndexList = append(dbTable.IndexList, dbIndex)
				}
			}

			schema.TableList = append(schema.TableList, dbTable)
		}
		// View statements.
		views, err := getViews(txn)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get views from database %q: %s", dbName, err)
		}
		for _, view := range views {
			var dbView db.View
			dbView.Name = fmt.Sprintf("%s.%s", view.schemaName, view.name)
			// Postgres does not store
			dbView.CreatedTs = time.Now().Unix()
			dbView.Definition = view.definition
			dbView.Comment = view.comment

			schema.ViewList = append(schema.ViewList, dbView)
		}

		if err := txn.Commit(); err != nil {
			return nil, nil, err
		}

		schemaList = append(schemaList, &schema)
	}

	return userList, schemaList, err
}

func (driver *Driver) getUserList(ctx context.Context) ([]*db.User, error) {
	// Query user info
	query := `
		SELECT usename AS role_name,
			CASE
				 WHEN usesuper AND usecreatedb THEN
				 CAST('superuser, create database' AS pg_catalog.text)
				 WHEN usesuper THEN
					CAST('superuser' AS pg_catalog.text)
				 WHEN usecreatedb THEN
					CAST('create database' AS pg_catalog.text)
				 ELSE
					CAST('' AS pg_catalog.text)
			END role_attributes
		FROM pg_catalog.pg_user
		ORDER BY role_name
			`
	var userList []*db.User
	userRows, err := driver.db.QueryContext(ctx, query)
	if err != nil {
		return nil, util.FormatErrorWithQuery(err, query)
	}
	defer userRows.Close()

	for userRows.Next() {
		var role string
		var attr string
		if err := userRows.Scan(
			&role,
			&attr,
		); err != nil {
			return nil, err
		}

		userList = append(userList, &db.User{
			Name:  role,
			Grant: attr,
		})
	}
	return userList, nil
}

// Execute executes a SQL statement.
func (driver *Driver) Execute(ctx context.Context, statement string) error {
	var remainingStmts []string
	f := func(stmt string) error {
		stmt = strings.TrimLeft(stmt, " \t")
		if strings.HasPrefix(stmt, "CREATE DATABASE ") {
			// We don't use transaction for creating databases in Postgres.
			// https://github.com/bytebase/bytebase/issues/202
			if _, err := driver.db.ExecContext(ctx, stmt); err != nil {
				return err
			}
		} else if strings.HasPrefix(stmt, "\\connect ") {
			// For the case of `\connect "dbname";`, we need to use GetDbConnection() instead of executing the statement.
			parts := strings.Split(stmt, `"`)
			if len(parts) != 3 {
				return fmt.Errorf("invalid statement %q", stmt)
			}
			_, err := driver.GetDbConnection(ctx, parts[1])
			return err
		} else {
			remainingStmts = append(remainingStmts, stmt)
		}
		return nil
	}
	sc := bufio.NewScanner(strings.NewReader(statement))
	if err := util.ApplyMultiStatements(sc, f); err != nil {
		return err
	}

	if len(remainingStmts) == 0 {
		return nil
	}

	tx, err := driver.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err = tx.ExecContext(ctx, strings.Join(remainingStmts, "\n")); err == nil {
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
	// Don't use `bytebase` when user gives database instead of instance.
	if !driver.strictUseDb() {
		exist, err := driver.hasBytebaseDatabase(ctx)
		if err != nil {
			return false, err
		}
		if !exist {
			return true, nil
		}
		if err := driver.switchDatabase(db.BytebaseDatabase); err != nil {
			return false, err
		}
	}

	const query = `
		SELECT
		    1
		FROM information_schema.tables
		WHERE table_name = 'migration_history'
	`

	return util.NeedsSetupMigrationSchema(ctx, driver.db, query)
}

func (driver *Driver) hasBytebaseDatabase(ctx context.Context) (bool, error) {
	databases, err := driver.getDatabases()
	if err != nil {
		return false, err
	}
	exist := false
	for _, database := range databases {
		if database.name == db.BytebaseDatabase {
			exist = true
			break
		}
	}
	return exist, nil
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

		// Only try to create `bytebase` db when user provide an instance
		if !driver.strictUseDb() {
			exist, err := driver.hasBytebaseDatabase(ctx)
			if err != nil {
				driver.l.Error("Failed to find bytebase database.",
					zap.Error(err),
					zap.String("environment", driver.connectionCtx.EnvironmentName),
					zap.String("database", driver.connectionCtx.InstanceName),
				)
				return fmt.Errorf("failed to find bytebase database error: %v", err)
			}

			if !exist {
				// Create `bytebase` database
				if _, err := driver.db.ExecContext(ctx, createBytebaseDatabaseStmt); err != nil {
					driver.l.Error("Failed to create bytebase database.",
						zap.Error(err),
						zap.String("environment", driver.connectionCtx.EnvironmentName),
						zap.String("database", driver.connectionCtx.InstanceName),
					)
					return util.FormatErrorWithQuery(err, createBytebaseDatabaseStmt)
				}
			}

			if err := driver.switchDatabase(db.BytebaseDatabase); err != nil {
				driver.l.Error("Failed to switch to bytebase database.",
					zap.Error(err),
					zap.String("environment", driver.connectionCtx.EnvironmentName),
					zap.String("database", driver.connectionCtx.InstanceName),
				)
				return fmt.Errorf("failed to switch to bytebase database error: %v", err)
			}
		}

		// Create `migration_history` table
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

// FindLargestVersionSinceBaseline will find the largest version since last baseline or branch.
func (driver Driver) FindLargestVersionSinceBaseline(ctx context.Context, tx *sql.Tx, namespace string) (*string, error) {
	largestBaselineSequence, err := driver.FindLargestSequence(ctx, tx, namespace, true /* baseline */)
	if err != nil {
		return nil, err
	}
	const getLargestVersionSinceLastBaselineQuery = `
		SELECT MAX(version) FROM migration_history
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
		SELECT MAX(sequence) FROM migration_history
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
	INSERT INTO migration_history (
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
		` + `"schema",` + `
		schema_prev,
		execution_duration_ns,
		issue_id,
		payload
	)
	VALUES ($1, EXTRACT(epoch from NOW()), $2, EXTRACT(epoch from NOW()), $3, $4, $5, $6, $7, 'PENDING', $8, $9, $10, $11, $12, 0, $13, $14)
	RETURNING id
	`
	var insertedID int64
	if err := tx.QueryRowContext(ctx, insertHistoryQuery,
		m.Creator,
		m.Creator,
		m.ReleaseVersion,
		m.Namespace,
		sequence,
		m.Source,
		m.Type,
		storedVersion,
		m.Description,
		statement,
		prevSchema,
		prevSchema,
		m.IssueID,
		m.Payload,
	).Scan(&insertedID); err != nil {
		return 0, err
	}
	return insertedID, nil
}

// UpdateHistoryAsDone will update the migration record as done.
func (Driver) UpdateHistoryAsDone(ctx context.Context, tx *sql.Tx, migrationDurationNs int64, updatedSchema string, insertedID int64) error {
	const updateHistoryAsDoneQuery = `
	UPDATE
		migration_history
	SET
		status = 'DONE',
		execution_duration_ns = $1,
		"schema" = $2
	WHERE id = $3
	`
	_, err := tx.ExecContext(ctx, updateHistoryAsDoneQuery, migrationDurationNs, updatedSchema, insertedID)
	return err
}

// UpdateHistoryAsFailed will update the migration record as failed.
func (Driver) UpdateHistoryAsFailed(ctx context.Context, tx *sql.Tx, migrationDurationNs int64, insertedID int64) error {
	const updateHistoryAsFailedQuery = `
	UPDATE
		migration_history
	SET
		status = 'FAILED',
		execution_duration_ns = $1
	WHERE id = $2
	`
	_, err := tx.ExecContext(ctx, updateHistoryAsFailedQuery, migrationDurationNs, insertedID)
	return err
}

// ExecuteMigration will execute the migration.
func (driver *Driver) ExecuteMigration(ctx context.Context, m *db.MigrationInfo, statement string) (int64, string, error) {
	if driver.strictUseDb() {
		return util.ExecuteMigration(ctx, driver.l, driver, m, statement, driver.strictDatabase)
	}
	return util.ExecuteMigration(ctx, driver.l, driver, m, statement, db.BytebaseDatabase)
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
		` + `"schema",` + `
		schema_prev,
		execution_duration_ns,
		issue_id,
		payload
		FROM migration_history `
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

	database := db.BytebaseDatabase
	if driver.strictUseDb() {
		database = driver.strictDatabase
	}
	history, err := util.FindMigrationHistoryList(ctx, query, params, driver, database, find, baseQuery)
	// TODO(d): remove this block once all existing customers all migrated to semantic versioning.
	// Skip this backfill for bytebase's database "bb" with user "bb". We will use the one in pg_engine.go instead.
	isBytebaseDatabase := strings.Contains(driver.baseDSN, "user=bb") && strings.Contains(driver.baseDSN, "host=/tmp")
	if err != nil && !isBytebaseDatabase {
		if !strings.Contains(err.Error(), "invalid stored version") {
			return nil, err
		}
		if err := driver.updateMigrationHistoryStorageVersion(ctx); err != nil {
			return nil, err
		}
		return util.FindMigrationHistoryList(ctx, query, params, driver, db.BytebaseDatabase, find, baseQuery)
	}
	return history, err
}

func (driver *Driver) updateMigrationHistoryStorageVersion(ctx context.Context) error {
	var sqldb *sql.DB
	var err error
	if !driver.strictUseDb() {
		sqldb, err = driver.GetDbConnection(ctx, db.BytebaseDatabase)
	}
	if err != nil {
		return err
	}

	query := `SELECT id, version FROM migration_history`
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
		UPDATE
			migration_history
		SET
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

// Dump dumps the database.
func (driver *Driver) Dump(ctx context.Context, database string, out io.Writer, schemaOnly bool) error {
	// pg_dump -d dbName --schema-only+

	// Find all dumpable databases
	databases, err := driver.getDatabases()
	if err != nil {
		return fmt.Errorf("failed to get databases: %s", err)
	}

	var dumpableDbNames []string
	if database != "" {
		exist := false
		for _, n := range databases {
			if n.name == database {
				exist = true
				break
			}
		}
		if !exist {
			return fmt.Errorf("database %s not found", database)
		}
		dumpableDbNames = []string{database}
	} else {
		for _, n := range databases {
			if systemDatabases[n.name] {
				continue
			}
			dumpableDbNames = append(dumpableDbNames, n.name)
		}
	}

	for _, dbName := range dumpableDbNames {
		includeUseDatabase := len(dumpableDbNames) > 1
		if err := driver.dumpOneDatabase(ctx, dbName, out, schemaOnly, includeUseDatabase); err != nil {
			return err
		}
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

func (driver *Driver) dumpOneDatabase(ctx context.Context, database string, out io.Writer, schemaOnly bool, includeUseDatabase bool) error {
	if err := driver.switchDatabase(database); err != nil {
		return err
	}

	version, err := driver.GetVersion(ctx)
	if err != nil {
		return err
	}
	semverVersion, err := semver.ParseTolerant(version)
	if err != nil {
		return err
	}

	txn, err := driver.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer txn.Rollback()

	// Database statement.
	if includeUseDatabase {
		// Database header.
		header := fmt.Sprintf(databaseHeaderFmt, database)
		if _, err := io.WriteString(out, header); err != nil {
			return err
		}
		// Use database statement.
		useStmt := fmt.Sprintf(useDatabaseFmt, database)
		if _, err := io.WriteString(out, useStmt); err != nil {
			return err
		}
	}

	// Schema statements.
	schemas, err := getPgSchemas(txn)
	if err != nil {
		return err
	}
	for _, schema := range schemas {
		if _, err := io.WriteString(out, schema.Statement()); err != nil {
			return err
		}
	}

	// Sequence statements.
	seqs, err := getSequences(txn)
	if err != nil {
		return fmt.Errorf("failed to get sequences from database %q: %s", database, err)
	}
	for _, seq := range seqs {
		if _, err := io.WriteString(out, seq.Statement()); err != nil {
			return err
		}
	}

	// Table statements.
	tables, err := getPgTables(txn)
	if err != nil {
		return fmt.Errorf("failed to get tables from database %q: %s", database, err)
	}

	constraints := make(map[string]bool)
	for _, tbl := range tables {
		if _, err := io.WriteString(out, tbl.Statement()); err != nil {
			return err
		}
		for _, constraint := range tbl.constraints {
			key := fmt.Sprintf("%s.%s.%s", constraint.schemaName, constraint.tableName, constraint.name)
			constraints[key] = true
		}
		if !schemaOnly {
			if err := exportTableData(txn, tbl, out); err != nil {
				return err
			}
		}
	}

	// View statements.
	views, err := getViews(txn)
	if err != nil {
		return fmt.Errorf("failed to get views from database %q: %s", database, err)
	}
	for _, view := range views {
		if _, err := io.WriteString(out, view.Statement()); err != nil {
			return err
		}
	}

	// Index statements.
	indices, err := getIndices(txn)
	if err != nil {
		return fmt.Errorf("failed to get indices from database %q: %s", database, err)
	}
	for _, idx := range indices {
		key := fmt.Sprintf("%s.%s.%s", idx.schemaName, idx.tableName, idx.name)
		if constraints[key] {
			continue
		}
		if _, err := io.WriteString(out, idx.Statement()); err != nil {
			return err
		}
	}

	// Function statements.
	fs, err := getFunctions(txn)
	if err != nil {
		return fmt.Errorf("failed to get functions from database %q: %s", database, err)
	}
	for _, f := range fs {
		if _, err := io.WriteString(out, f.Statement()); err != nil {
			return err
		}
	}

	// Trigger statements.
	triggers, err := getTriggers(txn)
	if err != nil {
		return fmt.Errorf("failed to get triggers from database %q: %s", database, err)
	}
	for _, tr := range triggers {
		if _, err := io.WriteString(out, tr.Statement()); err != nil {
			return err
		}
	}

	// Event statements.
	events, err := getEventTriggers(txn)
	if err != nil {
		return fmt.Errorf("failed to get event triggers from database %q: %s", database, err)
	}
	for _, evt := range events {
		if _, err := io.WriteString(out, evt.Statement(semverVersion.Major)); err != nil {
			return err
		}
	}

	if err := txn.Commit(); err != nil {
		return err
	}

	return nil
}

func (driver *Driver) switchDatabase(dbName string) error {
	if driver.db != nil {
		if err := driver.db.Close(); err != nil {
			return err
		}
	}

	dsn := driver.baseDSN + " dbname=" + dbName
	db, err := sql.Open(driverName, dsn)
	if err != nil {
		return err
	}
	driver.db = db
	return nil
}

// getDatabases gets all databases of an instance.
func (driver *Driver) getDatabases() ([]*pgDatabaseSchema, error) {
	var dbs []*pgDatabaseSchema
	rows, err := driver.db.Query("SELECT datname, pg_encoding_to_char(encoding), datcollate FROM pg_database;")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var d pgDatabaseSchema
		if err := rows.Scan(&d.name, &d.encoding, &d.collate); err != nil {
			return nil, err
		}
		dbs = append(dbs, &d)
	}
	return dbs, nil
}

func (driver *Driver) strictUseDb() bool {
	return len(driver.strictDatabase) != 0
}

// pgDatabaseSchema describes a pg database schema.
type pgDatabaseSchema struct {
	name     string
	encoding string
	collate  string
}

// pgSchema describes a pg schema, a namespace for all schemas.
type pgSchema struct {
	name        string
	schemaOwner string
}

// tableSchema describes the schema of a pg table.
type tableSchema struct {
	schemaName    string
	name          string
	tableowner    string
	comment       string
	rowCount      int64
	tableSizeByte int64
	indexSizeByte int64

	columns     []*columnSchema
	constraints []*tableConstraint
}

// columnSchema describes the schema of a pg table column.
type columnSchema struct {
	columnName             string
	dataType               string
	ordinalPosition        int
	characterMaximumLength string
	columnDefault          string
	isNullable             bool
	collationName          string
	comment                string
}

// tableConstraint describes constraint schema of a pg table.
type tableConstraint struct {
	name       string
	schemaName string
	tableName  string
	constraint string
}

// viewSchema describes the schema of a pg view.
type viewSchema struct {
	schemaName string
	name       string
	definition string
	comment    string
}

// indexSchema describes the schema of a pg index.
type indexSchema struct {
	schemaName string
	name       string
	tableName  string
	statement  string
	unique     bool
	// methodType such as btree.
	methodType        string
	columnExpressions []string
	comment           string
}

// sequencePgSchema describes the schema of a pg sequence.
type sequencePgSchema struct {
	schemaName   string
	name         string
	dataType     string
	startValue   string
	increment    string
	minimumValue string
	maximumValue string
	cycleOption  string
	cache        string
}

// functionSchema describes the schema of a pg function.
type functionSchema struct {
	schemaName string
	name       string
	statement  string
	language   string
	arguments  string
}

// triggerSchema describes the schema of a pg trigger.
type triggerSchema struct {
	name      string
	statement string
}

// eventTriggerSchema describes the schema of a pg event trigger.
type eventTriggerSchema struct {
	name     string
	enabled  string
	event    string
	owner    string
	tags     string
	funcName string
}

// Statement returns the create statement of a Postgres schema.
func (ps *pgSchema) Statement() string {
	return fmt.Sprintf(""+
		"--\n"+
		"-- Schema structure for %s\n"+
		"--\n"+
		"CREATE SCHEMA %s;\n\n", ps.name, ps.name)
}

// Statement returns the create statement of a table.
func (t *tableSchema) Statement() string {
	s := fmt.Sprintf(""+
		"--\n"+
		"-- Table structure for %s.%s\n"+
		"--\n"+
		"CREATE TABLE %s.%s (\n",
		t.schemaName, t.name, t.schemaName, t.name)
	var cols []string
	for _, v := range t.columns {
		cols = append(cols, "  "+v.Statement())
	}
	s += strings.Join(cols, ",\n")
	s += "\n);\n\n"

	// Add constraints such as primary key, unique, or checks.
	for _, constraint := range t.constraints {
		s += fmt.Sprintf("%s\n", constraint.Statement())
	}
	s += "\n"
	return s
}

// Statement returns the statement of a table column.
func (c *columnSchema) Statement() string {
	s := fmt.Sprintf("%s %s", c.columnName, c.dataType)
	if c.characterMaximumLength != "" {
		s += fmt.Sprintf("(%s)", c.characterMaximumLength)
	}
	if !c.isNullable {
		s = s + " NOT NULL"
	}
	if c.columnDefault != "" {
		s += fmt.Sprintf(" DEFAULT %s", c.columnDefault)
	}
	return s
}

// Statement returns the create statement of a table constraint.
func (c *tableConstraint) Statement() string {
	return fmt.Sprintf(""+
		"ALTER TABLE ONLY %s.%s\n"+
		"    ADD CONSTRAINT %s %s;\n",
		c.schemaName, c.tableName, c.name, c.constraint)
}

// Statement returns the create statement of a view.
func (v *viewSchema) Statement() string {
	return fmt.Sprintf(""+
		"--\n"+
		"-- View structure for %s.%s\n"+
		"--\n"+
		"CREATE VIEW %s.%s AS\n%s\n\n",
		v.schemaName, v.name, v.schemaName, v.name, v.definition)
}

// Statement returns the create statement of a sequence.
func (seq *sequencePgSchema) Statement() string {
	s := fmt.Sprintf(""+
		"--\n"+
		"-- Sequence structure for %s.%s\n"+
		"--\n"+
		"CREATE SEQUENCE %s.%s\n"+
		"    AS %s\n"+
		"    START WITH %s\n"+
		"    INCREMENT BY %s\n",
		seq.schemaName, seq.name, seq.schemaName, seq.name, seq.dataType, seq.startValue, seq.increment)
	if seq.minimumValue == "" {
		s += fmt.Sprintf("    MINVALUE %s\n", seq.minimumValue)
	} else {
		s += "    NO MINVALUE\n"
	}
	if seq.maximumValue == "" {
		s += fmt.Sprintf("    MAXVALUE %s\n", seq.maximumValue)
	} else {
		s += "    NO MAXVALUE\n"
	}
	s += fmt.Sprintf("    CACHE %s", seq.cache)
	switch seq.cycleOption {
	case "YES":
		s += "\n    CYCLE;\n"
	case "NO":
		s += ";\n"
	}
	s += "\n"
	return s
}

// Statement returns the create statement of an index.
func (idx indexSchema) Statement() string {
	return fmt.Sprintf(""+
		"--\n"+
		"-- Index structure for %s.%s\n"+
		"--\n"+
		"%s;\n\n",
		idx.schemaName, idx.name, idx.statement)
}

// Statement returns the create statement of a function.
func (f functionSchema) Statement() string {
	return fmt.Sprintf(""+
		"--\n"+
		"-- Function structure for %s.%s\n"+
		"--\n"+
		"%s;\n\n",
		f.schemaName, f.name, f.statement)
}

// Statement returns the create statement of a trigger.
func (t triggerSchema) Statement() string {
	return fmt.Sprintf(""+
		"--\n"+
		"-- Trigger structure for %s\n"+
		"--\n"+
		"%s;\n\n",
		t.name, t.statement)
}

// Statement returns the create statement of an event trigger.
func (t eventTriggerSchema) Statement(majorVersion uint64) string {
	s := fmt.Sprintf(""+
		"--\n"+
		"-- Event trigger structure for %s\n"+
		"--\n",
		t.name)
	s += fmt.Sprintf("CREATE EVENT TRIGGER %s ON %s", t.name, t.event)
	if t.tags != "" {
		s += fmt.Sprintf("\n         WHEN TAG IN (%s)", t.tags)
	}

	// See https://www.postgresql.org/docs/10/sql-createeventtrigger.html,
	// pg with major version below 10 uses PROCEDURE syntax.
	functionSyntax := "FUNCTION"
	if majorVersion == 10 {
		functionSyntax = "PROCEDURE"
	}

	s += fmt.Sprintf("\n   EXECUTE %s %s();\n", functionSyntax, t.funcName)

	if t.enabled != "O" {
		s += fmt.Sprintf("ALTER EVENT TRIGGER %s ", t.name)
		switch t.enabled {
		case "D":
			s += "DISABLE;\n"
		case "A":
			s += "ENABLE ALWAYS;\n"
		case "R":
			s += "ENABLE REPLICA;\n"
		default:
			s += "ENABLE;\n"
		}
	}
	s += "\n"
	return s
}

// getPgSchemas gets all schemas of a database.
func getPgSchemas(txn *sql.Tx) ([]*pgSchema, error) {
	var schemas []*pgSchema
	rows, err := txn.Query("SELECT schema_name, schema_owner FROM information_schema.schemata;")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var schema pgSchema
		if err := rows.Scan(&schema.name, &schema.schemaOwner); err != nil {
			return nil, err
		}
		schema.name = quoteIdentifier(schema.name)
		if ok := pgSystemSchema(schema.name); ok {
			continue
		}
		schemas = append(schemas, &schema)
	}
	return schemas, nil
}

// pgSystemSchema returns whether the schema is a system or user defined schema.
func pgSystemSchema(s string) bool {
	if common.HasPrefixes(s, "pg_toast", "pg_temp") {
		return true
	}
	switch s {
	case "pg_catalog":
		return true
	case "public":
		return true
	case "information_schema":
		return true
	}
	return false
}

// getTables gets all tables of a database.
func getPgTables(txn *sql.Tx) ([]*tableSchema, error) {
	constraints, err := getTableConstraints(txn)
	if err != nil {
		return nil, fmt.Errorf("getTableConstraints() got error: %v", err)
	}

	var tables []*tableSchema
	query := "" +
		"SELECT tbl.schemaname, tbl.tablename, tbl.tableowner, pg_table_size(c.oid), pg_indexes_size(c.oid) " +
		"FROM pg_catalog.pg_tables tbl, pg_catalog.pg_class c " +
		"WHERE schemaname NOT IN ('pg_catalog', 'information_schema') AND tbl.schemaname=c.relnamespace::regnamespace::text AND tbl.tablename = c.relname;"
	rows, err := txn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var tbl tableSchema
		var schemaname, tablename, tableowner string
		var tableSizeByte, indexSizeByte int64
		if err := rows.Scan(&schemaname, &tablename, &tableowner, &tableSizeByte, &indexSizeByte); err != nil {
			return nil, err
		}
		tbl.schemaName = quoteIdentifier(schemaname)
		tbl.name = quoteIdentifier(tablename)
		tbl.tableowner = tableowner
		tbl.tableSizeByte = tableSizeByte
		tbl.indexSizeByte = indexSizeByte

		tables = append(tables, &tbl)
	}

	for _, tbl := range tables {
		if err := getTable(txn, tbl); err != nil {
			return nil, fmt.Errorf("getTable(%q, %q) got error %v", tbl.schemaName, tbl.name, err)
		}
		columns, err := getTableColumns(txn, tbl.schemaName, tbl.name)
		if err != nil {
			return nil, fmt.Errorf("getTableColumns(%q, %q) got error %v", tbl.schemaName, tbl.name, err)
		}
		tbl.columns = columns

		key := fmt.Sprintf("%s.%s", tbl.schemaName, tbl.name)
		tbl.constraints = constraints[key]
	}
	return tables, nil
}

func getTable(txn *sql.Tx, tbl *tableSchema) error {
	countQuery := fmt.Sprintf("SELECT COUNT(1) FROM %s.%s;", tbl.schemaName, tbl.name)
	rows, err := txn.Query(countQuery)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		if err := rows.Scan(&tbl.rowCount); err != nil {
			return err
		}
	}

	commentQuery := fmt.Sprintf("SELECT obj_description('%s.%s'::regclass);", tbl.schemaName, tbl.name)
	crows, err := txn.Query(commentQuery)
	if err != nil {
		return err
	}
	defer crows.Close()

	for crows.Next() {
		var comment sql.NullString
		if err := crows.Scan(&comment); err != nil {
			return err
		}
		tbl.comment = comment.String
	}
	return nil
}

// getTableColumns gets the columns of a table.
func getTableColumns(txn *sql.Tx, schemaName, tableName string) ([]*columnSchema, error) {
	query := `
	SELECT
		cols.column_name,
		cols.data_type,
		cols.ordinal_position,
		cols.character_maximum_length,
		cols.column_default,
		cols.is_nullable,
		cols.collation_name,
		cols.udt_schema,
		cols.udt_name,
		(
			SELECT
					pg_catalog.col_description(c.oid, cols.ordinal_position::int)
			FROM pg_catalog.pg_class c
			WHERE
					c.oid     = (SELECT cols.table_name::regclass::oid) AND
					cols.table_schema=c.relnamespace::regnamespace::text AND
					cols.table_name = c.relname
		) as column_comment
	FROM INFORMATION_SCHEMA.COLUMNS AS cols
	WHERE table_schema=$1 AND table_name=$2;`
	rows, err := txn.Query(query, schemaName, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []*columnSchema
	for rows.Next() {
		var columnName, dataType, isNullable string
		var characterMaximumLength, columnDefault, collationName, udtSchema, udtName, comment sql.NullString
		var ordinalPosition int
		if err := rows.Scan(&columnName, &dataType, &ordinalPosition, &characterMaximumLength, &columnDefault, &isNullable, &collationName, &udtSchema, &udtName, &comment); err != nil {
			return nil, err
		}
		isNullBool, err := convertBoolFromYesNo(isNullable)
		if err != nil {
			return nil, err
		}
		c := columnSchema{
			columnName:             columnName,
			dataType:               dataType,
			ordinalPosition:        ordinalPosition,
			characterMaximumLength: characterMaximumLength.String,
			columnDefault:          columnDefault.String,
			isNullable:             isNullBool,
			collationName:          collationName.String,
			comment:                comment.String,
		}
		if dataType == "USER-DEFINED" {
			c.dataType = fmt.Sprintf("%s.%s", udtSchema.String, udtName.String)
		}
		columns = append(columns, &c)
	}
	return columns, nil
}

func convertBoolFromYesNo(s string) (bool, error) {
	switch s {
	case "YES":
		return true, nil
	case "NO":
		return false, nil
	default:
		return false, fmt.Errorf("unrecognized isNullable type %q", s)
	}
}

// getTableConstraints gets all table constraints of a database.
func getTableConstraints(txn *sql.Tx) (map[string][]*tableConstraint, error) {
	query := "" +
		"SELECT n.nspname, conrelid::regclass, conname, pg_get_constraintdef(c.oid) " +
		"FROM pg_constraint c " +
		"JOIN pg_namespace n ON n.oid = c.connamespace " +
		"WHERE n.nspname NOT IN ('pg_catalog', 'information_schema');"
	ret := make(map[string][]*tableConstraint)
	rows, err := txn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var constraint tableConstraint
		if err := rows.Scan(&constraint.schemaName, &constraint.tableName, &constraint.name, &constraint.constraint); err != nil {
			return nil, err
		}
		if strings.Contains(constraint.tableName, ".") {
			constraint.tableName = constraint.tableName[1+strings.Index(constraint.tableName, "."):]
		}
		constraint.schemaName, constraint.tableName, constraint.name = quoteIdentifier(constraint.schemaName), quoteIdentifier(constraint.tableName), quoteIdentifier(constraint.name)
		key := fmt.Sprintf("%s.%s", constraint.schemaName, constraint.tableName)
		ret[key] = append(ret[key], &constraint)
	}
	return ret, nil
}

// getViews gets all views of a database.
func getViews(txn *sql.Tx) ([]*viewSchema, error) {
	query := "" +
		"SELECT table_schema, table_name, view_definition FROM information_schema.views " +
		"WHERE table_schema NOT IN ('pg_catalog', 'information_schema');"
	var views []*viewSchema
	rows, err := txn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var view viewSchema
		var def sql.NullString
		if err := rows.Scan(&view.schemaName, &view.name, &def); err != nil {
			return nil, err
		}
		// Return error on NULL view definition.
		// https://github.com/bytebase/bytebase/issues/343
		if !def.Valid {
			return nil, fmt.Errorf("schema %q view %q has empty definition; please check whether proper privileges have been granted to Bytebase", view.schemaName, view.name)
		}
		view.schemaName, view.name, view.definition = quoteIdentifier(view.schemaName), quoteIdentifier(view.name), def.String
		views = append(views, &view)
	}

	for _, view := range views {
		if err = getView(txn, view); err != nil {
			return nil, fmt.Errorf("getPgView(%q, %q) got error %v", view.schemaName, view.name, err)
		}
	}
	return views, nil
}

// getView gets the schema of a view.
func getView(txn *sql.Tx, view *viewSchema) error {
	query := fmt.Sprintf("SELECT obj_description('%s.%s'::regclass);", view.schemaName, view.name)
	rows, err := txn.Query(query)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var comment sql.NullString
		if err := rows.Scan(&comment); err != nil {
			return err
		}
		view.comment = comment.String
	}
	return nil
}

// getIndices gets all indices of a database.
func getIndices(txn *sql.Tx) ([]*indexSchema, error) {
	query := "" +
		"SELECT schemaname, tablename, indexname, indexdef " +
		"FROM pg_indexes WHERE schemaname NOT IN ('pg_catalog', 'information_schema');"

	var indices []*indexSchema
	rows, err := txn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var idx indexSchema
		if err := rows.Scan(&idx.schemaName, &idx.tableName, &idx.name, &idx.statement); err != nil {
			return nil, err
		}
		idx.schemaName, idx.tableName, idx.name = quoteIdentifier(idx.schemaName), quoteIdentifier(idx.tableName), quoteIdentifier(idx.name)
		idx.unique = strings.Contains(idx.statement, " UNIQUE INDEX ")
		idx.methodType = getIndexMethodType(idx.statement)
		idx.columnExpressions, err = getIndexColumnExpressions(idx.statement)
		if err != nil {
			return nil, err
		}
		indices = append(indices, &idx)
	}

	for _, idx := range indices {
		if err = getIndex(txn, idx); err != nil {
			return nil, fmt.Errorf("getIndex(%q, %q) got error %v", idx.schemaName, idx.name, err)
		}
	}

	return indices, nil
}

func getIndex(txn *sql.Tx, idx *indexSchema) error {
	commentQuery := fmt.Sprintf("SELECT obj_description('%s.%s'::regclass);", idx.schemaName, idx.name)
	crows, err := txn.Query(commentQuery)
	if err != nil {
		return err
	}
	defer crows.Close()

	for crows.Next() {
		var comment sql.NullString
		if err := crows.Scan(&comment); err != nil {
			return err
		}
		idx.comment = comment.String
	}
	return nil
}

func getIndexMethodType(stmt string) string {
	re := regexp.MustCompile(`USING (\w+) `)
	matches := re.FindStringSubmatch(stmt)
	if len(matches) == 0 {
		return ""
	}
	return matches[1]
}

func getIndexColumnExpressions(stmt string) ([]string, error) {
	rc := regexp.MustCompile(`\((.*)\)`)
	rm := rc.FindStringSubmatch(stmt)
	if len(rm) == 0 {
		return nil, fmt.Errorf("invalid index statement: %q", stmt)
	}
	columnStmt := rm[1]

	var cols []string
	re := regexp.MustCompile(`\(\(.*\)\)`)
	for {
		if len(columnStmt) <= 0 {
			break
		}
		// Get a token
		token := ""
		// Expression has format of "((exp))".
		if strings.HasPrefix(columnStmt, "((") {
			token = re.FindString(columnStmt)
		} else {
			i := strings.Index(columnStmt, ",")
			if i < 0 {
				token = columnStmt
			} else {
				token = columnStmt[:i]
			}
		}
		// Strip token
		if len(token) == 0 {
			return nil, fmt.Errorf("invalid index statement: %q", stmt)
		}
		columnStmt = columnStmt[len(token):]
		cols = append(cols, strings.TrimSpace(token))

		// Trim space and remove a comma to prepare for the next tokenization.
		columnStmt = strings.TrimSpace(columnStmt)
		if len(columnStmt) > 0 && columnStmt[0] == ',' {
			columnStmt = columnStmt[1:]
		}
		columnStmt = strings.TrimSpace(columnStmt)
	}

	return cols, nil
}

// exportTableData gets the data of a table.
func exportTableData(txn *sql.Tx, tbl *tableSchema, out io.Writer) error {
	query := fmt.Sprintf("SELECT * FROM %s.%s;", tbl.schemaName, tbl.name)
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
		stmt := fmt.Sprintf("INSERT INTO %s.%s VALUES (%s);\n", tbl.schemaName, tbl.name, strings.Join(tokens, ", "))
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

// getSequences gets all sequences of a database.
func getSequences(txn *sql.Tx) ([]*sequencePgSchema, error) {
	caches := make(map[string]string)
	query := "SELECT seqclass.relnamespace::regnamespace::text, seqclass.relname, seq.seqcache " +
		"FROM pg_catalog.pg_class AS seqclass " +
		"JOIN pg_catalog.pg_sequence AS seq ON (seq.seqrelid = seqclass.oid);"
	rows, err := txn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var schemaName, seqName, cache string
		if err := rows.Scan(&schemaName, &seqName, &cache); err != nil {
			return nil, err
		}
		schemaName, seqName = quoteIdentifier(schemaName), quoteIdentifier(seqName)
		caches[fmt.Sprintf("%s.%s", schemaName, seqName)] = cache
	}

	var seqs []*sequencePgSchema
	query = "" +
		"SELECT sequence_schema, sequence_name, data_type, start_value, increment, minimum_value, maximum_value, cycle_option " +
		"FROM information_schema.sequences;"
	rows, err = txn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var seq sequencePgSchema

		if err := rows.Scan(&seq.schemaName, &seq.name, &seq.dataType, &seq.startValue, &seq.increment, &seq.minimumValue, &seq.maximumValue, &seq.cycleOption); err != nil {
			return nil, err
		}
		seq.schemaName, seq.name = quoteIdentifier(seq.schemaName), quoteIdentifier(seq.name)
		cache, ok := caches[fmt.Sprintf("%s.%s", seq.schemaName, seq.name)]
		if !ok {
			return nil, fmt.Errorf("cannot find cache value for sequence: %q.%q", seq.schemaName, seq.name)
		}
		seq.cache = cache
		seqs = append(seqs, &seq)
	}

	return seqs, nil
}

// getFunctions gets all functions of a database.
func getFunctions(txn *sql.Tx) ([]*functionSchema, error) {
	query := "" +
		"SELECT n.nspname, p.proname, l.lanname, " +
		"  CASE WHEN l.lanname = 'internal' THEN p.prosrc ELSE pg_get_functiondef(p.oid) END as definition, " +
		"  pg_get_function_arguments(p.oid) " +
		"FROM pg_proc p " +
		"LEFT JOIN pg_namespace n ON p.pronamespace = n.oid " +
		"LEFT JOIN pg_language l ON p.prolang = l.oid " +
		"LEFT JOIN pg_type t ON t.oid = p.prorettype " +
		"WHERE n.nspname NOT IN ('pg_catalog', 'information_schema');"

	var fs []*functionSchema
	rows, err := txn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var f functionSchema
		if err := rows.Scan(&f.schemaName, &f.name, &f.language, &f.statement, &f.arguments); err != nil {
			return nil, err
		}
		f.schemaName, f.name = quoteIdentifier(f.schemaName), quoteIdentifier(f.name)
		fs = append(fs, &f)
	}

	return fs, nil
}

// getTriggers gets all triggers of a database.
func getTriggers(txn *sql.Tx) ([]*triggerSchema, error) {
	query := "SELECT tgname, pg_get_triggerdef(t.oid) FROM pg_trigger AS t;"

	var triggers []*triggerSchema
	rows, err := txn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var t triggerSchema
		if err := rows.Scan(&t.name, &t.statement); err != nil {
			return nil, err
		}
		t.name = quoteIdentifier(t.name)
		triggers = append(triggers, &t)
	}

	return triggers, nil
}

// getEventTriggers gets all event triggers of a database.
func getEventTriggers(txn *sql.Tx) ([]*eventTriggerSchema, error) {
	query := "" +
		"SELECT evtname, evtenabled, evtevent, pg_get_userbyid(evtowner) AS evtowner, " +
		"  array_to_string(array(SELECT quote_literal(x) FROM unnest(evttags) as t(x)), ', ') AS evttags, " +
		"  e.evtfoid::regproc " +
		"FROM pg_event_trigger e;"

	var triggers []*eventTriggerSchema
	rows, err := txn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var t eventTriggerSchema
		if err := rows.Scan(&t.name, &t.enabled, &t.event, &t.owner, &t.tags, &t.funcName); err != nil {
			return nil, err
		}
		t.name = quoteIdentifier(t.name)
		triggers = append(triggers, &t)
	}

	return triggers, nil
}

// quoteIdentifier will quote identifiers including keywords, capital characters, or special characters.
func quoteIdentifier(s string) string {
	quote := false
	if reserved[strings.ToUpper(s)] {
		quote = true
	}
	if !ident.MatchString(s) {
		quote = true
	}
	if quote {
		return fmt.Sprintf("\"%s\"", strings.ReplaceAll(s, "\"", "\"\""))
	}
	return s

}
