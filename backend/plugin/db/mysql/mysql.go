// Package mysql is the plugin for MySQL driver.
package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
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
	// Sequence is available to TiDB only.
	sequenceTableType = "SEQUENCE"

	_ db.Driver = (*Driver)(nil)
)

func init() {
	db.Register(storepb.Engine_MYSQL, newDriver)
	db.Register(storepb.Engine_TIDB, newDriver)
	db.Register(storepb.Engine_MARIADB, newDriver)
	db.Register(storepb.Engine_OCEANBASE, newDriver)
}

// Driver is the MySQL driver.
type Driver struct {
	connectionCtx db.ConnectionContext
	connCfg       db.ConnectionConfig
	dbType        storepb.Engine
	dbBinDir      string
	binlogDir     string
	db            *sql.DB
	databaseName  string
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
func (driver *Driver) Open(_ context.Context, dbType storepb.Engine, connCfg db.ConnectionConfig, connCtx db.ConnectionContext) (db.Driver, error) {
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
	tlsKey := "storepb.Engine_MYSQL.tls"
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
	driver.dbType = dbType
	driver.db = db
	// TODO(d): remove the work-around once we have clean-up the migration connection hack.
	db.SetConnMaxLifetime(2 * time.Hour)
	db.SetMaxOpenConns(50)
	db.SetMaxIdleConns(15)
	driver.connectionCtx = connCtx
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

// GetType returns the database type.
func (driver *Driver) GetType() storepb.Engine {
	return driver.dbType
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
	pos := strings.Index(version, "-")
	if pos == -1 {
		return version, "", nil
	}
	return version[:pos], version[pos:], nil
}

// Execute executes a SQL statement.
func (driver *Driver) Execute(ctx context.Context, statement string, _ bool, opts db.ExecuteOptions) (int64, error) {
	statement, err := mysqlparser.DealWithDelimiter(statement)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to deal with delimiter")
	}
	conn, err := driver.db.Conn(ctx)
	if err != nil {
		return 0, err
	}
	defer conn.Close()

	if opts.BeginFunc != nil {
		if err := opts.BeginFunc(ctx, conn); err != nil {
			return 0, err
		}
	}

	if opts.IndividualSubmission && len(statement) <= common.MaxSheetCheckSize {
		return executeChunkedSubmission(ctx, conn, statement, opts)
	}

	return executeBatchSubmission(ctx, conn, statement, opts)
}

func executeChunkedSubmission(ctx context.Context, conn *sql.Conn, statement string, opts db.ExecuteOptions) (int64, error) {
	list, err := mysqlparser.SplitSQL(statement)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to split sql")
	}
	if len(list) == 0 {
		return 0, nil
	}

	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to begin execute transaction")
	}
	defer tx.Rollback()

	chunks, err := util.ChunkedSQLScript(list, common.MaxSheetCheckSize)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to chunk sql")
	}

	currentIndex := 0
	var totalRowsAffected int64
	for _, chunk := range chunks {
		if len(chunk) == 0 {
			continue
		}
		// Start the current chunk.

		// Set the progress information for the current chunk.
		if opts.UpdateExecutionStatus != nil {
			opts.UpdateExecutionStatus(&v1pb.TaskRun_ExecutionDetail{
				CommandsTotal:     int32(len(list)),
				CommandsCompleted: int32(currentIndex),
				CommandStartPosition: &v1pb.TaskRun_ExecutionDetail_Position{
					// TODO(rebelice): should find the first non-comment and blank line.
					Line: int32(chunk[0].BaseLine),
					// TODO(rebelice): we should also set the column position.
				},
				CommandEndPosition: &v1pb.TaskRun_ExecutionDetail_Position{
					// TODO(rebelice): should find the first non-comment and blank line.
					Line: int32(chunk[len(chunk)-1].BaseLine),
					// TODO(rebelice): we should also set the column position.
				},
			})
		}

		var chunkBuf strings.Builder
		for _, sql := range chunk {
			if _, err := chunkBuf.WriteString(sql.Text); err != nil {
				return 0, errors.Wrapf(err, "failed to write chunk buffer")
			}
		}

		currentIndex += len(chunk)
		sqlResult, err := tx.ExecContext(ctx, chunkBuf.String())
		if err != nil {
			return 0, errors.Wrapf(err, "failed to execute context in a transaction")
		}
		rowsAffected, err := sqlResult.RowsAffected()
		if err != nil {
			// Since we cannot differentiate DDL and DML yet, we have to ignore the error.
			slog.Debug("rowsAffected returns error", log.BBError(err))
		}
		totalRowsAffected += rowsAffected
	}

	if err := tx.Commit(); err != nil {
		return 0, errors.Wrapf(err, "failed to commit execute transaction")
	}

	return totalRowsAffected, nil
}

func executeBatchSubmission(ctx context.Context, conn *sql.Conn, statement string, _ db.ExecuteOptions) (int64, error) {
	tx, err := conn.BeginTx(ctx, nil)
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
		slog.Debug("rowsAffected returns error", log.BBError(err))
	}
	totalRowsAffected += rowsAffected

	if err := tx.Commit(); err != nil {
		return 0, errors.Wrapf(err, "failed to commit execute transaction")
	}

	return totalRowsAffected, nil
}

// QueryConn queries a SQL statement in a given connection.
func (driver *Driver) QueryConn(ctx context.Context, conn *sql.Conn, statement string, queryContext *db.QueryContext) ([]*v1pb.QueryResult, error) {
	singleSQLs, err := base.SplitMultiSQL(storepb.Engine_MYSQL, statement)
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

func (driver *Driver) querySingleSQL(ctx context.Context, conn *sql.Conn, singleSQL base.SingleSQL, queryContext *db.QueryContext) (*v1pb.QueryResult, error) {
	if singleSQL.Empty {
		return nil, nil
	}
	statement := strings.TrimLeft(strings.TrimRight(singleSQL.Text, " \n\t;"), " \n\t")
	isExplain := strings.HasPrefix(statement, "EXPLAIN")

	stmt := statement
	if !isExplain && queryContext.Limit > 0 {
		stmt = driver.getStatementWithResultLimit(stmt, queryContext.Limit)
	}

	if driver.dbType == storepb.Engine_TIDB && queryContext.ReadOnly {
		// TiDB doesn't support READ ONLY transactions. We have to skip the flag for it.
		// https://github.com/pingcap/tidb/issues/34626
		queryContext.ReadOnly = false
	}

	if queryContext.SensitiveSchemaInfo != nil {
		for _, database := range queryContext.SensitiveSchemaInfo.DatabaseList {
			if len(database.SchemaList) == 0 {
				continue
			}
			if len(database.SchemaList) > 1 {
				return nil, errors.Errorf("MySQL schema info should only have one schema per database, but got %d, %v", len(database.SchemaList), database.SchemaList)
			}
			if database.SchemaList[0].Name != "" {
				return nil, errors.Errorf("MySQL schema info should have empty schema name, but got %s", database.SchemaList[0].Name)
			}
		}
	}

	startTime := time.Now()
	result, err := util.Query(ctx, driver.dbType, conn, stmt, queryContext)
	if err != nil {
		return nil, err
	}
	result.Latency = durationpb.New(time.Since(startTime))
	result.Statement = statement
	return result, nil
}

// RunStatement runs a SQL statement in a given connection.
func (*Driver) RunStatement(ctx context.Context, conn *sql.Conn, statement string) ([]*v1pb.QueryResult, error) {
	return util.RunStatement(ctx, storepb.Engine_MYSQL, conn, statement)
}
