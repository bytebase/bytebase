// Package oracle is the plugin for Oracle driver.
package oracle

import (
	"context"
	"crypto/rand"
	"database/sql"
	"fmt"
	"log/slog"
	"math/big"
	"regexp"
	"strconv"
	"strings"
	"time"

	// Import go-ora Oracle driver.

	"github.com/pkg/errors"
	goora "github.com/sijms/go-ora/v2"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	plsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/plsql"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

var (
	_ db.Driver = (*Driver)(nil)
)

const dbVersion12 = 12

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

func newDriver(db.DriverConfig) db.Driver {
	return &Driver{}
}

// Open opens a Oracle driver.
func (driver *Driver) Open(ctx context.Context, _ storepb.Engine, config db.ConnectionConfig) (db.Driver, error) {
	port, err := strconv.Atoi(config.Port)
	if err != nil {
		return nil, errors.Errorf("invalid port %q", config.Port)
	}
	options := make(map[string]string)
	options["CONNECTION TIMEOUT"] = "0"
	if config.SID != "" {
		options["SID"] = config.SID
	}
	dsn := goora.BuildUrl(config.Host, port, config.ServiceName, config.Username, config.Password, options)
	db, err := sql.Open("oracle", dsn)
	if err != nil {
		return nil, err
	}
	if config.Database != "" {
		if _, err := db.ExecContext(ctx, fmt.Sprintf("ALTER SESSION SET CURRENT_SCHEMA = \"%s\"", config.Database)); err != nil {
			return nil, errors.Wrapf(err, "failed to set current schema to %q", config.Database)
		}
	}
	driver.db = db
	driver.databaseName = config.Database
	driver.serviceName = config.ServiceName
	driver.connectionCtx = config.ConnectionContext
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
func (*Driver) GetType() storepb.Engine {
	return storepb.Engine_ORACLE
}

// GetDB gets the database.
func (driver *Driver) GetDB() *sql.DB {
	return driver.db
}

// Execute executes the migration, `beforeCommitTxFunc` will be called before transaction commit and after executing `statement`.
//
// Callers can use `beforeCommitTx` to do some extra work before transaction commit, like get the transaction id.
// Any error returned by `beforeCommitTx` will rollback the transaction, so it is the callers' responsibility to return nil if the error occurs in `beforeCommitTx` is not fatal.
func (driver *Driver) Execute(ctx context.Context, statement string, opts db.ExecuteOptions) (int64, error) {
	if opts.CreateDatabase {
		return 0, errors.New("create database is not supported for Oracle")
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

		indexes := []int32{int32(i)}
		opts.LogCommandExecute(indexes)

		sqlResult, err := tx.ExecContext(ctx, singleSQL.Text)
		if err != nil {
			opts.LogCommandResponse(indexes, 0, nil, err.Error())
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
			rowsAffected = 0
		}
		opts.LogCommandResponse(indexes, int32(rowsAffected), []int32{int32(rowsAffected)}, "")
		totalRowsAffected += rowsAffected
	}

	if err := tx.Commit(); err != nil {
		return 0, errors.Wrapf(err, "failed to commit transaction")
	}
	return totalRowsAffected, nil
}

// QueryConn queries a SQL statement in a given connection.
func (driver *Driver) QueryConn(ctx context.Context, conn *sql.Conn, statement string, queryContext *db.QueryContext) ([]*v1pb.QueryResult, error) {
	// Oracle does not support transaction isolation level for read-only queries.
	if queryContext != nil {
		queryContext.ReadOnly = false
	}

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

func (driver *Driver) getOracleStatementWithResultLimit(stmt string, queryContext *db.QueryContext) (string, error) {
	engineVersion := driver.connectionCtx.EngineVersion
	versionIdx := strings.Index(engineVersion, ".")
	if versionIdx < 0 {
		return "", errors.New("instance version number is invalid")
	}
	versionNumber, err := strconv.Atoi(engineVersion[:versionIdx])
	if err != nil {
		return "", err
	}
	if queryContext != nil {
		switch {
		case versionNumber < dbVersion12:
			return getStatementWithResultLimitFor11g(stmt, queryContext.Limit), nil
		default:
			res, err := getStatementWithResultLimitFor12c(stmt, queryContext.Limit)
			if err != nil {
				return "", err
			}
			return res, nil
		}
	}

	return stmt, nil
}

func (driver *Driver) querySingleSQL(ctx context.Context, conn *sql.Conn, singleSQL base.SingleSQL, queryContext *db.QueryContext) (*v1pb.QueryResult, error) {
	statement := strings.TrimRight(singleSQL.Text, " \n\t;")

	if queryContext != nil && queryContext.Explain {
		startTime := time.Now()
		randNum, err := rand.Int(rand.Reader, big.NewInt(999))
		if err != nil {
			return nil, errors.Wrapf(err, "failed to generate random statement ID")
		}
		randomID := fmt.Sprintf("%d%d", startTime.UnixMilli(), randNum.Int64())

		statement = fmt.Sprintf("EXPLAIN PLAN SET STATEMENT_ID = '%s' FOR %s", randomID, statement)
		if _, err := conn.ExecContext(ctx, statement); err != nil {
			return nil, err
		}
		explainQuery := fmt.Sprintf(`SELECT LPAD(' ', LEVEL-1) || OPERATION || ' (' || OPTIONS || ')' "Operation", OBJECT_NAME "Object", OPTIMIZER "Optimizer", COST "Cost", CARDINALITY "Cardinality", BYTES "Bytes", PARTITION_START "Partition Start", PARTITION_ID "Partition ID", ACCESS_PREDICATES "Access Predicates",FILTER_PREDICATES "Filter Predicates" FROM PLAN_TABLE START WITH ID = 0 AND statement_id = '%s' CONNECT BY PRIOR ID=PARENT_ID AND statement_id = '%s' ORDER BY id`, randomID, randomID)
		result, err := util.Query(ctx, storepb.Engine_ORACLE, conn, explainQuery, queryContext)
		if err != nil {
			return nil, err
		}
		result.Latency = durationpb.New(time.Since(startTime))
		result.Statement = statement
		return result, nil
	}

	if queryContext != nil && queryContext.Limit > 0 {
		stmt, err := driver.getOracleStatementWithResultLimit(statement, queryContext)
		if err != nil {
			slog.Error("fail to add limit clause", "statement", statement, log.BBError(err))
			stmt = getStatementWithResultLimitFor11g(stmt, queryContext.Limit)
		}
		statement = stmt
	}

	startTime := time.Now()
	result, err := util.Query(ctx, storepb.Engine_ORACLE, conn, statement, queryContext)
	if err != nil {
		return nil, err
	}
	result.Latency = durationpb.New(time.Since(startTime))
	result.Statement = statement
	return result, nil
}

// RunStatement runs a SQL statement in a given connection.
func (*Driver) RunStatement(ctx context.Context, conn *sql.Conn, statement string) ([]*v1pb.QueryResult, error) {
	return util.RunStatement(ctx, storepb.Engine_ORACLE, conn, statement)
}

type oracleVersion struct {
	first  int
	second int
}

func parseVersion(banner string) (*oracleVersion, error) {
	re := regexp.MustCompile(`(\d+)\.(\d+)`)
	match := re.FindStringSubmatch(banner)
	if len(match) >= 3 {
		firstVersion, err := strconv.Atoi(match[1])
		if err != nil {
			return nil, errors.Errorf("failed to parse first version from banner: %s", banner)
		}
		secondVersion, err := strconv.Atoi(match[2])
		if err != nil {
			return nil, errors.Errorf("failed to parse second version from banner: %s", banner)
		}
		return &oracleVersion{first: firstVersion, second: secondVersion}, nil
	}
	return nil, errors.Errorf("failed to parse version from banner: %s", banner)
}
