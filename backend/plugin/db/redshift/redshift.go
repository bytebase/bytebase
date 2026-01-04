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
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	pgparser "github.com/bytebase/bytebase/backend/plugin/parser/pg"
)

var (
	excludedDatabaseList = map[string]bool{
		// Skip internal databases from cloud service providers
		// aws
		"padb_harvest":   true,
		"awsdatacatalog": true,
		"sys:internal":   true,
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

func newDriver() db.Driver {
	return &Driver{}
}

// Open opens a Postgres driver.
func (d *Driver) Open(_ context.Context, _ storepb.Engine, config db.ConnectionConfig) (db.Driver, error) {
	if config.DataSource.Username == "" {
		return nil, errors.Errorf("user must be set")
	}

	if (config.DataSource.GetSslCert() == "" && config.DataSource.GetSslKey() != "") ||
		(config.DataSource.GetSslCert() != "" && config.DataSource.GetSslKey() == "") {
		return nil, errors.Errorf("ssl-cert and ssl-key must be both set or unset")
	}

	pgxConnConfig, err := pgx.ParseConfig(fmt.Sprintf("host=%s port=%s", config.DataSource.Host, config.DataSource.Port))
	if err != nil {
		return nil, err
	}
	pgxConnConfig.User = config.DataSource.Username
	pgxConnConfig.Password = config.Password
	if config.ConnectionContext.DatabaseName != "" {
		pgxConnConfig.Database = config.ConnectionContext.DatabaseName
	} else if config.DataSource.GetDatabase() != "" {
		pgxConnConfig.Database = config.DataSource.GetDatabase()
	} else {
		pgxConnConfig.Database = "dev"
	}

	if config.DataSource.GetSslCert() != "" {
		tlscfg, err := util.GetTLSConfig(config.DataSource)
		if err != nil {
			return nil, err
		}
		pgxConnConfig.TLSConfig = tlscfg
	}
	if config.DataSource.GetSshHost() != "" {
		sshClient, err := util.GetSSHClient(config.DataSource)
		if err != nil {
			return nil, err
		}
		d.sshClient = sshClient

		pgxConnConfig.DialFunc = func(_ context.Context, network, addr string) (net.Conn, error) {
			conn, err := sshClient.Dial(network, addr)
			if err != nil {
				return nil, err
			}
			return &util.NoDeadlineConn{Conn: conn}, nil
		}
	}
	d.databaseName = config.ConnectionContext.DatabaseName
	d.datashare = config.ConnectionContext.DataShare
	d.config = config

	// Datashare doesn't support read-only transactions.
	if config.ConnectionContext.ReadOnly && !d.datashare {
		pgxConnConfig.RuntimeParams["default_transaction_read_only"] = "true"
	}

	d.connectionString = stdlib.RegisterConnConfig(pgxConnConfig)
	db, err := sql.Open(driverName, d.connectionString)
	if err != nil {
		return nil, err
	}
	d.db = db
	return d, nil
}

// Close closes the database and prevents new queries from starting.
// Close then waits for all queries that have started processing on the server to finish.
func (d *Driver) Close(context.Context) error {
	stdlib.UnregisterConnConfig(d.connectionString)
	var err error
	err = multierr.Append(err, d.db.Close())
	if d.sshClient != nil {
		err = multierr.Append(err, d.sshClient.Close())
	}
	return err
}

// Ping verifies a connection to the database is still alive, establishing a connection if necessary.
func (d *Driver) Ping(ctx context.Context) error {
	return d.db.PingContext(ctx)
}

// GetDB gets the database.
func (d *Driver) GetDB() *sql.DB {
	return d.db
}

// Execute will execute the statement. For CREATE DATABASE statement, some types of databases such as Postgres
// will not use transactions to execute the statement but will still use transactions to execute the rest of statements.
func (d *Driver) Execute(ctx context.Context, statement string, opts db.ExecuteOptions) (int64, error) {
	if d.datashare {
		return 0, errors.Errorf("datashare database cannot be updated")
	}
	if opts.CreateDatabase {
		if err := d.createDatabaseExecute(ctx, statement); err != nil {
			return 0, err
		}
		return 0, nil
	}

	// Parse transaction mode from the script
	config, cleanedStatement := base.ParseTransactionConfig(statement)
	statement = cleanedStatement
	transactionMode := config.Mode

	// Apply default when transaction mode is not specified
	if transactionMode == common.TransactionModeUnspecified {
		transactionMode = common.GetDefaultTransactionMode()
	}

	var commands []base.Statement
	oneshot := true
	if len(statement) <= common.MaxSheetCheckSize {
		singleSQLs, err := pgparser.SplitSQL(statement)
		if err != nil {
			return 0, err
		}
		commands = base.FilterEmptyStatements(singleSQLs)
		if len(commands) <= common.MaximumCommands {
			oneshot = false
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
		return d.executeInAutoCommitMode(ctx, commands, opts)
	}
	return d.executeInTransactionMode(ctx, commands, opts)
}

func (d *Driver) createDatabaseExecute(ctx context.Context, statement string) error {
	databaseName, err := getDatabaseInCreateDatabaseStatement(statement)
	if err != nil {
		return err
	}
	databases, err := d.getDatabases(ctx)
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
		if _, err := d.db.ExecContext(ctx, s); err != nil {
			return err
		}
	}
	return nil
}

// executeInTransactionMode executes statements within a single transaction
func (d *Driver) executeInTransactionMode(ctx context.Context, commands []base.Statement, opts db.ExecuteOptions) (int64, error) {
	totalRowsAffected := int64(0)
	if len(commands) == 0 {
		return 0, nil
	}

	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		opts.LogTransactionControl(storepb.TaskRunLog_TransactionControl_BEGIN, err.Error())
		return 0, err
	}
	opts.LogTransactionControl(storepb.TaskRunLog_TransactionControl_BEGIN, "")

	committed := false
	defer func() {
		err := tx.Rollback()
		if committed {
			return
		}
		var rerr string
		if err != nil && !errors.Is(err, sql.ErrTxDone) {
			rerr = err.Error()
			slog.Debug("failed to rollback transaction", log.BBError(err))
		}
		opts.LogTransactionControl(storepb.TaskRunLog_TransactionControl_ROLLBACK, rerr)
	}()

	for i, command := range commands {
		opts.LogCommandExecute(command.Range, command.Text)
		// Log the query statement in char code to see if there are some control characters that cause issues.
		var charCode []rune
		for _, r := range command.Text {
			charCode = append(charCode, r)
		}
		slog.Debug("executing command", slog.Any("command", charCode), slog.Int("index", i))
		sqlResult, err := tx.ExecContext(ctx, command.Text)
		if err != nil {
			opts.LogCommandResponse(0, nil, err.Error())
			return 0, err
		}
		rowsAffected, err := sqlResult.RowsAffected()
		if err != nil {
			// Since we cannot differentiate DDL and DML yet, we have to ignore the error.
			slog.Debug("rowsAffected returns error", log.BBError(err))
		}
		opts.LogCommandResponse(rowsAffected, nil, "")
		totalRowsAffected += rowsAffected
	}

	if err := tx.Commit(); err != nil {
		opts.LogTransactionControl(storepb.TaskRunLog_TransactionControl_COMMIT, err.Error())
		return 0, err
	}
	opts.LogTransactionControl(storepb.TaskRunLog_TransactionControl_COMMIT, "")
	committed = true

	return totalRowsAffected, nil
}

// executeInAutoCommitMode executes statements sequentially in auto-commit mode
func (d *Driver) executeInAutoCommitMode(ctx context.Context, commands []base.Statement, opts db.ExecuteOptions) (int64, error) {
	totalRowsAffected := int64(0)

	for i, command := range commands {
		opts.LogCommandExecute(command.Range, command.Text)
		// Log the query statement in char code to see if there are some control characters that cause issues.
		var charCode []rune
		for _, r := range command.Text {
			charCode = append(charCode, r)
		}
		slog.Debug("executing command", slog.Any("command", charCode), slog.Int("index", i))
		sqlResult, err := d.db.ExecContext(ctx, command.Text)
		if err != nil {
			opts.LogCommandResponse(0, nil, err.Error())
			// In auto-commit mode, we stop at the first error
			// The database is left in a partially migrated state
			return totalRowsAffected, err
		}
		rowsAffected, err := sqlResult.RowsAffected()
		if err != nil {
			// Since we cannot differentiate DDL and DML yet, we have to ignore the error.
			slog.Debug("rowsAffected returns error", log.BBError(err))
		}
		opts.LogCommandResponse(rowsAffected, nil, "")
		totalRowsAffected += rowsAffected
	}

	return totalRowsAffected, nil
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
func (d *Driver) GetCurrentDatabaseOwner() (string, error) {
	const query = `
		SELECT
			u.usename
		FROM
			pg_database as d JOIN pg_user as u ON (d.datdba = u.usesysid)
		WHERE d.datname = current_database();
	`
	var owner string
	rows, err := d.db.Query(query)
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
func (d *Driver) QueryConn(ctx context.Context, conn *sql.Conn, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
	singleSQLs, err := pgparser.SplitSQL(statement)
	if err != nil {
		return nil, err
	}
	singleSQLs = base.FilterEmptyStatements(singleSQLs)
	if len(singleSQLs) == 0 {
		return nil, nil
	}

	// If the queryContext.Schema is not empty, set the search path for the database connection to the specified schema.
	// Reference: https://docs.aws.amazon.com/redshift/latest/dg/r_search_path.html
	if queryContext.Schema != "" {
		if _, err := conn.ExecContext(ctx, fmt.Sprintf("set search_path to %s;", queryContext.Schema)); err != nil {
			return nil, err
		}
	}

	var results []*v1pb.QueryResult
	for _, singleSQL := range singleSQLs {
		statement := singleSQL.Text
		if d.datashare {
			statement = strings.ReplaceAll(statement, fmt.Sprintf("%s.", d.databaseName), "")
		}
		if queryContext.Explain {
			statement = fmt.Sprintf("EXPLAIN %s", statement)
		} else if queryContext.Limit > 0 {
			statement = getStatementWithResultLimit(statement, queryContext.Limit)
		}

		_, allQuery, err := base.ValidateSQLForEditor(storepb.Engine_REDSHIFT, statement)
		if err != nil {
			return nil, err
		}
		startTime := time.Now()
		queryResult, err := func() (*v1pb.QueryResult, error) {
			if allQuery {
				rows, err := conn.QueryContext(ctx, statement)
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

func getStatementWithResultLimit(stmt string, limit int) string {
	return fmt.Sprintf("WITH result AS (%s) SELECT * FROM result LIMIT %d;", util.TrimStatement(stmt), limit)
}
