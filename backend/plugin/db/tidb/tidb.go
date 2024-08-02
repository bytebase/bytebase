// Package tidb is the plugin for TiDB driver.
package tidb

import (
	"context"
	"database/sql"
	"database/sql/driver"
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
	tidbparser "github.com/bytebase/bytebase/backend/plugin/parser/tidb"
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
	db.Register(storepb.Engine_TIDB, newDriver)
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
func (d *Driver) Open(_ context.Context, dbType storepb.Engine, connCfg db.ConnectionConfig) (db.Driver, error) {
	defer func() {
		for _, f := range d.openCleanUp {
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
		d.sshClient = sshClient
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
		d.openCleanUp = append(d.openCleanUp, func() { mysql.DeregisterTLSConfig(tlsKey) })
		params = append(params, fmt.Sprintf("tls=%s", tlsKey))
	}

	dsn := fmt.Sprintf("%s:%s@%s(%s:%s)/%s?%s", connCfg.Username, connCfg.Password, protocol, connCfg.Host, connCfg.Port, connCfg.Database, strings.Join(params, "&"))
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	d.dbType = dbType
	d.db = db
	// TODO(d): remove the work-around once we have clean-up the migration connection hack.
	db.SetConnMaxLifetime(2 * time.Hour)
	db.SetMaxOpenConns(50)
	db.SetMaxIdleConns(15)
	d.connectionCtx = connCfg.ConnectionContext
	d.connCfg = connCfg
	d.databaseName = connCfg.Database

	return d, nil
}

// Close closes the driver.
func (d *Driver) Close(context.Context) error {
	var err error
	err = multierr.Append(err, d.db.Close())
	if d.sshClient != nil {
		err = multierr.Append(err, d.sshClient.Close())
	}
	return err
}

// Ping pings the database.
func (d *Driver) Ping(ctx context.Context) error {
	return d.db.PingContext(ctx)
}

// GetDB gets the database.
func (d *Driver) GetDB() *sql.DB {
	return d.db
}

// getVersion gets the version.
func (d *Driver) getVersion(ctx context.Context) (string, string, error) {
	query := "SELECT VERSION()"
	var version string
	if err := d.db.QueryRowContext(ctx, query).Scan(&version); err != nil {
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
func (d *Driver) Execute(ctx context.Context, statement string, opts db.ExecuteOptions) (int64, error) {
	statement, err := mysqlparser.DealWithDelimiter(statement)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to deal with delimiter")
	}

	conn, err := d.db.Conn(ctx)
	if err != nil {
		return 0, err
	}
	defer conn.Close()

	connectionID, err := getConnectionID(ctx, conn)
	if err != nil {
		return 0, err
	}
	slog.Debug("connectionID", slog.String("connectionID", connectionID))

	var remainingSQLsIndex, nonTransactionStmtsIndex []int
	var nonTransactionStmts []string
	var totalCommands int
	var commands []base.SingleSQL
	var originalIndex []int32
	oneshot := true
	if len(statement) <= common.MaxSheetCheckSize {
		singleSQLs, err := tidbparser.SplitSQL(statement)
		if err != nil {
			return 0, errors.Wrapf(err, "failed to split sql")
		}
		singleSQLs, originalIndex = base.FilterEmptySQLWithIndexes(singleSQLs)
		if len(singleSQLs) == 0 {
			return 0, nil
		}
		totalCommands = len(singleSQLs)
		if totalCommands <= common.MaximumCommands {
			oneshot = false
			// Find non-transactional statements.
			// TiDB cannot run create table and create index in a single transaction.
			var remainingSQLs []base.SingleSQL
			for i, singleSQL := range singleSQLs {
				if isNonTransactionStatement(singleSQL.Text) {
					nonTransactionStmts = append(nonTransactionStmts, singleSQL.Text)
					nonTransactionStmtsIndex = append(nonTransactionStmtsIndex, i)
					continue
				}
				remainingSQLs = append(remainingSQLs, singleSQL)
				remainingSQLsIndex = append(remainingSQLsIndex, i)
			}
			commands = remainingSQLs
		}
	}
	if oneshot {
		commands = []base.SingleSQL{
			{
				Text: statement,
			},
		}
		originalIndex = []int32{0}
		remainingSQLsIndex = []int{0}
	}

	var totalRowsAffected int64
	if err := conn.Raw(func(driverConn any) error {
		//nolint
		exer := driverConn.(driver.ExecerContext)
		//nolint
		txer := driverConn.(driver.ConnBeginTx)
		tx, err := txer.BeginTx(ctx, driver.TxOptions{})
		if err != nil {
			return err
		}
		defer tx.Rollback()

		for i, command := range commands {
			// Set the progress information for the current chunk.
			if opts.UpdateExecutionStatus != nil {
				opts.UpdateExecutionStatus(&v1pb.TaskRun_ExecutionDetail{
					CommandsTotal:     int32(totalCommands),
					CommandsCompleted: int32(i),
					CommandStartPosition: &v1pb.TaskRun_ExecutionDetail_Position{
						Line:   int32(command.FirstStatementLine),
						Column: int32(command.FirstStatementColumn),
					},
					CommandEndPosition: &v1pb.TaskRun_ExecutionDetail_Position{
						Line:   int32(command.LastLine),
						Column: int32(command.LastColumn),
					},
				})
			}

			indexes := []int32{originalIndex[remainingSQLsIndex[i]]}
			opts.LogCommandExecute(indexes)

			sqlWithBytebaseAppComment := util.MySQLPrependBytebaseAppComment(command.Text)
			sqlResult, err := exer.ExecContext(ctx, sqlWithBytebaseAppComment, nil)
			if err != nil {
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					slog.Info("cancel connection", slog.String("connectionID", connectionID))
					if err := d.StopConnectionByID(connectionID); err != nil {
						slog.Error("failed to cancel connection", slog.String("connectionID", connectionID), log.BBError(err))
					}
				}

				opts.LogCommandResponse(indexes, 0, nil, err.Error())

				return &db.ErrorWithPosition{
					Err: errors.Wrapf(err, "failed to execute context in a transaction"),
					Start: &storepb.TaskRunResult_Position{
						Line:   int32(command.FirstStatementLine),
						Column: int32(command.FirstStatementColumn),
					},
					End: &storepb.TaskRunResult_Position{
						Line:   int32(command.LastLine),
						Column: int32(command.LastColumn),
					},
				}
			}

			allRowsAffected := sqlResult.(mysql.Result).AllRowsAffected()
			var rowsAffected int64
			var allRowsAffectedInt32 []int32
			for _, a := range allRowsAffected {
				rowsAffected += a
				allRowsAffectedInt32 = append(allRowsAffectedInt32, int32(a))
			}
			totalRowsAffected += rowsAffected

			opts.LogCommandResponse(indexes, int32(rowsAffected), allRowsAffectedInt32, "")
		}

		if err := tx.Commit(); err != nil {
			return errors.Wrapf(err, "failed to commit execute transaction")
		}
		return nil
	}); err != nil {
		return 0, err
	}

	// Run non-transaction statements at the end.
	for i, stmt := range nonTransactionStmts {
		indexes := []int32{int32(originalIndex[nonTransactionStmtsIndex[i]])}
		opts.LogCommandExecute(indexes)
		if _, err := d.db.ExecContext(ctx, stmt); err != nil {
			opts.LogCommandResponse(indexes, 0, []int32{0}, err.Error())
			return 0, err
		}
		opts.LogCommandResponse(indexes, 0, []int32{0}, "")
	}
	return totalRowsAffected, nil
}

var (
	// CREATE INDEX CONCURRENTLY cannot run inside a transaction block.
	// CREATE [ UNIQUE ] [ SPATIAL ] [ FULLTEXT ] INDEX ... ON table_name ...
	createIndexReg = regexp.MustCompile(`(?i)CREATE(\s+(UNIQUE\s+)?(SPATIAL\s+)?(FULLTEXT\s+)?)INDEX(\s+)`)
)

func isNonTransactionStatement(stmt string) bool {
	return len(createIndexReg.FindString(stmt)) > 0
}

// QueryConn queries a SQL statement in a given connection.
func (d *Driver) QueryConn(ctx context.Context, conn *sql.Conn, statement string, queryContext *db.QueryContext) ([]*v1pb.QueryResult, error) {
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
		if queryContext != nil && queryContext.Explain {
			statement = fmt.Sprintf("EXPLAIN %s", statement)
		} else if queryContext != nil && queryContext.Limit > 0 {
			statement = getStatementWithResultLimit(statement, queryContext.Limit)
		}
		sqlWithBytebaseAppComment := util.MySQLPrependBytebaseAppComment(statement)

		_, allQuery, err := base.ValidateSQLForEditor(storepb.Engine_TIDB, statement)
		if err != nil {
			return nil, err
		}
		startTime := time.Now()
		queryResult, err := func() (*v1pb.QueryResult, error) {
			if allQuery {
				rows, err := conn.QueryContext(ctx, sqlWithBytebaseAppComment)
				if err != nil {
					return nil, util.FormatErrorWithQuery(err, statement)
				}
				defer rows.Close()
				r, err := util.RowsToQueryResult(rows)
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
				if err := d.StopConnectionByID(connectionID); err != nil {
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

func (d *Driver) StopConnectionByID(id string) error {
	// We cannot use placeholder parameter because TiDB doesn't accept it.
	_, err := d.db.Exec(fmt.Sprintf("KILL QUERY %s", id))
	return err
}

func getConnectionID(ctx context.Context, conn *sql.Conn) (string, error) {
	var id string
	if err := conn.QueryRowContext(ctx, `SELECT CONNECTION_ID();`).Scan(&id); err != nil {
		return "", err
	}
	return id, nil
}

// RunStatement runs a SQL statement in a given connection.
func (d *Driver) RunStatement(ctx context.Context, conn *sql.Conn, statement string) ([]*v1pb.QueryResult, error) {
	return d.QueryConn(ctx, conn, statement, nil)
}
