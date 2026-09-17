// Package clickhouse is the plugin for ClickHouse driver.
package clickhouse

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"time"

	clickhouse "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/plugin/parser/standard"
)

var (
	systemDatabases = map[string]bool{
		"system":             true,
		"information_schema": true,
		"INFORMATION_SCHEMA": true,
	}
	systemDatabaseClause = func() string {
		var l []string
		for k := range systemDatabases {
			l = append(l, fmt.Sprintf("'%s'", k))
		}
		return strings.Join(l, ", ")
	}()

	_ db.Driver = (*Driver)(nil)
)

func init() {
	db.Register(storepb.Engine_CLICKHOUSE, newDriver)
}

// Driver is the ClickHouse driver.
type Driver struct {
	connectionCtx db.ConnectionContext
	dbType        storepb.Engine
	databaseName  string

	db *sql.DB
}

func newDriver() db.Driver {
	return &Driver{}
}

// Open opens a ClickHouse driver.
func (d *Driver) Open(_ context.Context, dbType storepb.Engine, config db.ConnectionConfig) (db.Driver, error) {
	addr := fmt.Sprintf("%s:%s", config.DataSource.Host, config.DataSource.Port)
	tlsConfig, err := util.GetTLSConfig(config.DataSource)
	if err != nil {
		return nil, errors.Wrap(err, "sql: tls config error")
	}
	// Default user name is "default".
	conn := clickhouse.OpenDB(&clickhouse.Options{
		Addr: []string{addr},
		Auth: clickhouse.Auth{
			Database: config.ConnectionContext.DatabaseName,
			Username: config.DataSource.GetUsername(),
			Password: config.Password,
		},
		TLS: tlsConfig,
		Settings: clickhouse.Settings{
			// Use a relative long value to avoid timeout on resource-intenstive query. Example failure:
			// failed: code: 160, message: Estimated query execution time (xxx seconds) is too long. Maximum: yyy. Estimated rows to process: zzzzzzzzz
			"max_execution_time": 300,
		},
		DialTimeout: 10 * time.Second,
	})

	slog.Debug("Opening ClickHouse driver",
		slog.String("addr", addr),
		slog.String("environment", config.ConnectionContext.EnvironmentID),
		slog.String("database", config.ConnectionContext.InstanceID),
	)

	d.dbType = dbType
	d.db = conn
	d.databaseName = config.ConnectionContext.DatabaseName
	d.connectionCtx = config.ConnectionContext
	return d, nil
}

// Close closes the driver.
func (d *Driver) Close(context.Context) error {
	return d.db.Close()
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
	return version, nil
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

	singleSQLs, err := standard.SplitSQL(statement)
	if err != nil {
		return 0, err
	}
	singleSQLs = base.FilterEmptyStatements(singleSQLs)
	if len(singleSQLs) == 0 {
		return 0, nil
	}

	// Execute based on transaction mode
	if transactionMode == common.TransactionModeOff {
		return d.executeInAutoCommitMode(ctx, singleSQLs, opts)
	}
	return d.executeInTransactionMode(ctx, singleSQLs, opts)
}

func (d *Driver) executeInTransactionMode(ctx context.Context, singleSQLs []base.Statement, opts db.ExecuteOptions) (int64, error) {
	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}

	committed := false
	defer func() {
		if !committed {
			err := tx.Rollback()
			var rerr string
			if err != nil && !errors.Is(err, sql.ErrTxDone) {
				rerr = err.Error()
				slog.Debug("failed to rollback transaction", log.BBError(err))
			}
			opts.LogTransactionControl(storepb.TaskRunLog_TransactionControl_ROLLBACK, rerr)
		}
	}()

	// Log transaction start
	opts.LogTransactionControl(storepb.TaskRunLog_TransactionControl_BEGIN, "")

	totalRowsAffected := int64(0)
	for _, singleSQL := range singleSQLs {
		opts.LogCommandExecute(singleSQL.Range, singleSQL.Text)
		sqlResult, err := tx.ExecContext(ctx, singleSQL.Text)
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

func (d *Driver) executeInAutoCommitMode(ctx context.Context, singleSQLs []base.Statement, opts db.ExecuteOptions) (int64, error) {
	totalRowsAffected := int64(0)
	for _, singleSQL := range singleSQLs {
		opts.LogCommandExecute(singleSQL.Range, singleSQL.Text)
		sqlResult, err := d.db.ExecContext(ctx, singleSQL.Text)
		if err != nil {
			opts.LogCommandResponse(0, nil, err.Error())
			return totalRowsAffected, err
		}
		rowsAffected, err := sqlResult.RowsAffected()
		if err != nil {
			// Since we cannot differentiate DDL and DML yet, we have to ignore the error.
			slog.Debug("rowsAffected returns error", log.BBError(err))
		} else {
			totalRowsAffected += rowsAffected
		}
		opts.LogCommandResponse(rowsAffected, nil, "")
	}
	return totalRowsAffected, nil
}
