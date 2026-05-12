// Package oracle is the plugin for Oracle driver.
package oracle

import (
	"context"
	"crypto/rand"
	"database/sql"
	"fmt"
	"log/slog"
	"math/big"
	"strconv"
	"time"

	// Import go-ora Oracle driver.

	"github.com/pkg/errors"
	goora "github.com/sijms/go-ora/v2"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	plsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/plsql"
)

var (
	_ db.Driver = (*Driver)(nil)
)

func init() {
	db.Register(storepb.Engine_ORACLE, newDriver)
}

// Driver is the Oracle driver.
type Driver struct {
	db            *sql.DB
	databaseName  string
	serviceName   string
	connectionCtx db.ConnectionContext
}

func newDriver() db.Driver {
	return &Driver{}
}

// GetVersion gets the Oracle version.
func (d *Driver) GetVersion() (*plsqlparser.Version, error) {
	return plsqlparser.ParseVersion(d.connectionCtx.EngineVersion)
}

// Open opens a Oracle driver.
func (d *Driver) Open(ctx context.Context, _ storepb.Engine, config db.ConnectionConfig) (db.Driver, error) {
	port, err := strconv.Atoi(config.DataSource.Port)
	if err != nil {
		return nil, errors.Errorf("invalid port %q", config.DataSource.Port)
	}
	options := make(map[string]string)
	options["CONNECTION TIMEOUT"] = "0"
	if config.DataSource.GetSid() != "" {
		options["SID"] = config.DataSource.GetSid()
	}
	for key, value := range config.DataSource.GetExtraConnectionParameters() {
		options[key] = value
	}
	dsn := goora.BuildUrl(config.DataSource.Host, port, config.DataSource.GetServiceName(), config.DataSource.Username, config.Password, options)
	db, err := sql.Open("oracle", dsn)
	if err != nil {
		return nil, err
	}
	if config.ConnectionContext.DatabaseName != "" {
		if _, err := db.ExecContext(ctx, fmt.Sprintf("ALTER SESSION SET CURRENT_SCHEMA = \"%s\"", config.ConnectionContext.DatabaseName)); err != nil {
			return nil, errors.Wrapf(err, "failed to set current schema to %q", config.ConnectionContext.DatabaseName)
		}
	}
	d.db = db
	d.databaseName = config.ConnectionContext.DatabaseName
	d.serviceName = config.DataSource.GetServiceName()
	d.connectionCtx = config.ConnectionContext
	return d, nil
}

// Close closes the driver.
func (d *Driver) Close(_ context.Context) error {
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

// Execute executes the migration, `beforeCommitTxFunc` will be called before transaction commit and after executing `statement`.
//
// Callers can use `beforeCommitTx` to do some extra work before transaction commit, like get the transaction id.
// Any error returned by `beforeCommitTx` will rollback the transaction, so it is the callers' responsibility to return nil if the error occurs in `beforeCommitTx` is not fatal.
func (d *Driver) Execute(ctx context.Context, statement string, opts db.ExecuteOptions) (int64, error) {
	if opts.CreateDatabase {
		return 0, errors.New("create database is not supported for Oracle")
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
	if len(statement) <= common.MaxSheetCheckSize {
		// Use Oracle sql parser.
		singleSQLs, err := plsqlparser.SplitSQL(statement)
		if err != nil {
			return 0, errors.Wrapf(err, "failed to split sql")
		}
		singleSQLs = base.FilterEmptyStatements(singleSQLs)
		if len(singleSQLs) == 0 {
			return 0, nil
		}
		commands = singleSQLs
	} else {
		commands = []base.Statement{
			{
				Text: statement,
			},
		}
	}

	conn, err := d.db.Conn(ctx)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to get connection")
	}
	defer conn.Close()

	// Execute based on transaction mode
	if transactionMode == common.TransactionModeOff {
		return d.executeInAutoCommitMode(ctx, conn, commands, opts)
	}
	return d.executeInTransactionMode(ctx, conn, commands, opts)
}

// executeInTransactionMode executes statements within a single transaction
func (*Driver) executeInTransactionMode(ctx context.Context, conn *sql.Conn, commands []base.Statement, opts db.ExecuteOptions) (int64, error) {
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		opts.LogTransactionControl(storepb.TaskRunLog_TransactionControl_BEGIN, err.Error())
		return 0, errors.Wrapf(err, "failed to begin transaction")
	}
	opts.LogTransactionControl(storepb.TaskRunLog_TransactionControl_BEGIN, "")

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

	totalRowsAffected := int64(0)
	for _, command := range commands {
		opts.LogCommandExecute(command.Range, command.Text)

		sqlResult, err := tx.ExecContext(ctx, command.Text)
		if err != nil {
			opts.LogCommandResponse(0, nil, err.Error())
			return 0, err
		}
		rowsAffected, err := sqlResult.RowsAffected()
		if err != nil {
			// Since we cannot differentiate DDL and DML yet, we have to ignore the error.
			slog.Debug("rowsAffected returns error", log.BBError(err))
			rowsAffected = 0
		}
		opts.LogCommandResponse(rowsAffected, []int64{rowsAffected}, "")
		totalRowsAffected += rowsAffected
	}

	if err := tx.Commit(); err != nil {
		opts.LogTransactionControl(storepb.TaskRunLog_TransactionControl_COMMIT, err.Error())
		return 0, errors.Wrapf(err, "failed to commit transaction")
	}
	opts.LogTransactionControl(storepb.TaskRunLog_TransactionControl_COMMIT, "")
	committed = true
	return totalRowsAffected, nil
}

// executeInAutoCommitMode executes statements sequentially in auto-commit mode
func (*Driver) executeInAutoCommitMode(ctx context.Context, conn *sql.Conn, commands []base.Statement, opts db.ExecuteOptions) (int64, error) {
	totalRowsAffected := int64(0)
	for _, command := range commands {
		opts.LogCommandExecute(command.Range, command.Text)

		sqlResult, err := conn.ExecContext(ctx, command.Text)
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
			rowsAffected = 0
		}
		opts.LogCommandResponse(rowsAffected, []int64{rowsAffected}, "")
		totalRowsAffected += rowsAffected
	}
	return totalRowsAffected, nil
}

// QueryConn queries a SQL statement in a given connection.
func (d *Driver) QueryConn(ctx context.Context, conn *sql.Conn, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
	singleSQLs, err := plsqlparser.SplitSQL(statement)
	if err != nil {
		return nil, err
	}
	singleSQLs = base.FilterEmptyStatements(singleSQLs)
	if len(singleSQLs) == 0 {
		return nil, nil
	}

	var results []*v1pb.QueryResult
	for _, singleSQL := range singleSQLs {
		statement := singleSQL.Text
		if queryContext.Explain {
			startTime := time.Now()
			randNum, err := rand.Int(rand.Reader, big.NewInt(999))
			if err != nil {
				return nil, errors.Wrapf(err, "failed to generate random statement ID")
			}
			randomID := fmt.Sprintf("%d%d", startTime.UnixMilli(), randNum.Int64())

			if _, err := conn.ExecContext(ctx, fmt.Sprintf("EXPLAIN PLAN SET STATEMENT_ID = '%s' FOR %s", randomID, statement)); err != nil {
				return nil, err
			}
			statement = fmt.Sprintf(`SELECT LPAD(' ', LEVEL-1) || OPERATION || ' (' || OPTIONS || ')' "Operation", OBJECT_NAME "Object", OPTIMIZER "Optimizer", COST "Cost", CARDINALITY "Cardinality", BYTES "Bytes", PARTITION_START "Partition Start", PARTITION_ID "Partition ID", ACCESS_PREDICATES "Access Predicates" FROM PLAN_TABLE START WITH ID = 0 AND statement_id = '%s' CONNECT BY PRIOR ID=PARENT_ID AND statement_id = '%s' ORDER BY id`, randomID, randomID)
		}

		if !queryContext.Explain && queryContext.Limit > 0 {
			statement = addResultLimit(statement, queryContext.Limit, d.connectionCtx.EngineVersion)
		}

		_, allQuery, err := base.ValidateSQLForEditor(storepb.Engine_ORACLE, statement)
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
