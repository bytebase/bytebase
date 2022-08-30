package store

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/blang/semver/v4"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/bytebase/bytebase/common/log"
	dbdriver "github.com/bytebase/bytebase/plugin/db"

	// Register postgres driver.
	_ "github.com/bytebase/bytebase/plugin/db/pg"

	"github.com/bytebase/bytebase/common"
)

const (
	// The schema version consists of major version and minor version.
	// Backward compatible schema change increases the minor version, while backward non-compatible schema change increase the major version.
	// majorSchemaVersion and majorSchemaVersion defines the schema version this version of code can handle.
	// We reserve 4 least significant digits for minor version.
	// e.g.
	// 10001 -> Major version 1, minor version 1
	// 11001 -> Major version 1, minor version 1001
	// 20001 -> Major version 2, minor version 1
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
	majorSchemaVersion = 1
)

//go:embed migration
var migrationFS embed.FS

//go:embed demo
var demoFS embed.FS

// DB represents the database connection.
type DB struct {
	db *sql.DB

	// db.connCfg is the connection configuration to a Postgres database.
	// The user has superuser privilege to the database.
	connCfg dbdriver.ConnectionConfig

	// Dir to load demo data
	demoDataDir string

	// Dir for postgres instance
	pgBaseDir string

	// If true, database will be opened in readonly mode
	readonly bool

	// Bytebase server release version
	serverVersion string

	// mode is the mode of the release such as prod or dev.
	mode common.ReleaseMode

	// Returns the current time. Defaults to time.Now().
	// Can be mocked for tests.
	Now func() time.Time
}

// NewDB returns a new instance of DB associated with the given datasource name.
func NewDB(connCfg dbdriver.ConnectionConfig, pgBaseDir, demoDataDir string, readonly bool, serverVersion string, mode common.ReleaseMode) *DB {
	db := &DB{
		connCfg:       connCfg,
		demoDataDir:   demoDataDir,
		pgBaseDir:     pgBaseDir,
		readonly:      readonly,
		Now:           time.Now,
		serverVersion: serverVersion,
		mode:          mode,
	}
	return db
}

// Open opens the database connection.
func (db *DB) Open(ctx context.Context) (err error) {
	d, err := dbdriver.Open(
		ctx,
		dbdriver.Postgres,
		dbdriver.DriverConfig{PgInstanceDir: db.pgBaseDir},
		db.connCfg,
		dbdriver.ConnectionContext{},
	)
	if err != nil {
		return err
	}

	var databaseName string
	if db.connCfg.StrictUseDb {
		databaseName = db.connCfg.Database
	} else {
		databaseName = db.connCfg.Username
	}

	if db.readonly {
		log.Info("Database is opened in readonly mode. Skip migration and demo data setup.")
		// The database storing metadata is the same as user name.
		db.db, err = d.GetDBConnection(ctx, databaseName)
		if err != nil {
			return errors.Wrapf(err, "failed to connect to database %q which may not be setup yet", databaseName)
		}
		return nil
	}

	// We are also using our own migration core to manage our own schema's migration history.
	// So here we will create a "bytebase" database to store the migration history if the target
	// db instance does not have one yet.
	if err := d.SetupMigrationIfNeeded(ctx); err != nil {
		return err
	}

	verBefore, err := getLatestVersion(ctx, d, databaseName)
	if err != nil {
		return errors.Wrap(err, "failed to get current schema version")
	}

	if err := migrate(ctx, d, verBefore, db.mode, db.connCfg.StrictUseDb, db.serverVersion, databaseName); err != nil {
		return errors.Wrap(err, "failed to migrate")
	}

	verAfter, err := getLatestVersion(ctx, d, databaseName)
	if err != nil {
		return errors.Wrap(err, "failed to get current schema version")
	}
	log.Info(fmt.Sprintf("Current schema version after migration: %s", verAfter))

	db.db, err = d.GetDBConnection(ctx, databaseName)
	if err != nil {
		return errors.Wrapf(err, "failed to connect to database %q", db.connCfg.Username)
	}

	if err := db.setupDemoData(); err != nil {
		return errors.Wrapf(err, "failed to setup demo data."+
			" It could be Bytebase is running against an old Bytebase schema. If you are developing Bytebase, you can remove pgdata"+
			" directory under the same directory where the Bytebase binary resides. and restart again to let"+
			" Bytebase create the latest schema. If you are running in production and don't want to reset the data, you can contact support@bytebase.com for help")
	}

	return nil
}

// getLatestVersion returns the latest schema version in semantic versioning format.
// We expect our own migration history to use semantic versions.
// If there's no migration history, version will be nil.
func getLatestVersion(ctx context.Context, d dbdriver.Driver, database string) (*semver.Version, error) {
	limit := 1
	history, err := d.FindMigrationHistoryList(ctx, &dbdriver.MigrationHistoryFind{
		Database: &database,
		Limit:    &limit,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get migration history")
	}
	if len(history) == 0 {
		return nil, nil
	}

	for _, h := range history {
		if h.Status != dbdriver.Done {
			continue
		}
		v, err := semver.Make(h.Version)
		if err != nil {
			return nil, errors.Wrapf(err, "invalid version %q", h.Version)
		}
		return &v, nil
	}

	return nil, errors.Errorf("failed to find a successful migration history")
}

// setupDemoData loads the setupDemoData data for testing.
func (db *DB) setupDemoData() error {
	if db.demoDataDir == "" {
		log.Debug("Skip setting up demo data. Demo data directory not specified.")
		return nil
	}
	log.Info(fmt.Sprintf("Setting up demo data from %q...", db.demoDataDir))
	names, err := fs.Glob(demoFS, fmt.Sprintf("%s/*.sql", db.demoDataDir))
	if err != nil {
		return err
	}

	// We separate demo data for each table into their own demo data file.
	// And there exists foreign key dependency among tables, so we
	// name the data file as 10001_xxx.sql, 10002_xxx.sql. Here we sort
	// the file name so they are loaded accordingly.
	sort.Strings(names)

	// Loop over all data files and execute them in order.
	for _, name := range names {
		if err := db.applyDataFile(name); err != nil {
			return errors.Wrapf(err, "applyDataFile error: name=%q", name)
		}
	}
	log.Info("Completed demo data setup.")
	return nil
}

// applyDataFile runs a single demo data file within a transaction.
func (db *DB) applyDataFile(name string) error {
	log.Info(fmt.Sprintf("Applying data file %s...", name))
	tx, err := db.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Read and execute migration file.
	if buf, err := fs.ReadFile(demoFS, name); err != nil {
		return err
	} else if _, err := tx.Exec(string(buf)); err != nil {
		return err
	}

	return tx.Commit()
}

const (
	latestSchemaFile = "LATEST.sql"
	latestDataFile   = "LATEST_DATA.sql"
)

// migrate sets up migration tracking and executes pending migration files.
//
// Migration files are embedded in the migration folder and are executed
// in lexicographical order.
//
// We prepend each migration file with version = xxx; Each migration
// file run in a transaction to prevent partial migrations.
//
// The procedure follows https://github.com/bytebase/bytebase/blob/main/docs/schema-update-guide.md.
func migrate(ctx context.Context, d dbdriver.Driver, curVer *semver.Version, mode common.ReleaseMode, strictDb bool, serverVersion, databaseName string) error {
	log.Info("Apply database migration if needed...")
	if curVer == nil {
		log.Info("The database schema has not been setup.")
	} else {
		log.Info(fmt.Sprintf("Current schema version before migration: %s", curVer))
		major := curVer.Major
		if major != majorSchemaVersion {
			return errors.Errorf("current major schema version %d is different from the major schema version %d this release %s expects", major, majorSchemaVersion, serverVersion)
		}
	}

	// Calculate prod cutoffSchemaVersion.
	cutoffSchemaVersion, err := getProdCutoffVersion()
	if err != nil {
		return errors.Wrapf(err, "failed to get cutoff version")
	}
	log.Info(fmt.Sprintf("The prod cutoff schema version: %s", cutoffSchemaVersion))

	var histories []*dbdriver.MigrationHistory
	// Because dev migrations don't use semantic versioning, we have to look at all migration history to
	// figure out whether the migration statement has already been applied.
	if mode == common.ReleaseModeDev {
		h, err := d.FindMigrationHistoryList(ctx, &dbdriver.MigrationHistoryFind{
			Database: &databaseName,
		})
		if err != nil {
			return err
		}
		histories = h
	}

	if curVer == nil {
		// Initial schema setup if not yet setup.
		latestSchemaPath := fmt.Sprintf("migration/%s/%s", common.ReleaseModeProd, latestSchemaFile)
		buf, err := migrationFS.ReadFile(latestSchemaPath)
		if err != nil {
			return errors.Wrapf(err, "failed to read latest schema %q", latestSchemaPath)
		}
		latestDataPath := fmt.Sprintf("migration/%s/%s", common.ReleaseModeProd, latestDataFile)
		dataBuf, err := migrationFS.ReadFile(latestDataPath)
		if err != nil {
			return errors.Wrapf(err, "failed to read latest data %q", latestSchemaPath)
		}

		var stmt string
		var createDatabase bool
		if strictDb {
			// User gives only database instead of instance, we cannot create database again.
			stmt = fmt.Sprintf("%s\n%s", buf, dataBuf)
			createDatabase = false
		} else {
			// We will create the database together with initial schema and data migration.
			stmt = fmt.Sprintf("CREATE DATABASE %s;\n\\connect \"%s\";\n%s\n%s", databaseName, databaseName, buf, dataBuf)
			createDatabase = true
		}

		if _, _, err := d.ExecuteMigration(
			ctx,
			&dbdriver.MigrationInfo{
				ReleaseVersion:        serverVersion,
				UseSemanticVersion:    true,
				Version:               cutoffSchemaVersion.String(),
				SemanticVersionSuffix: common.DefaultMigrationVersion(),
				Namespace:             databaseName,
				Database:              databaseName,
				Environment:           "", /* unused in execute migration */
				Source:                dbdriver.LIBRARY,
				Type:                  dbdriver.Migrate,
				Description:           fmt.Sprintf("Initial migration version %s server version %s with file %s.", cutoffSchemaVersion, serverVersion, latestSchemaPath),
				CreateDatabase:        createDatabase,
				Force:                 true,
			},
			stmt,
		); err != nil {
			return errors.Wrapf(err, "failed to migrate initial schema version %q", latestSchemaPath)
		}
		log.Info(fmt.Sprintf("Completed database initial migration with version %s.", cutoffSchemaVersion))
	} else {
		// Apply migrations if needed.
		retVersion := *curVer
		names, err := fs.Glob(migrationFS, fmt.Sprintf("migration/%s/*", common.ReleaseModeProd))
		if err != nil {
			return err
		}

		minorVersions, messages, err := getMinorMigrationVersions(names, *curVer)
		if err != nil {
			return err
		}
		for _, message := range messages {
			log.Info(message)
		}

		for _, minorVersion := range minorVersions {
			log.Info(fmt.Sprintf("Starting minor version migration cycle from %s ...", minorVersion))
			names, err := fs.Glob(migrationFS, fmt.Sprintf("migration/%s/%d.%d/*.sql", common.ReleaseModeProd, minorVersion.Major, minorVersion.Minor))
			if err != nil {
				return err
			}
			patchVersions, err := getPatchVersions(minorVersion, *curVer, names)
			if err != nil {
				return err
			}

			for _, pv := range patchVersions {
				buf, err := fs.ReadFile(migrationFS, pv.filename)
				if err != nil {
					return errors.Wrapf(err, "failed to read migration file %q", pv.filename)
				}
				// This happens when a migration file is moved from dev to release and we should not reapply the migration.
				// For example,
				//   before - prod: 1.2; dev: 123.sql, something else.
				//   after - prod 1.3 with 123.sql; dev: something else.
				// When dev starts, it will try to apply version 1.3 including 123.sql. If we don't skip, the same statement will be re-applied and most likely to fail.
				if mode == common.ReleaseModeDev && migrationExists(string(buf), histories) {
					log.Info(fmt.Sprintf("Skip migrating migration file %s that's already migrated.", pv.filename))
					continue
				}
				log.Info(fmt.Sprintf("Migrating %s...", pv.version))
				if _, _, err := d.ExecuteMigration(
					ctx,
					&dbdriver.MigrationInfo{
						ReleaseVersion:        serverVersion,
						UseSemanticVersion:    true,
						Version:               pv.version.String(),
						SemanticVersionSuffix: common.DefaultMigrationVersion(),
						Namespace:             databaseName,
						Database:              databaseName,
						Environment:           "", /* unused in execute migration */
						Source:                dbdriver.LIBRARY,
						Type:                  dbdriver.Migrate,
						Description:           fmt.Sprintf("Migrate version %s server version %s with files %s.", pv.version, serverVersion, pv.filename),
						Force:                 true,
					},
					string(buf),
				); err != nil {
					return errors.Wrapf(err, "failed to migrate schema version %q", pv.version)
				}
				retVersion = pv.version
			}
		}
		if retVersion.EQ(*curVer) {
			log.Info(fmt.Sprintf("Database schema is at version %s; nothing to migrate.", *curVer))
		} else {
			log.Info(fmt.Sprintf("Completed database migration from version %s to %s.", *curVer, retVersion))
		}
	}

	if mode == common.ReleaseModeDev {
		if err := migrateDev(ctx, d, serverVersion, databaseName, cutoffSchemaVersion, histories); err != nil {
			return errors.Wrapf(err, "failed to migrate dev schema")
		}
	}
	return nil
}

func getProdCutoffVersion() (semver.Version, error) {
	minorPathPrefix := fmt.Sprintf("migration/%s/*", common.ReleaseModeProd)
	names, err := fs.Glob(migrationFS, minorPathPrefix)
	if err != nil {
		return semver.Version{}, err
	}

	versions, err := getMinorVersions(names)
	if err != nil {
		return semver.Version{}, err
	}
	if len(versions) == 0 {
		return semver.Version{}, errors.Errorf("migration path %s has no minor version", minorPathPrefix)
	}
	minorVersion := versions[len(versions)-1]

	patchPathPrefix := fmt.Sprintf("migration/%s/%d.%d", common.ReleaseModeProd, minorVersion.Major, minorVersion.Minor)
	names, err = fs.Glob(migrationFS, fmt.Sprintf("%s/*.sql", patchPathPrefix))
	if err != nil {
		return semver.Version{}, err
	}
	patchVersions, err := getPatchVersions(minorVersion, semver.Version{} /* currentVersion */, names)
	if err != nil {
		return semver.Version{}, err
	}
	if len(patchVersions) == 0 {
		return semver.Version{}, errors.Errorf("migration path %s has no patch version", patchPathPrefix)
	}
	return patchVersions[len(patchVersions)-1].version, nil
}

func migrateDev(ctx context.Context, d dbdriver.Driver, serverVersion, databaseName string, cutoffSchemaVersion semver.Version, histories []*dbdriver.MigrationHistory) error {
	devMigrations, err := getDevMigrations()
	if err != nil {
		return err
	}

	var migrations []devMigration
	// Skip migrations that are already applied, otherwise the migration reattempt will most likely to fail with already exists error.
	for _, m := range devMigrations {
		if migrationExists(m.statement, histories) {
			log.Info(fmt.Sprintf("Skip migrating dev migration file %s that's already migrated.", m.filename))
		} else {
			migrations = append(migrations, m)
		}
	}
	if len(migrations) == 0 {
		log.Info("Skip dev mode migration; no new version.")
		return nil
	}

	for _, m := range migrations {
		log.Info(fmt.Sprintf("Migrating dev %s...", m.filename))
		// We expect to use semantic versioning for dev environment too because getLatestVersion() always expect to get the latest version in semantic format.
		if _, _, err := d.ExecuteMigration(
			ctx,
			&dbdriver.MigrationInfo{
				ReleaseVersion:        serverVersion,
				UseSemanticVersion:    true,
				Version:               cutoffSchemaVersion.String(),
				SemanticVersionSuffix: fmt.Sprintf("dev%s", m.version),
				Namespace:             databaseName,
				Database:              databaseName,
				Environment:           "", /* unused in execute migration */
				Source:                dbdriver.LIBRARY,
				Type:                  dbdriver.Migrate,
				Description:           fmt.Sprintf("Migrate version %s server version %s with files %s.", m.version, serverVersion, m.filename),
				Force:                 true,
			},
			m.statement,
		); err != nil {
			return errors.Wrapf(err, "failed to migrate schema version %q", m.version)
		}
	}

	return nil
}

type devMigration struct {
	filename  string
	version   string
	statement string
}

func getDevMigrations() ([]devMigration, error) {
	names, err := fs.Glob(migrationFS, fmt.Sprintf("migration/%s/*", common.ReleaseModeDev))
	if err != nil {
		return nil, err
	}

	var devMigrations []devMigration
	for _, name := range names {
		baseName := filepath.Base(name)
		if baseName == latestSchemaFile || baseName == latestDataFile {
			continue
		}

		parts := strings.Split(baseName, "__")
		if len(parts) != 2 {
			return nil, errors.Errorf("invalid migration file name %q", name)
		}
		buf, err := fs.ReadFile(migrationFS, name)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to read dev migration file %q", name)
		}

		devMigrations = append(devMigrations, devMigration{
			filename:  name,
			version:   parts[0],
			statement: string(buf),
		})
	}
	sort.Slice(devMigrations, func(i, j int) bool {
		return devMigrations[i].version < devMigrations[j].version
	})
	return devMigrations, nil
}

func migrationExists(statement string, histories []*dbdriver.MigrationHistory) bool {
	for _, history := range histories {
		if history.Statement == statement {
			return true
		}
	}
	return false
}

type patchVersion struct {
	version  semver.Version
	filename string
}

// getPatchVersions gets the patch versions above the current version in a minor version directory.
func getPatchVersions(minorVersion semver.Version, currentVersion semver.Version, names []string) ([]patchVersion, error) {
	var patchVersions []patchVersion
	for _, name := range names {
		baseName := filepath.Base(name)
		parts := strings.Split(baseName, "__")
		if len(parts) != 2 {
			return nil, errors.Errorf("migration filename %q should include '__'", name)
		}
		patch, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			return nil, errors.Wrapf(err, "migration filename prefix %q should be four digits integer such as '0000'", parts[0])
		}
		version := minorVersion
		version.Patch = uint64(patch)
		if version.LE(currentVersion) {
			continue
		}

		patchVersions = append(patchVersions,
			patchVersion{
				version:  version,
				filename: name,
			},
		)
	}
	if len(patchVersions) == 0 {
		return nil, nil
	}
	// Sort patch version in ascending order.
	sort.Slice(patchVersions, func(i, j int) bool {
		return patchVersions[i].version.LT(patchVersions[j].version)
	})
	return patchVersions, nil
}

// getMinorMigrationVersions gets all the prod minor versions since currentVersion (included).
func getMinorMigrationVersions(names []string, currentVersion semver.Version) ([]semver.Version, []string, error) {
	versions, err := getMinorVersions(names)
	if err != nil {
		return nil, nil, err
	}

	// We should still include the version with the same minor version with currentVersion in case we have missed some patches.
	currentVersion.Patch = 0

	var migrateVersions []semver.Version
	var messages []string
	for _, version := range versions {
		// If the migration version is less than to the current version, we will skip the migration since it's already applied.
		// We should still double check the current version in case there's any patch needed.
		if version.LT(currentVersion) {
			messages = append(messages, fmt.Sprintf("Skip migration %s; the current schema version %s is higher.", version, currentVersion))
			continue
		}
		migrateVersions = append(migrateVersions, version)
	}
	return migrateVersions, messages, nil
}

// getMinorVersions returns the minor versions based on file names in the prod directory.
func getMinorVersions(names []string) ([]semver.Version, error) {
	var versions []semver.Version
	for _, name := range names {
		baseName := filepath.Base(name)
		if baseName == latestSchemaFile || baseName == latestDataFile {
			continue
		}
		// Convert minor version to semantic version format, e.g. "1.12" will be "1.12.0".
		s := fmt.Sprintf("%s.0", baseName)
		v, err := semver.Make(s)
		if err != nil {
			return nil, errors.Wrapf(err, "invalid migration file path %q", name)
		}
		versions = append(versions, v)
	}
	// Sort the migration semantic version in ascending order.
	sort.Slice(versions, func(i, j int) bool {
		return versions[i].LT(versions[j])
	})
	return versions, nil
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
		Tx:  ptx,
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

// Replace mutiple whitespace characters including /t/n with a single space.
var pattern = regexp.MustCompile(`\s+`)

func cleanQuery(query string) string {
	return pattern.ReplaceAllString(query, " ")
}

// PrepareContext overrides sql.Tx PrepareContext.
func (tx *Tx) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	log.Debug("PrepareContext", zap.String("query", cleanQuery(query)))
	return tx.Tx.PrepareContext(ctx, query)
}

// ExecContext overrides sql.Tx ExecContext.
func (tx *Tx) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	log.Debug("ExecContext", zap.String("query", cleanQuery(query)))
	return tx.Tx.ExecContext(ctx, query, args...)
}

// QueryContext overrides sql.Tx QueryContext.
func (tx *Tx) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	log.Debug("QueryContext", zap.String("query", cleanQuery(query)))
	return tx.Tx.QueryContext(ctx, query, args...)
}

// QueryRowContext overrides sql.Tx QueryRowContext.
func (tx *Tx) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	log.Debug("QueryRowContext", zap.String("query", cleanQuery(query)))
	return tx.Tx.QueryRowContext(ctx, query, args...)
}

// FormatError returns err as a Bytebase error, if possible.
// Otherwise returns the original error.
func FormatError(err error) error {
	if err == nil {
		return nil
	}

	if strings.Contains(err.Error(), "unique constraint") {
		switch {
		case strings.Contains(err.Error(), "idx_principal_unique_email"):
			return common.Errorf(common.Conflict, "email already exists")
		case strings.Contains(err.Error(), "idx_setting_unique_name"):
			return common.Errorf(common.Conflict, "setting name already exists")
		case strings.Contains(err.Error(), "idx_member_unique_principal_id"):
			return common.Errorf(common.Conflict, "member already exists")
		case strings.Contains(err.Error(), "idx_environment_unique_name"):
			return common.Errorf(common.Conflict, "environment name already exists")
		case strings.Contains(err.Error(), "idx_policy_unique_environment_id_type"):
			return common.Errorf(common.Conflict, "policy environment and type already exists")
		case strings.Contains(err.Error(), "idx_project_unique_key"):
			return common.Errorf(common.Conflict, "project key already exists")
		case strings.Contains(err.Error(), "idx_project_member_unique_project_id_role_provider_principal_id"):
			return common.Errorf(common.Conflict, "project member already exists")
		case strings.Contains(err.Error(), "idx_project_webhook_unique_project_id_url"):
			return common.Errorf(common.Conflict, "webhook url already exists")
		case strings.Contains(err.Error(), "idx_instance_user_unique_instance_id_name"):
			return common.Errorf(common.Conflict, "instance id and name already exists")
		case strings.Contains(err.Error(), "idx_db_unique_instance_id_name"):
			return common.Errorf(common.Conflict, "database name already exists")
		case strings.Contains(err.Error(), "idx_tbl_unique_database_id_name"):
			return common.Errorf(common.Conflict, "database id and name already exists")
		case strings.Contains(err.Error(), "idx_col_unique_database_id_table_id_name"):
			return common.Errorf(common.Conflict, "database id, table id and name already exists")
		case strings.Contains(err.Error(), "idx_idx_unique_database_id_table_id_name_expression"):
			return common.Errorf(common.Conflict, "database id, table id, name and expression already exists")
		case strings.Contains(err.Error(), "idx_vw_unique_database_id_name"):
			return common.Errorf(common.Conflict, "database id and name already exists")
		case strings.Contains(err.Error(), "idx_data_source_unique_database_id_name"):
			return common.Errorf(common.Conflict, "data source name already exists")
		case strings.Contains(err.Error(), "idx_backup_unique_database_id_name"):
			return common.Errorf(common.Conflict, "backup name already exists")
		case strings.Contains(err.Error(), "idx_backup_setting_unique_database_id"):
			return common.Errorf(common.Conflict, "database id already exists")
		case strings.Contains(err.Error(), "idx_bookmark_unique_creator_id_link"):
			return common.Errorf(common.Conflict, "bookmark already exists")
		case strings.Contains(err.Error(), "idx_repository_unique_project_id"):
			return common.Errorf(common.Conflict, "project has already linked repository")
		case strings.Contains(err.Error(), "idx_repository_unique_webhook_endpoint_id"):
			return common.Errorf(common.Conflict, "webhook endpoint already exists")
		case strings.Contains(err.Error(), "idx_label_key_unique_key"):
			return common.Errorf(common.Conflict, "label key already exists")
		case strings.Contains(err.Error(), "idx_label_value_unique_key_value"):
			return common.Errorf(common.Conflict, "label key value already exists")
		case strings.Contains(err.Error(), "idx_db_label_unique_database_id_key"):
			return common.Errorf(common.Conflict, "database id and key already exists")
		case strings.Contains(err.Error(), "idx_deployment_config_unique_project_id"):
			return common.Errorf(common.Conflict, "project deployment configuration already exists")
		case strings.Contains(err.Error(), "issue_subscriber_pkey"):
			return common.Errorf(common.Conflict, "issue subscriber already exists")
		}
	}
	return err
}
