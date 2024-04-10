// Package obo is for OceanBase Oracle mode
package obo

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/exp/slog"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	plsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/plsql"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"

	// Register OceanBase Oracle mode driver.
	_ "github.com/mattn/go-oci8"
)

func init() {
	db.Register(storepb.Engine_OCEANBASE_ORACLE, newDriver)
}

type Driver struct {
	db           *sql.DB
	databaseName string
}

func newDriver(db.DriverConfig) db.Driver {
	return &Driver{}
}

func (driver *Driver) Open(_ context.Context, _ storepb.Engine, config db.ConnectionConfig) (db.Driver, error) {
	databaseName := func() string {
		if config.Database != "" {
			return config.Database
		}
		i := strings.Index(config.Username, "@")
		if i == -1 {
			return config.Username
		}
		return config.Username[:i]
	}()

	// Usename format: {user}@{tenant}#{cluster}
	// User is required, others are optional.
	dsn := fmt.Sprintf("%s/%s@%s:%s/%s", url.PathEscape(config.Username), url.PathEscape(config.Password), config.Host, config.Port, url.PathEscape(databaseName))

	db, err := sql.Open("oci8", dsn)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to open connection")
	}

	driver.db = db
	driver.databaseName = databaseName
	return driver, nil
}

func (driver *Driver) Close(context.Context) error {
	return driver.db.Close()
}

func (driver *Driver) Ping(ctx context.Context) error {
	return driver.db.PingContext(ctx)
}

func (*Driver) GetType() storepb.Engine {
	return storepb.Engine_OCEANBASE_ORACLE
}

func (driver *Driver) GetDB() *sql.DB {
	return driver.db
}

func (driver *Driver) Execute(ctx context.Context, statement string, opts db.ExecuteOptions) (int64, error) {
	if opts.CreateDatabase {
		return 0, errors.New("create database is not supported for OceanBase Oracle mode")
	}

	// Use Oracle sql parser.
	singleSQLs, err := plsqlparser.SplitSQL(statement)
	if err != nil {
		return 0, err
	}
	singleSQLs = base.FilterEmptySQL(singleSQLs)
	if len(singleSQLs) == 0 {
		return 0, nil
	}

	conn, err := driver.db.Conn(ctx)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to get connection")
	}
	defer conn.Close()
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to begin transaction")
	}
	defer tx.Rollback()

	totalCommands := len(singleSQLs)
	totalRowsAffected := int64(0)
	for i, singleSQL := range singleSQLs {
		// Start the current chunk.
		// Set the progress information for the current chunk.
		if opts.UpdateExecutionStatus != nil {
			opts.UpdateExecutionStatus(&v1pb.TaskRun_ExecutionDetail{
				CommandsTotal:     int32(totalCommands),
				CommandsCompleted: int32(i),
				CommandStartPosition: &v1pb.TaskRun_ExecutionDetail_Position{
					Line:   int32(singleSQL.FirstStatementLine),
					Column: int32(singleSQL.FirstStatementColumn),
				},
				CommandEndPosition: &v1pb.TaskRun_ExecutionDetail_Position{
					Line:   int32(singleSQL.LastLine),
					Column: int32(singleSQL.LastColumn),
				},
			})
		}

		sqlResult, err := tx.ExecContext(ctx, singleSQL.Text)
		if err != nil {
			return 0, &db.ErrorWithPosition{
				Err: errors.Wrapf(err, "failed to execute context in a transaction"),
				Start: &storepb.TaskRunResult_Position{
					Line:   int32(singleSQL.FirstStatementLine),
					Column: int32(singleSQL.FirstStatementColumn),
				},
				End: &storepb.TaskRunResult_Position{
					Line:   int32(singleSQL.LastLine),
					Column: int32(singleSQL.LastColumn),
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

	if opts.EndTransactionFunc != nil {
		if err := opts.EndTransactionFunc(tx); err != nil {
			return 0, errors.Wrapf(err, "failed to execute beforeCommitTx")
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, errors.Wrapf(err, "failed to commit transaction")
	}
	return totalRowsAffected, nil
}

func (driver *Driver) QueryConn(ctx context.Context, conn *sql.Conn, statement string, queryContext *db.QueryContext) ([]*v1pb.QueryResult, error) {
	// Oracle does not support transaction isolation level for read-only queries.
	queryContext.ReadOnly = false

	singleSQLs, err := plsqlparser.SplitSQL(statement)
	if err != nil {
		return nil, err
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

func (*Driver) querySingleSQL(ctx context.Context, conn *sql.Conn, singleSQL base.SingleSQL, queryContext *db.QueryContext) (*v1pb.QueryResult, error) {
	statement := strings.TrimRight(singleSQL.Text, " \n\t;")
	if !strings.HasPrefix(strings.ToUpper(statement), "EXPLAIN") && queryContext.Limit > 0 {
		statement = fmt.Sprintf("SELECT * FROM (%s) WHERE ROWNUM <= %d", statement, queryContext.Limit)
	}

	if queryContext.SensitiveSchemaInfo != nil {
		for _, database := range queryContext.SensitiveSchemaInfo.DatabaseList {
			if len(database.SchemaList) == 0 {
				continue
			}
			if len(database.SchemaList) > 1 {
				return nil, errors.Errorf("Oracle schema info should only have one schema per database, but got %d, %v", len(database.SchemaList), database.SchemaList)
			}
			if database.SchemaList[0].Name != database.Name {
				return nil, errors.Errorf("Oracle schema info should have the same database name and schema name, but got %s and %s", database.Name, database.SchemaList[0].Name)
			}
		}
	}

	startTime := time.Now()
	result, err := util.Query(ctx, storepb.Engine_OCEANBASE_ORACLE, conn, statement, queryContext)
	if err != nil {
		return nil, err
	}
	result.Latency = durationpb.New(time.Since(startTime))
	result.Statement = statement
	return result, nil
}

func (*Driver) RunStatement(ctx context.Context, conn *sql.Conn, statement string) ([]*v1pb.QueryResult, error) {
	return util.RunStatement(ctx, storepb.Engine_OCEANBASE_ORACLE, conn, statement)
}
