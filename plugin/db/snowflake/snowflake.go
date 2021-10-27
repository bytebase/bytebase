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
	_ "github.com/snowflakedb/gosnowflake"
	"go.uber.org/zap"
)

//go:embed snowflake_migration_schema.sql
var migrationSchema string

var (
	systemSchemas = map[string]bool{
		"information_schema": true,
	}

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

func (driver *Driver) SyncSchema(ctx context.Context) ([]*db.DBUser, []*db.DBSchema, error) {
	// TODO(spinningbot): implement it.
	// Query user info
	userList, err := driver.getUserList(ctx)
	if err != nil {
		return nil, nil, err
	}

	// Query db info
	tx, err := driver.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, nil, err
	}
	defer tx.Rollback()

	if _, err = tx.ExecContext(ctx, "Use ROLE accountadmin"); err != nil {
		return nil, nil, err
	}

	query := `
		SELECT 
			DATABASE_NAME
		FROM SNOWFLAKE.INFORMATION_SCHEMA.DATABASES`
	rows, err := tx.QueryContext(ctx, query)
	if err != nil {
		return nil, nil, util.FormatErrorWithQuery(err, query)
	}
	defer rows.Close()

	schemaList := make([]*db.DBSchema, 0)
	for rows.Next() {
		var schema db.DBSchema
		if err := rows.Scan(
			&schema.Name,
		); err != nil {
			return nil, nil, err
		}

		schemaList = append(schemaList, &schema)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	return userList, schemaList, nil
}

func (driver *Driver) getUserList(ctx context.Context) ([]*db.DBUser, error) {
	// Query user info
	tx, err := driver.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if _, err = tx.ExecContext(ctx, "Use ROLE accountadmin"); err != nil {
		return nil, err
	}

	query := `
	  SELECT
			name
		FROM SNOWFLAKE.ACCOUNT_USAGE.USERS
	`
	userList := make([]*db.DBUser, 0)
	userRows, err := tx.QueryContext(ctx, query)

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
	tx, err := driver.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, statement)

	if err := tx.Commit(); err != nil {
		return err
	}

	return err
}

// Migration related
func (driver *Driver) NeedsSetupMigration(ctx context.Context) (bool, error) {
	// TODO(spinningbot): implement it.
	return false, nil
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

// ExecuteMigration will execute the migration for MySQL.
func (driver *Driver) ExecuteMigration(ctx context.Context, m *db.MigrationInfo, statement string) (int64, string, error) {
	// TODO(spinningbot): implement it.
	return int64(0), "", nil
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
