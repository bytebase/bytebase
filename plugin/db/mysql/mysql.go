package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/common/log"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/plugin/db/util"
	"github.com/go-sql-driver/mysql"
	"go.uber.org/zap"
)

var (
	baseTableType = "BASE TABLE"
	viewTableType = "VIEW"

	_ db.Driver = (*Driver)(nil)
)

func init() {
	db.Register(db.MySQL, newDriver)
	db.Register(db.TiDB, newDriver)
}

// Driver is the MySQL driver.
type Driver struct {
	connectionCtx db.ConnectionContext
	connCfg       db.ConnectionConfig
	dbType        db.Type
	resourceDir   string
	binlogDir     string
	db            *sql.DB

	replayBinlogCounter *common.CountingReader
}

func newDriver(dc db.DriverConfig) db.Driver {
	return &Driver{
		resourceDir: dc.ResourceDir,
		binlogDir:   dc.BinlogDir,
	}
}

// Open opens a MySQL driver.
func (driver *Driver) Open(_ context.Context, dbType db.Type, connCfg db.ConnectionConfig, connCtx db.ConnectionContext) (db.Driver, error) {
	db, err := connect(dbType, connCfg, connCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MySQL database, error: %w", err)
	}
	driver.db = db
	driver.dbType = dbType
	driver.connectionCtx = connCtx
	driver.connCfg = connCfg

	return driver, nil
}

func connect(dbType db.Type, connCfg db.ConnectionConfig, connCtx db.ConnectionContext) (*sql.DB, error) {
	params := []string{"multiStatements=true"}

	tlsKey := "db.mysql.tls"
	tlsConfig, err := connCfg.TLSConfig.GetSslConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get TLS config, error: %w", err)
	}
	if tlsConfig != nil {
		if err := mysql.RegisterTLSConfig(tlsKey, tlsConfig); err != nil {
			return nil, fmt.Errorf("sql: failed to register tls config: %v", err)
		}
		// TLS config is only used during sql.Open, so should be safe to deregister afterwards.
		defer mysql.DeregisterTLSConfig(tlsKey)
		params = append(params, fmt.Sprintf("tls=%s", tlsKey))
	}

	loggedDSN := getDSN(dbType, connCfg, params, true)
	log.Debug("Opening MySQL driver",
		zap.String("dsn", loggedDSN),
		zap.String("environment", connCtx.EnvironmentName),
		zap.String("database", connCtx.InstanceName),
	)

	dsn := getDSN(dbType, connCfg, params, false)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	return db, nil
}

func getDSN(dbType db.Type, connCfg db.ConnectionConfig, params []string, redact bool) string {
	protocol := "tcp"
	if strings.HasPrefix(connCfg.Host, "/") {
		protocol = "unix"
	}

	port := connCfg.Port
	if port == "" {
		port = "3306"
		if dbType == db.TiDB {
			port = "4000"
		}
	}

	password := ""
	if connCfg.Password != "" {
		if redact {
			password = ":<<redacted password>>"
		} else {
			password = ":" + connCfg.Password
		}
	}

	return fmt.Sprintf("%s%s@%s(%s:%s)/%s?%s", connCfg.Username, password, protocol, connCfg.Host, port, connCfg.Database, strings.Join(params, "&"))
}

// Close closes the driver.
func (driver *Driver) Close(context.Context) error {
	return driver.db.Close()
}

// Ping pings the database.
func (driver *Driver) Ping(ctx context.Context) error {
	return driver.db.PingContext(ctx)
}

// GetDBConnection gets a database connection.
func (driver *Driver) GetDBConnection(context.Context, string) (*sql.DB, error) {
	return connect(driver.dbType, driver.connCfg, driver.connectionCtx)
}

// getDatabases gets all databases of an instance.
func getDatabases(ctx context.Context, txn *sql.Tx) ([]string, error) {
	var dbNames []string
	query := "SHOW DATABASES"
	rows, err := txn.QueryContext(ctx, query)
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
	if err := rows.Err(); err != nil {
		return nil, util.FormatErrorWithQuery(err, query)
	}
	return dbNames, nil
}

// getVersion gets the version.
func (driver *Driver) getVersion(ctx context.Context) (string, error) {
	query := "SELECT VERSION()"
	var version string
	if err := driver.db.QueryRowContext(ctx, query).Scan(&version); err != nil {
		if err == sql.ErrNoRows {
			return "", common.FormatDBErrorEmptyRowWithQuery(query)
		}
		return "", util.FormatErrorWithQuery(err, query)
	}
	return version, nil
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
	return util.Query(ctx, driver.db, statement, limit)
}
