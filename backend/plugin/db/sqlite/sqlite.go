// Package sqlite is the plugin for SQLite driver.
package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	// Import sqlite3 driver.
	_ "github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

var (
	bytebaseDatabase = "bytebase"

	_ db.Driver = (*Driver)(nil)
)

func init() {
	db.Register(db.SQLite, newDriver)
}

// Driver is the SQLite driver.
type Driver struct {
	dir           string
	db            *sql.DB
	connectionCtx db.ConnectionContext
	databaseName  string
}

func newDriver(db.DriverConfig) db.Driver {
	return &Driver{}
}

// Open opens a SQLite driver.
func (driver *Driver) Open(_ context.Context, _ db.Type, config db.ConnectionConfig, connCtx db.ConnectionContext) (db.Driver, error) {
	// Host is the directory (instance) containing all SQLite databases.
	driver.dir = config.Host

	// If config.Database is empty, we will get a connection to in-memory database.
	db, err := createDBConnection(driver.dir, config.Database)
	if err != nil {
		return nil, err
	}
	driver.db = db
	driver.connectionCtx = connCtx
	driver.databaseName = config.Database
	return driver, nil
}

// Close closes the driver.
func (driver *Driver) Close(context.Context) error {
	if driver.db != nil {
		return driver.db.Close()
	}
	return nil
}

// Ping pings the database.
func (driver *Driver) Ping(ctx context.Context) error {
	return driver.db.PingContext(ctx)
}

// GetType returns the database type.
func (*Driver) GetType() db.Type {
	return db.SQLite
}

// GetDB gets the database.
func (driver *Driver) GetDB() *sql.DB {
	return driver.db
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

func (driver *Driver) getDatabases() ([]string, error) {
	files, err := os.ReadDir(driver.dir)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read directory %q", driver.dir)
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
func (driver *Driver) Execute(ctx context.Context, statement string, createDatabase bool, _ db.ExecuteOptions) (int64, error) {
	if createDatabase {
		parts := strings.Split(statement, `'`)
		if len(parts) != 3 {
			return 0, errors.Errorf("invalid statement %q", statement)
		}
		db, err := createDBConnection(driver.dir, parts[1])
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

	tx, err := driver.db.BeginTx(ctx, nil)
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
		log.Debug("rowsAffected returns error", zap.Error(err))
		return 0, nil
	}

	return rowsAffected, nil
}

// QueryConn queries a SQL statement in a given connection.
func (*Driver) QueryConn(ctx context.Context, conn *sql.Conn, statement string, queryContext *db.QueryContext) ([]*v1pb.QueryResult, error) {
	startTime := time.Now()
	result, err := util.Query(ctx, db.SQLite, conn, statement, queryContext)
	if err != nil {
		return nil, err
	}
	result.Latency = durationpb.New(time.Since(startTime))
	result.Statement = strings.TrimRight(statement, " \n\t;")

	return []*v1pb.QueryResult{result}, nil
}

// RunStatement runs a SQL statement.
func (*Driver) RunStatement(_ context.Context, _ *sql.Conn, _ string) ([]*v1pb.QueryResult, error) {
	return nil, errors.New("not implemented")
}
