package store

import (
	"bytes"
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/blang/semver/v4"
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
	majorSchemaVervion = 1
)

//go:embed migration
var migrationFS embed.FS

//go:embed demo
var demoFS embed.FS

// DB represents the database connection.
type DB struct {
	db *sql.DB

	l *zap.Logger

	// db.connCfg is the connection configuration to a Postgres database.
	// The user has superuser privilege to the database.
	connCfg dbdriver.ConnectionConfig

	// Dir to load demo data
	demoDataDir string

	// If true, database will be opened in readonly mode
	readonly bool

	// Bytebase server release version
	serverVersion string

	// schemaVersion is the version of Bytebase schema.
	schemaVersion semver.Version

	// Returns the current time. Defaults to time.Now().
	// Can be mocked for tests.
	Now func() time.Time
}

// NewDB returns a new instance of DB associated with the given datasource name.
func NewDB(logger *zap.Logger, connCfg dbdriver.ConnectionConfig, demoDataDir string, readonly bool, serverVersion string, schemaVersion semver.Version) *DB {
	db := &DB{
		l:             logger,
		connCfg:       connCfg,
		demoDataDir:   demoDataDir,
		readonly:      readonly,
		Now:           time.Now,
		serverVersion: serverVersion,
		schemaVersion: schemaVersion,
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
		db.l.Info("Database is opened in readonly mode. Skip migration and demo data setup.")
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
	// TODO(d): remove this block once all existing customers all migrated to semantic versioning.
	if _, err := getLatestVersion(ctx, d, databaseName); err != nil {
		// Convert existing record to semantic versioning format.
		if !strings.Contains(err.Error(), "invalid stored version") {
			return err
		}
		db.l.Info("Migrating migration history version storage format to semantic version.")
		db, err := d.GetDbConnection(ctx, "bytebase")
		if err != nil {
			return err
		}
		if _, err := db.ExecContext(ctx, "UPDATE migration_history SET version = '0001.0000.0000-20210113000000' WHERE id = 1 AND version = '10000';"); err != nil {
			return err
		}
		if _, err := db.ExecContext(ctx, "UPDATE migration_history SET version = '0001.0000.0000-20210113000001' WHERE id = 2 AND version = '10001';"); err != nil {
			return err
		}
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

	if err := db.setupDemoData(); err != nil {
		return fmt.Errorf("failed to setup demo data: %w."+
			" It could be Bytebase is running against an old Bytebase schema. If you are developing Bytebase, you can remove pgdata"+
			" directory under the same directory where the bytebase binary resides. and restart again to let"+
			" Bytebase create the latest schema. If you are running in production and don't want to reset the data, you can contact support@bytebase.com for help",
			err)
	}

	return nil
}

// getLatestVersion returns the latest schema version in semantic versioning format.
// If there's no migration history, version will be nil.
func getLatestVersion(ctx context.Context, d dbdriver.Driver, database string) (*semver.Version, error) {
	limit := 1
	history, err := d.FindMigrationHistoryList(ctx, &dbdriver.MigrationHistoryFind{
		Database: &database,
		Limit:    &limit,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get migration history, error: %v", err)
	}
	if len(history) == 0 {
		return nil, nil
	}

	v, err := semver.Make(history[0].Version)
	if err != nil {
		return nil, fmt.Errorf("invalid version %q, error: %v", history[0].Version, err)
	}

	return &v, nil
}

// setupDemoData loads the setupDemoData data for testing
func (db *DB) setupDemoData() error {
	if db.demoDataDir == "" {
		db.l.Debug("Skip setting up demo data. Demo data directory not specified.")
		return nil
	}
	db.l.Info(fmt.Sprintf("Setting up demo data from %q...", db.demoDataDir))
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
			return fmt.Errorf("applyDataFile error: name=%q err=%w", name, err)
		}
	}
	db.l.Info("Completed demo data setup.")
	return nil
}

// applyDataFile runs a single demo data file within a transaction.
func (db *DB) applyDataFile(name string) error {
	db.l.Info(fmt.Sprintf("Applying data file %s...", name))
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
	latestSchemaFile = "latest.sql"
	latestDataFile   = "latest_data.sql"
)

// migrate sets up migration tracking and executes pending migration files.
//
// Migration files are embedded in the migration folder and are executed
// in lexicographical order.
//
// We prepend each migration file with version = xxx; Each migration
// file run in a transaction to prevent partial migrations.
func (db *DB) migrate(ctx context.Context, d dbdriver.Driver, curVer *semver.Version, databaseName string) error {
	db.l.Info("Apply database migration if needed...")
	db.l.Info(fmt.Sprintf("The release cutoff schema version: %s", db.schemaVersion))
	if curVer == nil {
		db.l.Info("The database schema has not been setup.")
	} else {
		db.l.Info(fmt.Sprintf("Current schema version before migration: %s", curVer))
		major := (*curVer).Major
		if major != majorSchemaVervion {
			return fmt.Errorf("current major schema version %d is different from the major schema version %d this release %s expects", major, majorSchemaVervion, db.serverVersion)
		}
	}

	// Initial schema setup.
	if curVer == nil {
		latestSchemaPath := fmt.Sprintf("migration/%s/%s", db.schemaVersion.String(), latestSchemaFile)
		buf, err := migrationFS.ReadFile(latestSchemaPath)
		if err != nil {
			return fmt.Errorf("failed to read latest schema %q, error %w", latestSchemaPath, err)
		}
		latestDataPath := fmt.Sprintf("migration/%s/%s", db.schemaVersion.String(), latestDataFile)
		dataBuf, err := migrationFS.ReadFile(latestDataPath)
		if err != nil {
			return fmt.Errorf("failed to read latest data %q, error %w", latestSchemaPath, err)
		}
		// We will create the database together with initial schema and data migration.
		stmt := fmt.Sprintf("CREATE DATABASE %s;\n\\connect \"%s\";\n%s\n%s", databaseName, databaseName, buf, dataBuf)
		if _, _, err := d.ExecuteMigration(
			ctx,
			&dbdriver.MigrationInfo{
				ReleaseVersion:        db.serverVersion,
				UseSemanticVersion:    true,
				Version:               db.schemaVersion.String(),
				SemanticVersionSuffix: time.Now().Format("20060102150405"),
				Namespace:             databaseName,
				Database:              databaseName,
				Environment:           "", /* unused in execute migration */
				Source:                dbdriver.LIBRARY,
				Type:                  dbdriver.Migrate,
				Description:           fmt.Sprintf("Initial migration version %s server version %s with file %s.", db.schemaVersion, db.serverVersion, latestSchemaPath),
				CreateDatabase:        true,
			},
			stmt,
		); err != nil {
			return fmt.Errorf("failed to migrate initial schema version %q, error: %v", latestSchemaPath, err)
		}
		db.l.Info("Completed database initial migration.")
		return nil
	}

	// Apply migrations
	versionNames, err := fs.Glob(migrationFS, "migration/*")
	if err != nil {
		return err
	}
	var versions []semver.Version
	for _, name := range versionNames {
		v, err := semver.Make(strings.TrimPrefix(name, "migration/"))
		if err != nil {
			return fmt.Errorf("invalid migration file path %q, error %w", name, err)
		}
		versions = append(versions, v)
	}
	// Sort the migration semantic version in ascending order.
	sort.Slice(versions, func(i, j int) bool {
		return versions[i].LT(versions[j])
	})

	migrateVersions, messages := getMigrationVersions(versions, db.schemaVersion, *curVer)
	for _, message := range messages {
		db.l.Info(message)
	}

	for _, version := range migrateVersions {
		// Migrate migration files.
		db.l.Info(fmt.Sprintf("Migrating %s...", version))
		names, err := fs.Glob(migrationFS, fmt.Sprintf("migration/%s/*.sql", version))
		if err != nil {
			return err
		}
		var stmtBuf bytes.Buffer
		var baseNames []string
		for _, name := range names {
			// Skip the latest sql file.
			baseName := filepath.Base(name)
			if baseName == latestSchemaFile {
				continue
			}
			if baseName == latestDataFile {
				continue
			}
			baseNames = append(baseNames, baseName)
			buf, err := fs.ReadFile(migrationFS, name)
			if err != nil {
				return fmt.Errorf("failed to read migration file %q, error %w", name, err)
			}
			if _, err := stmtBuf.Write(buf); err != nil {
				return fmt.Errorf("failed to write buffer for migration file %q, error %w", name, err)
			}
			if _, err := stmtBuf.WriteString("\n\n"); err != nil {
				return fmt.Errorf("failed to write newline buffer for migration file %q, error %w", name, err)
			}
			db.l.Debug(fmt.Sprintf("Reading migration file %q.", name))
		}
		if _, _, err := d.ExecuteMigration(
			ctx,
			&dbdriver.MigrationInfo{
				ReleaseVersion:        db.serverVersion,
				UseSemanticVersion:    true,
				Version:               version.String(),
				SemanticVersionSuffix: time.Now().Format("20060102150405"),
				Namespace:             databaseName,
				Database:              databaseName,
				Environment:           "", /* unused in execute migration */
				Source:                dbdriver.LIBRARY,
				Type:                  dbdriver.Migrate,
				Description:           fmt.Sprintf("Migrate version %s server version %s with files %s.", version, db.serverVersion, strings.Join(baseNames, ", ")),
			},
			stmtBuf.String(),
		); err != nil {
			return fmt.Errorf("failed to migrate schema version %q, error: %v", version, err)
		}
	}
	db.l.Info("Completed database migration.")
	return nil
}

func getMigrationVersions(versions []semver.Version, releaseCutSchemaVersion, currentVersion semver.Version) ([]semver.Version, []string) {
	var migrateVersions []semver.Version
	var messages []string
	for _, version := range versions {
		// If the migration version is greater than the schema version this build expects, we will skip the migration.
		if version.GT(releaseCutSchemaVersion) {
			messages = append(messages, fmt.Sprintf("Skip this migration: %s; the corresponding migration version %s is bigger than maximum schema version %s.", version, version, releaseCutSchemaVersion))
			continue
		}
		// If the migration version is less than or equal to the current version, we will skip the migration since it's already applied.
		if version.LE(currentVersion) {
			messages = append(messages, fmt.Sprintf("Skip this migration: %s; the current schema version %s is higher.", version, currentVersion))
			continue
		}
		migrateVersions = append(migrateVersions, version)
	}
	return migrateVersions, messages
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

	if strings.Contains(err.Error(), "unique constraint") {
		switch {
		case strings.Contains(err.Error(), "idx_principal_unique_email"):
			return common.Errorf(common.Conflict, fmt.Errorf("email already exists"))
		case strings.Contains(err.Error(), "idx_setting_unique_name"):
			return common.Errorf(common.Conflict, fmt.Errorf("setting name already exists"))
		case strings.Contains(err.Error(), "idx_member_unique_principal_id"):
			return common.Errorf(common.Conflict, fmt.Errorf("member already exists"))
		case strings.Contains(err.Error(), "idx_environment_unique_name"):
			return common.Errorf(common.Conflict, fmt.Errorf("environment name already exists"))
		case strings.Contains(err.Error(), "idx_policy_unique_environment_id_type"):
			return common.Errorf(common.Conflict, fmt.Errorf("policy environment and type already exists"))
		case strings.Contains(err.Error(), "idx_project_unique_key"):
			return common.Errorf(common.Conflict, fmt.Errorf("project key already exists"))
		case strings.Contains(err.Error(), "idx_project_member_unique_project_id_role_provider_principal_id"):
			return common.Errorf(common.Conflict, fmt.Errorf("project member already exists"))
		case strings.Contains(err.Error(), "idx_project_webhook_unique_project_id_url"):
			return common.Errorf(common.Conflict, fmt.Errorf("webhook url already exists"))
		case strings.Contains(err.Error(), "idx_instance_user_unique_instance_id_name"):
			return common.Errorf(common.Conflict, fmt.Errorf("instance id and name already exists"))
		case strings.Contains(err.Error(), "idx_db_unique_instance_id_name"):
			return common.Errorf(common.Conflict, fmt.Errorf("database name already exists"))
		case strings.Contains(err.Error(), "idx_tbl_unique_database_id_name"):
			return common.Errorf(common.Conflict, fmt.Errorf("database id and name already exists"))
		case strings.Contains(err.Error(), "idx_col_unique_database_id_table_id_name"):
			return common.Errorf(common.Conflict, fmt.Errorf("database id, table id and name already exists"))
		case strings.Contains(err.Error(), "idx_idx_unique_database_id_table_id_name_expression"):
			return common.Errorf(common.Conflict, fmt.Errorf("database id, table id, name and expression already exists"))
		case strings.Contains(err.Error(), "idx_vw_unique_database_id_name"):
			return common.Errorf(common.Conflict, fmt.Errorf("database id and name already exists"))
		case strings.Contains(err.Error(), "idx_data_source_unique_database_id_name"):
			return common.Errorf(common.Conflict, fmt.Errorf("data source name already exists"))
		case strings.Contains(err.Error(), "idx_backup_unique_database_id_name"):
			return common.Errorf(common.Conflict, fmt.Errorf("backup name already exists"))
		case strings.Contains(err.Error(), "idx_backup_setting_unique_database_id"):
			return common.Errorf(common.Conflict, fmt.Errorf("database id already exists"))
		case strings.Contains(err.Error(), "idx_bookmark_unique_creator_id_link"):
			return common.Errorf(common.Conflict, fmt.Errorf("bookmark already exists"))
		case strings.Contains(err.Error(), "idx_repository_unique_project_id"):
			return common.Errorf(common.Conflict, fmt.Errorf("project has already linked repository"))
		case strings.Contains(err.Error(), "idx_repository_unique_webhook_endpoint_id"):
			return common.Errorf(common.Conflict, fmt.Errorf("webhook endpoint already exists"))
		case strings.Contains(err.Error(), "idx_label_key_unique_key"):
			return common.Errorf(common.Conflict, fmt.Errorf("label key already exists"))
		case strings.Contains(err.Error(), "idx_label_value_unique_key_value"):
			return common.Errorf(common.Conflict, fmt.Errorf("label key value already exists"))
		case strings.Contains(err.Error(), "idx_db_label_unique_database_id_key"):
			return common.Errorf(common.Conflict, fmt.Errorf("database id and key already exists"))
		case strings.Contains(err.Error(), "idx_deployment_config_unique_project_id"):
			return common.Errorf(common.Conflict, fmt.Errorf("project deployment configuration already exists"))
		case strings.Contains(err.Error(), "issue_subscriber_pkey"):
			return common.Errorf(common.Conflict, fmt.Errorf("issue subscriber already exists"))
		}
	}
	return err
}
