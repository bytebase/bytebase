// Package cockroachdb is the plugin for CockroachDB driver.
package cockroachdb

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/cockroachdb/cockroach-go/v2/crdb"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pkg/errors"
	"go.uber.org/multierr"
	"golang.org/x/crypto/ssh"
	"google.golang.org/protobuf/types/known/durationpb"

	pgquery "github.com/pganalyze/pg_query_go/v6"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	crdbparser "github.com/bytebase/bytebase/backend/plugin/parser/cockroachdb"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

var (
	// driverName is the driver name that our driver dependence register, now is "pgx".
	driverName = "pgx"

	_ db.Driver = (*Driver)(nil)
)

func init() {
	db.Register(storepb.Engine_COCKROACHDB, newDriver)
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
	connectionCtx    db.ConnectionContext
}

func newDriver() db.Driver {
	return &Driver{}
}

// Open opens a Postgres driver.
func (d *Driver) Open(ctx context.Context, _ storepb.Engine, config db.ConnectionConfig) (db.Driver, error) {
	pgxConnConfig, err := getCockroachConnectionConfig(config)
	if err != nil {
		return nil, err
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
	if config.ConnectionContext.DatabaseName != "" {
		pgxConnConfig.Database = config.ConnectionContext.DatabaseName
	} else if config.DataSource.GetDatabase() != "" {
		pgxConnConfig.Database = config.DataSource.GetDatabase()
	} else {
		pgxConnConfig.Database = "postgres"
	}
	d.config = config

	d.connectionString = stdlib.RegisterConnConfig(pgxConnConfig)
	db, err := sql.Open(driverName, d.connectionString)
	if err != nil {
		return nil, err
	}
	d.db = db
	if config.ConnectionContext.UseDatabaseOwner {
		owner, err := d.GetCurrentDatabaseOwner(ctx)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get database owner")
		}
		if err := crdb.Execute(func() error {
			_, err := d.db.ExecContext(ctx, fmt.Sprintf("SET ROLE \"%s\";", owner))
			return err
		}); err != nil {
			return nil, errors.Wrapf(err, "failed to set role to database owner %q", owner)
		}
	}
	d.connectionCtx = config.ConnectionContext
	return d, nil
}

// getRoutingIDFromCockroachCloudURL returns the routing ID from the Cockroach Cloud URL, returns empty string if not found.
func getRoutingIDFromCockroachCloudURL(host string) string {
	host = strings.TrimSpace(host)
	if !strings.HasSuffix(host, "cockroachlabs.cloud") {
		return ""
	}
	parts := strings.Split(host, ".")
	// routing-id[.xxx].cockroachlabs.cloud
	if len(parts) > 2 {
		return parts[0]
	}
	return ""
}

func getCockroachConnectionConfig(config db.ConnectionConfig) (*pgx.ConnConfig, error) {
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

	routingID := getRoutingIDFromCockroachCloudURL(config.DataSource.Host)
	if routingID != "" {
		connStr += fmt.Sprintf(" options='--cluster=%s'", routingID)
	}

	connConfig, err := pgx.ParseConfig(connStr)
	if err != nil {
		return nil, err
	}
	connConfig.User = config.DataSource.Username
	connConfig.Password = config.Password
	connConfig.Database = config.ConnectionContext.DatabaseName

	tlscfg, err := util.GetTLSConfig(config.DataSource)
	if err != nil {
		return nil, err
	}
	if tlscfg != nil {
		connConfig.TLSConfig = tlscfg
	}
	if config.ConnectionContext.ReadOnly {
		connConfig.RuntimeParams["default_transaction_read_only"] = "true"
		connConfig.RuntimeParams["application_name"] = "bytebase"
	}

	return connConfig, nil
}

// Close closes the driver.
func (d *Driver) Close(context.Context) error {
	stdlib.UnregisterConnConfig(d.connectionString)
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

// getDatabases gets all databases of an instance.
func (d *Driver) getDatabases(ctx context.Context) ([]*storepb.DatabaseSchemaMetadata, error) {
	var databases []*storepb.DatabaseSchemaMetadata
	if err := crdb.Execute(func() error {
		rows, err := d.db.QueryContext(ctx, "SELECT datname, pg_encoding_to_char(encoding), datcollate FROM pg_database;")
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			database := &storepb.DatabaseSchemaMetadata{}
			if err := rows.Scan(&database.Name, &database.CharacterSet, &database.Collation); err != nil {
				return err
			}
			databases = append(databases, database)
		}
		err = rows.Err()
		return err
	}); err != nil {
		return nil, err
	}

	return databases, nil
}

func (d *Driver) getVersion(ctx context.Context) (string, error) {
	// https://www.cockroachlabs.com/docs/v25.1/cluster-settings#setting-version
	query := "SHOW CLUSTER SETTING version;"
	var version string
	if err := d.db.QueryRowContext(ctx, query).Scan(&version); err != nil {
		if err == sql.ErrNoRows {
			return "", common.FormatDBErrorEmptyRowWithQuery(query)
		}
		return "", util.FormatErrorWithQuery(err, query)
	}
	return version, nil
}

// Execute will execute the statement. For CREATE DATABASE statement, some types of databases such as Postgres
// will not use transactions to execute the statement but will still use transactions to execute the rest of statements.
func (d *Driver) Execute(ctx context.Context, statement string, opts db.ExecuteOptions) (int64, error) {
	if opts.CreateDatabase {
		if err := d.createDatabaseExecute(ctx, statement); err != nil {
			return 0, err
		}
		return 0, nil
	}

	owner, err := d.GetCurrentDatabaseOwner(ctx)
	if err != nil {
		return 0, err
	}

	var commands []base.SingleSQL
	var originalIndex []int32
	var nonTransactionAndSetRoleStmts []string
	var nonTransactionAndSetRoleStmtsIndex []int32
	var isPlsql bool

	singleSQLs, err := crdbparser.SplitSQLStatement(statement)
	if err != nil {
		return 0, err
	}
	for i, singleSQL := range singleSQLs {
		commands = append(commands, base.SingleSQL{Text: singleSQL})
		originalIndex = append(originalIndex, int32(i))
	}

	// If the statement is a single statement and is a PL/pgSQL block,
	// we should execute it as a single statement without transaction.
	// If the statement is a PL/pgSQL block, we should execute it as a single statement.
	// https://www.postgresql.org/docs/current/plpgsql-control-structures.html
	if len(singleSQLs) == 1 && isPlSQLBlock(singleSQLs[0]) {
		isPlsql = true
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

	conn, err := d.db.Conn(ctx)
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
		// USE SET SESSION ROLE to set the role for the current session.
		if err := crdb.Execute(func() error {
			_, err := conn.ExecContext(ctx, fmt.Sprintf("SET SESSION ROLE '%s'", owner))
			return err
		}); err != nil {
			return 0, errors.Wrapf(err, "failed to set role to database owner %q", owner)
		}
		opts.LogCommandExecute([]int32{0})
		if err := crdb.Execute(func() error {
			_, err := conn.ExecContext(ctx, statement)
			return err
		}); err != nil {
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

			// Set the current transaction role to the database owner so that the owner of created objects will be the same as the database owner.
			if _, err := tx.Exec(ctx, fmt.Sprintf("SET LOCAL ROLE '%s'", owner)); err != nil {
				return err
			}

			for i, command := range commands {
				indexes := []int32{originalIndex[i]}
				opts.LogCommandExecute(indexes)

				rr := tx.Conn().PgConn().Exec(ctx, command.Text)
				results, err := rr.ReadAll()
				if err != nil {
					opts.LogCommandResponse(indexes, 0, nil, err.Error())

					return &db.ErrorWithPosition{
						Err:   errors.Wrapf(err, "failed to execute context in a transaction"),
						Start: command.Start,
						End:   command.End,
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

	// USE SET SESSION ROLE to set the role for the current session.
	if err := crdb.Execute(func() error {
		_, err := conn.ExecContext(ctx, fmt.Sprintf("SET SESSION ROLE '%s'", owner))
		return err
	}); err != nil {
		return 0, errors.Wrapf(err, "failed to set role to database owner %q", owner)
	}
	// Run non-transaction statements at the end.
	for i, stmt := range nonTransactionAndSetRoleStmts {
		indexes := []int32{nonTransactionAndSetRoleStmtsIndex[i]}
		opts.LogCommandExecute(indexes)
		if err := crdb.Execute(func() error {
			_, err := conn.ExecContext(ctx, stmt)
			return err
		}); err != nil {
			opts.LogCommandResponse(indexes, 0, []int32{0}, err.Error())
			return 0, err
		}
		opts.LogCommandResponse(indexes, 0, []int32{0}, "")
	}
	return totalRowsAffected, nil
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
		if err := crdb.Execute(func() error {
			_, err := d.db.ExecContext(ctx, s)
			return err
		}); err != nil {
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
func (d *Driver) GetCurrentDatabaseOwner(ctx context.Context) (string, error) {
	const query = `
		SELECT
			u.rolname
		FROM
			pg_roles AS u JOIN pg_database AS d ON (d.datdba = u.oid)
		WHERE
			d.datname = current_database();
		`
	var owner string
	if err := d.db.QueryRowContext(ctx, query).Scan(&owner); err != nil {
		return "", err
	}
	return owner, nil
}

// QueryConn queries a SQL statement in a given connection.
func (*Driver) QueryConn(ctx context.Context, conn *sql.Conn, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
	singleSQLs, err := crdbparser.SplitSQLStatement(statement)
	if err != nil {
		return nil, err
	}
	if len(singleSQLs) == 0 {
		return nil, nil
	}

	var results []*v1pb.QueryResult
	for _, singleSQL := range singleSQLs {
		statement := singleSQL
		if queryContext.Explain {
			statement = fmt.Sprintf("EXPLAIN %s", statement)
		} else if queryContext.Limit > 0 {
			statement = getStatementWithResultLimit(statement, queryContext.Limit)
		}

		_, allQuery, err := base.ValidateSQLForEditor(storepb.Engine_POSTGRES, statement)
		if err != nil {
			return nil, err
		}

		// If the queryContext.Schema is not empty, set the search path for the database connection to the specified schema.
		if queryContext.Schema != "" {
			if err := crdb.Execute(func() error {
				_, err := conn.ExecContext(ctx, fmt.Sprintf("SET search_path TO %s;", queryContext.Schema))
				return err
			}); err != nil {
				return nil, err
			}
		}

		startTime := time.Now()
		queryResult, err := func() (*v1pb.QueryResult, error) {
			if allQuery {
				var r *v1pb.QueryResult
				if err := crdb.Execute(func() error {
					rows, err := conn.QueryContext(ctx, statement)
					if err != nil {
						return err
					}
					defer rows.Close()
					r, err = util.RowsToQueryResult(rows, makeValueByTypeName, convertValue, queryContext.MaximumSQLResultSize)
					if err != nil {
						return err
					}
					err = rows.Err()
					return err
				}); err != nil {
					return nil, err
				}
				return r, nil
			}

			var sqlResult sql.Result
			if err := crdb.Execute(func() error {
				var err error
				sqlResult, err = conn.ExecContext(ctx, statement)
				return err
			}); err != nil {
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
