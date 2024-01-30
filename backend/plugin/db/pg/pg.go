// Package pg is the plugin for PostgreSQL driver.
package pg

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net"
	"regexp"
	"strconv"
	"strings"
	"time"

	// Import pg driver.
	// init() in pgx/v5/stdlib will register it's pgx driver.
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
	// driverName is the driver name that our driver dependence register, now is "pgx".
	driverName = "pgx"

	_ db.Driver = (*Driver)(nil)
)

func init() {
	db.Register(storepb.Engine_POSTGRES, newDriver)
}

// Driver is the Postgres driver.
type Driver struct {
	dbBinDir string
	config   db.ConnectionConfig

	db        *sql.DB
	sshClient *ssh.Client
	// connectionString is the connection string registered by pgx.
	// Unregister connectionString if we don't need it.
	connectionString string
	databaseName     string
	connectionCtx    db.ConnectionContext
}

func newDriver(config db.DriverConfig) db.Driver {
	return &Driver{
		dbBinDir: config.DbBinDir,
	}
}

// Open opens a Postgres driver.
func (driver *Driver) Open(ctx context.Context, _ storepb.Engine, config db.ConnectionConfig) (db.Driver, error) {
	// Require username for Postgres, as the guessDSN 1st guess is to use the username as the connecting database
	// if database name is not explicitly specified.
	if config.Username == "" {
		return nil, errors.Errorf("user must be set")
	}

	if config.Host == "" {
		return nil, errors.Errorf("host must be set")
	}

	if config.Port == "" {
		return nil, errors.Errorf("port must be set")
	}

	if (config.TLSConfig.SslCert == "" && config.TLSConfig.SslKey != "") ||
		(config.TLSConfig.SslCert != "" && config.TLSConfig.SslKey == "") {
		return nil, errors.Errorf("ssl-cert and ssl-key must be both set or unset")
	}

	connStr := fmt.Sprintf("host=%s port=%s", config.Host, config.Port)
	// TODO(tianzhou): this work-around is no longer needed probably.
	// https://neon.tech/docs/connect/connectivity-issues#c-set-verify-full-for-golang-based-clients
	if strings.HasSuffix(config.Host, ".neon.tech") {
		connStr += " sslmode=verify-full"
	}
	connConfig, err := pgx.ParseConfig(connStr)
	if err != nil {
		return nil, err
	}
	connConfig.Config.User = config.Username
	connConfig.Config.Password = config.Password
	connConfig.Config.Database = config.Database
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
	if config.ReadOnly {
		connConfig.RuntimeParams["default_transaction_read_only"] = "true"
	}

	driver.databaseName = config.Database
	if config.Database == "" {
		databaseName, cfg, err := guessDSN(connConfig, config.Username)
		if err != nil {
			return nil, err
		}
		connConfig = cfg
		driver.databaseName = databaseName
	}
	driver.config = config

	driver.connectionString = stdlib.RegisterConnConfig(connConfig)
	db, err := sql.Open(driverName, driver.connectionString)
	if err != nil {
		return nil, err
	}
	driver.db = db
	if config.ConnectionContext.UseDatabaseOwner {
		owner, err := driver.GetCurrentDatabaseOwner(ctx)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get database owner")
		}
		if _, err := db.ExecContext(ctx, fmt.Sprintf("SET ROLE \"%s\";", owner)); err != nil {
			return nil, errors.Wrapf(err, "failed to set role to database owner %q", owner)
		}
	}
	driver.connectionCtx = config.ConnectionContext
	return driver, nil
}

type noDeadlineConn struct{ net.Conn }

func (*noDeadlineConn) SetDeadline(time.Time) error      { return nil }
func (*noDeadlineConn) SetReadDeadline(time.Time) error  { return nil }
func (*noDeadlineConn) SetWriteDeadline(time.Time) error { return nil }

// guessDSN will guess a valid DB connection and its database name.
func guessDSN(baseConnConfig *pgx.ConnConfig, username string) (string, *pgx.ConnConfig, error) {
	// Some postgres server default behavior is to use username as the database name if not specified,
	// while some postgres server explicitly requires the database name to be present (e.g. render.com).
	guesses := []string{"postgres", username, "template1"}
	//  dsn+" dbname=bytebase"
	for _, guessDatabase := range guesses {
		connConfig := *baseConnConfig
		connConfig.Database = guessDatabase
		if err := func() error {
			connectionString := stdlib.RegisterConnConfig(&connConfig)
			defer stdlib.UnregisterConnConfig(connectionString)
			db, err := sql.Open(driverName, connectionString)
			if err != nil {
				return err
			}
			defer db.Close()
			return db.Ping()
		}(); err != nil {
			slog.Debug("guessDSN attempt failed", log.BBError(err))
			continue
		}
		return guessDatabase, &connConfig, nil
	}
	return "", nil, errors.Errorf("cannot connect to the instance, make sure the connection info is correct")
}

// Close closes the driver.
func (driver *Driver) Close(context.Context) error {
	stdlib.UnregisterConnConfig(driver.connectionString)
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
func (*Driver) GetType() storepb.Engine {
	return storepb.Engine_POSTGRES
}

// GetDB gets the database.
func (driver *Driver) GetDB() *sql.DB {
	return driver.db
}

// getDatabases gets all databases of an instance.
func (driver *Driver) getDatabases(ctx context.Context) ([]*storepb.DatabaseSchemaMetadata, error) {
	var databases []*storepb.DatabaseSchemaMetadata
	rows, err := driver.db.QueryContext(ctx, "SELECT datname, pg_encoding_to_char(encoding), datcollate FROM pg_database;")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		database := &storepb.DatabaseSchemaMetadata{}
		if err := rows.Scan(&database.Name, &database.CharacterSet, &database.Collation); err != nil {
			return nil, err
		}
		databases = append(databases, database)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return databases, nil
}

// getVersion gets the version of Postgres server.
func (driver *Driver) getVersion(ctx context.Context) (string, error) {
	// SHOW server_version_num returns an integer such as 100005, which means 10.0.5.
	// It is more convenient to use SHOW server_version to get the version string.
	// PostgreSQL supports it since 8.2.
	// https://www.postgresql.org/docs/current/functions-info.html
	query := "SHOW server_version_num"
	var version string
	if err := driver.db.QueryRowContext(ctx, query).Scan(&version); err != nil {
		if err == sql.ErrNoRows {
			return "", common.FormatDBErrorEmptyRowWithQuery(query)
		}
		return "", util.FormatErrorWithQuery(err, query)
	}
	versionNum, err := strconv.Atoi(version)
	if err != nil {
		return "", err
	}
	// https://www.postgresql.org/docs/current/libpq-status.html#LIBPQ-PQSERVERVERSION
	// Convert to semantic version.
	major, minor, patch := versionNum/1_00_00, (versionNum/100)%100, versionNum%100
	return fmt.Sprintf("%d.%d.%d", major, minor, patch), nil
}

func (driver *Driver) getPGStatStatementsVersion(ctx context.Context) (string, error) {
	query := "select extversion from pg_extension where extname = 'pg_stat_statements'"
	var version string
	if err := driver.db.QueryRowContext(ctx, query).Scan(&version); err != nil {
		if err == sql.ErrNoRows {
			return "", common.FormatDBErrorEmptyRowWithQuery(query)
		}
		return "", util.FormatErrorWithQuery(err, query)
	}
	return version, nil
}

// Execute will execute the statement. For CREATE DATABASE statement, some types of databases such as Postgres
// will not use transactions to execute the statement but will still use transactions to execute the rest of statements.
func (driver *Driver) Execute(ctx context.Context, statement string, opts db.ExecuteOptions) (int64, error) {
	if opts.CreateDatabase {
		if err := driver.createDatabaseExecute(ctx, statement); err != nil {
			return 0, err
		}
		return 0, nil
	}

	owner, err := driver.GetCurrentDatabaseOwner(ctx)
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
		if isIgnoredStatement(singleSQL.Text) {
			continue
		}
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
			slog.Info("Use superuser privilege to run privileged statements", slog.String("statement", singleSQL.Text))
			singleSQL.Text = fmt.Sprintf("SET LOCAL ROLE NONE;%sSET LOCAL ROLE '%s';", singleSQL.Text, owner)
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
		if _, err := tx.ExecContext(ctx, fmt.Sprintf("SET LOCAL ROLE '%s'", owner)); err != nil {
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

func isIgnoredStatement(stmt string) bool {
	// Extensions created in AWS Aurora PostgreSQL are owned by rdsadmin.
	// We don't have privileges to comment on the extension and have to ignore it.
	upperCaseStmt := strings.ToUpper(strings.TrimLeft(stmt, " \n\t"))
	return strings.HasPrefix(upperCaseStmt, "COMMENT ON EXTENSION")
}

var (
	// DROP DATABASE cannot run inside a transaction block.
	// DROP DATABASE [ IF EXISTS ] name [ [ WITH ] ( option [, ...] ) ]。
	dropDatabaseReg = regexp.MustCompile(`(?i)DROP DATABASE`)
	// CREATE INDEX CONCURRENTLY cannot run inside a transaction block.
	// CREATE [ UNIQUE ] INDEX [ CONCURRENTLY ] [ [ IF NOT EXISTS ] name ] ON [ ONLY ] table_name [ USING method ] ...
	createIndexReg = regexp.MustCompile(`(?i)CREATE(\s+(UNIQUE\s+)?)INDEX(\s+)CONCURRENTLY`)
	// DROP INDEX CONCURRENTLY cannot run inside a transaction block.
	// DROP INDEX [ CONCURRENTLY ] [ IF EXISTS ] name [, ...] [ CASCADE | RESTRICT ].
	dropIndexReg = regexp.MustCompile(`(?i)DROP(\s+)INDEX(\s+)CONCURRENTLY`)
	// VACUUM cannot run inside a transaction block.
	// VACUUM [ ( option [, ...] ) ] [ table_and_columns [, ...] ]
	// VACUUM [ FULL ] [ FREEZE ] [ VERBOSE ] [ ANALYZE ] [ table_and_columns [, ...] ].
	vacuumReg = regexp.MustCompile(`(?i)VACUUM`)
)

func isNonTransactionStatement(stmt string) bool {
	if len(dropDatabaseReg.FindString(stmt)) > 0 {
		return true
	}
	if len(createIndexReg.FindString(stmt)) > 0 {
		return true
	}
	if len(dropIndexReg.FindString(stmt)) > 0 {
		return true
	}
	return len(vacuumReg.FindString(stmt)) > 0
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

// GetCurrentDatabaseOwner gets the role of the current database.
func (driver *Driver) GetCurrentDatabaseOwner(ctx context.Context) (string, error) {
	const query = `
		SELECT
			u.rolname
		FROM
			pg_roles AS u JOIN pg_database AS d ON (d.datdba = u.oid)
		WHERE
			d.datname = current_database();
		`
	var owner string
	if err := driver.db.QueryRowContext(ctx, query).Scan(&owner); err != nil {
		return "", err
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
	// To handle cases where there are comments in the query.
	// eg. select * from t1 -- this is comment;
	// Add two new line symbol here.
	return fmt.Sprintf("WITH result AS (\n%s\n) SELECT * FROM result LIMIT %d;", stmt, limit)
}

func (*Driver) querySingleSQL(ctx context.Context, conn *sql.Conn, singleSQL base.SingleSQL, queryContext *db.QueryContext) (*v1pb.QueryResult, error) {
	statement := strings.Trim(singleSQL.Text, " \n\t;")

	stmt := statement
	if !strings.HasPrefix(stmt, "EXPLAIN") && !strings.HasPrefix(stmt, "SET") && queryContext.Limit > 0 {
		stmt = getStatementWithResultLimit(stmt, queryContext.Limit)
	}

	startTime := time.Now()
	result, err := util.QueryV2(ctx, conn, stmt, queryContext)
	if err != nil {
		return nil, err
	}
	result.Latency = durationpb.New(time.Since(startTime))
	result.Statement = statement
	return result, nil
}

// RunStatement runs a SQL statement in a given connection.
func (*Driver) RunStatement(ctx context.Context, conn *sql.Conn, statement string) ([]*v1pb.QueryResult, error) {
	return util.RunStatement(ctx, storepb.Engine_POSTGRES, conn, statement)
}
