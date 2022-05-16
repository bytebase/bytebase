package snowflake

import (
	"bufio"
	"context"
	"database/sql"
	"fmt"
	"io"
	"strings"

	// embed will embeds the migration schema.
	_ "embed"

	"github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/plugin/db/util"
	snow "github.com/snowflakedb/gosnowflake"
	"go.uber.org/zap"
)

//go:embed snowflake_migration_schema.sql
var migrationSchema string

var (
	systemSchemas = map[string]bool{
		"information_schema": true,
	}
	bytebaseDatabase = "BYTEBASE"
	sysAdminRole     = "SYSADMIN"
	accountAdminRole = "ACCOUNTADMIN"

	_ db.Driver              = (*Driver)(nil)
	_ util.MigrationExecutor = (*Driver)(nil)
)

func init() {
	db.Register(db.Snowflake, newDriver)
}

// Driver is the Snowflake driver.
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

// Open opens a Snowflake driver.
func (driver *Driver) Open(ctx context.Context, dbType db.Type, config db.ConnectionConfig, connCtx db.ConnectionContext) (db.Driver, error) {
	prefixParts, loggedPrefixParts := []string{config.Username}, []string{config.Username}
	if config.Password != "" {
		prefixParts = append(prefixParts, config.Password)
		loggedPrefixParts = append(loggedPrefixParts, "<<redacted password>>")
	}

	var account, host string
	// Host can also be account e.g. xma12345, or xma12345@host_ip where host_ip is the proxy server IP.
	if strings.Contains(config.Host, "@") {
		parts := strings.Split(config.Host, "@")
		if len(parts) != 2 {
			return nil, fmt.Errorf("driver.Open() has invalid host %q", config.Host)
		}
		account, host = parts[0], parts[1]
	} else {
		account = config.Host
	}

	var params []string
	var suffix string
	if host != "" {
		suffix = fmt.Sprintf("%s:%s", host, config.Port)
		params = append(params, fmt.Sprintf("account=%s", account))
	} else {
		suffix = account
	}

	dsn := fmt.Sprintf("%s@%s/%s", strings.Join(prefixParts, ":"), suffix, config.Database)
	loggedDSN := fmt.Sprintf("%s@%s/%s", strings.Join(loggedPrefixParts, ":"), suffix, config.Database)
	if len(params) > 0 {
		dsn = fmt.Sprintf("%s?%s", dsn, strings.Join(params, "&"))
		loggedDSN = fmt.Sprintf("%s?%s", loggedDSN, strings.Join(params, "&"))
	}
	driver.l.Debug("Opening Snowflake driver",
		zap.String("dsn", loggedDSN),
		zap.String("environment", connCtx.EnvironmentName),
		zap.String("database", connCtx.InstanceName),
	)
	db, err := sql.Open("snowflake", dsn)
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
	query := "SELECT CURRENT_VERSION()"
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

func (driver *Driver) useRole(ctx context.Context, role string) error {
	query := fmt.Sprintf("USE ROLE %s", role)
	if _, err := driver.db.ExecContext(ctx, query); err != nil {
		return util.FormatErrorWithQuery(err, query)
	}
	return nil
}

// SyncSchema synces the schema.
func (driver *Driver) SyncSchema(ctx context.Context) ([]*db.User, []*db.Schema, error) {
	// Query user info
	if err := driver.useRole(ctx, accountAdminRole); err != nil {
		return nil, nil, err
	}

	userList, err := driver.getUserList(ctx)
	if err != nil {
		return nil, nil, err
	}

	// Query db info
	databases, err := driver.getDatabases(ctx)
	if err != nil {
		return nil, nil, err
	}

	var schemaList []*db.Schema
	for _, database := range databases {
		if database == bytebaseDatabase {
			continue
		}

		var schema db.Schema
		schema.Name = database
		tableList, viewList, err := driver.syncTableSchema(ctx, database)
		if err != nil {
			return nil, nil, err
		}
		schema.TableList, schema.ViewList = tableList, viewList

		schemaList = append(schemaList, &schema)
	}

	return userList, schemaList, nil
}

func (driver *Driver) syncTableSchema(ctx context.Context, database string) ([]db.Table, []db.View, error) {
	// Query table info
	var excludedSchemaList []string

	// Skip all system schemas.
	for k := range systemSchemas {
		excludedSchemaList = append(excludedSchemaList, fmt.Sprintf("'%s'", k))
	}
	excludeWhere := fmt.Sprintf("LOWER(TABLE_SCHEMA) NOT IN (%s)", strings.Join(excludedSchemaList, ", "))

	// Query column info
	query := fmt.Sprintf(`
		SELECT
			TABLE_SCHEMA,
			TABLE_NAME,
			IFNULL(COLUMN_NAME, ''),
			ORDINAL_POSITION,
			COLUMN_DEFAULT,
			IS_NULLABLE,
			DATA_TYPE,
			IFNULL(CHARACTER_SET_NAME, ''),
			IFNULL(COLLATION_NAME, ''),
			IFNULL(COMMENT, '')
		FROM %s.INFORMATION_SCHEMA.COLUMNS
		WHERE %s`, database, excludeWhere)
	columnRows, err := driver.db.QueryContext(ctx, query)
	if err != nil {
		return nil, nil, util.FormatErrorWithQuery(err, query)
	}
	defer columnRows.Close()

	// schemaName.tableName -> columnList map
	columnMap := make(map[string][]db.Column)
	for columnRows.Next() {
		var schemaName string
		var tableName string
		var nullable string
		var defaultStr sql.NullString
		var column db.Column
		if err := columnRows.Scan(
			&schemaName,
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

		key := fmt.Sprintf("%s.%s", schemaName, tableName)
		columnMap[key] = append(columnMap[key], column)
	}

	query = fmt.Sprintf(`
		SELECT
			TABLE_SCHEMA,
			TABLE_NAME,
			DATE_PART(EPOCH_SECOND, CREATED),
			DATE_PART(EPOCH_SECOND, LAST_ALTERED),
			TABLE_TYPE,
			ROW_COUNT,
			BYTES,
			IFNULL(COMMENT, '')
		FROM %s.INFORMATION_SCHEMA.TABLES
		WHERE TABLE_TYPE = 'BASE TABLE' AND %s`, database, excludeWhere)
	tableRows, err := driver.db.QueryContext(ctx, query)
	if err != nil {
		return nil, nil, util.FormatErrorWithQuery(err, query)
	}
	defer tableRows.Close()

	var tables []db.Table
	for tableRows.Next() {
		var schemaName, tableName string
		var table db.Table
		if err := tableRows.Scan(
			&schemaName,
			&tableName,
			&table.CreatedTs,
			&table.UpdatedTs,
			&table.Type,
			&table.RowCount,
			&table.DataSize,
			&table.Comment,
		); err != nil {
			return nil, nil, err
		}

		table.Name = fmt.Sprintf("%s.%s", schemaName, tableName)
		table.ColumnList = columnMap[table.Name]
		tables = append(tables, table)
	}
	if err := tableRows.Err(); err != nil {
		return nil, nil, err
	}

	query = fmt.Sprintf(`
	SELECT
		TABLE_SCHEMA,
		TABLE_NAME,
		DATE_PART(EPOCH_SECOND, CREATED),
		DATE_PART(EPOCH_SECOND, LAST_ALTERED),
		IFNULL(VIEW_DEFINITION, ''),
		IFNULL(COMMENT, '')
	FROM %s.INFORMATION_SCHEMA.VIEWS
	WHERE %s`, database, excludeWhere)
	viewRows, err := driver.db.QueryContext(ctx, query)
	if err != nil {
		return nil, nil, util.FormatErrorWithQuery(err, query)
	}
	defer viewRows.Close()

	var views []db.View
	for viewRows.Next() {
		var schemaName, viewName string
		var createdTs, updatedTs sql.NullInt64
		var view db.View
		if err := viewRows.Scan(
			&schemaName,
			&viewName,
			&createdTs,
			&updatedTs,
			&view.Definition,
			&view.Comment,
		); err != nil {
			return nil, nil, err
		}
		view.Name = fmt.Sprintf("%s.%s", schemaName, viewName)
		if createdTs.Valid {
			view.CreatedTs = createdTs.Int64
		}
		if updatedTs.Valid {
			view.UpdatedTs = updatedTs.Int64
		}
		views = append(views, view)
	}
	if err := viewRows.Err(); err != nil {
		return nil, nil, err
	}

	return tables, views, nil
}

func (driver *Driver) getDatabases(ctx context.Context) ([]string, error) {
	txn, err := driver.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer txn.Rollback()

	databases, err := getDatabasesTxn(ctx, txn)
	if err != nil {
		return nil, err
	}

	if err := txn.Commit(); err != nil {
		return nil, err
	}

	return databases, nil
}

func getDatabasesTxn(ctx context.Context, tx *sql.Tx) ([]string, error) {
	if _, err := tx.ExecContext(ctx, fmt.Sprintf("USE ROLE %s", accountAdminRole)); err != nil {
		return nil, err
	}

	// Filter inbound shared databases because they are immutable and we cannot get their DDLs.
	inboundDatabases := make(map[string]bool)
	shareRows, err := tx.Query("SHOW SHARES;")
	if err != nil {
		return nil, err
	}
	defer shareRows.Close()

	cols, err := shareRows.ColumnTypes()
	if err != nil {
		return nil, err
	}
	// created_on, kind, name, database_name.
	if len(cols) < 4 {
		return nil, nil
	}
	values := make([]*sql.NullString, len(cols))
	refs := make([]interface{}, len(cols))
	for i := 0; i < len(cols); i++ {
		refs[i] = &values[i]
	}
	for shareRows.Next() {
		if err := shareRows.Scan(refs...); err != nil {
			return nil, err
		}
		if values[1].String == "INBOUND" {
			inboundDatabases[values[3].String] = true
		}
	}
	if err := shareRows.Err(); err != nil {
		return nil, err
	}

	query := `
		SELECT
			DATABASE_NAME
		FROM SNOWFLAKE.INFORMATION_SCHEMA.DATABASES`
	rows, err := tx.QueryContext(ctx, query)
	if err != nil {
		return nil, util.FormatErrorWithQuery(err, query)
	}
	defer rows.Close()

	var databases []string
	for rows.Next() {
		var name string
		if err := rows.Scan(
			&name,
		); err != nil {
			return nil, err
		}

		if _, ok := inboundDatabases[name]; !ok {
			databases = append(databases, name)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return databases, nil
}

func (driver *Driver) getUserList(ctx context.Context) ([]*db.User, error) {
	query := `
		SELECT
			GRANTEE_NAME,
			ROLE
		FROM SNOWFLAKE.ACCOUNT_USAGE.GRANTS_TO_USERS
`
	grants := make(map[string][]string)

	grantRows, err := driver.db.QueryContext(ctx, query)
	if err != nil {
		return nil, util.FormatErrorWithQuery(err, query)
	}
	defer grantRows.Close()

	for grantRows.Next() {
		var name, role string
		if err := grantRows.Scan(
			&name,
			&role,
		); err != nil {
			return nil, err
		}
		grants[name] = append(grants[name], role)
	}

	// Query user info
	query = `
	  SELECT
			name
		FROM SNOWFLAKE.ACCOUNT_USAGE.USERS
	`
	var userList []*db.User
	userRows, err := driver.db.QueryContext(ctx, query)

	if err != nil {
		return nil, util.FormatErrorWithQuery(err, query)
	}
	defer userRows.Close()

	for userRows.Next() {
		var name string
		if err := userRows.Scan(
			&name,
		); err != nil {
			return nil, err
		}

		userList = append(userList, &db.User{
			Name:  name,
			Grant: strings.Join(grants[name], ", "),
		})
	}
	return userList, nil
}

// Execute executes a SQL statement.
func (driver *Driver) Execute(ctx context.Context, statement string) error {
	count := 0
	f := func(stmt string) error {
		count++
		return nil
	}
	sc := bufio.NewScanner(strings.NewReader(statement))
	if err := util.ApplyMultiStatements(sc, f); err != nil {
		return err
	}

	if count <= 0 {
		return nil
	}

	if err := driver.useRole(ctx, sysAdminRole); err != nil {
		return nil
	}
	tx, err := driver.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	mctx, err := snow.WithMultiStatement(ctx, count)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(mctx, statement); err != nil {
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
	exist, err := driver.hasBytebaseDatabase(ctx)
	if err != nil {
		return false, err
	}
	if !exist {
		return true, nil
	}

	const query = `
		SELECT
		    1
		FROM BYTEBASE.INFORMATION_SCHEMA.TABLES
		WHERE TABLE_SCHEMA='PUBLIC' AND TABLE_NAME = 'MIGRATION_HISTORY'
	`
	return util.NeedsSetupMigrationSchema(ctx, driver.db, query)
}

func (driver *Driver) hasBytebaseDatabase(ctx context.Context) (bool, error) {
	databases, err := driver.getDatabases(ctx)
	if err != nil {
		return false, err
	}
	exist := false
	for _, database := range databases {
		if database == bytebaseDatabase {
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
		// Should use role SYSADMIN.
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
		SELECT MAX(version) FROM bytebase.public.migration_history
		WHERE namespace = ? AND sequence >= ?
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
		SELECT MAX(sequence) FROM bytebase.public.migration_history
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
func (Driver) InsertPendingHistory(ctx context.Context, tx *sql.Tx, sequence int, prevSchema string, m *db.MigrationInfo, storedVersion, statement string) (int64, error) {
	const insertHistoryQuery = `
		INSERT INTO bytebase.public.migration_history (
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
			schema,
			schema_prev,
			execution_duration_ns,
			issue_id,
			payload
		)
		VALUES (?, DATE_PART(EPOCH_SECOND, CURRENT_TIMESTAMP()), ?, DATE_PART(EPOCH_SECOND, CURRENT_TIMESTAMP()), ?, ?, ?, ?,  ?, 'PENDING', ?, ?, ?, ?, ?, 0, ?, ?)
	`
	var insertedID int64
	maxIDQuery := "SELECT MAX(id)+1 FROM bytebase.public.migration_history"
	rows, err := tx.QueryContext(ctx, maxIDQuery)
	if err != nil {
		return int64(0), util.FormatErrorWithQuery(err, maxIDQuery)
	}
	defer rows.Close()
	var id sql.NullInt64
	for rows.Next() {
		if err := rows.Scan(
			&id,
		); err != nil {
			return int64(0), util.FormatErrorWithQuery(err, maxIDQuery)
		}
	}
	if err := rows.Err(); err != nil {
		return int64(0), util.FormatErrorWithQuery(err, maxIDQuery)
	}
	if id.Valid {
		insertedID = id.Int64
	} else {
		insertedID = 1
	}

	_, err = tx.ExecContext(ctx, insertHistoryQuery,
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
	)
	if err != nil {
		return int64(0), util.FormatErrorWithQuery(err, insertHistoryQuery)
	}
	return insertedID, nil
}

// UpdateHistoryAsDone will update the migration record as done.
func (Driver) UpdateHistoryAsDone(ctx context.Context, tx *sql.Tx, migrationDurationNs int64, updatedSchema string, insertedID int64) error {
	const updateHistoryAsDoneQuery = `
		UPDATE
			bytebase.public.migration_history
		SET
			status = 'DONE',
			execution_duration_ns = ?,
			schema = ?
		WHERE id = ?
	`
	_, err := tx.ExecContext(ctx, updateHistoryAsDoneQuery, migrationDurationNs, updatedSchema, insertedID)
	return err
}

// UpdateHistoryAsFailed will update the migration record as failed.
func (Driver) UpdateHistoryAsFailed(ctx context.Context, tx *sql.Tx, migrationDurationNs int64, insertedID int64) error {
	const updateHistoryAsFailedQuery = `
		UPDATE
			bytebase.public.migration_history
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
	if err := driver.useRole(ctx, sysAdminRole); err != nil {
		return int64(0), "", err
	}
	return util.ExecuteMigration(ctx, driver.l, driver, m, statement, bytebaseDatabase)
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
		schema,
		schema_prev,
		execution_duration_ns,
		issue_id,
		payload
		FROM bytebase.public.migration_history `
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
		db.FormatParamNameInQuestionMark(paramNames) +
		`ORDER BY created_ts DESC`
	if v := find.Limit; v != nil {
		query += fmt.Sprintf(" LIMIT %d", *v)
	}
	// TODO(zp):  modified param `bytebaseDatabase` of `util.FindMigrationHistoryList` when we support *snowflake* database level.
	history, err := util.FindMigrationHistoryList(ctx, query, params, driver, bytebaseDatabase, find, baseQuery)
	// TODO(d): remove this block once all existing customers all migrated to semantic versioning.
	if err != nil {
		if !strings.Contains(err.Error(), "invalid stored version") {
			return nil, err
		}
		if err := driver.updateMigrationHistoryStorageVersion(ctx); err != nil {
			return nil, err
		}
		return util.FindMigrationHistoryList(ctx, query, params, driver, bytebaseDatabase, find, baseQuery)
	}
	return history, err
}

func (driver *Driver) updateMigrationHistoryStorageVersion(ctx context.Context) error {
	sqldb, err := driver.GetDbConnection(ctx, "bytebase")
	if err != nil {
		return err
	}
	query := `SELECT id, version FROM bytebase.public.migration_history`
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
			bytebase.public.migration_history
		SET
			version = ?
		WHERE id = ? AND version = ?
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
		"-- Snowflake database structure for %s\n" +
		"--\n"
)

// Dump dumps the database.
func (driver *Driver) Dump(ctx context.Context, database string, out io.Writer, schemaOnly bool) (string, error) {
	txn, err := driver.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return "{}", err
	}
	defer txn.Rollback()

	if err := dumpTxn(ctx, txn, database, out, schemaOnly); err != nil {
		return "{}", err
	}

	if err := txn.Commit(); err != nil {
		return "{}", err
	}

	return "{}", nil
}

// dumpTxn will dump the input database. schemaOnly isn't supported yet and true by default.
func dumpTxn(ctx context.Context, txn *sql.Tx, database string, out io.Writer, schemaOnly bool) error {
	// Find all dumpable databases
	var dumpableDbNames []string
	if database != "" {
		dumpableDbNames = []string{database}
	} else {
		var err error
		dumpableDbNames, err = getDatabasesTxn(ctx, txn)
		if err != nil {
			return fmt.Errorf("failed to get databases: %s", err)
		}
	}

	// Use ACCOUNTADMIN role to dump database;
	if _, err := txn.ExecContext(ctx, fmt.Sprintf("USE ROLE %s", accountAdminRole)); err != nil {
		return err
	}

	for _, dbName := range dumpableDbNames {
		// includeCreateDatabaseStmt should be false if dumping a single database.
		dumpSingleDatabase := len(dumpableDbNames) == 1
		dbName = strings.ToUpper(dbName)
		if err := dumpOneDatabase(ctx, txn, dbName, out, schemaOnly, dumpSingleDatabase); err != nil {
			return err
		}
	}

	return nil
}

// dumpOneDatabase will dump the database DDL schema for a database.
// Note: this operation is not supported on shared databases, e.g. SNOWFLAKE_SAMPLE_DATA.
func dumpOneDatabase(ctx context.Context, txn *sql.Tx, database string, out io.Writer, schemaOnly bool, dumpSingleDatabase bool) error {
	if !dumpSingleDatabase {
		// Database header.
		header := fmt.Sprintf(databaseHeaderFmt, database)
		if _, err := io.WriteString(out, header); err != nil {
			return err
		}
	}

	query := fmt.Sprintf(`SELECT GET_DDL('DATABASE', '%s', true)`, database)
	rows, err := txn.QueryContext(ctx, query)
	if err != nil {
		return util.FormatErrorWithQuery(err, query)
	}
	defer rows.Close()

	var databaseDDL string
	for rows.Next() {
		if err := rows.Scan(
			&databaseDDL,
		); err != nil {
			return err
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	// Transform1: if dumpSingleDatabase, we should remove `create or replace database` statement.
	if dumpSingleDatabase {
		lines := strings.Split(databaseDDL, "\n")
		if len(lines) >= 2 {
			lines = lines[2:]
		}
		databaseDDL = strings.Join(lines, "\n")
	}

	// Transform2: remove "create or replace schema PUBLIC;\n\n" because it's created by default.
	schemaStmt := fmt.Sprintf("create or replace schema %s.PUBLIC;", database)
	databaseDDL = strings.ReplaceAll(databaseDDL, schemaStmt+"\n\n", "")
	// If this is the last statement.
	databaseDDL = strings.ReplaceAll(databaseDDL, schemaStmt, "")

	var lines []string
	for _, line := range strings.Split(databaseDDL, "\n") {
		if strings.HasPrefix(strings.ToLower(line), "create ") {
			// Transform3: Remove "DEMO_DB." quantifier.
			line = strings.ReplaceAll(line, fmt.Sprintf(" %s.", database), " ")

			// Transform4 (Important!): replace all `create or replace ` with `create ` to not break existing schema by any chance.
			line = strings.ReplaceAll(line, "create or replace ", "create ")
		}
		lines = append(lines, line)
	}
	databaseDDL = strings.Join(lines, "\n")

	if _, err := io.WriteString(out, databaseDDL); err != nil {
		return err
	}

	return nil
}

// Restore restores a database.
func (driver *Driver) Restore(ctx context.Context, sc *bufio.Scanner) (err error) {
	if err := driver.useRole(ctx, sysAdminRole); err != nil {
		return nil
	}
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
