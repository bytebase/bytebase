// Package sqlite is the plugin for SQLite driver.
package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path"
	"strings"

	// Import sqlite3 driver.
	_ "github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/bytebase/bytebase/common/log"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/plugin/db/util"
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
}

func newDriver(db.DriverConfig) db.Driver {
	return &Driver{}
}

// Open opens a SQLite driver.
func (driver *Driver) Open(ctx context.Context, _ db.Type, config db.ConnectionConfig, connCtx db.ConnectionContext) (db.Driver, error) {
	// Host is the directory (instance) containing all SQLite databases.
	driver.dir = config.Host

	// If config.Database is empty, we will get a connection to in-memory database.
	if _, err := driver.GetDBConnection(ctx, config.Database); err != nil {
		return nil, err
	}
	driver.connectionCtx = connCtx
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

// SwitchDatabase switches the connected database.
func (*Driver) SwitchDatabase(context.Context, string) (func() error, error) {
	noop := func() error { return nil }
	return noop, nil
}

// GetDBConnection gets a database connection.
// If database is empty, we will get a connect to in-memory database.
func (driver *Driver) GetDBConnection(_ context.Context, database string) (*sql.DB, error) {
	if driver.db != nil {
		if err := driver.db.Close(); err != nil {
			return nil, err
		}
	}

	dns := path.Join(driver.dir, fmt.Sprintf("%s.db", database))
	if database == "" {
		dns = ":memory:"
	}
	db, err := sql.Open("sqlite3", dns)
	if err != nil {
		return nil, err
	}
	driver.db = db
	return db, nil
}

// getVersion gets the version.
func (driver *Driver) getVersion(ctx context.Context) (string, error) {
	var version string
	if err := driver.db.QueryRowContext(ctx, "SELECT sqlite_version();").Scan(&version); err != nil {
		return "", err
	}
	return version, nil
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
		databases = append(databases, strings.TrimRight(file.Name(), ".db"))
	}
	return databases, nil
}

func (driver *Driver) hasBytebaseDatabase() (bool, error) {
	databases, err := driver.getDatabases()
	if err != nil {
		return false, err
	}
	for _, database := range databases {
		if database == bytebaseDatabase {
			return true, nil
		}
	}
	return false, nil
}

// Execute executes a SQL statement.
func (driver *Driver) Execute(ctx context.Context, statement string, _ bool) (int64, error) {
	var remainingStmts []string
	f := func(stmt string) error {
		// This is a fake CREATE DATABASE statement. Engine driver will recognize it and establish a connection to create the database.
		stmt = strings.TrimLeft(stmt, " \t")
		if strings.HasPrefix(stmt, "CREATE DATABASE ") {
			parts := strings.Split(stmt, `'`)
			if len(parts) != 3 {
				return errors.Errorf("invalid statement %q", stmt)
			}
			db, err := driver.GetDBConnection(ctx, parts[1])
			if err != nil {
				return err
			}
			// We need to query to persist the database file.
			if _, err := db.ExecContext(ctx, "SELECT 1;"); err != nil {
				return err
			}
		} else if !strings.HasPrefix(stmt, "USE ") { // ignore the fake use database statement.
			remainingStmts = append(remainingStmts, stmt)
		}
		return nil
	}

	if err := util.ApplyMultiStatements(strings.NewReader(statement), f); err != nil {
		return 0, err
	}

	if len(remainingStmts) == 0 {
		return 0, nil
	}

	tx, err := driver.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	sqlResult, err := tx.ExecContext(ctx, strings.Join(remainingStmts, "\n"))
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

// Query queries a SQL statement.
func (driver *Driver) Query(ctx context.Context, statement string, queryContext *db.QueryContext) ([]interface{}, error) {
	return util.Query(ctx, db.SQLite, driver.db, statement, queryContext)
}
