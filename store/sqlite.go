package store

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/bytebase/bytebase/common"
	_ "github.com/mattn/go-sqlite3"
	"go.uber.org/zap"
)

const (
	// We use SQLite "PRAGMA user_version" to manage the schema version. The schema version consists of
	// major version and minor version. Backward compatible schema change increases the minor version,
	// while backward non-compatible schema change increase the majar version.
	// MAX_MAJOR_SCHEMA_VERSION defines the maximum major schema version this version of code can handle.
	// We reserve 4 least significant digits for minor version.
	// e.g.
	// 10001 -> Major verion 1, minor version 1
	// 10002 -> Major verion 1, minor version 2
	// 20001 -> Major verion 2, minor version 1
	//
	// If MAX_MAJOR_SCHEMA_VERSION is 2, then it can handle version up to 29999 and report error if encountering
	// version >= 30000.
	//
	// The migration file follows the name pattern of {{version_number}}__{{description}}, and inside each migration
	// file, the first line is: PRAGMA user_version = {{version_number}};
	//
	// Notes about rollback
	//
	// The migration script is bundled with the code. If the new release contains new migration, it will be applied
	// upon startup. It could happen we push out a bad release. If it only involves minor version migration change,
	// then it's safe to use the older release because of the backward compatibility guarantee. However, if it
	// involves a major version migration change, the rollback is much harder, and we can only do this during
	// major app version change (like announce Bytebase 2.0 after 1.0), where we can allocate enough resource for
	// the migration. But hopefully, we would never do any major migration change at all. In other words, major
	// migration change is very costly and we should do it as the last resort.
	MAX_MAJOR_SCHEMA_VERSION = 1
)

// If both debug and sqlite_trace build tags are enabled, then sqliteDriver will be set to "sqlite3_trace" in sqlite_trace.go
var sqliteDriver = "sqlite3"

// Allocate 32MB cache
var pragmaList = []string{"_foreign_keys=1", "_journal_mode=WAL", "_cache_size=33554432"}

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

	// Force reset seed, true for testing and demo
	forceResetSeed bool

	// If true, database will be opened in readonly mode
	readonly bool

	// Returns the current time. Defaults to time.Now().
	// Can be mocked for tests.
	Now func() time.Time
}

// NewDB returns a new instance of DB associated with the given datasource name.
func NewDB(logger *zap.Logger, dsn string, seedDir string, forceResetSeed bool, readonly bool) *DB {
	if readonly {
		pragmaList = append(pragmaList, "mode=ro")
	}
	db := &DB{
		l:              logger,
		DSN:            strings.Join([]string{dsn, strings.Join(pragmaList, "&")}, "?"),
		seedDir:        seedDir,
		forceResetSeed: forceResetSeed,
		readonly:       readonly,
		Now:            time.Now,
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
	if db.readonly {
		db.l.Info("Database is opened in readonly mode. Skip migration and seeding.")
	} else {
		majorBeforeMigration, minorBeforeMigration, err := db.version()
		if err != nil {
			return fmt.Errorf("failed to get current schema version: %w", err)
		}

		if err := db.migrate(); err != nil {
			return fmt.Errorf("failed to migrate: %w", err)
		}

		majorAfterMigration, minorAfterMigration, err := db.version()
		if err != nil {
			return fmt.Errorf("failed to get current schema version: %w", err)
		}

		if err := db.seed(majorBeforeMigration, minorBeforeMigration, majorAfterMigration, minorAfterMigration); err != nil {
			return fmt.Errorf("failed to seed: %w."+
				" It could be Bytebase is running against an old Bytebase schema. If you are developing Bytebase, you can remove bytebase_dev.db,"+
				" bytebase_dev.db-shm, bytebase_dev.db-wal under the same directory where the bytebase binary resides. and restart again to let"+
				" Bytebase create the latest schema. If you are running in production and don't want to reset the data, you can contact support@bytebase.com for help",
				err)
		}
	}

	return nil
}

func (db *DB) version() (major int, minor int, err error) {
	rows, err := db.Db.Query("PRAGMA user_version")
	if err != nil {
		return 0, 0, err
	}

	var version int
	rows.Next()
	if err := rows.Scan(&version); err != nil {
		return 0, 0, err
	}
	return version / 10000, version % 10000, nil
}

// seed loads the seed data for testing
func (db *DB) seed(majorBeforeMigration int, minorBeforeMigration int, majorAfterMigration int, minorAfterMigration int) error {
	db.l.Info(fmt.Sprintf("Seeding database from %s, force: %t ...", db.seedDir, db.forceResetSeed))
	names, err := fs.Glob(seedFS, fmt.Sprintf("%s/*.sql", db.seedDir))
	if err != nil {
		return err
	}

	// We separate seed data for each table into their own seed file.
	// And there exists foreign key dependency among tables, so we
	// name the seed file as 10001_xxx.sql, 10002_xxx.sql. Here we sort
	// the file name so they are loaded accordingly.
	sort.Strings(names)

	// Loop over all seed files and execute them in order.
	for _, name := range names {
		versionPrefix := strings.Split(filepath.Base(name), "__")[0]
		version, err := strconv.Atoi(versionPrefix)
		if err != nil {
			return fmt.Errorf("invalid seed file format %s, expected number prefix", filepath.Base(name))
		}
		beforeVersion := majorBeforeMigration*10000 + minorBeforeMigration
		afterVersion := majorAfterMigration*10000 + minorAfterMigration
		if db.forceResetSeed || version > beforeVersion && version <= afterVersion {
			if err := db.seedFile(name); err != nil {
				return fmt.Errorf("seed error: name=%q err=%w", name, err)
			}
		} else {
			db.l.Info(fmt.Sprintf("Skip this seed file: %s. The corresponding seed version %d.%d is not in the applicable range (%d.%d, %d.%d].",
				name, version/10000, version%10000, majorBeforeMigration, minorBeforeMigration, majorAfterMigration, minorAfterMigration))
		}
	}
	db.l.Info("Completed database seeding.")
	return nil
}

// seedFile runs a single seed file within a transaction.
func (db *DB) seedFile(name string) error {
	db.l.Info(fmt.Sprintf("Seeding %s...", name))
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
// We prepend each migration file with PRAGMA user_version = xxx; Each migration
// file run in a transaction to prevent partial migrations.
func (db *DB) migrate() error {
	db.l.Info("Apply database migration if needed...")

	major, minor, err := db.version()
	if err != nil {
		return fmt.Errorf("failed to get current schema version: %w", err)
	}

	db.l.Info(fmt.Sprintf("Current schema version before migration: %d.%d", major, minor))

	if major > MAX_MAJOR_SCHEMA_VERSION {
		return fmt.Errorf("current major schema version %d is higher than the max major schema version %d this code can handle ", major, MAX_MAJOR_SCHEMA_VERSION)
	}

	// Apply migrations
	names, err := fs.Glob(migrationFS, fmt.Sprintf("%s/*.sql", "migration"))
	if err != nil {
		return err
	}

	// Sort the migration up file in ascending order.
	sort.Strings(names)

	for _, name := range names {
		versionPrefix := strings.Split(filepath.Base(name), "__")[0]
		version, err := strconv.Atoi(versionPrefix)
		if err != nil {
			return fmt.Errorf("invalid migration file format %s, expected number prefix", filepath.Base(name))
		}
		fileMajor, fileMinor := version/10000, version%10000
		if fileMajor > major || (fileMajor == major && fileMinor > minor) {
			if err := db.migrateFile(name, true); err != nil {
				return fmt.Errorf("migration error: name=%q err=%w", name, err)
			}
		} else {
			db.l.Info(fmt.Sprintf("Skip this migration file: %s. The corresponding migration version %d.%d has already been applied.", name, fileMajor, fileMinor))
		}
	}

	major, minor, err = db.version()
	if err != nil {
		return fmt.Errorf("failed to get current schema version: %w", err)
	}
	db.l.Info(fmt.Sprintf("Current schema version after migration: %d.%d", major, minor))

	// This is a sanity check to prevent us setting the incorrect user_version in the migration script.
	// e.g. We set PRAGMA user_version = 20001 while our code can only handle major version 1.
	if major != MAX_MAJOR_SCHEMA_VERSION {
		return fmt.Errorf("current schema major version %d does not match the expected schema major version %d after migration, make sure to set the correct PRAGMA user_version in the migration script", major, MAX_MAJOR_SCHEMA_VERSION)
	}

	db.l.Info("Completed database migration.")
	return nil
}

// migrateFile runs a migration file within a transaction.
func (db *DB) migrateFile(name string, up bool) error {
	if up {
		db.l.Info(fmt.Sprintf("Migrating %s...", name))
	} else {
		db.l.Info(fmt.Sprintf("Migrating %s...", name))
	}

	tx, err := db.Db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Read and execute migration file.
	if buf, err := fs.ReadFile(migrationFS, name); err != nil {
		return err
	} else if _, err := tx.Exec(string(buf)); err != nil {
		return err
	}

	return tx.Commit()
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
		return common.Errorf(common.ECONFLICT, "email already exists")
	case "UNIQUE constraint failed: member.principal_id":
		return common.Errorf(common.ECONFLICT, "member already exists")
	case "UNIQUE constraint failed: environment.name":
		return common.Errorf(common.ECONFLICT, "environment name already exists")
	case "UNIQUE constraint failed: project.key":
		return common.Errorf(common.ECONFLICT, "project key already exists")
	case "UNIQUE constraint failed: project_webhook.project_id, project_webhook.url":
		return common.Errorf(common.ECONFLICT, "webhook url already exists")
	case "UNIQUE constraint failed: project_member.project_id, project_member.principal_id":
		return common.Errorf(common.ECONFLICT, "project member already exists")
	case "UNIQUE constraint failed: db.instance_id, db.name":
		return common.Errorf(common.ECONFLICT, "database name already exists")
	case "UNIQUE constraint failed: data_source.instance_id, data_source.name":
		return common.Errorf(common.ECONFLICT, "data source name already exists")
	case "UNIQUE constraint failed: backup.database_id, backup.name":
		return common.Errorf(common.ECONFLICT, "backup name already exists")
	case "UNIQUE constraint failed: bookmark.creator_id, bookmark.link":
		return common.Errorf(common.ECONFLICT, "bookmark already exists")
	case "UNIQUE constraint failed: repository.project_id":
		return common.Errorf(common.ECONFLICT, "project has already linked repository")
	case "UNIQUE constraint failed: issue_subscriber.issue_id, issue_subscriber.subscriber_id":
		return common.Errorf(common.ECONFLICT, "issue subscriber already exists")
	default:
		return err
	}
}
