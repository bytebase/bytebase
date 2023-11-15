// Package dm is the plugin for DM driver.
package dm

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	// Import go-dm DM driver.
	_ "gitee.com/chunanyong/dm"
	"github.com/pkg/errors"
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

func init() {
	db.Register(storepb.Engine_DM, newDriver)
}

// Driver is the DM driver.
type Driver struct {
	db           *sql.DB
	databaseName string
}

func newDriver(db.DriverConfig) db.Driver {
	return &Driver{}
}

// Open opens a DM driver.
func (driver *Driver) Open(_ context.Context, _ storepb.Engine, config db.ConnectionConfig, _ db.ConnectionContext) (db.Driver, error) {
	port, err := strconv.Atoi(config.Port)
	if err != nil {
		return nil, errors.Errorf("invalid port %q", config.Port)
	}
	dsn := fmt.Sprintf("dm://%s:%s@%s:%d", config.Username, config.Password, config.Host, port)
	if config.Database != "" {
		dsn = fmt.Sprintf("%s?schema=%s", dsn, config.Database)
	}
	db, err := sql.Open("dm", dsn)
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
func (*Driver) GetType() storepb.Engine {
	return storepb.Engine_DM
}

// GetDB gets the database.
func (driver *Driver) GetDB() *sql.DB {
	return driver.db
}

// Execute executes the migration, `beforeCommitTxFunc` will be called before transaction commit and after executing `statement`.
//
// Callers can use `beforeCommitTx` to do some extra work before transaction commit, like get the transaction id.
// Any error returned by `beforeCommitTx` will rollback the transaction, so it is the callers' responsibility to return nil if the error occurs in `beforeCommitTx` is not fatal.
func (driver *Driver) Execute(ctx context.Context, statement string, _ bool, opts db.ExecuteOptions) (int64, error) {
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

	totalRowsAffected := int64(0)
	f := func(stmt string) error {
		sqlResult, err := tx.ExecContext(ctx, stmt)
		if err != nil {
			return err
		}
		rowsAffected, err := sqlResult.RowsAffected()
		if err != nil {
			// Since we cannot differentiate DDL and DML yet, we have to ignore the error.
			slog.Debug("rowsAffected returns error", log.BBError(err))
		} else {
			totalRowsAffected += rowsAffected
		}
		return nil
	}

	// use oracle sql parser
	if _, err := plsqlparser.SplitMultiSQLStream(strings.NewReader(statement), f); err != nil {
		return 0, err
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

// QueryConn queries a SQL statement in a given connection.
func (driver *Driver) QueryConn(ctx context.Context, conn *sql.Conn, statement string, queryContext *db.QueryContext) ([]*v1pb.QueryResult, error) {
	singleSQLs, err := plsqlparser.SplitSQL(statement)
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

func getDMStatementWithResultLimit(stmt string, limit int) string {
	return fmt.Sprintf("SELECT * FROM (%s) WHERE ROWNUM <= %d", stmt, limit)
}

func (*Driver) querySingleSQL(ctx context.Context, conn *sql.Conn, singleSQL base.SingleSQL, queryContext *db.QueryContext) (*v1pb.QueryResult, error) {
	statement := strings.TrimRight(singleSQL.Text, " \n\t;")

	stmt := statement
	if !strings.HasPrefix(strings.ToUpper(stmt), "EXPLAIN") && queryContext.Limit > 0 {
		stmt = getDMStatementWithResultLimit(stmt, queryContext.Limit)
	}

	if queryContext.ReadOnly {
		// DM does not support transaction isolation level for read-only queries.(also like Oracle :)
		queryContext.ReadOnly = false
	}

	if queryContext.SensitiveSchemaInfo != nil {
		for _, database := range queryContext.SensitiveSchemaInfo.DatabaseList {
			if len(database.SchemaList) == 0 {
				continue
			}
			if len(database.SchemaList) > 1 {
				return nil, errors.Errorf("DM schema info should only have one schema per database, but got %d, %v", len(database.SchemaList), database.SchemaList)
			}
			if database.SchemaList[0].Name != database.Name {
				return nil, errors.Errorf("DM schema info should have the same database name and schema name, but got %s and %s", database.Name, database.SchemaList[0].Name)
			}
		}
	}

	startTime := time.Now()
	result, err := util.Query(ctx, storepb.Engine_DM, conn, stmt, queryContext)
	if err != nil {
		return nil, err
	}
	result.Latency = durationpb.New(time.Since(startTime))
	result.Statement = statement
	return result, nil
}

// RunStatement runs a SQL statement in a given connection.
// and like usual,use Oracle sql parser.
func (*Driver) RunStatement(ctx context.Context, conn *sql.Conn, statement string) ([]*v1pb.QueryResult, error) {
	return util.RunStatement(ctx, storepb.Engine_ORACLE, conn, statement)
}
