// Package risingwave is the plugin for RisingWave driver.
package risingwave

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net"
	"regexp"
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
	db.Register(storepb.Engine_RISINGWAVE, newDriver)
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
}

func newDriver() db.Driver {
	return &Driver{}
}

// Open opens a RisingWave driver.
func (d *Driver) Open(_ context.Context, _ storepb.Engine, config db.ConnectionConfig) (db.Driver, error) {
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
	pgxConnConfig, err := pgx.ParseConfig(connStr)
	if err != nil {
		return nil, err
	}
	pgxConnConfig.User = config.DataSource.Username
	pgxConnConfig.Password = config.Password
	pgxConnConfig.Database = config.ConnectionContext.DatabaseName
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
	if config.ConnectionContext.ReadOnly {
		pgxConnConfig.RuntimeParams["default_transaction_read_only"] = "true"
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
	return d, nil
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
	rows, err := d.db.QueryContext(ctx, "SELECT datname, pg_encoding_to_char(encoding), datcollate FROM pg_database;")
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
func (d *Driver) getVersion(ctx context.Context) (string, error) {
	// Likes PostgreSQL 9.5-RisingWave-1.1.0 (f41ff20612323dc56f654939cfa3be9ca684b52f)
	// We will return 1.1.0
	regexp := regexp.MustCompile(`(?m)PostgreSQL (?P<PG_VERSION>.*)-RisingWave-(?P<RISINGWAVE_VERSION>.*) \((?P<BUILD_SHA>.*)\)$`)
	query := "SELECT version();"
	var version string
	if err := d.db.QueryRowContext(ctx, query).Scan(&version); err != nil {
		if err == sql.ErrNoRows {
			return "", common.FormatDBErrorEmptyRowWithQuery(query)
		}
		return "", util.FormatErrorWithQuery(err, query)
	}
	matches := regexp.FindStringSubmatch(version)
	if len(matches) != 4 {
		return "", errors.Errorf("cannot parse version %q", version)
	}
	return matches[2], nil
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
		tx, err := d.db.BeginTx(ctx, nil)
		if err != nil {
			return 0, err
		}
		defer tx.Rollback()

		for _, command := range commands {
			sqlResult, err := tx.ExecContext(ctx, command.Text)
			if err != nil {
				return 0, &db.ErrorWithPosition{
					Err:   errors.Wrapf(err, "failed to execute context in a transaction"),
					Start: command.Start,
					End:   command.End,
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

	// If the queryContext.Schema is not empty, set the search path for the database connection to the specified schema.
	// Reference: https://docs.risingwave.com/docs/current/view-configure-runtime-parameters
	if queryContext.Schema != "" {
		if _, err := conn.ExecContext(ctx, fmt.Sprintf("SET search_path TO %s;", queryContext.Schema)); err != nil {
			return nil, err
		}
	}

	var results []*v1pb.QueryResult
	for _, singleSQL := range singleSQLs {
		statement := singleSQL.Text
		if queryContext.Explain {
			statement = fmt.Sprintf("EXPLAIN %s", statement)
		} else if queryContext.Limit > 0 {
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
	return fmt.Sprintf("WITH result AS (%s) SELECT * FROM result LIMIT %d;", stmt, limit)
}
