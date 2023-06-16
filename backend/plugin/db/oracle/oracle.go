// Package oracle is the plugin for Oracle driver.
package oracle

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	// Import go-ora Oracle driver.
	"github.com/pkg/errors"
	go_ora "github.com/sijms/go-ora/v2"
	"go.uber.org/zap"

	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	parser "github.com/bytebase/bytebase/backend/plugin/parser/sql"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

var (
	_ db.Driver = (*Driver)(nil)
)

func init() {
	db.Register(db.Oracle, newDriver)
}

// Driver is the Oracle driver.
type Driver struct {
	db           *sql.DB
	databaseName string
}

func newDriver(db.DriverConfig) db.Driver {
	return &Driver{}
}

// Open opens a Snowflake driver.
func (driver *Driver) Open(_ context.Context, _ db.Type, config db.ConnectionConfig, _ db.ConnectionContext) (db.Driver, error) {
	port, err := strconv.Atoi(config.Port)
	if err != nil {
		return nil, errors.Errorf("invalid port %q", config.Port)
	}
	options := make(map[string]string)
	if config.SID != "" {
		options["SID"] = config.SID
	}
	dsn := go_ora.BuildUrl(config.Host, port, config.ServiceName, config.Username, config.Password, options)
	db, err := sql.Open("oracle", dsn)
	if err != nil {
		return nil, err
	}
	driver.db = db
	driver.databaseName = config.Database
	return driver, nil
}

// Close closes the driver.
func (driver *Driver) Close(_ context.Context) error {
	return driver.db.Close()
}

// Ping pings the database.
func (driver *Driver) Ping(ctx context.Context) error {
	return driver.db.PingContext(ctx)
}

// GetType returns the database type.
func (*Driver) GetType() db.Type {
	return db.Oracle
}

// GetDB gets the database.
func (driver *Driver) GetDB() *sql.DB {
	return driver.db
}

// Execute executes a SQL statement and returns the affected rows.
func (*Driver) Execute(_ context.Context, _ *sql.Conn, _ string, _ bool) (int64, error) {
	return 0, errors.Errorf("Oracle driver Execute() is not supported")
}

// ExecuteWithBeforeCommit executes the migration, `beforeCommitTxFunc` will be called before transaction commit and after executing `statement`.
//
// Callers can use `beforeCommitTx` to do some extra work before transaction commit, like get the transaction id.
// Any error returned by `beforeCommitTx` will rollback the transaction, so it is the callers' responsibility to return nil if the error occurs in `beforeCommitTx` is not fatal.
func (driver *Driver) ExecuteWithBeforeCommit(ctx context.Context, statement string, beforeCommitTxFunc func(tx *sql.Tx) error) (migrationHistoryID string, updatedSchema string, resErr error) {
	conn, err := driver.db.Conn(ctx)
	if err != nil {
		return "", "", errors.Wrapf(err, "failed to get connection")
	}
	defer conn.Close()
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return "", "", errors.Wrapf(err, "failed to begin transaction")
	}
	defer tx.Rollback()

	totalRowsAffected := int64(0)
	f := func(stmt string) error {
		// The underlying oracle golang driver go-ora does not support semicolon, so we should trim the suffix semicolon.
		stmt = strings.TrimSuffix(stmt, ";")
		sqlResult, err := tx.ExecContext(ctx, stmt)
		if err != nil {
			return err
		}
		rowsAffected, err := sqlResult.RowsAffected()
		if err != nil {
			// Since we cannot differentiate DDL and DML yet, we have to ignore the error.
			log.Debug("rowsAffected returns error", zap.Error(err))
		} else {
			totalRowsAffected += rowsAffected
		}
		return nil
	}

	if _, err := parser.SplitMultiSQLStream(parser.Oracle, strings.NewReader(statement), f); err != nil {
		return "", "", err
	}

	if beforeCommitTxFunc != nil {
		if err := beforeCommitTxFunc(tx); err != nil {
			return "", "", errors.Wrapf(err, "failed to execute beforeCommitTx")
		}
	}

	if err := tx.Commit(); err != nil {
		return "", "", errors.Wrapf(err, "failed to commit transaction")
	}
	return "", "", nil
}

// QueryConn querys a SQL statement in a given connection.
func (*Driver) QueryConn(ctx context.Context, conn *sql.Conn, statement string, queryContext *db.QueryContext) ([]any, error) {
	return util.Query(ctx, db.Oracle, conn, statement, queryContext)
}

// QueryConn2 queries a SQL statement in a given connection.
func (driver *Driver) QueryConn2(ctx context.Context, conn *sql.Conn, statement string, queryContext *db.QueryContext) ([]*v1pb.QueryResult, error) {
	singleSQLs, err := parser.SplitMultiSQL(parser.Oracle, statement)
	if err != nil {
		return nil, err
	}
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

func getOracleStatementWithResultLimit(stmt string, limit int) string {
	return fmt.Sprintf("SELECT * FROM (%s) WHERE ROWNUM <= %d", stmt, limit)
}

func (*Driver) querySingleSQL(ctx context.Context, conn *sql.Conn, singleSQL parser.SingleSQL, queryContext *db.QueryContext) (*v1pb.QueryResult, error) {
	statement := singleSQL.Text
	statement = strings.TrimRight(statement, " \n\t;")
	if !strings.HasPrefix(strings.ToUpper(statement), "EXPLAIN") && queryContext.Limit > 0 {
		statement = getOracleStatementWithResultLimit(statement, queryContext.Limit)
	}

	if queryContext.ReadOnly {
		// Oracle does not support transaction isolation level for read-only queries.
		queryContext.ReadOnly = false
	}

	return util.Query2(ctx, db.Oracle, conn, statement, queryContext)
}

// RunStatement runs a SQL statement in a given connection.
func (*Driver) RunStatement(ctx context.Context, conn *sql.Conn, statement string) ([]*v1pb.QueryResult, error) {
	return util.RunStatement(ctx, parser.Oracle, conn, statement)
}
