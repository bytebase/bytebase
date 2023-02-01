// Package mysql is the plugin for MySQL driver.
package mysql

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/go-sql-driver/mysql"
	"github.com/pkg/errors"
	"go.uber.org/multierr"
	"go.uber.org/zap"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	bbparser "github.com/bytebase/bytebase/backend/plugin/parser"
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
	dbBinDir      string
	binlogDir     string
	db            *sql.DB
	// migrationConn is used to execute migrations.
	// Use a single connection for executing migrations in the lifetime of the driver can keep the thread ID unchanged.
	// So that it's easy to get the thread ID for rollback SQL.
	migrationConn *sql.Conn

	replayedBinlogBytes *common.CountingReader
	restoredBackupBytes *common.CountingReader
}

func newDriver(dc db.DriverConfig) db.Driver {
	return &Driver{
		dbBinDir:  dc.DbBinDir,
		binlogDir: dc.BinlogDir,
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

	tlsKey := "db.mysql.tls"
	if tlsConfig != nil {
		if err := mysql.RegisterTLSConfig(tlsKey, tlsConfig); err != nil {
			return nil, errors.Wrap(err, "sql: failed to register tls config")
		}
		// TLS config is only used during sql.Open, so should be safe to deregister afterwards.
		defer mysql.DeregisterTLSConfig(tlsKey)
		params = append(params, fmt.Sprintf("tls=%s", tlsKey))
	}
	dsn := fmt.Sprintf("%s@%s(%s:%s)/%s?%s", connCfg.Username, protocol, connCfg.Host, port, connCfg.Database, strings.Join(params, "&"))
	if connCfg.Password != "" {
		dsn = fmt.Sprintf("%s:%s@%s(%s:%s)/%s?%s", connCfg.Username, connCfg.Password, protocol, connCfg.Host, port, connCfg.Database, strings.Join(params, "&"))
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	conn, err := db.Conn(ctx)
	if err != nil {
		var errList error
		errList = multierr.Append(errList, err)
		errList = multierr.Append(errList, db.Close())
		return nil, errList
	}
	driver.dbType = dbType
	driver.db = db
	driver.migrationConn = conn
	driver.connectionCtx = connCtx
	driver.connCfg = connCfg

	return driver, nil
}

// Close closes the driver.
func (driver *Driver) Close(context.Context) error {
	var err error
	err = multierr.Append(err, driver.db.Close())
	err = multierr.Append(err, driver.migrationConn.Close())
	return err
}

// Ping pings the database.
func (driver *Driver) Ping(ctx context.Context) error {
	return driver.db.PingContext(ctx)
}

// GetType returns the database type.
func (driver *Driver) GetType() db.Type {
	return driver.dbType
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
func (driver *Driver) Execute(ctx context.Context, statement string, _ bool) (int64, error) {
	trunks, err := splitAndTransformDelimiter(statement)
	if err != nil {
		return 0, err
	}

	tx, err := driver.migrationConn.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	var totalRowsAffected int64
	for _, trunk := range trunks {
		sqlResult, err := tx.ExecContext(ctx, trunk)
		if err != nil {
			return 0, err
		}
		rowsAffected, err := sqlResult.RowsAffected()
		if err != nil {
			// Since we cannot differentiate DDL and DML yet, we have to ignore the error.
			log.Debug("rowsAffected returns error", zap.Error(err))
		}
		totalRowsAffected += rowsAffected
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}

	return totalRowsAffected, nil
}

// GetMigrationConnID gets the ID of the connection executing migrations.
func (driver *Driver) GetMigrationConnID(ctx context.Context) (string, error) {
	var id string
	if err := driver.migrationConn.QueryRowContext(ctx, "SELECT CONNECTION_ID();").Scan(&id); err != nil {
		return "", errors.Wrap(err, "failed to get the connection ID")
	}
	return id, nil
}

// Query queries a SQL statement.
func (driver *Driver) Query(ctx context.Context, statement string, queryContext *db.QueryContext) ([]interface{}, error) {
	singleSQLs, err := bbparser.SplitMultiSQL(bbparser.MySQL, statement, true /* filterEmptyStatement */)
	if err != nil {
		return nil, err
	}
	if len(singleSQLs) == 0 {
		return nil, nil
	}
	// https://dev.mysql.com/doc/c-api/8.0/en/mysql-affected-rows.html
	// If the statement is an INSERT, UPDATE, or DELETE statement, we will call execute instead of query and return the number of rows affected.
	if len(singleSQLs) == 1 && util.IsAffectedRowsStatement(singleSQLs[0].Text) {
		affectedRows, err := driver.Execute(ctx, singleSQLs[0].Text, false)
		if err != nil {
			return nil, err
		}
		field := []string{"Affected Rows"}
		types := []string{"INT"}
		rows := [][]interface{}{{affectedRows}}
		return []interface{}{field, types, rows}, nil
	}
	return util.Query(ctx, driver.dbType, driver.db, statement, queryContext)
}

const querySize = 2 * 1024 * 1024 // 2M.

// splitAndTransformDelimiter transform the delimiter to the MySQL default delimiter.
func splitAndTransformDelimiter(statement string) ([]string, error) {
	var trunks []string

	var out bytes.Buffer
	statements, err := bbparser.SplitMultiSQL(bbparser.MySQL, statement, true /* filterEmptyStatement */)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to split SQL statements")
	}
	delimiter := `;`
	for _, singleSQL := range statements {
		stmt := singleSQL.Text
		if bbparser.IsDelimiter(stmt) {
			delimiter, err = bbparser.ExtractDelimiter(stmt)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to extract delimiter")
			}
			continue
		}
		if delimiter != ";" {
			// Trim delimiter
			stmt = fmt.Sprintf("%s;", stmt[:len(stmt)-len(delimiter)])
		}
		if _, err = out.Write([]byte(stmt)); err != nil {
			return nil, errors.Wrapf(err, "failed to write SQL statement")
		}

		if out.Len() > querySize {
			trunks = append(trunks, out.String())
			out.Reset()
		}
	}
	if out.Len() > 0 {
		trunks = append(trunks, out.String())
	}
	return trunks, nil
}
