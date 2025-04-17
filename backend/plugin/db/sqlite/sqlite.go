// Package sqlite is the plugin for SQLite driver.
package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path"
	"strings"
	"time"

	// Import sqlite3 driver.
	_ "github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

var (
	_ db.Driver = (*Driver)(nil)
)

func init() {
	db.Register(storepb.Engine_SQLITE, newDriver)
}

// Driver is the SQLite driver.
type Driver struct {
	dir           string
	db            *sql.DB
	connectionCtx db.ConnectionContext
	databaseName  string
}

func newDriver() db.Driver {
	return &Driver{}
}

// Open opens a SQLite driver.
func (d *Driver) Open(_ context.Context, _ storepb.Engine, config db.ConnectionConfig) (db.Driver, error) {
	// Host is the directory (instance) containing all SQLite databases.
	d.dir = config.DataSource.Host

	// If config.Database is empty, we will get a connection to in-memory database.
	db, err := createDBConnection(d.dir, config.ConnectionContext.DatabaseName)
	if err != nil {
		return nil, err
	}
	d.db = db
	d.connectionCtx = config.ConnectionContext
	d.databaseName = config.ConnectionContext.DatabaseName
	return d, nil
}

// Close closes the driver.
func (d *Driver) Close(context.Context) error {
	if d.db != nil {
		return d.db.Close()
	}
	return nil
}

// Ping pings the database.
func (d *Driver) Ping(ctx context.Context) error {
	return d.db.PingContext(ctx)
}

// GetDB gets the database.
func (d *Driver) GetDB() *sql.DB {
	return d.db
}

// createDBConnection gets a database connection.
// If database is empty, we will get a connect to in-memory database.
func createDBConnection(dir, database string) (*sql.DB, error) {
	dns := path.Join(dir, fmt.Sprintf("%s.db", database))
	if database == "" {
		dns = ":memory:"
	}
	db, err := sql.Open("sqlite3", dns)
	if err != nil {
		return nil, err
	}
	return db, nil
}

func (d *Driver) getDatabases() ([]string, error) {
	files, err := os.ReadDir(d.dir)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read directory %q", d.dir)
	}
	var databases []string
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".db") {
			continue
		}
		databases = append(databases, strings.TrimSuffix(file.Name(), ".db"))
	}
	return databases, nil
}

// Execute executes a SQL statement.
func (d *Driver) Execute(ctx context.Context, statement string, opts db.ExecuteOptions) (int64, error) {
	if opts.CreateDatabase {
		parts := strings.Split(statement, `'`)
		if len(parts) != 3 {
			return 0, errors.Errorf("invalid statement %q", statement)
		}
		db, err := createDBConnection(d.dir, parts[1])
		if err != nil {
			return 0, err
		}
		defer db.Close()
		// We need to query to persist the database file.
		if _, err := db.ExecContext(ctx, "SELECT 1;"); err != nil {
			return 0, err
		}
		return 0, nil
	}

	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	sqlResult, err := tx.ExecContext(ctx, statement)
	if err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	rowsAffected, err := sqlResult.RowsAffected()
	if err != nil {
		// Since we cannot differentiate DDL and DML yet, we have to ignore the error.
		slog.Debug("rowsAffected returns error", log.BBError(err))
		return 0, nil
	}

	return rowsAffected, nil
}

// QueryConn queries a SQL statement in a given connection.
func (*Driver) QueryConn(ctx context.Context, conn *sql.Conn, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
	startTime := time.Now()
	rows, err := conn.QueryContext(ctx, statement)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result, err := util.RowsToQueryResult(rows, util.MakeCommonValueByTypeName, util.ConvertCommonValue, queryContext.MaximumSQLResultSize)
	if err != nil {
		// nolint
		return []*v1pb.QueryResult{
			{
				Error: err.Error(),
			},
		}, nil
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	result.Latency = durationpb.New(time.Since(startTime))
	result.Statement = statement
	result.RowsCount = int64(len(result.Rows))
	return []*v1pb.QueryResult{result}, nil
}
