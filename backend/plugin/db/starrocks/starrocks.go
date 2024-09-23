// Package starrocks is the plugin for starrocks driver.
package starrocks

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net"
	"regexp"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"go.uber.org/multierr"
	"golang.org/x/crypto/ssh"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

var (
	baseTableType = "BASE TABLE"
	viewTableType = "VIEW"

	_ db.Driver = (*Driver)(nil)
)

func init() {
	db.Register(storepb.Engine_STARROCKS, newDriver)
	db.Register(storepb.Engine_DORIS, newDriver)
}

// Driver is the MySQL driver.
type Driver struct {
	connectionCtx db.ConnectionContext
	connCfg       db.ConnectionConfig
	dbType        storepb.Engine
	dbBinDir      string
	db            *sql.DB
	databaseName  string
	sshClient     *ssh.Client

	// Called upon driver.Open() finishes.
	openCleanUp []func()
}

func newDriver(dc db.DriverConfig) db.Driver {
	return &Driver{
		dbBinDir: dc.DbBinDir,
	}
}

// Open opens a MySQL driver.
func (driver *Driver) Open(_ context.Context, dbType storepb.Engine, connCfg db.ConnectionConfig) (db.Driver, error) {
	defer func() {
		for _, f := range driver.openCleanUp {
			f()
		}
	}()

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
		mysql.RegisterDialContext("mysql+tcp", func(_ context.Context, addr string) (net.Conn, error) {
			return sshClient.Dial("tcp", addr)
		})
		protocol = "mysql+tcp"
	}

	// TODO(zp): mysql and mysqlbinlog doesn't support SSL yet. We need to write certs to temp files and load them as CLI flags.
	tlsConfig, err := connCfg.TLSConfig.GetSslConfig()
	if err != nil {
		return nil, errors.Wrap(err, "sql: tls config error")
	}
	tlsKey := uuid.NewString()
	if tlsConfig != nil {
		if err := mysql.RegisterTLSConfig(tlsKey, tlsConfig); err != nil {
			return nil, errors.Wrap(err, "sql: failed to register tls config")
		}
		// TLS config is only used during sql.Open, so should be safe to deregister afterwards.
		driver.openCleanUp = append(driver.openCleanUp, func() { mysql.DeregisterTLSConfig(tlsKey) })
		params = append(params, fmt.Sprintf("tls=%s", tlsKey))
	}

	dsn := fmt.Sprintf("%s:%s@%s(%s:%s)/%s?%s", connCfg.Username, connCfg.Password, protocol, connCfg.Host, connCfg.Port, connCfg.Database, strings.Join(params, "&"))
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	driver.dbType = dbType
	driver.db = db
	// TODO(d): remove the work-around once we have clean-up the migration connection hack.
	db.SetConnMaxLifetime(2 * time.Hour)
	db.SetMaxOpenConns(50)
	db.SetMaxIdleConns(15)
	driver.connectionCtx = connCfg.ConnectionContext
	driver.connCfg = connCfg
	driver.databaseName = connCfg.Database

	return driver, nil
}

// Close closes the driver.
func (driver *Driver) Close(context.Context) error {
	var err error
	err = multierr.Append(err, driver.db.Close())
	if driver.sshClient != nil {
		err = multierr.Append(err, driver.sshClient.Close())
	}
	return err
}

// Ping pings the database.
func (driver *Driver) Ping(ctx context.Context) error {
	return driver.db.PingContext(ctx)
}

// GetDB gets the database.
func (driver *Driver) GetDB() *sql.DB {
	return driver.db
}

// getVersion gets the version.
func (driver *Driver) getVersion(ctx context.Context) (string, string, error) {
	query := "SELECT VERSION()"
	var version string
	if err := driver.db.QueryRowContext(ctx, query).Scan(&version); err != nil {
		if err == sql.ErrNoRows {
			return "", "", common.FormatDBErrorEmptyRowWithQuery(query)
		}
		return "", "", util.FormatErrorWithQuery(err, query)
	}

	return parseVersion(version)
}

func parseVersion(version string) (string, string, error) {
	if loc := regexp.MustCompile(`^\d+.\d+.\d+`).FindStringIndex(version); loc != nil {
		return version[loc[0]:loc[1]], version[loc[1]:], nil
	}
	return "", "", errors.Errorf("failed to parse version %q", version)
}

// Execute executes a SQL statement.
func (driver *Driver) Execute(ctx context.Context, statement string, _ db.ExecuteOptions) (int64, error) {
	statement, err := mysqlparser.DealWithDelimiter(statement)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to deal with delimiter")
	}
	conn, err := driver.db.Conn(ctx)
	if err != nil {
		return 0, err
	}
	defer conn.Close()

	connectionID, err := getConnectionID(ctx, conn)
	if err != nil {
		return 0, err
	}
	slog.Debug("connectionID", slog.String("connectionID", connectionID))

	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to begin execute transaction")
	}
	defer tx.Rollback()

	sqlResult, err := tx.ExecContext(ctx, statement)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			slog.Info("cancel connection", slog.String("connectionID", connectionID))
			if err := driver.StopConnectionByID(connectionID); err != nil {
				slog.Error("failed to cancel connection", slog.String("connectionID", connectionID), log.BBError(err))
			}
		}

		return 0, err
	}
	rowsAffected, err := sqlResult.RowsAffected()
	if err != nil {
		// Since we cannot differentiate DDL and DML yet, we have to ignore the error.
		slog.Debug("rowsAffected returns error", log.BBError(err))
	}
	if err := tx.Commit(); err != nil {
		return 0, errors.Wrapf(err, "failed to commit execute transaction")
	}

	return rowsAffected, nil
}

// QueryConn queries a SQL statement in a given connection.
func (driver *Driver) QueryConn(ctx context.Context, conn *sql.Conn, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
	singleSQLs, err := base.SplitMultiSQL(storepb.Engine_MYSQL, statement)
	if err != nil {
		return nil, err
	}
	singleSQLs = base.FilterEmptySQL(singleSQLs)
	if len(singleSQLs) == 0 {
		return nil, nil
	}

	connectionID, err := getConnectionID(ctx, conn)
	if err != nil {
		return nil, err
	}
	slog.Debug("connectionID", slog.String("connectionID", connectionID))

	var results []*v1pb.QueryResult
	for _, singleSQL := range singleSQLs {
		statement := singleSQL.Text
		if queryContext.Explain {
			statement = fmt.Sprintf("EXPLAIN %s", statement)
		} else if queryContext.Limit > 0 {
			statement = getStatementWithResultLimit(statement, queryContext.Limit)
		}
		sqlWithBytebaseAppComment := util.MySQLPrependBytebaseAppComment(statement)

		_, allQuery, err := base.ValidateSQLForEditor(storepb.Engine_MYSQL, statement)
		if err != nil {
			// TODO(d): need to make parser compatible.
			slog.Error("failed to validate sql", slog.String("statement", statement), log.BBError(err))
			allQuery = true
		}
		startTime := time.Now()
		queryResult, err := func() (*v1pb.QueryResult, error) {
			if allQuery {
				rows, err := conn.QueryContext(ctx, sqlWithBytebaseAppComment)
				if err != nil {
					return nil, err
				}
				defer rows.Close()
				r, err := rowsToQueryResult(rows, driver.connCfg.MaximumSQLResultSize)
				if err != nil {
					return nil, err
				}
				if err := rows.Err(); err != nil {
					return nil, err
				}
				return r, nil
			}

			sqlResult, err := conn.ExecContext(ctx, statement)
			if err != nil {
				return nil, err
			}
			affectedRows, err := sqlResult.RowsAffected()
			if err != nil {
				slog.Info("rowsAffected returns error", log.BBError(err))
			}
			return util.BuildAffectedRowsResult(affectedRows), nil
		}()
		stop := false
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				slog.Info("cancel connection", slog.String("connectionID", connectionID))
				if err := driver.StopConnectionByID(connectionID); err != nil {
					slog.Error("failed to cancel connection", slog.String("connectionID", connectionID), log.BBError(err))
				}
			}
			queryResult = &v1pb.QueryResult{
				Error: err.Error(),
			}
			stop = true
		}
		queryResult.Statement = statement
		queryResult.Latency = durationpb.New(time.Since(startTime))
		results = append(results, queryResult)
		if stop {
			break
		}
	}

	return results, nil
}

func (driver *Driver) StopConnectionByID(id string) error {
	_, err := driver.db.Exec(fmt.Sprintf("KILL QUERY %s", id))
	return err
}

func getConnectionID(ctx context.Context, conn *sql.Conn) (string, error) {
	var id string
	if err := conn.QueryRowContext(ctx, `SELECT CONNECTION_ID();`).Scan(&id); err != nil {
		return "", err
	}
	return id, nil
}
