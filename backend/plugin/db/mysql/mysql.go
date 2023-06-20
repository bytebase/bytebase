// Package mysql is the plugin for MySQL driver.
package mysql

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/pkg/errors"
	"go.uber.org/multierr"
	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	bbparser "github.com/bytebase/bytebase/backend/plugin/parser/sql"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

var (
	baseTableType = "BASE TABLE"
	viewTableType = "VIEW"
	// Sequence is available to TiDB only.
	sequenceTableType = "SEQUENCE"

	_ db.Driver = (*Driver)(nil)
)

func init() {
	db.Register(db.MySQL, newDriver)
	db.Register(db.TiDB, newDriver)
	db.Register(db.MariaDB, newDriver)
	db.Register(db.OceanBase, newDriver)
}

// Driver is the MySQL driver.
type Driver struct {
	connectionCtx db.ConnectionContext
	connCfg       db.ConnectionConfig
	dbType        db.Type
	dbBinDir      string
	binlogDir     string
	db            *sql.DB
	databaseName  string
	// migrationConn is used to execute migrations.
	// Use a single connection for executing migrations in the lifetime of the driver can keep the thread ID unchanged.
	// So that it's easy to get the thread ID for rollback SQL.
	migrationConn *sql.Conn
	sshClient     *ssh.Client

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

	params := []string{"multiStatements=true", "maxAllowedPacket=0"}
	if connCfg.SSHConfig.Host != "" {
		sshClient, err := util.GetSSHClient(connCfg.SSHConfig)
		if err != nil {
			return nil, err
		}
		driver.sshClient = sshClient
		// Now we register the dialer with the ssh connection as a parameter.
		mysql.RegisterDialContext("mysql+tcp", func(ctx context.Context, addr string) (net.Conn, error) {
			return sshClient.Dial("tcp", addr)
		})
		protocol = "mysql+tcp"
	}

	// TODO(zp): mysql and mysqlbinlog doesn't support SSL yet. We need to write certs to temp files and load them as CLI flags.
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

	dsn := fmt.Sprintf("%s:%s@%s(%s:%s)/%s?%s", connCfg.Username, connCfg.Password, protocol, connCfg.Host, connCfg.Port, connCfg.Database, strings.Join(params, "&"))
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
	// TODO(d): remove the work-around once we have clean-up the migration connection hack.
	db.SetConnMaxLifetime(2 * time.Hour)
	db.SetMaxOpenConns(50)
	db.SetMaxIdleConns(15)
	driver.migrationConn = conn
	driver.connectionCtx = connCtx
	driver.connCfg = connCfg
	driver.databaseName = connCfg.Database

	return driver, nil
}

// Close closes the driver.
func (driver *Driver) Close(context.Context) error {
	var err error
	err = multierr.Append(err, driver.db.Close())
	err = multierr.Append(err, driver.migrationConn.Close())
	if driver.sshClient != nil {
		err = multierr.Append(err, driver.sshClient.Close())
	}
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

// GetDB gets the database.
func (driver *Driver) GetDB() *sql.DB {
	return driver.db
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
	tx, err := driver.migrationConn.BeginTx(ctx, nil)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to begin execute transaction")
	}
	defer tx.Rollback()

	var totalRowsAffected int64
	sqlResult, err := tx.ExecContext(ctx, statement)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to execute context in a transaction")
	}
	rowsAffected, err := sqlResult.RowsAffected()
	if err != nil {
		// Since we cannot differentiate DDL and DML yet, we have to ignore the error.
		log.Debug("rowsAffected returns error", zap.Error(err))
	}
	totalRowsAffected += rowsAffected

	if err := tx.Commit(); err != nil {
		return 0, errors.Wrapf(err, "failed to commit execute transaction")
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

// QueryConn querys a SQL statement in a given connection.
func (driver *Driver) QueryConn(ctx context.Context, conn *sql.Conn, statement string, queryContext *db.QueryContext) ([]any, error) {
	singleSQLs, err := bbparser.SplitMultiSQL(bbparser.MySQL, statement)
	if err != nil {
		return nil, err
	}
	if len(singleSQLs) == 0 {
		return nil, nil
	}
	// https://dev.mysql.com/doc/c-api/8.0/en/mysql-affected-rows.html
	// If the statement is an INSERT, UPDATE, or DELETE statement, we will call execute instead of query and return the number of rows affected.
	if len(singleSQLs) == 1 && util.IsAffectedRowsStatement(singleSQLs[0].Text) {
		sqlResult, err := conn.ExecContext(ctx, singleSQLs[0].Text)
		if err != nil {
			return nil, err
		}
		affectedRows, err := sqlResult.RowsAffected()
		if err != nil {
			log.Info("rowsAffected returns error", zap.Error(err))
		}

		field := []string{"Affected Rows"}
		types := []string{"INT"}
		rows := [][]any{{affectedRows}}
		return []any{field, types, rows}, nil
	}
	return util.Query(ctx, driver.dbType, conn, statement, queryContext)
}

const querySize = 2 * 1024 * 1024 // 2M.

// splitAndTransformDelimiter transform the delimiter to the MySQL default delimiter.
func splitAndTransformDelimiter(statement string) ([]string, error) {
	var trunks []string

	var out bytes.Buffer
	statements, err := bbparser.SplitMultiSQLAndNormalize(bbparser.MySQL, statement)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to split SQL statements")
	}
	for _, singleSQL := range statements {
		stmt := singleSQL.Text
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

// QueryConn2 queries a SQL statement in a given connection.
func (driver *Driver) QueryConn2(ctx context.Context, conn *sql.Conn, statement string, queryContext *db.QueryContext) ([]*v1pb.QueryResult, error) {
	singleSQLs, err := bbparser.SplitMultiSQL(bbparser.MySQL, statement)
	if err != nil {
		return nil, err
	}
	if len(singleSQLs) == 0 {
		return nil, nil
	}

	var results []*v1pb.QueryResult
	for _, singleSQL := range singleSQLs {
		result, err := driver.querySingleSQL(ctx, conn, singleSQL, queryContext)
		if err != nil {
			results = append(results, &v1pb.QueryResult{
				Error: err.Error(),
			})
		} else {
			results = append(results, result)
		}
	}

	return results, nil
}

func (driver *Driver) getStatementWithResultLimit(stmt string, limit int) (string, error) {
	switch driver.dbType {
	case db.MySQL, db.MariaDB:
		// MySQL 5.7 doesn't support WITH clause.
		return fmt.Sprintf("SELECT * FROM (%s) result LIMIT %d;", stmt, limit), nil
	case db.TiDB:
		return fmt.Sprintf("WITH result AS (%s) SELECT * FROM result LIMIT %d;", stmt, limit), nil
	default:
		return "", errors.Errorf("unsupported database type %s", driver.dbType)
	}
}

func (driver *Driver) querySingleSQL(ctx context.Context, conn *sql.Conn, singleSQL bbparser.SingleSQL, queryContext *db.QueryContext) (*v1pb.QueryResult, error) {
	if singleSQL.Empty {
		return nil, nil
	}
	statement := strings.TrimRight(singleSQL.Text, " \n\t;")
	if !strings.HasPrefix(statement, "EXPLAIN") && queryContext.Limit > 0 {
		var err error
		statement, err = driver.getStatementWithResultLimit(statement, queryContext.Limit)
		if err != nil {
			return nil, err
		}
	}

	if driver.dbType == db.TiDB && queryContext.ReadOnly {
		// TiDB doesn't support READ ONLY transactions. We have to skip the flag for it.
		// https://github.com/pingcap/tidb/issues/34626
		queryContext.ReadOnly = false
	}

	return util.Query2(ctx, driver.dbType, conn, statement, queryContext)
}

// RunStatement runs a SQL statement in a given connection.
func (*Driver) RunStatement(ctx context.Context, conn *sql.Conn, statement string) ([]*v1pb.QueryResult, error) {
	return util.RunStatement(ctx, bbparser.MySQL, conn, statement)
}
