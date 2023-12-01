// Package mssql is the plugin for MSSQL driver.
package mssql

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"strings"
	"time"

	// Import go-ora Oracle driver.
	_ "github.com/microsoft/go-mssqldb"
	_ "github.com/microsoft/go-mssqldb/integratedauth/krb5"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	tsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/tsql"
	tsqlbatch "github.com/bytebase/bytebase/backend/plugin/parser/tsql/batch"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

var (
	_ db.Driver = (*Driver)(nil)
)

func init() {
	db.Register(storepb.Engine_MSSQL, newDriver)
}

// Driver is the MSSQL driver.
type Driver struct {
	db           *sql.DB
	databaseName string
}

func newDriver(db.DriverConfig) db.Driver {
	return &Driver{}
}

// Open opens a MSSQL driver.
func (driver *Driver) Open(_ context.Context, _ storepb.Engine, config db.ConnectionConfig, _ db.ConnectionContext) (db.Driver, error) {
	query := url.Values{}
	query.Add("app name", "Bytebase")
	if config.Database != "" {
		query.Add("database", config.Database)
	}
	u := &url.URL{
		Scheme:   "sqlserver",
		User:     url.UserPassword(config.Username, config.Password),
		Host:     fmt.Sprintf("%s:%s", config.Host, config.Port),
		RawQuery: query.Encode(),
	}
	db, err := sql.Open("sqlserver", u.String())
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
	return storepb.Engine_MSSQL
}

// GetDB gets the database.
func (driver *Driver) GetDB() *sql.DB {
	return driver.db
}

// Execute executes a SQL statement and returns the affected rows.
func (driver *Driver) Execute(ctx context.Context, statement string, createDatabase bool, _ db.ExecuteOptions) (int64, error) {
	if createDatabase {
		if _, err := driver.db.ExecContext(ctx, statement); err != nil {
			return 0, err
		}
		return 0, nil
	}
	tx, err := driver.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	totalAffectRows := int64(0)

	// Split to batches to support some client commands like GO.
	s := strings.Split(statement, "\n")
	scanner := func() (string, error) {
		if len(s) > 0 {
			z := s[0]
			s = s[1:]
			return z, nil
		}
		return "", io.EOF
	}
	batch := tsqlbatch.NewBatch(scanner)
	execute := func(ctx context.Context, tx *sql.Tx, s string) (int64, error) {
		sqlResult, err := tx.ExecContext(ctx, s)
		if err != nil {
			return 0, errors.Wrapf(err, "failed to execute statement: %s", s)
		}
		rowsAffected, err := sqlResult.RowsAffected()
		if err != nil {
			// Since we cannot differentiate DDL and DML yet, we have to ignore the error.
			slog.Debug("rowsAffected returns error", log.BBError(err))
			return 0, nil
		}
		return rowsAffected, nil
	}

	for {
		command, err := batch.Next()
		if err != nil {
			if err == io.EOF {
				// Try send the last batch to server.
				v := batch.String()
				if v != "" {
					rowsAffected, err := execute(ctx, tx, v)
					if err != nil {
						return 0, errors.Wrapf(err, "failed to execute statement: %s", v)
					}
					totalAffectRows += rowsAffected
				}
				break
			}
			return 0, errors.Wrapf(err, "failed to execute batch statement: %s", batch.String())
		}
		if command != nil {
			switch v := command.(type) {
			case *tsqlbatch.GoCommand:
				// Try send the batch to server.
				for i := uint(0); i < v.Count; i++ {
					stmt := batch.String()
					rowsAffected, err := execute(ctx, tx, stmt)
					if err != nil {
						return 0, err
					}
					totalAffectRows += rowsAffected
				}
			default:
				return 0, errors.Errorf("unsupported command type: %T", v)
			}
			batch.Reset(nil)
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return totalAffectRows, nil
}

// QueryConn queries a SQL statement in a given connection.
func (driver *Driver) QueryConn(ctx context.Context, conn *sql.Conn, statement string, queryContext *db.QueryContext) ([]*v1pb.QueryResult, error) {
	singleSQLs, err := tsqlparser.SplitSQL(statement)
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

func (*Driver) querySingleSQL(ctx context.Context, conn *sql.Conn, singleSQL base.SingleSQL, queryContext *db.QueryContext) (*v1pb.QueryResult, error) {
	statement := strings.TrimRight(singleSQL.Text, " \n\t;")

	stmt := statement
	if !strings.HasPrefix(stmt, "EXPLAIN") && queryContext.Limit > 0 {
		var err error
		stmt, err = getMSSQLStatementWithResultLimit(stmt, queryContext.Limit)
		if err != nil {
			slog.Error("fail to add limit clause", "statement", statement, log.BBError(err))
			stmt = fmt.Sprintf("WITH result AS (%s) SELECT TOP %d * FROM result;", stmt, queryContext.Limit)
		}
	}

	if queryContext.ReadOnly {
		// MSSQL does not support transaction isolation level for read-only queries.
		queryContext.ReadOnly = false
	}
	startTime := time.Now()
	result, err := util.Query(ctx, storepb.Engine_MSSQL, conn, stmt, queryContext)
	if err != nil {
		return nil, err
	}
	result.Latency = durationpb.New(time.Since(startTime))
	result.Statement = statement
	return result, nil
}

// RunStatement runs a SQL statement.
func (*Driver) RunStatement(ctx context.Context, conn *sql.Conn, statement string) ([]*v1pb.QueryResult, error) {
	return util.RunStatement(ctx, storepb.Engine_MSSQL, conn, statement)
}
