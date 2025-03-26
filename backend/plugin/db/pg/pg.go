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
	"unicode"

	"cloud.google.com/go/cloudsqlconn"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/rds/auth"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pkg/errors"
	"go.uber.org/multierr"
	"golang.org/x/crypto/ssh"
	"google.golang.org/protobuf/types/known/durationpb"

	pgquery "github.com/pganalyze/pg_query_go/v5"

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
	var pgxConnConfig *pgx.ConnConfig
	var err error

	switch config.DataSource.GetAuthenticationType() {
	case storepb.DataSource_GOOGLE_CLOUD_SQL_IAM:
		pgxConnConfig, err = getCloudSQLConnectionConfig(ctx, config)
	case storepb.DataSource_AWS_RDS_IAM:
		pgxConnConfig, err = getRDSConnectionConfig(ctx, config)
	default:
		pgxConnConfig, err = getPGConnectionConfig(config)
	}
	if err != nil {
		return nil, err
	}
	pgxConnConfig.RuntimeParams["application_name"] = "bytebase"
	if config.ConnectionContext.ReadOnly {
		pgxConnConfig.RuntimeParams["default_transaction_read_only"] = "true"
	}

	if config.DataSource.GetSshHost() != "" {
		sshClient, err := util.GetSSHClient(config.DataSource)
		if err != nil {
			return nil, err
		}
		driver.sshClient = sshClient

		pgxConnConfig.Config.DialFunc = func(_ context.Context, network, addr string) (net.Conn, error) {
			conn, err := sshClient.Dial(network, addr)
			if err != nil {
				return nil, err
			}
			return &util.NoDeadlineConn{Conn: conn}, nil
		}
	}

	driver.databaseName = config.ConnectionContext.DatabaseName
	if config.ConnectionContext.DatabaseName != "" {
		pgxConnConfig.Database = config.ConnectionContext.DatabaseName
	} else if config.DataSource.GetDatabase() != "" {
		pgxConnConfig.Database = config.DataSource.GetDatabase()
	} else {
		pgxConnConfig.Database = "postgres"
	}
	driver.config = config

	driver.connectionString = stdlib.RegisterConnConfig(pgxConnConfig)
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
		if _, err := driver.db.ExecContext(ctx, fmt.Sprintf("SET ROLE \"%s\";", owner)); err != nil {
			return nil, errors.Wrapf(err, "failed to set role to database owner %q", owner)
		}
	}
	driver.connectionCtx = config.ConnectionContext
	return driver, nil
}

func getPGConnectionConfig(config db.ConnectionConfig) (*pgx.ConnConfig, error) {
	if config.DataSource.Username == "" {
		return nil, errors.Errorf("user must be set")
	}

	if config.DataSource.Host == "" {
		return nil, errors.Errorf("host must be set")
	}

	if config.DataSource.Port == "" {
		return nil, errors.Errorf("port must be set")
	}

	if (config.DataSource.GetSslCert() == "" && config.DataSource.GetSslKey() != "") ||
		(config.DataSource.GetSslCert() != "" && config.DataSource.GetSslKey() == "") {
		return nil, errors.Errorf("ssl-cert and ssl-key must be both set or unset")
	}

	connStr := fmt.Sprintf("host=%s port=%s", config.DataSource.Host, config.DataSource.Port)
	if config.DataSource.GetUseSsl() {
		connStr += fmt.Sprintf(" sslmode=%s", util.GetPGSSLMode(config.DataSource))
	}

	// Add target_session_attrs=read-write if specified in ExtraConnectionParameters
	if len(config.DataSource.GetExtraConnectionParameters()) > 0 {
		for key, value := range config.DataSource.GetExtraConnectionParameters() {
			connStr += fmt.Sprintf(" %s=%s", key, value)
		}
	}

	connConfig, err := pgx.ParseConfig(connStr)
	if err != nil {
		return nil, err
	}
	connConfig.Config.User = config.DataSource.Username
	connConfig.Config.Password = config.Password
	connConfig.Config.Database = config.ConnectionContext.DatabaseName

	tlscfg, err := util.GetTLSConfig(config.DataSource)
	if err != nil {
		return nil, err
	}
	if tlscfg != nil {
		connConfig.TLSConfig = tlscfg
	}

	return connConfig, nil
}

func getRDSConnectionPassword(ctx context.Context, conf db.ConnectionConfig) (string, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return "", errors.Wrap(err, "load aws config failed")
	}

	dbEndpoint := fmt.Sprintf("%s:%s", conf.DataSource.Host, conf.DataSource.Port)
	authenticationToken, err := auth.BuildAuthToken(
		ctx, dbEndpoint, conf.DataSource.GetRegion(), conf.DataSource.Username, cfg.Credentials)
	if err != nil {
		return "", errors.Wrap(err, "failed to create authentication token")
	}

	return authenticationToken, nil
}

// getRDSConnectionConfig returns connection config for AWS RDS.
//
// https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/UsingWithRDS.IAMDBAuth.Connecting.Go.html
func getRDSConnectionConfig(ctx context.Context, conf db.ConnectionConfig) (*pgx.ConnConfig, error) {
	password, err := getRDSConnectionPassword(ctx, conf)
	if err != nil {
		return nil, err
	}

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s",
		conf.DataSource.Host, conf.DataSource.Port, conf.DataSource.Username, password,
	)
	return pgx.ParseConfig(dsn)
}

// getCloudSQLConnectionConfig returns config for Cloud SQL connector.
// refs:
// https://cloud.google.com/sql/docs/postgres/connect-connectors
// https://github.com/GoogleCloudPlatform/golang-samples/blob/main/cloudsql/postgres/database-sql/cloudsql.go
func getCloudSQLConnectionConfig(ctx context.Context, conf db.ConnectionConfig) (*pgx.ConnConfig, error) {
	d, err := cloudsqlconn.NewDialer(ctx, cloudsqlconn.WithIAMAuthN())
	if err != nil {
		return nil, err
	}

	dsn := fmt.Sprintf("user=%s", conf.DataSource.Username)
	config, err := pgx.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}
	config.DialFunc = func(ctx context.Context, _, _ string) (net.Conn, error) {
		return d.Dial(ctx, conf.DataSource.Host)
	}

	return config, nil
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

// GetDB gets the database.
func (driver *Driver) GetDB() *sql.DB {
	return driver.db
}

// getDatabases gets all databases of an instance.
func (driver *Driver) getDatabases(ctx context.Context) ([]*storepb.DatabaseSchemaMetadata, error) {
	var databases []*storepb.DatabaseSchemaMetadata
	rows, err := driver.db.QueryContext(ctx, "SELECT datname, pg_encoding_to_char(encoding), datcollate, pg_catalog.pg_get_userbyid(datdba) as db_owner FROM pg_database;")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		database := &storepb.DatabaseSchemaMetadata{}
		if err := rows.Scan(&database.Name, &database.CharacterSet, &database.Collation, &database.Owner); err != nil {
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

	var commands []base.SingleSQL
	var originalIndex []int32
	var nonTransactionAndSetRoleStmts []string
	var nonTransactionAndSetRoleStmtsIndex []int32
	var isPlsql bool
	oneshot := true
	// HACK(p0ny): always split for pg
	//nolint
	if true || len(statement) <= common.MaxSheetCheckSize {
		singleSQLs, err := pgparser.SplitSQL(statement)
		if err != nil {
			return 0, err
		}
		commands, originalIndex = base.FilterEmptySQLWithIndexes(singleSQLs)

		// If the statement is a single statement and is a PL/pgSQL block,
		// we should execute it as a single statement without transaction.
		// If the statement is a PL/pgSQL block, we should execute it as a single statement.
		// https://www.postgresql.org/docs/current/plpgsql-control-structures.html
		if len(singleSQLs) == 1 && isPlSQLBlock(singleSQLs[0].Text) {
			isPlsql = true
		}
		// HACK(p0ny): always split for pg
		//nolint
		if false && len(commands) <= common.MaximumCommands {
			oneshot = false
		}

		var tmpCommands []base.SingleSQL
		var tmpOriginalIndex []int32
		for i, command := range commands {
			switch {
			case isSetRoleStatement(command.Text):
				nonTransactionAndSetRoleStmts = append(nonTransactionAndSetRoleStmts, command.Text)
				nonTransactionAndSetRoleStmtsIndex = append(nonTransactionAndSetRoleStmtsIndex, originalIndex[i])
			case IsNonTransactionStatement(command.Text):
				nonTransactionAndSetRoleStmts = append(nonTransactionAndSetRoleStmts, command.Text)
				nonTransactionAndSetRoleStmtsIndex = append(nonTransactionAndSetRoleStmtsIndex, originalIndex[i])
				continue
			case isSuperuserStatement(command.Text):
				// Use superuser privilege to run privileged statements.
				slog.Info("Use superuser privilege to run privileged statements", slog.String("statement", command.Text))
				ct := command.Text
				if !strings.HasSuffix(strings.TrimRightFunc(ct, unicode.IsSpace), ";") {
					ct += ";"
				}
				command.Text = fmt.Sprintf("SET LOCAL ROLE NONE;%sSET LOCAL ROLE '%s';", ct, owner)
			}
			tmpCommands = append(tmpCommands, command)
			tmpOriginalIndex = append(tmpOriginalIndex, originalIndex[i])
		}
		commands, originalIndex = tmpCommands, tmpOriginalIndex
	}
	// HACK(p0ny): always split for pg
	//nolint
	if false && oneshot {
		commands = []base.SingleSQL{
			{
				Text: statement,
			},
		}
		originalIndex = []int32{0}
	}

	conn, err := driver.db.Conn(ctx)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to get connection")
	}
	defer conn.Close()

	if opts.SetConnectionID != nil {
		var pid string
		if err := conn.QueryRowContext(ctx, "SELECT pg_backend_pid()").Scan(&pid); err != nil {
			return 0, errors.Wrapf(err, "failed to get connection id")
		}
		opts.SetConnectionID(pid)

		if opts.DeleteConnectionID != nil {
			defer opts.DeleteConnectionID()
		}
	}

	if isPlsql {
		if driver.connectionCtx.UseDatabaseOwner {
			// USE SET SESSION ROLE to set the role for the current session.
			if _, err := conn.ExecContext(ctx, fmt.Sprintf("SET SESSION ROLE '%s'", owner)); err != nil {
				return 0, errors.Wrapf(err, "failed to set role to database owner %q", owner)
			}
		}
		opts.LogCommandExecute([]int32{0})
		if _, err := conn.ExecContext(ctx, statement); err != nil {
			opts.LogCommandResponse([]int32{0}, 0, []int32{0}, err.Error())
			return 0, err
		}
		opts.LogCommandResponse([]int32{0}, 0, []int32{0}, "")

		return 0, nil
	}

	totalRowsAffected := int64(0)

	totalCommands := len(commands)
	if totalCommands > 0 {
		err = conn.Raw(func(driverConn any) error {
			conn := driverConn.(*stdlib.Conn).Conn()

			tx, err := conn.Begin(ctx)
			if err != nil {
				opts.LogTransactionControl(storepb.TaskRunLog_TransactionControl_BEGIN, err.Error())
				return errors.Wrapf(err, "failed to begin transaction")
			}
			opts.LogTransactionControl(storepb.TaskRunLog_TransactionControl_BEGIN, "")

			committed := false
			defer func() {
				err := tx.Rollback(ctx)
				if committed {
					return
				}
				var rerr string
				if err != nil {
					rerr = err.Error()
				}
				opts.LogTransactionControl(storepb.TaskRunLog_TransactionControl_ROLLBACK, rerr)
			}()

			if driver.connectionCtx.UseDatabaseOwner {
				// Set the current transaction role to the database owner so that the owner of created objects will be the same as the database owner.
				if _, err := tx.Exec(ctx, fmt.Sprintf("SET LOCAL ROLE '%s'", owner)); err != nil {
					return err
				}
			}

			for i, command := range commands {
				indexes := []int32{originalIndex[i]}
				opts.LogCommandExecute(indexes)

				rr := tx.Conn().PgConn().Exec(ctx, command.Text)
				results, err := rr.ReadAll()
				if err != nil {
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

				var rowsAffected int64
				var allRowsAffected []int32
				for _, result := range results {
					ra := result.CommandTag.RowsAffected()
					allRowsAffected = append(allRowsAffected, int32(ra))
					rowsAffected += ra
				}
				opts.LogCommandResponse(indexes, int32(rowsAffected), allRowsAffected, "")

				totalRowsAffected += rowsAffected
			}

			if err := tx.Commit(ctx); err != nil {
				opts.LogTransactionControl(storepb.TaskRunLog_TransactionControl_COMMIT, err.Error())
				return errors.Wrapf(err, "failed to commit transaction")
			}
			opts.LogTransactionControl(storepb.TaskRunLog_TransactionControl_COMMIT, "")
			committed = true

			return nil
		})
		if err != nil {
			return 0, err
		}
	}

	if driver.connectionCtx.UseDatabaseOwner {
		// USE SET SESSION ROLE to set the role for the current session.
		if _, err := conn.ExecContext(ctx, fmt.Sprintf("SET SESSION ROLE '%s'", owner)); err != nil {
			return 0, errors.Wrapf(err, "failed to set role to database owner %q", owner)
		}
	}
	// Run non-transaction statements at the end.
	for i, stmt := range nonTransactionAndSetRoleStmts {
		indexes := []int32{nonTransactionAndSetRoleStmtsIndex[i]}
		opts.LogCommandExecute(indexes)
		if _, err := conn.ExecContext(ctx, stmt); err != nil {
			opts.LogCommandResponse(indexes, 0, []int32{0}, err.Error())
			return 0, err
		}
		opts.LogCommandResponse(indexes, 0, []int32{0}, "")
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
	vacuumReg = regexp.MustCompile(`(?i)^\s*VACUUM`)
	// SET ROLE is a special statement that should be run before any other statements containing inside a transaction block or not.
	setRoleReg = regexp.MustCompile(`(?i)SET\s+((SESSION|LOCAL)\s+)?ROLE`)
)

func isSetRoleStatement(stmt string) bool {
	return len(setRoleReg.FindString(stmt)) > 0
}

func IsNonTransactionStatement(stmt string) bool {
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

func isSuperuserStatement(stmt string) bool {
	upperCaseStmt := strings.ToUpper(strings.TrimLeftFunc(stmt, unicode.IsSpace))
	if strings.HasPrefix(upperCaseStmt, "GRANT") || strings.HasPrefix(upperCaseStmt, "CREATE EXTENSION") || strings.HasPrefix(upperCaseStmt, "CREATE EVENT TRIGGER") || strings.HasPrefix(upperCaseStmt, "COMMENT ON EVENT TRIGGER") {
		return true
	}
	return false
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
func (*Driver) QueryConn(ctx context.Context, conn *sql.Conn, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
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
		if queryContext.Explain {
			statement = fmt.Sprintf("EXPLAIN %s", statement)
		} else if queryContext.Limit > 0 {
			// Quick fix for do not add limit to non-select statement.
			t := util.TrimStatement(statement)
			if strings.HasPrefix(strings.ToUpper(t), "SELECT") || strings.HasPrefix(strings.ToUpper(t), "WITH") {
				statement = getStatementWithResultLimit(statement, queryContext.Limit)
			}
		}

		_, allQuery, err := base.ValidateSQLForEditor(storepb.Engine_POSTGRES, statement)
		if err != nil {
			return nil, err
		}

		// Sanitize the schema name by escaping any quotes.
		safeSchemeName := strings.ReplaceAll(queryContext.Schema, "\"", "\"\"")

		// If the queryContext.Schema is not empty, set the search path for the database connection to the specified schema.
		if queryContext.Schema != "" {
			if _, err := conn.ExecContext(ctx, fmt.Sprintf(`SET search_path TO "%s";`, safeSchemeName)); err != nil {
				return nil, err
			}
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
			return util.BuildAffectedRowsResult(affectedRows), nil
		}()
		stop := false
		if err != nil {
			queryResult = &v1pb.QueryResult{
				Error:         err.Error(),
				DetailedError: getPgError(err),
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

func getPgError(e error) *v1pb.QueryResult_PostgresError_ {
	if e == nil {
		return nil
	}
	var pge *pgconn.PgError
	if errors.As(e, &pge) {
		return &v1pb.QueryResult_PostgresError_{
			PostgresError: &v1pb.QueryResult_PostgresError{
				Severity:         pge.Severity,
				Code:             pge.Code,
				Message:          pge.Message,
				Detail:           pge.Detail,
				Hint:             pge.Hint,
				Position:         pge.Position,
				InternalPosition: pge.InternalPosition,
				InternalQuery:    pge.InternalQuery,
				Where:            pge.Where,
				SchemaName:       pge.SchemaName,
				TableName:        pge.TableName,
				ColumnName:       pge.ColumnName,
				DataTypeName:     pge.DataTypeName,
				ConstraintName:   pge.ConstraintName,
				File:             pge.File,
				Line:             pge.Line,
				Routine:          pge.Routine,
			},
		}
	}
	return nil
}

func getStatementWithResultLimit(stmt string, limit int) string {
	// To handle cases where there are comments in the query.
	// eg. select * from t1 -- this is comment;
	// Add two new line symbol here.
	return fmt.Sprintf("WITH result AS (\n%s\n) SELECT * FROM result LIMIT %d;", util.TrimStatement(stmt), limit)
}

func isPlSQLBlock(stmt string) bool {
	tree, err := pgquery.Parse(stmt)
	if err != nil {
		return false
	}

	if len(tree.Stmts) != 1 {
		return false
	}

	if _, ok := tree.Stmts[0].Stmt.Node.(*pgquery.Node_DoStmt); ok {
		return true
	}

	return false
}
