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

	dbdriver "github.com/bytebase/bytebase/plugin/db"

	"github.com/bytebase/bytebase/common"
	"go.uber.org/zap"
)

const (
	// The schema version consists of major version and minor version.
	// Backward compatible schema change increases the minor version, while backward non-compatible schema change increase the majar version.
	// majorSchemaVervion and majorSchemaVervion defines the schema version this version of code can handle.
	// We reserve 4 least significant digits for minor version.
	// e.g.
	// 10001 -> Major verion 1, minor version 1
	// 11001 -> Major verion 1, minor version 1001
	// 20001 -> Major verion 2, minor version 1
	//
	// The migration file follows the name pattern of {{version_number}}__{{description}}.
	//
	// Though minor version is backward compatible, we require the schema version must match both the MAJOR and MINOR version,
	// otherwise, Bytebase will fail to start. We choose this because otherwise failed minor migration changes like adding an
	// index is hard to detect.
	//
	// If the new release requires a higher MAJOR version then the schema file, then the code will abort immediately. We
	// will require a separate process to upgrade the schema.
	// If the new release requires a higher MINOR version than the schema file, then it will apply the migration upon
	// startup.
	majorSchemaVervion          = 1
	createDatabaseSchemaVersion = "10000"
)

//go:embed migration
var migrationFS embed.FS

//go:embed seed
var seedFS embed.FS

// DB represents the database connection.
type DB struct {
	db *sql.DB

	l *zap.Logger

	// db.connCfg is the connection configuration to a Postgres database.
	// The user has superuser privilege to the database.
	connCfg dbdriver.ConnectionConfig

	// Dir to load seed data
	seedDir string

	// Force reset seed, true for testing and demo
	forceResetSeed bool

	// If true, database will be opened in readonly mode
	readonly bool

	// Bytebase release version
	releaseVersion string

	// Returns the current time. Defaults to time.Now().
	// Can be mocked for tests.
	Now func() time.Time
}

// NewDB returns a new instance of DB associated with the given datasource name.
func NewDB(logger *zap.Logger, dsn string, connCfg dbdriver.ConnectionConfig, seedDir string, forceResetSeed bool, readonly bool, releaseVersion string) *DB {
	db := &DB{
		l:              logger,
		connCfg:        connCfg,
		seedDir:        seedDir,
		forceResetSeed: forceResetSeed,
		readonly:       readonly,
		Now:            time.Now,
		releaseVersion: releaseVersion,
	}
	return db
}

// Open opens the database connection.
func (db *DB) Open(ctx context.Context) (err error) {
	d, err := dbdriver.Open(
		ctx,
		dbdriver.Postgres,
		dbdriver.DriverConfig{Logger: db.l},
		db.connCfg,
		dbdriver.ConnectionContext{},
	)
	if err != nil {
		return err
	}
	databaseName := db.connCfg.Username

	if db.readonly {
		db.l.Info("Database is opened in readonly mode. Skip migration and seeding.")
		// The database storing metadata is the same as user name.
		db.db, err = d.GetDbConnection(ctx, databaseName)
		if err != nil {
			return fmt.Errorf("failed to connect to database %q which may not be setup yet, error: %v", databaseName, err)
		}
		return nil
	}

	if err := d.SetupMigrationIfNeeded(ctx); err != nil {
		return err
	}
	verBefore, err := getLatestVersion(ctx, d, databaseName)
	if err != nil {
		return fmt.Errorf("failed to get current schema version: %w", err)
	}

	if err := db.migrate(ctx, d, verBefore, databaseName); err != nil {
		return fmt.Errorf("failed to migrate: %w", err)
	}

	verAfter, err := getLatestVersion(ctx, d, databaseName)
	if err != nil {
		return fmt.Errorf("failed to get current schema version: %w", err)
	}
	db.l.Info(fmt.Sprintf("Current schema version after migration: %s", verAfter))

	db.db, err = d.GetDbConnection(ctx, databaseName)
	if err != nil {
		return fmt.Errorf("failed to connect to database %q, error: %v", db.connCfg.Username, err)
	}

	if err := db.seed(verBefore, verAfter); err != nil {
		return fmt.Errorf("failed to seed: %w."+
			" It could be Bytebase is running against an old Bytebase schema. If you are developing Bytebase, you can remove pgdata"+
			" directory under the same directory where the bytebase binary resides. and restart again to let"+
			" Bytebase create the latest schema. If you are running in production and don't want to reset the data, you can contact support@bytebase.com for help",
			err)
	}

	return nil
}

func getLatestVersion(ctx context.Context, d dbdriver.Driver, database string) (ver version, err error) {
	limit := 1
	history, err := d.FindMigrationHistoryList(ctx, &dbdriver.MigrationHistoryFind{
		Database: &database,
		Limit:    &limit,
	})
	if err != nil {
		return version{}, fmt.Errorf("failed to get migration history, error: %v", err)
	}
	if len(history) == 0 {
		return version{major: 0, minor: 0}, nil
	}

	v, err := strconv.Atoi(history[0].Version)
	if err != nil {
		return version{}, fmt.Errorf("invalid version %q, error: %v", history[0].Version, err)
	}

	return versionFromInt(v), nil
}

// seed loads the seed data for testing
func (db *DB) seed(verBefore, verAfter version) error {
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
		ver := versionFromInt(version)
		if db.forceResetSeed || ver.biggerThan(verBefore) && !ver.biggerThan(verAfter) {
			if err := db.seedFile(name); err != nil {
				return fmt.Errorf("seed error: name=%q err=%w", name, err)
			}
		} else {
			db.l.Info(fmt.Sprintf("Skip this seed file: %s. The corresponding seed version %s is not in the applicable range (%s, %s].",
				name, ver, verBefore, verAfter))
		}
	}
	db.l.Info("Completed database seeding.")
	return nil
}

// seedFile runs a single seed file within a transaction.
func (db *DB) seedFile(name string) error {
	db.l.Info(fmt.Sprintf("Seeding %s...", name))
	tx, err := db.db.Begin()
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
// Migration files are embedded in the migration folder and are executed
// in lexicographical order.
//
// We prepend each migration file with version = xxx; Each migration
// file run in a transaction to prevent partial migrations.
func (db *DB) migrate(ctx context.Context, d dbdriver.Driver, curVer version, databaseName string) error {
	db.l.Info("Apply database migration if needed...")
	db.l.Info(fmt.Sprintf("Current schema version before migration: %s", curVer))

	// major version is 0 when the store isn't yet setup for the first time.
	if curVer.major != 0 && curVer.major != majorSchemaVervion {
		return fmt.Errorf("current major schema version %d is different from the major schema version %d this release %s expects", curVer.major, majorSchemaVervion, db.releaseVersion)
	}

	if curVer.major == 0 && curVer.minor == 0 {
		createDatabaseStatement := fmt.Sprintf("CREATE DATABASE %s", databaseName)
		if _, _, err := d.ExecuteMigration(
			ctx,
			&dbdriver.MigrationInfo{
				ReleaseVersion: db.releaseVersion,
				Version:        createDatabaseSchemaVersion,
				Namespace:      databaseName,
				Database:       databaseName,
				Environment:    "", /* unused in execute migration */
				Source:         dbdriver.LIBRARY,
				Type:           dbdriver.Baseline,
				Description:    fmt.Sprintf("Create database %s.", databaseName),
				CreateDatabase: true,
			},
			createDatabaseStatement,
		); err != nil {
			return fmt.Errorf("failed to migrate create database schema, error: %v", err)
		}
	}

	// Apply migrations
	names, err := fs.Glob(migrationFS, "migration/*.sql")
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
		v := versionFromInt(version)
		if v.biggerThan(curVer) {
			// Migrate migration files.
			db.l.Info(fmt.Sprintf("Migrating %s...", name))
			buf, err := fs.ReadFile(migrationFS, name)
			if err != nil {
				return err
			}
			if _, _, err := d.ExecuteMigration(
				ctx,
				&dbdriver.MigrationInfo{
					ReleaseVersion: db.releaseVersion,
					Version:        fmt.Sprintf("%d", version),
					Namespace:      databaseName,
					Database:       databaseName,
					Environment:    "", /* unused in execute migration */
					Source:         dbdriver.LIBRARY,
					Type:           dbdriver.Migrate,
					Description:    fmt.Sprintf("Migrate %s.", filepath.Base(name)),
				},
				string(buf),
			); err != nil {
				return fmt.Errorf("failed to migrate schema version %q, error: %v", filepath.Base(name), err)
			}
		} else {
			db.l.Info(fmt.Sprintf("Skip this migration file: %s. The corresponding migration version %s has already been applied.", name, v))
		}
	}

	db.l.Info("Completed database migration.")
	return nil
}

// Close closes the database connection.
func (db *DB) Close() error {
	// Close database.
	if db.db != nil {
		if err := db.db.Close(); err != nil {
			return err
		}
	}
	return nil
}

// BeginTx starts a transaction and returns a wrapper Tx type. This type
// provides a reference to the database and a fixed timestamp at the start of
// the transaction. The timestamp allows us to mock time during tests as well.
func (db *DB) BeginTx(ctx context.Context, opts *sql.TxOptions) (*Tx, error) {
	ptx, err := db.db.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}

	// Return wrapper Tx that includes the transaction start time.
	return &Tx{
		PTx: ptx,
		db:  db,
		now: db.Now().UTC().Truncate(time.Second),
	}, nil
}

// Tx wraps the SQL Tx object to provide a timestamp at the start of the transaction.
type Tx struct {
	PTx *sql.Tx
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
		return common.Errorf(common.Conflict, fmt.Errorf("email already exists"))
	case "UNIQUE constraint failed: member.principal_id":
		return common.Errorf(common.Conflict, fmt.Errorf("member already exists"))
	case "UNIQUE constraint failed: environment.name":
		return common.Errorf(common.Conflict, fmt.Errorf("environment name already exists"))
	case "UNIQUE constraint failed: project.key":
		return common.Errorf(common.Conflict, fmt.Errorf("project key already exists"))
	case "UNIQUE constraint failed: project_webhook.project_id, project_webhook.url":
		return common.Errorf(common.Conflict, fmt.Errorf("webhook url already exists"))
	case "UNIQUE constraint failed: project_member.project_id, project_member.principal_id":
		return common.Errorf(common.Conflict, fmt.Errorf("project member already exists"))
	case "UNIQUE constraint failed: db.instance_id, db.name":
		return common.Errorf(common.Conflict, fmt.Errorf("database name already exists"))
	case "UNIQUE constraint failed: data_source.instance_id, data_source.name":
		return common.Errorf(common.Conflict, fmt.Errorf("data source name already exists"))
	case "UNIQUE constraint failed: backup.database_id, backup.name":
		return common.Errorf(common.Conflict, fmt.Errorf("backup name already exists"))
	case "UNIQUE constraint failed: bookmark.creator_id, bookmark.link":
		return common.Errorf(common.Conflict, fmt.Errorf("bookmark already exists"))
	case "UNIQUE constraint failed: repository.project_id":
		return common.Errorf(common.Conflict, fmt.Errorf("project has already linked repository"))
	case "UNIQUE constraint failed: issue_subscriber.issue_id, issue_subscriber.subscriber_id":
		return common.Errorf(common.Conflict, fmt.Errorf("issue subscriber already exists"))
	default:
		return err
	}
}
