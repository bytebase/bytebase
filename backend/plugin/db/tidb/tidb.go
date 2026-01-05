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
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	tidbparser "github.com/bytebase/bytebase/backend/plugin/parser/tidb"
)

var (
	baseTableType = "BASE TABLE"
	viewTableType = "VIEW"

	_ db.Driver = (*Driver)(nil)
)

func init() {
	db.Register(storepb.Engine_TIDB, newDriver)
}

// Driver is the MySQL driver.
type Driver struct {
	dbType       storepb.Engine
	db           *sql.DB
	databaseName string
	sshClient    *ssh.Client

	// Called upon driver.Open() finishes.
	openCleanUp []func()
}

func newDriver() db.Driver {
	return &Driver{}
}

// Open opens a MySQL driver.
func (d *Driver) Open(_ context.Context, dbType storepb.Engine, connCfg db.ConnectionConfig) (db.Driver, error) {
	defer func() {
		for _, f := range d.openCleanUp {
			f()
		}
	}()

	protocol := "tcp"
	if strings.HasPrefix(connCfg.DataSource.Host, "/") {
		protocol = "unix"
	}
	params := []string{"multiStatements=true", "maxAllowedPacket=0"}
	if connCfg.DataSource.GetSshHost() != "" {
		sshClient, err := util.GetSSHClient(connCfg.DataSource)
		if err != nil {
			return nil, err
		}
		d.sshClient = sshClient
		// Now we register the dialer with the ssh connection as a parameter.
		protocol = "mysql-tcp-" + uuid.NewString()[:8]
		// Now we register the dialer with the ssh connection as a parameter.
		mysql.RegisterDialContext(protocol, func(_ context.Context, addr string) (net.Conn, error) {
			return sshClient.Dial("tcp", addr)
		})
	}

	tlscfg, err := util.GetTLSConfig(connCfg.DataSource)
	if err != nil {
		return nil, errors.Wrap(err, "sql: tls config error")
	}
	tlsKey := uuid.NewString()
	if tlscfg != nil {
		if err := mysql.RegisterTLSConfig(tlsKey, tlscfg); err != nil {
			return nil, errors.Wrap(err, "sql: failed to register tls config")
		}
		// TLS config is only used during sql.Open, so should be safe to deregister afterwards.
		d.openCleanUp = append(d.openCleanUp, func() { mysql.DeregisterTLSConfig(tlsKey) })
		params = append(params, fmt.Sprintf("tls=%s", tlsKey))
	}

	dsn := fmt.Sprintf("%s:%s@%s(%s:%s)/%s?%s", connCfg.DataSource.Username, connCfg.Password, protocol, connCfg.DataSource.Host, connCfg.DataSource.Port, connCfg.ConnectionContext.DatabaseName, strings.Join(params, "&"))
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
	d.databaseName = connCfg.ConnectionContext.DatabaseName
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
func (d *Driver) getVersion(ctx context.Context) (string, error) {
	query := "SELECT VERSION()"
	var version string
	if err := d.db.QueryRowContext(ctx, query).Scan(&version); err != nil {
		if err == sql.ErrNoRows {
			return "", common.FormatDBErrorEmptyRowWithQuery(query)
		}
		return "", util.FormatErrorWithQuery(err, query)
	}

	return parseVersion(version)
}

func parseVersion(version string) (string, error) {
	// Examples: 8.0.11-TiDB-v8.5.0, 8.0.11-TiDB-v7.5.2-serverless.
	if loc := regexp.MustCompile(`v\d+\.\d+\.\d+`).FindStringIndex(version); loc != nil {
		return version[loc[0]:loc[1]], nil
	}
	return "", errors.Errorf("failed to parse version %q", version)
}

// Execute executes a SQL statement.
func (d *Driver) Execute(ctx context.Context, statement string, opts db.ExecuteOptions) (int64, error) {
	// Parse transaction mode from the script
	config, cleanedStatement := base.ParseTransactionConfig(statement)
	statement = cleanedStatement
	transactionMode := config.Mode

	// Apply default when transaction mode is not specified
	if transactionMode == common.TransactionModeUnspecified {
		transactionMode = common.GetDefaultTransactionMode()
	}

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

	var nonTransactionStmts []base.Statement
	var totalCommands int
	var commands []base.Statement
	oneshot := true
	if len(statement) <= common.MaxSheetCheckSize {
		singleSQLs, err := tidbparser.SplitSQL(statement)
		if err != nil {
			return 0, errors.Wrapf(err, "failed to split sql")
		}
		singleSQLs = base.FilterEmptyStatements(singleSQLs)
		if len(singleSQLs) == 0 {
			return 0, nil
		}
		totalCommands = len(singleSQLs)
		if totalCommands <= common.MaximumCommands {
			oneshot = false
			// Find non-transactional statements.
			// TiDB cannot run create table and create index in a single transaction.
			var remainingSQLs []base.Statement
			for _, singleSQL := range singleSQLs {
				if isNonTransactionStatement(singleSQL.Text) {
					nonTransactionStmts = append(nonTransactionStmts, singleSQL)
					continue
				}
				remainingSQLs = append(remainingSQLs, singleSQL)
			}
			commands = remainingSQLs
		}
	}
	if oneshot {
		commands = []base.Statement{
			{
				Text: statement,
			},
		}
	}

	// Execute based on transaction mode
	if transactionMode == common.TransactionModeOff {
		return d.executeInAutoCommitMode(ctx, conn, commands, nonTransactionStmts, opts, connectionID)
	}
	return d.executeInTransactionMode(ctx, conn, commands, nonTransactionStmts, opts, connectionID)
}

// executeInTransactionMode executes statements within a single transaction
func (d *Driver) executeInTransactionMode(ctx context.Context, conn *sql.Conn, commands []base.Statement, nonTransactionStmts []base.Statement, opts db.ExecuteOptions, connectionID string) (int64, error) {
	var totalRowsAffected int64

	if err := conn.Raw(func(driverConn any) error {
		//nolint
		exer := driverConn.(driver.ExecerContext)
		//nolint
		txer := driverConn.(driver.ConnBeginTx)
		tx, err := txer.BeginTx(ctx, driver.TxOptions{})
		if err != nil {
			opts.LogTransactionControl(storepb.TaskRunLog_TransactionControl_BEGIN, err.Error())
			return err
		} else {
			opts.LogTransactionControl(storepb.TaskRunLog_TransactionControl_BEGIN, "")
		}

		committed := false
		defer func() {
			err := tx.Rollback()
			if committed {
				return
			}
			var rerr string
			if err != nil {
				rerr = err.Error()
			}
			opts.LogTransactionControl(storepb.TaskRunLog_TransactionControl_ROLLBACK, rerr)
		}()

		for _, command := range commands {
			opts.LogCommandExecute(command.Range, command.Text)

			sqlWithBytebaseAppComment := util.MySQLPrependBytebaseAppComment(command.Text)
			sqlResult, err := exer.ExecContext(ctx, sqlWithBytebaseAppComment, nil)
			if err != nil {
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					slog.Info("cancel connection", slog.String("connectionID", connectionID))
					if err := d.StopConnectionByID(connectionID); err != nil {
						slog.Error("failed to cancel connection", slog.String("connectionID", connectionID), log.BBError(err))
					}
				}

				opts.LogCommandResponse(0, nil, err.Error())

				return err
			}

			allRowsAffected := sqlResult.(mysql.Result).AllRowsAffected()
			var rowsAffected int64
			var allRowsAffectedInt64 []int64
			for _, a := range allRowsAffected {
				rowsAffected += a
				allRowsAffectedInt64 = append(allRowsAffectedInt64, a)
			}
			totalRowsAffected += rowsAffected

			opts.LogCommandResponse(rowsAffected, allRowsAffectedInt64, "")
		}

		if err := tx.Commit(); err != nil {
			opts.LogTransactionControl(storepb.TaskRunLog_TransactionControl_COMMIT, err.Error())
			return errors.Wrapf(err, "failed to commit execute transaction")
		} else {
			opts.LogTransactionControl(storepb.TaskRunLog_TransactionControl_COMMIT, "")
			committed = true
		}
		return nil
	}); err != nil {
		return 0, err
	}

	// Run non-transaction statements at the end.
	for _, stmt := range nonTransactionStmts {
		opts.LogCommandExecute(stmt.Range, stmt.Text)
		if _, err := d.db.ExecContext(ctx, stmt.Text); err != nil {
			opts.LogCommandResponse(0, []int64{0}, err.Error())
			return 0, err
		}
		opts.LogCommandResponse(0, []int64{0}, "")
	}
	return totalRowsAffected, nil
}

// executeInAutoCommitMode executes statements sequentially in auto-commit mode
func (d *Driver) executeInAutoCommitMode(ctx context.Context, conn *sql.Conn, commands []base.Statement, nonTransactionStmts []base.Statement, opts db.ExecuteOptions, connectionID string) (int64, error) {
	var totalRowsAffected int64

	// Execute all statements (including non-transactional ones)
	// In auto-commit mode, execute commands followed by non-transactional statements
	allCommands := make([]base.Statement, 0, len(commands)+len(nonTransactionStmts))
	allCommands = append(allCommands, commands...)
	allCommands = append(allCommands, nonTransactionStmts...)

	if err := conn.Raw(func(driverConn any) error {
		//nolint
		exer := driverConn.(driver.ExecerContext)

		for _, command := range allCommands {
			opts.LogCommandExecute(command.Range, command.Text)

			sqlWithBytebaseAppComment := util.MySQLPrependBytebaseAppComment(command.Text)
			sqlResult, err := exer.ExecContext(ctx, sqlWithBytebaseAppComment, nil)
			if err != nil {
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					slog.Info("cancel connection", slog.String("connectionID", connectionID))
					if err := d.StopConnectionByID(connectionID); err != nil {
						slog.Error("failed to cancel connection", slog.String("connectionID", connectionID), log.BBError(err))
					}
				}

				opts.LogCommandResponse(0, nil, err.Error())
				// In auto-commit mode, we stop at the first error
				// The database is left in a partially migrated state
				return err
			}

			allRowsAffected := sqlResult.(mysql.Result).AllRowsAffected()
			var rowsAffected int64
			var allRowsAffectedInt64 []int64
			for _, a := range allRowsAffected {
				rowsAffected += a
				allRowsAffectedInt64 = append(allRowsAffectedInt64, a)
			}
			totalRowsAffected += rowsAffected

			opts.LogCommandResponse(rowsAffected, allRowsAffectedInt64, "")
		}

		return nil
	}); err != nil {
		return 0, err
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
func (d *Driver) QueryConn(ctx context.Context, conn *sql.Conn, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
	singleSQLs, err := base.SplitMultiSQL(storepb.Engine_TIDB, statement)
	if err != nil {
		return nil, err
	}
	singleSQLs = base.FilterEmptyStatements(singleSQLs)
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

		_, allQuery, err := base.ValidateSQLForEditor(storepb.Engine_TIDB, statement)
		if err != nil {
			return nil, err
		}
		startTime := time.Now()
		queryResult, err := func() (*v1pb.QueryResult, error) {
			if allQuery {
				rows, err := conn.QueryContext(ctx, sqlWithBytebaseAppComment)
				if err != nil {
					return nil, err
				}
				defer rows.Close()
				r, err := util.RowsToQueryResult(rows, makeValueByTypeName, convertValue, queryContext.MaximumSQLResultSize)
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
			return util.BuildAffectedRowsResult(affectedRows, nil), nil
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
		queryResult.RowsCount = int64(len(queryResult.Rows))
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
