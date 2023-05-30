// Package mssql is the plugin for MSSQL driver.
package mssql

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"strings"

	// Import go-ora Oracle driver.
	_ "github.com/microsoft/go-mssqldb"
	_ "github.com/microsoft/go-mssqldb/integratedauth/krb5"
	"github.com/pkg/errors"

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
	db.Register(db.MSSQL, newDriver)
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
func (driver *Driver) Open(_ context.Context, _ db.Type, config db.ConnectionConfig, _ db.ConnectionContext) (db.Driver, error) {
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
func (*Driver) GetType() db.Type {
	return db.MSSQL
}

// GetDB gets the database.
func (driver *Driver) GetDB() *sql.DB {
	return driver.db
}

// Execute executes a SQL statement and returns the affected rows.
func (driver *Driver) Execute(ctx context.Context, statement string, createDatabase bool) (int64, error) {
	if createDatabase {
		if _, err := driver.db.ExecContext(ctx, statement); err != nil {
			return 0, err
		}
		return 0, nil
	}
	totalRowsAffected := int64(0)
	tx, err := driver.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	f := func(stmt string) error {
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

	if _, err := parser.SplitMultiSQLStream(parser.MSSQL, strings.NewReader(statement), f); err != nil {
		return 0, err
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return totalRowsAffected, nil
}

// QueryConn querys a SQL statement in a given connection.
func (*Driver) QueryConn(ctx context.Context, conn *sql.Conn, statement string, queryContext *db.QueryContext) ([]any, error) {
	return util.Query(ctx, db.MSSQL, conn, statement, queryContext)
}

// QueryConn2 queries a SQL statement in a given connection.
func (*Driver) QueryConn2(_ context.Context, _ *sql.Conn, _ string, _ *db.QueryContext) ([]*v1pb.QueryResult, error) {
	return nil, errors.New("not implemented")
}
