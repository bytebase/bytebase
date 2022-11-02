// Package mysql is the plugin for MySQL driver.
package mysql

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"io"
	"strings"

	"github.com/go-sql-driver/mysql"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/plugin/db/util"
	bbparser "github.com/bytebase/bytebase/plugin/parser"
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
	// conn is used to execute migrations.
	// Use a single connection for executing migrations in the lifetime of the driver can keep the thread ID unchanged.
	// So that it's easy to get the thread ID for rollback SQL.
	conn *sql.Conn

	replayedBinlogBytes *common.CountingReader
	restoredBackupBytes *common.CountingReader
}

func newDriver(dc db.DriverConfig) db.Driver {
	return &Driver{
		resourceDir: dc.ResourceDir,
		binlogDir:   dc.BinlogDir,
	}
}

// Open opens a MySQL driver.
func (driver *Driver) Open(ctx context.Context, dbType db.Type, connCfg db.ConnectionConfig, connCtx db.ConnectionContext) (db.Driver, error) {
	protocol := "tcp"
	if strings.HasPrefix(connCfg.Host, "/") {
		protocol = "unix"
	}

	params := []string{"multiStatements=true"}

	port := connCfg.Port
	if port == "" {
		port = "3306"
		if dbType == db.TiDB {
			port = "4000"
		}
	}

	tlsConfig, err := connCfg.TLSConfig.GetSslConfig()

	if err != nil {
		return nil, errors.Wrap(err, "sql: tls config error")
	}

	dsn := fmt.Sprintf("%s@%s(%s:%s)/%s?%s", connCfg.Username, protocol, connCfg.Host, port, connCfg.Database, strings.Join(params, "&"))
	if connCfg.Password != "" {
		dsn = fmt.Sprintf("%s:%s@%s(%s:%s)/%s?%s", connCfg.Username, connCfg.Password, protocol, connCfg.Host, port, connCfg.Database, strings.Join(params, "&"))
	}
	tlsKey := "db.mysql.tls"
	if tlsConfig != nil {
		if err := mysql.RegisterTLSConfig(tlsKey, tlsConfig); err != nil {
			return nil, errors.Wrap(err, "sql: failed to register tls config")
		}
		// TLS config is only used during sql.Open, so should be safe to deregister afterwards.
		defer mysql.DeregisterTLSConfig(tlsKey)
		dsn += fmt.Sprintf("?tls=%s", tlsKey)
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	conn, err := db.Conn(ctx)
	if err != nil {
		return nil, err
	}
	driver.dbType = dbType
	driver.db = db
	driver.conn = conn
	driver.connectionCtx = connCtx
	driver.connCfg = connCfg

	return driver, nil
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
	return driver.db, nil
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
	var buf bytes.Buffer
	if err := transformDelimiter(&buf, statement); err != nil {
		return err
	}
	transformedStatement := buf.String()
	tx, err := driver.conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, transformedStatement)

	if err == nil {
		if err := tx.Commit(); err != nil {
			return err
		}
	}

	return err
}

// Query queries a SQL statement.
func (driver *Driver) Query(ctx context.Context, statement string, limit int, readOnly bool) ([]interface{}, error) {
	return util.Query(ctx, driver.dbType, driver.db, statement, limit, readOnly)
}

// transformDelimiter transform the delimiter to the MySQL default delimiter.
func transformDelimiter(out io.Writer, statement string) error {
	statements, err := bbparser.SplitMultiSQL(bbparser.MySQL, statement)
	if err != nil {
		return errors.Wrapf(err, "failed to split SQL statements")
	}
	delimiter := `;`
	for _, singleSQL := range statements {
		stmt := singleSQL.Text
		if bbparser.IsDelimiter(stmt) {
			delimiter, err = bbparser.ExtractDelimiter(stmt)
			if err != nil {
				return errors.Wrapf(err, "failed to extract delimiter")
			}
			continue
		}
		if delimiter != ";" {
			// Trim delimiter
			stmt = fmt.Sprintf("%s;", stmt[:len(stmt)-len(delimiter)])
		}
		if _, err = out.Write([]byte(stmt)); err != nil {
			return errors.Wrapf(err, "failed to write SQL statement")
		}
	}
	return nil
}
