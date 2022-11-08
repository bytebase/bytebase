// Package snowflake is the plugin for Snowflake driver.
package snowflake

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/common/log"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/plugin/db/util"

	snow "github.com/snowflakedb/gosnowflake"
	"go.uber.org/zap"
)

var (
	bytebaseDatabase = "BYTEBASE"
	sysAdminRole     = "SYSADMIN"
	accountAdminRole = "ACCOUNTADMIN"

	_ db.Driver = (*Driver)(nil)
)

func init() {
	db.Register(db.Snowflake, newDriver)
}

// Driver is the Snowflake driver.
type Driver struct {
	connectionCtx db.ConnectionContext
	dbType        db.Type

	db *sql.DB

	collector prometheus.Collector
}

func newDriver(db.DriverConfig) db.Driver {
	return &Driver{}
}

// Open opens a Snowflake driver.
func (driver *Driver) Open(_ context.Context, dbType db.Type, config db.ConnectionConfig, connCtx db.ConnectionContext) (db.Driver, error) {
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
			return nil, errors.Errorf("driver.Open() has invalid host %q", config.Host)
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
	log.Debug("Opening Snowflake driver",
		zap.String("dsn", loggedDSN),
		zap.String("environment", connCtx.EnvironmentName),
		zap.String("database", connCtx.InstanceName),
	)
	db, err := sql.Open("snowflake", dsn)
	if err != nil {
		panic(err)
	}

	if driver.collector == nil {
		// Create a new collector, the name will be used as a label on the metrics
		driver.collector = util.NewStatsCollector(string(dbType), config.Database, db)
		// Register it with Prometheus
		prometheus.MustRegister(driver.collector)
	}

	driver.dbType = dbType
	driver.db = db
	driver.connectionCtx = connCtx

	return driver, nil
}

// Close closes the driver.
func (driver *Driver) Close(context.Context) error {
	if driver.collector != nil {
		prometheus.Unregister(driver.collector)
	}
	return driver.db.Close()
}

// Ping pings the database.
func (driver *Driver) Ping(ctx context.Context) error {
	return driver.db.PingContext(ctx)
}

// GetDBConnection gets a database connection.
func (driver *Driver) GetDBConnection(context.Context, string) (*sql.DB, error) {
	return driver.db, nil
}

// getVersion gets the version.
func (driver *Driver) getVersion(ctx context.Context) (string, error) {
	query := "SELECT CURRENT_VERSION()"
	var version string
	if err := driver.db.QueryRowContext(ctx, query).Scan(&version); err != nil {
		if err == sql.ErrNoRows {
			return "", common.FormatDBErrorEmptyRowWithQuery(query)
		}
		return "", util.FormatErrorWithQuery(err, query)
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
	shareQuery := "SHOW SHARES"
	shareRows, err := tx.Query(shareQuery)
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
		return nil, util.FormatErrorWithQuery(err, shareQuery)
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
		return nil, util.FormatErrorWithQuery(err, query)
	}
	return databases, nil
}

// Execute executes a SQL statement.
func (driver *Driver) Execute(ctx context.Context, statement string, _ bool) error {
	count := 0
	f := func(stmt string) error {
		count++
		return nil
	}

	if err := util.ApplyMultiStatements(strings.NewReader(statement), f); err != nil {
		return err
	}

	if count <= 0 {
		return nil
	}

	if err := driver.useRole(ctx, sysAdminRole); err != nil {
		return err
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
func (driver *Driver) Query(ctx context.Context, statement string, limit int, readOnly bool) ([]interface{}, error) {
	return util.Query(ctx, db.Snowflake, driver.db, statement, limit, readOnly)
}
