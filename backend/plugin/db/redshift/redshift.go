// Package redshift is the plugin for RedShift driver.
package redshift

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net"
	"regexp"
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

		connConfig.Config.DialFunc = func(ctx context.Context, network, addr string) (net.Conn, error) {
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

// GetType returns the database type.
func (*Driver) GetType() storepb.Engine {
	return storepb.Engine_REDSHIFT
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
	singleSQLs, err := pgparser.SplitSQL(statement)
	if err != nil {
		return 0, err
	}
	singleSQLs = base.FilterEmptySQL(singleSQLs)
	if len(singleSQLs) == 0 {
		return 0, nil
	}

	var remainingSQLs []base.SingleSQL
	var nonTransactionStmts []string
	for _, singleSQL := range singleSQLs {
		if isNonTransactionStatement(singleSQL.Text) {
			nonTransactionStmts = append(nonTransactionStmts, singleSQL.Text)
			continue
		}

		if isSuperuserStatement(singleSQL.Text) {
			// CREATE EVENT TRIGGER statement only supports EXECUTE PROCEDURE in version 10 and before, while newer version supports both EXECUTE { FUNCTION | PROCEDURE }.
			// Since we use pg_dump version 14, the dump uses a new style even for an old version of PostgreSQL.
			// We should convert EXECUTE FUNCTION to EXECUTE PROCEDURE to make the restoration work on old versions.
			// https://www.postgresql.org/docs/14/sql-createeventtrigger.html
			if strings.Contains(strings.ToUpper(singleSQL.Text), "CREATE EVENT TRIGGER") {
				singleSQL.Text = strings.ReplaceAll(singleSQL.Text, "EXECUTE FUNCTION", "EXECUTE PROCEDURE")
			}
			// Use superuser privilege to run privileged statements.
			singleSQL.Text = fmt.Sprintf("SET SESSION AUTHORIZATION NONE;%sSET SESSION AUTHORIZATION '%s';", singleSQL.Text, owner)
		}
		remainingSQLs = append(remainingSQLs, singleSQL)
	}

	totalRowsAffected := int64(0)
	if len(remainingSQLs) != 0 {
		var totalCommands int
		var chunks [][]base.SingleSQL
		if opts.ChunkedSubmission && len(statement) <= common.MaxSheetCheckSize {
			totalCommands = len(remainingSQLs)
			ret, err := util.ChunkedSQLScript(remainingSQLs, common.MaxSheetChunksCount)
			if err != nil {
				return 0, errors.Wrapf(err, "failed to chunk sql")
			}
			chunks = ret
		} else {
			chunks = [][]base.SingleSQL{
				remainingSQLs,
			}
		}
		currentIndex := 0

		tx, err := driver.db.BeginTx(ctx, nil)
		if err != nil {
			return 0, err
		}
		defer tx.Rollback()
		// Set the current transaction role to the database owner so that the owner of created objects will be the same as the database owner.
		if _, err := tx.ExecContext(ctx, fmt.Sprintf("SET SESSION AUTHORIZATION '%s'", owner)); err != nil {
			return 0, err
		}

		for _, chunk := range chunks {
			if len(chunk) == 0 {
				continue
			}
			// Start the current chunk.
			// Set the progress information for the current chunk.
			if opts.UpdateExecutionStatus != nil {
				opts.UpdateExecutionStatus(&v1pb.TaskRun_ExecutionDetail{
					CommandsTotal:     int32(totalCommands),
					CommandsCompleted: int32(currentIndex),
					CommandStartPosition: &v1pb.TaskRun_ExecutionDetail_Position{
						Line:   int32(chunk[0].FirstStatementLine),
						Column: int32(chunk[0].FirstStatementColumn),
					},
					CommandEndPosition: &v1pb.TaskRun_ExecutionDetail_Position{
						Line:   int32(chunk[len(chunk)-1].LastLine),
						Column: int32(chunk[len(chunk)-1].LastColumn),
					},
				})
			}

			chunkText, err := util.ConcatChunk(chunk)
			if err != nil {
				return 0, err
			}

			sqlResult, err := tx.ExecContext(ctx, chunkText)
			if err != nil {
				return 0, &db.ErrorWithPosition{
					Err: errors.Wrapf(err, "failed to execute context in a transaction"),
					Start: &storepb.TaskRunResult_Position{
						Line:   int32(chunk[0].FirstStatementLine),
						Column: int32(chunk[0].FirstStatementColumn),
					},
					End: &storepb.TaskRunResult_Position{
						Line:   int32(chunk[len(chunk)-1].LastLine),
						Column: int32(chunk[len(chunk)-1].LastColumn),
					},
				}
			}
			rowsAffected, err := sqlResult.RowsAffected()
			if err != nil {
				// Since we cannot differentiate DDL and DML yet, we have to ignore the error.
				slog.Debug("rowsAffected returns error", log.BBError(err))
			}
			totalRowsAffected += rowsAffected
			currentIndex += len(chunk)
		}

		// Restore the current transaction role to the current user.
		if _, err := tx.ExecContext(ctx, "SET SESSION AUTHORIZATION DEFAULT"); err != nil {
			slog.Warn("Failed to restore the current transaction role to the current user", log.BBError(err))
		}

		if err := tx.Commit(); err != nil {
			return 0, err
		}
	}

	// Run non-transaction statements at the end.
	for _, stmt := range nonTransactionStmts {
		if _, err := driver.db.ExecContext(ctx, stmt); err != nil {
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
	if strings.HasPrefix(upperCaseStmt, "GRANT") || strings.HasPrefix(upperCaseStmt, "CREATE EXTENSION") || strings.HasPrefix(upperCaseStmt, "CREATE EVENT TRIGGER") || strings.HasPrefix(upperCaseStmt, "COMMENT ON EVENT TRIGGER") {
		return true
	}
	return false
}

func isNonTransactionStatement(stmt string) bool {
	// CREATE INDEX CONCURRENTLY cannot run inside a transaction block.
	// CREATE [ UNIQUE ] INDEX [ CONCURRENTLY ] [ [ IF NOT EXISTS ] name ] ON [ ONLY ] table_name [ USING method ] ...
	createIndexReg := regexp.MustCompile(`(?i)CREATE(\s+(UNIQUE\s+)?)INDEX(\s+)CONCURRENTLY`)
	if len(createIndexReg.FindString(stmt)) > 0 {
		return true
	}

	// DROP INDEX CONCURRENTLY cannot run inside a transaction block.
	// DROP INDEX [ CONCURRENTLY ] [ IF EXISTS ] name [, ...] [ CASCADE | RESTRICT ]
	dropIndexReg := regexp.MustCompile(`(?i)DROP(\s+)INDEX(\s+)CONCURRENTLY`)
	return len(dropIndexReg.FindString(stmt)) > 0
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
	singleSQLs = base.FilterEmptySQL(singleSQLs)
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

func getStatementWithResultLimit(stmt string, limit int) string {
	return fmt.Sprintf("WITH result AS (%s) SELECT * FROM result LIMIT %d;", stmt, limit)
}

func (driver *Driver) querySingleSQL(ctx context.Context, conn *sql.Conn, singleSQL base.SingleSQL, queryContext *db.QueryContext) (*v1pb.QueryResult, error) {
	statement := strings.Trim(singleSQL.Text, " \n\t;")

	stmt := statement
	if !strings.HasPrefix(stmt, "EXPLAIN") && queryContext.Limit > 0 {
		stmt = getStatementWithResultLimit(stmt, queryContext.Limit)
	}
	// Datashare doesn't support read-only transactions.
	if driver.datashare {
		queryContext.ReadOnly = false
		queryContext.ShareDB = true
	}

	startTime := time.Now()
	result, err := util.Query(ctx, storepb.Engine_REDSHIFT, conn, stmt, queryContext)
	if err != nil {
		return nil, err
	}
	result.Latency = durationpb.New(time.Since(startTime))
	result.Statement = statement
	return result, nil
}

// RunStatement runs a SQL statement in a given connection.
func (*Driver) RunStatement(ctx context.Context, conn *sql.Conn, statement string) ([]*v1pb.QueryResult, error) {
	return util.RunStatement(ctx, storepb.Engine_REDSHIFT, conn, statement)
}
