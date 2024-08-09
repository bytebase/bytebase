// Package redshift is the plugin for RedShift driver.
package redshift

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pkg/errors"
	"go.uber.org/multierr"
	"golang.org/x/crypto/ssh"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	pgparser "github.com/bytebase/bytebase/backend/plugin/parser/pg"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

var (
	excludedDatabaseList = map[string]bool{
		// Skip internal databases from cloud service providers
		// aws
		"padb_harvest": true,
		// system templates.
		"template0": true,
		"template1": true,
	}

	// driverName is the driver name that our driver dependence register, now is "pgx".
	driverName = "pgx"

	_ db.Driver = (*Driver)(nil)
)

func init() {
	db.Register(storepb.Engine_REDSHIFT, newDriver)
}

// Driver is the Postgres driver.
type Driver struct {
	config db.ConnectionConfig

	db        *sql.DB
	sshClient *ssh.Client
	// connectionString is the connection string registered by pgx.
	// Unregister connectionString if we don't need it.
	connectionString string
	databaseName     string
	datashare        bool
}

func newDriver(db.DriverConfig) db.Driver {
	return &Driver{}
}

// Open opens a Postgres driver.
func (driver *Driver) Open(_ context.Context, _ storepb.Engine, config db.ConnectionConfig) (db.Driver, error) {
	// Require username for Postgres, as the guessDSN 1st guess is to use the username as the connecting database
	// if database name is not explicitly specified.
	if config.Username == "" {
		return nil, errors.Errorf("user must be set")
	}

	if (config.TLSConfig.SslCert == "" && config.TLSConfig.SslKey != "") ||
		(config.TLSConfig.SslCert != "" && config.TLSConfig.SslKey == "") {
		return nil, errors.Errorf("ssl-cert and ssl-key must be both set or unset")
	}

	connConfig, err := pgx.ParseConfig(fmt.Sprintf("host=%s port=%s", config.Host, config.Port))
	if err != nil {
		return nil, err
	}
	connConfig.Config.User = config.Username
	connConfig.Config.Password = config.Password
	connConfig.Config.Database = config.Database
	if config.ConnectionDatabase != "" {
		connConfig.Config.Database = config.ConnectionDatabase
	}
	if config.TLSConfig.SslCert != "" {
		cfg, err := config.TLSConfig.GetSslConfig()
		if err != nil {
			return nil, err
		}
		connConfig.TLSConfig = cfg
	}
	if config.SSHConfig.Host != "" {
		sshClient, err := util.GetSSHClient(config.SSHConfig)
		if err != nil {
			return nil, err
		}
		driver.sshClient = sshClient

		connConfig.Config.DialFunc = func(_ context.Context, network, addr string) (net.Conn, error) {
			conn, err := sshClient.Dial(network, addr)
			if err != nil {
				return nil, err
			}
			return &noDeadlineConn{Conn: conn}, nil
		}
	}
	driver.databaseName = config.Database
	driver.datashare = config.ConnectionDatabase != ""
	driver.config = config

	// Datashare doesn't support read-only transactions.
	if config.ReadOnly && !driver.datashare {
		connConfig.RuntimeParams["default_transaction_read_only"] = "true"
	}

	driver.connectionString = stdlib.RegisterConnConfig(connConfig)
	db, err := sql.Open(driverName, driver.connectionString)
	if err != nil {
		return nil, err
	}
	driver.db = db
	return driver, nil
}

type noDeadlineConn struct{ net.Conn }

func (*noDeadlineConn) SetDeadline(time.Time) error      { return nil }
func (*noDeadlineConn) SetReadDeadline(time.Time) error  { return nil }
func (*noDeadlineConn) SetWriteDeadline(time.Time) error { return nil }

// Close closes the database and prevents new queries from starting.
// Close then waits for all queries that have started processing on the server to finish.
func (driver *Driver) Close(context.Context) error {
	stdlib.UnregisterConnConfig(driver.connectionString)
	var err error
	err = multierr.Append(err, driver.db.Close())
	if driver.sshClient != nil {
		err = multierr.Append(err, driver.sshClient.Close())
	}
	return err
}

// Ping verifies a connection to the database is still alive, establishing a connection if necessary.
func (driver *Driver) Ping(ctx context.Context) error {
	return driver.db.PingContext(ctx)
}

// GetDB gets the database.
func (driver *Driver) GetDB() *sql.DB {
	return driver.db
}

// Execute will execute the statement. For CREATE DATABASE statement, some types of databases such as Postgres
// will not use transactions to execute the statement but will still use transactions to execute the rest of statements.
func (driver *Driver) Execute(ctx context.Context, statement string, opts db.ExecuteOptions) (int64, error) {
	if driver.datashare {
		return 0, errors.Errorf("datashare database cannot be updated")
	}
	if opts.CreateDatabase {
		if err := driver.createDatabaseExecute(ctx, statement); err != nil {
			return 0, err
		}
		return 0, nil
	}

	owner, err := driver.GetCurrentDatabaseOwner()
	if err != nil {
		return 0, err
	}

	var commands []base.SingleSQL
	oneshot := true
	if len(statement) <= common.MaxSheetCheckSize {
		singleSQLs, err := pgparser.SplitSQL(statement)
		if err != nil {
			return 0, err
		}
		commands = base.FilterEmptySQL(singleSQLs)
		if len(commands) <= common.MaximumCommands {
			oneshot = false
			for _, singleSQL := range commands {
				if isSuperuserStatement(singleSQL.Text) {
					// Use superuser privilege to run privileged statements.
					singleSQL.Text = fmt.Sprintf("SET SESSION AUTHORIZATION NONE;%sSET SESSION AUTHORIZATION '%s';", singleSQL.Text, owner)
				}
			}
		}
	}
	if oneshot {
		commands = []base.SingleSQL{
			{
				Text: statement,
			},
		}
	}
	totalRowsAffected := int64(0)
	if len(commands) != 0 {
		totalCommands := len(commands)
		tx, err := driver.db.BeginTx(ctx, nil)
		if err != nil {
			return 0, err
		}
		defer tx.Rollback()
		// Set the current transaction role to the database owner so that the owner of created objects will be the same as the database owner.
		if _, err := tx.ExecContext(ctx, fmt.Sprintf("SET SESSION AUTHORIZATION '%s'", owner)); err != nil {
			return 0, err
		}

		for i, command := range commands {
			// Start the current chunk.
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

			sqlResult, err := tx.ExecContext(ctx, command.Text)
			if err != nil {
				return 0, &db.ErrorWithPosition{
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
			rowsAffected, err := sqlResult.RowsAffected()
			if err != nil {
				// Since we cannot differentiate DDL and DML yet, we have to ignore the error.
				slog.Debug("rowsAffected returns error", log.BBError(err))
			}
			totalRowsAffected += rowsAffected
		}

		if err := tx.Commit(); err != nil {
			return 0, err
		}
	}

	return totalRowsAffected, nil
}

func (driver *Driver) createDatabaseExecute(ctx context.Context, statement string) error {
	databaseName, err := getDatabaseInCreateDatabaseStatement(statement)
	if err != nil {
		return err
	}
	databases, err := driver.getDatabases(ctx)
	if err != nil {
		return err
	}
	for _, database := range databases {
		if database.Name == databaseName {
			// Database already exists.
			return nil
		}
	}

	for _, s := range strings.Split(statement, "\n") {
		if _, err := driver.db.ExecContext(ctx, s); err != nil {
			return err
		}
	}
	return nil
}

func isSuperuserStatement(stmt string) bool {
	upperCaseStmt := strings.ToUpper(strings.TrimLeft(stmt, " \n\t"))
	return strings.HasPrefix(upperCaseStmt, "GRANT")
}

func getDatabaseInCreateDatabaseStatement(createDatabaseStatement string) (string, error) {
	raw := strings.TrimRight(createDatabaseStatement, ";")
	raw = strings.TrimPrefix(raw, "CREATE DATABASE")
	tokens := strings.Fields(raw)
	if len(tokens) == 0 {
		return "", errors.Errorf("database name not found")
	}
	databaseName := strings.TrimLeft(tokens[0], `"`)
	databaseName = strings.TrimRight(databaseName, `"`)
	return databaseName, nil
}

// GetCurrentDatabaseOwner returns the current database owner name.
func (driver *Driver) GetCurrentDatabaseOwner() (string, error) {
	const query = `
		SELECT
			u.usename
		FROM
			pg_database as d JOIN pg_user as u ON (d.datdba = u.usesysid)
		WHERE d.datname = current_database();
	`
	var owner string
	rows, err := driver.db.Query(query)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	for rows.Next() {
		if err := rows.Scan(&owner); err != nil {
			return "", err
		}
	}
	if err := rows.Err(); err != nil {
		return "", err
	}
	if owner == "" {
		return "", errors.Errorf("cannot find the current database owner because the query result is empty")
	}
	return owner, nil
}

// QueryConn queries a SQL statement in a given connection.
func (driver *Driver) QueryConn(ctx context.Context, conn *sql.Conn, statement string, queryContext *db.QueryContext) ([]*v1pb.QueryResult, error) {
	singleSQLs, err := pgparser.SplitSQL(statement)
	if err != nil {
		return nil, err
	}
	singleSQLs = base.FilterEmptySQL(singleSQLs)
	if len(singleSQLs) == 0 {
		return nil, nil
	}

	var results []*v1pb.QueryResult
	for _, singleSQL := range singleSQLs {
		statement := singleSQL.Text
		if driver.datashare {
			statement = strings.ReplaceAll(statement, fmt.Sprintf("%s.", queryContext.CurrentDatabase), "")
		}
		if queryContext != nil && queryContext.Explain {
			statement = fmt.Sprintf("EXPLAIN %s", statement)
		} else if queryContext != nil && queryContext.Limit > 0 {
			statement = getStatementWithResultLimit(statement, queryContext.Limit)
		}

		_, allQuery, err := base.ValidateSQLForEditor(storepb.Engine_POSTGRES, statement)
		if err != nil {
			return nil, err
		}
		startTime := time.Now()
		queryResult, err := func() (*v1pb.QueryResult, error) {
			if allQuery {
				rows, err := conn.QueryContext(ctx, statement)
				if err != nil {
					return nil, util.FormatErrorWithQuery(err, statement)
				}
				defer rows.Close()
				r, err := util.RowsToQueryResult(rows, driver.config.MaximumSQLResultSize)
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

func getStatementWithResultLimit(stmt string, limit int) string {
	return fmt.Sprintf("WITH result AS (%s) SELECT * FROM result LIMIT %d;", stmt, limit)
}
