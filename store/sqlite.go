package store

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/bytebase/bytebase"
	"github.com/golang-migrate/migrate/v4/source/httpfs"
	_ "github.com/mattn/go-sqlite3"
	"go.uber.org/zap"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// If both debug and sqlite_trace build tags are enabled, then sqliteDriver will be set to "sqlite3_trace"
var sqliteDriver = "sqlite3"

var pragmaList = []string{"_foreign_keys=1", "_journal_mode=WAL"}

//go:embed migration
var migrationFS embed.FS

//go:embed seed
var seedFS embed.FS

// DB represents the database connection.
type DB struct {
	Db *sql.DB

	l *zap.Logger

	// Datasource name.
	DSN string

	// Dir to load seed data
	seedDir string

	// Returns the current time. Defaults to time.Now().
	// Can be mocked for tests.
	Now func() time.Time
}

// NewDB returns a new instance of DB associated with the given datasource name.
func NewDB(logger *zap.Logger, dsn string, seedDir string) *DB {
	db := &DB{
		l:       logger,
		DSN:     strings.Join([]string{dsn, strings.Join(pragmaList, "&")}, "?"),
		seedDir: seedDir,
		Now:     time.Now,
	}
	return db
}

// Open opens the database connection.
func (db *DB) Open() (err error) {
	// Ensure a DSN is set before attempting to open the database.
	if db.DSN == "" {
		return fmt.Errorf("dsn required")
	}

	// Connect to the database.
	if db.Db, err = sql.Open(sqliteDriver, db.DSN); err != nil {
		return err
	}

	if err := db.migrate(); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}

	if err := db.seed(); err != nil {
		return fmt.Errorf("seed: %w", err)
	}

	return nil
}

// seed loads the seed data for testing
func (db *DB) seed() error {
	db.l.Info(fmt.Sprintf("Seeding database from %s...", db.seedDir))
	names, err := fs.Glob(seedFS, fmt.Sprintf("%s/*.sql", db.seedDir))
	if err != nil {
		return err
	}

	// We separate seed data for each table into their own seed file.
	// And there exists foreign key dependency among tables, so we
	// name the seed file as 01_xxx.sql, 02_xxx.sql. Here we sort
	// the file name so they are loaded accordingly.
	sort.Strings(names)

	// Loop over all seed files and execute them in order.
	for _, name := range names {
		if err := db.seedFile(name); err != nil {
			return fmt.Errorf("seed error: name=%q err=%w", name, err)
		}
	}
	db.l.Info("Completed database seeding.")
	return nil
}

// seedFile runs a single seed file within a transaction.
func (db *DB) seedFile(name string) error {
	db.l.Debug(fmt.Sprintf("Seeding %s...", name))
	tx, err := db.Db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Read and execute migration file.
	if buf, err := fs.ReadFile(seedFS, name); err != nil {
		return err
	} else if _, err := tx.Exec(string(buf)); err != nil {
		return err
	}

	return tx.Commit()
}

// migrate sets up migration tracking and executes pending migration files.
//
// Migration files are embedded in the sqlite/migration folder and are executed
// in lexigraphical order.
//
// Once a migration is run, its name is stored in the 'migrations' table so it
// is not re-executed. Migrations run in a transaction to prevent partial
// migrations.
func (db *DB) migrate() error {
	db.l.Info("Apply database migration if needed...")

	source, err := httpfs.New(http.FS(migrationFS), "migration")
	if err != nil {
		return err
	}

	m, err := migrate.NewWithSourceInstance(
		"httpfs",
		source,
		"sqlite3://"+db.DSN)
	if err != nil {
		return err
	}

	v1, dirty1, err := m.Version()
	if err != nil {
		if err != migrate.ErrNilVersion {
			return err
		}
	}
	db.l.Info(fmt.Sprintf("Database version before migration down: %v, dirty: %v", v1, dirty1))

	if err := m.Down(); err != nil {
		if err == migrate.ErrNoChange {
			db.l.Info("No need to migrate down.")
		} else {
			return fmt.Errorf("migrate down error: %w", err)
		}
	}

	v2, dirty2, err := m.Version()
	db.l.Info(fmt.Sprintf("Database version before migration up: %v, dirty: %v", v2, dirty2))
	if err != nil {
		if err != migrate.ErrNilVersion {
			return err
		}
	}

	if err := m.Up(); err != nil {
		if err == migrate.ErrNoChange {
			db.l.Info("No need to migrate up.")
		} else {
			return fmt.Errorf("migrate up error: %w", err)
		}
	}

	v3, dirty3, err := m.Version()
	if err != nil {
		if err != migrate.ErrNilVersion {
			return err
		}
	}
	db.l.Info(fmt.Sprintf("Database version after migration: %v, dirty: %v", v3, dirty3))

	db.l.Info("Completed database migration.")

	return nil
}

// Close closes the database connection.
func (db *DB) Close() error {
	// Close database.
	if db.Db != nil {
		return db.Db.Close()
	}
	return nil
}

// BeginTx starts a transaction and returns a wrapper Tx type. This type
// provides a reference to the database and a fixed timestamp at the start of
// the transaction. The timestamp allows us to mock time during tests as well.
func (db *DB) BeginTx(ctx context.Context, opts *sql.TxOptions) (*Tx, error) {
	tx, err := db.Db.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}

	// Return wrapper Tx that includes the transaction start time.
	return &Tx{
		Tx:  tx,
		db:  db,
		now: db.Now().UTC().Truncate(time.Second),
	}, nil
}

// Tx wraps the SQL Tx object to provide a timestamp at the start of the transaction.
type Tx struct {
	*sql.Tx
	db  *DB
	now time.Time
}

// FormatError returns err as a bytebase error, if possible.
// Otherwise returns the original error.
func FormatError(err error) error {
	if err == nil {
		return nil
	}

	switch err.Error() {
	case "UNIQUE constraint failed: principal.email":
		return bytebase.Errorf(bytebase.ECONFLICT, "email already exists")
	case "UNIQUE constraint failed: member.principal_id":
		return bytebase.Errorf(bytebase.ECONFLICT, "member already exists")
	case "UNIQUE constraint failed: environment.name":
		return bytebase.Errorf(bytebase.ECONFLICT, "environment name already exists")
	case "UNIQUE constraint failed: project.key":
		return bytebase.Errorf(bytebase.ECONFLICT, "project key already exists")
	case "UNIQUE constraint failed: project_member.project_id, project_member.principal_id":
		return bytebase.Errorf(bytebase.ECONFLICT, "project member already exists")
	case "UNIQUE constraint failed: db.instance_id, db.name":
		return bytebase.Errorf(bytebase.ECONFLICT, "database name already exists")
	case "UNIQUE constraint failed: data_source.instance_id, data_source.name":
		return bytebase.Errorf(bytebase.ECONFLICT, "data source name already exists")
	case "UNIQUE constraint failed: bookmark.creator_id, bookmark.link":
		return bytebase.Errorf(bytebase.ECONFLICT, "bookmark already exists")
	case "UNIQUE constraint failed: repo.project_id":
		return bytebase.Errorf(bytebase.ECONFLICT, "project has already linked repository")
	default:
		return err
	}
}
