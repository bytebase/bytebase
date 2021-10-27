package snowflake

import (
	"bufio"
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"io"
	"strings"

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

	_ db.Driver = (*Driver)(nil)
)

func init() {
	db.Register(db.Snowflake, newDriver)
}

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

func (driver *Driver) Open(ctx context.Context, dbType db.Type, config db.ConnectionConfig, connCtx db.ConnectionContext) (db.Driver, error) {
	prefixParts, loggedPrefixParts := []string{config.Username}, []string{config.Username}
	if config.Password != "" {
		prefixParts = append(prefixParts, config.Password)
		loggedPrefixParts = append(loggedPrefixParts, "<<redacted password>>")
	}
	// Host can also be account e.g. xma12345.
	suffixParts := []string{config.Host}
	// TODO(spinningbot): support port later. suffixParts = append(suffixParts, config.Port)

	dsn := fmt.Sprintf("%s@%s/%s", strings.Join(prefixParts, ":"), strings.Join(suffixParts, ":"), config.Database)
	loggedDSN := fmt.Sprintf("%s@%s/%s", strings.Join(loggedPrefixParts, ":"), strings.Join(suffixParts, ":"), config.Database)
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

func (driver *Driver) Close(ctx context.Context) error {
	return driver.db.Close()
}

func (driver *Driver) Ping(ctx context.Context) error {
	return driver.db.PingContext(ctx)
}

func (driver *Driver) GetDbConnection(ctx context.Context, database string) (*sql.DB, error) {
	return driver.db, nil
}

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

func (driver *Driver) UseRole(ctx context.Context, role string) error {
	query := fmt.Sprintf("USE ROLE %s", role)
	if _, err := driver.db.ExecContext(ctx, query); err != nil {
		return util.FormatErrorWithQuery(err, query)
	}
	return nil
}

func (driver *Driver) SyncSchema(ctx context.Context) ([]*db.DBUser, []*db.DBSchema, error) {
	// TODO(spinningbot): implement it.
	// Query user info
	if err := driver.UseRole(ctx, accountAdminRole); err != nil {
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

	var schemaList []*db.DBSchema
	for _, database := range databases {
		if database == bytebaseDatabase {
			continue
		}

		var schema db.DBSchema
		schema.Name = database

		schemaList = append(schemaList, &schema)
	}

	return userList, schemaList, nil
}

func (driver *Driver) getDatabases(ctx context.Context) ([]string, error) {
	if err := driver.UseRole(ctx, accountAdminRole); err != nil {
		return nil, err
	}
	query := `
		SELECT 
			DATABASE_NAME
		FROM SNOWFLAKE.INFORMATION_SCHEMA.DATABASES`
	rows, err := driver.db.QueryContext(ctx, query)
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

		databases = append(databases, name)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return databases, nil
}

func (driver *Driver) getUserList(ctx context.Context) ([]*db.DBUser, error) {
	// Query user info
	query := `
	  SELECT
			name
		FROM SNOWFLAKE.ACCOUNT_USAGE.USERS
	`
	userList := make([]*db.DBUser, 0)
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

		userList = append(userList, &db.DBUser{
			Name: name,
			// TODO(spinningbot): show grants.
			Grant: "",
		})
	}
	return userList, nil
}

func (driver *Driver) Execute(ctx context.Context, statement string) error {
	if err := driver.UseRole(ctx, sysAdminRole); err != nil {
		return nil
	}
	tx, err := driver.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	count := 0
	f := func(stmt string) error {
		count++
		return nil
	}
	sc := bufio.NewScanner(strings.NewReader(statement))
	if err := util.ApplyMultiStatements(sc, f); err != nil {
		return err
	}

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

// Migration related
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

// ExecuteMigration will execute the migration for MySQL.
func (driver *Driver) ExecuteMigration(ctx context.Context, m *db.MigrationInfo, statement string) (int64, string, error) {
	if err := driver.UseRole(ctx, sysAdminRole); err != nil {
		return int64(0), "", err
	}
	insertHistoryQuery := `
		INSERT INTO bytebase.public.migration_history (
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
			schema,
			schema_prev,
			execution_duration,
			issue_id,
			payload
		)
		VALUES (?, DATE_PART(EPOCH_SECOND, CURRENT_TIMESTAMP()), ?, DATE_PART(EPOCH_SECOND, CURRENT_TIMESTAMP()), ?, ?, ?, ?,  ?, 'PENDING', ?, ?, ?, ?, ?, 0, ?, ?)
`
	updateHistoryAsDoneQuery := `
		UPDATE
			bytebase.public.migration_history
		SET
			status = 'DONE',
			execution_duration = ?,
			schema = ?
		WHERE id = ?
	`

	updateHistoryAsFailedQuery := `
		UPDATE
			bytebase.public.migration_history
		SET
			status = 'FAILED',
			execution_duration = ?
		WHERE id = ?
	`

	args := util.MigrationExecutionArgs{
		InsertHistoryQuery:         insertHistoryQuery,
		UpdateHistoryAsDoneQuery:   updateHistoryAsDoneQuery,
		UpdateHistoryAsFailedQuery: updateHistoryAsFailedQuery,
		TablePrefix:                "bytebase.public.",
	}
	return util.ExecuteMigration(ctx, driver.l, db.Snowflake, driver, m, statement, args)
}

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
		schema,
		schema_prev,
		execution_duration,
		issue_id,
		payload
		FROM bytebase.public.migration_history `
	return util.FindMigrationHistoryList(ctx, db.MySQL, driver, find, baseQuery)
}

func (driver *Driver) Dump(ctx context.Context, database string, out io.Writer, schemaOnly bool) error {
	// TODO(spinningbot): implement it.
	return nil
}

func (driver *Driver) Restore(ctx context.Context, sc *bufio.Scanner) (err error) {
	// TODO(spinningbot): implement it.
	return nil
}
