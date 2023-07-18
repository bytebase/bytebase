// Package migrator handles store schema migration.
package migrator

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

	"github.com/blang/semver/v4"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	dbdriver "github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
)

//go:embed migration
var migrationFS embed.FS

//go:embed demo
var demoFS embed.FS

// MigrateSchema migrates the schema for metadata database.
func MigrateSchema(ctx context.Context, storeDB *store.DB, strictUseDb bool, pgBinDir, demoName, serverVersion string, mode common.ReleaseMode) (*semver.Version, error) {
	databaseName := storeDB.ConnCfg.Database
	if !strictUseDb {
		// The database storing metadata is the same as user name.
		databaseName = storeDB.ConnCfg.Username
	}
	metadataConnConfig := storeDB.ConnCfg
	if !strictUseDb {
		metadataConnConfig.Database = databaseName
	}
	metadataDriver, err := dbdriver.Open(
		ctx,
		dbdriver.Postgres,
		dbdriver.DriverConfig{DbBinDir: pgBinDir},
		metadataConnConfig,
		dbdriver.ConnectionContext{},
	)
	if err != nil {
		return nil, err
	}
	storeInstance := store.New(storeDB)
	// Calculate prod cutoffSchemaVersion.
	cutoffSchemaVersion, err := getProdCutoffVersion()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get cutoff version")
	}
	log.Info(fmt.Sprintf("The prod cutoff schema version: %s", cutoffSchemaVersion))
	if err := initializeSchema(ctx, storeInstance, metadataDriver, cutoffSchemaVersion, serverVersion); err != nil {
		return nil, err
	}

	bytebaseConnConfig := storeDB.ConnCfg
	if !strictUseDb {
		// BytebaseDatabase was the database installed in the controlled database server.
		bytebaseConnConfig.Database = "bytebase"
	}
	bytebaseDriver, err := dbdriver.Open(
		ctx,
		dbdriver.Postgres,
		dbdriver.DriverConfig{DbBinDir: pgBinDir},
		bytebaseConnConfig,
		dbdriver.ConnectionContext{},
	)
	if err != nil {
		return nil, err
	}
	defer bytebaseDriver.Close(ctx)

	verBefore, err := getLatestVersion(ctx, storeInstance)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get current schema version")
	}

	if err := migrate(ctx, storeInstance, metadataDriver, cutoffSchemaVersion, verBefore, mode, serverVersion, databaseName); err != nil {
		return nil, errors.Wrap(err, "failed to migrate")
	}

	verAfter, err := getLatestVersion(ctx, storeInstance)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get current schema version")
	}
	log.Info(fmt.Sprintf("Current schema version after migration: %s", verAfter))

	if err := setupDemoData(demoName, metadataDriver.GetDB()); err != nil {
		return nil, errors.Wrapf(err, "failed to setup demo data."+
			" It could be Bytebase is running against an old Bytebase schema. If you are developing Bytebase, you can remove pgdata"+
			" directory under the same directory where the Bytebase binary resides. and restart again to let"+
			" Bytebase create the latest schema. If you are running in production and don't want to reset the data, you can contact support@bytebase.com for help")
	}

	return &verAfter, nil
}

func initializeSchema(ctx context.Context, storeInstance *store.Store, metadataDriver dbdriver.Driver, cutoffSchemaVersion semver.Version, serverVersion string) error {
	// We use environment table to determine whether we've initialized the schema.
	var exists bool
	if err := metadataDriver.GetDB().QueryRowContext(ctx,
		`SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'environment')`,
	).Scan(&exists); err != nil {
		return err
	}
	if exists {
		return nil
	}
	log.Info("The database schema has not been setup.")

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

	// We will create the database together with initial schema and data migration.
	stmt := fmt.Sprintf("%s\n%s", buf, dataBuf)

	storedVersion, err := util.ToStoredVersion(true /* UseSemanticVersion */, cutoffSchemaVersion.String(), common.DefaultMigrationVersion())
	if err != nil {
		return err
	}
	if _, err := metadataDriver.GetDB().ExecContext(ctx, stmt); err != nil {
		return err
	}
	if err := storeInstance.CreateInstanceChangeHistory(ctx, &store.InstanceChangeHistoryMessage{
		CreatorID:      api.SystemBotID,
		InstanceUID:    nil,
		DatabaseUID:    nil,
		IssueUID:       nil,
		ReleaseVersion: serverVersion,
		// Sequence starts from 1.
		Sequence:            1,
		Source:              dbdriver.LIBRARY,
		Type:                dbdriver.Migrate,
		Status:              dbdriver.Done,
		Version:             storedVersion,
		Description:         fmt.Sprintf("Initial migration version %s server version %s with file %s.", cutoffSchemaVersion, serverVersion, latestSchemaPath),
		Statement:           stmt,
		Schema:              stmt,
		SchemaPrev:          "",
		ExecutionDurationNs: 0,
		Payload:             nil,
	}); err != nil {
		return err
	}
	log.Info(fmt.Sprintf("Completed database initial migration with version %s.", cutoffSchemaVersion))
	return nil
}

// getLatestVersion returns the latest schema version in semantic versioning format.
// We expect our own migration history to use semantic versions.
// If there's no migration history, version will be nil.
func getLatestVersion(ctx context.Context, storeInstance *store.Store) (semver.Version, error) {
	// We look back the past migration history records and return the latest successful (DONE) migration version.
	histories, err := storeInstance.ListInstanceChangeHistoryForMigrator(ctx, &store.FindInstanceChangeHistoryMessage{
		// Metadata database has instanceID nil;
		InstanceID: nil,
		ShowFull:   true,
	})
	if err != nil {
		return semver.Version{}, errors.Wrap(err, "failed to get migration history")
	}
	if len(histories) == 0 {
		return semver.Version{}, errors.Errorf("migration history should exist for metadata database")
	}

	for _, h := range histories {
		if h.Status != dbdriver.Done {
			stmt := h.Statement
			// Only print out first 200 chars.
			if len(stmt) > 200 {
				stmt = stmt[:200]
			}
			// Non-success migration history record is an anomaly, in the case where the actual
			// migration has been applied, the followup migration will likely fail because the
			// schema has already been applied. Thus emitting a warning here will assist debugging.
			log.Warn(fmt.Sprintf("Found %s migration history", h.Status),
				zap.String("type", string(h.Type)),
				zap.String("version", h.Version),
				zap.String("description", h.Description),
				zap.String("statement", stmt),
			)
			continue
		}
		_, version, _, err := util.FromStoredVersion(h.Version)
		if err != nil {
			return semver.Version{}, err
		}
		v, err := semver.Make(version)
		if err != nil {
			return semver.Version{}, errors.Wrapf(err, "invalid version %q", h.Version)
		}
		return v, nil
	}

	return semver.Version{}, errors.Errorf("failed to find a successful migration history to determine the schema version")
}

// setupDemoData loads the demo data.
func setupDemoData(demoName string, db *sql.DB) error {
	if demoName == "" {
		log.Debug("Skip setting up demo data. Demo not specified.")
		return nil
	}

	log.Info(fmt.Sprintf("Setting up demo %q...", demoName))

	// Reset existing demo data.
	if err := applyDataFile("demo/reset.sql", db); err != nil {
		return errors.Wrapf(err, "Failed to reset demo data")
	}

	names, err := fs.Glob(demoFS, fmt.Sprintf("demo/%s/*.sql", demoName))
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
		if err := applyDataFile(name, db); err != nil {
			return errors.Wrapf(err, "Failed to load demo data: %q", name)
		}
	}
	log.Info("Completed demo data setup.")
	return nil
}

// applyDataFile runs a single demo data file within a transaction.
func applyDataFile(name string, db *sql.DB) error {
	log.Info(fmt.Sprintf("Applying data file %s...", name))
	tx, err := db.Begin()
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
func migrate(ctx context.Context, storeInstance *store.Store, metadataDriver dbdriver.Driver, cutoffSchemaVersion, curVer semver.Version, mode common.ReleaseMode, serverVersion, databaseName string) error {
	log.Info("Apply database migration if needed...")
	log.Info(fmt.Sprintf("Current schema version before migration: %s", curVer))

	var histories []*store.InstanceChangeHistoryMessage
	// Because dev migrations don't use semantic versioning, we have to look at all migration history to
	// figure out whether the migration statement has already been applied.
	if mode == common.ReleaseModeDev {
		h, err := storeInstance.ListInstanceChangeHistoryForMigrator(ctx, &store.FindInstanceChangeHistoryMessage{
			// Metadata database has instanceID nil;
			InstanceID: nil,
			ShowFull:   true,
		})
		if err != nil {
			return err
		}
		histories = h
	}

	// Apply migrations if needed.
	retVersion := curVer
	names, err := fs.Glob(migrationFS, fmt.Sprintf("migration/%s/*", common.ReleaseModeProd))
	if err != nil {
		return err
	}

	minorVersions, err := getMinorMigrationVersions(names, curVer)
	if err != nil {
		return err
	}

	for _, minorVersion := range minorVersions {
		log.Info(fmt.Sprintf("Starting minor version migration cycle from %s ...", minorVersion))
		names, err := fs.Glob(migrationFS, fmt.Sprintf("migration/%s/%d.%d/*.sql", common.ReleaseModeProd, minorVersion.Major, minorVersion.Minor))
		if err != nil {
			return err
		}
		patchVersions, err := getPatchVersions(minorVersion, curVer, names)
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
			mi := &dbdriver.MigrationInfo{
				InstanceID:            nil,
				CreatorID:             api.SystemBotID,
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
			}
			if _, _, err := utils.ExecuteMigrationDefault(ctx, storeInstance, metadataDriver, mi, string(buf), nil, dbdriver.ExecuteOptions{}); err != nil {
				return err
			}
			retVersion = pv.version
		}
		if retVersion.EQ(curVer) {
			log.Info(fmt.Sprintf("Database schema is at version %s; nothing to migrate.", curVer))
		} else {
			log.Info(fmt.Sprintf("Completed database migration from version %s to %s.", curVer, retVersion))
		}
	}

	if mode == common.ReleaseModeDev {
		if err := migrateDev(ctx, storeInstance, metadataDriver, serverVersion, databaseName, cutoffSchemaVersion, histories); err != nil {
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

func migrateDev(ctx context.Context, storeInstance *store.Store, metadataDriver dbdriver.Driver, serverVersion, databaseName string, cutoffSchemaVersion semver.Version, histories []*store.InstanceChangeHistoryMessage) error {
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
		mi := &dbdriver.MigrationInfo{
			InstanceID:            nil,
			CreatorID:             api.SystemBotID,
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
		}
		if _, _, err := utils.ExecuteMigrationDefault(ctx, storeInstance, metadataDriver, mi, m.statement, nil, dbdriver.ExecuteOptions{}); err != nil {
			return err
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

		parts := strings.Split(baseName, "##")
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

func migrationExists(statement string, histories []*store.InstanceChangeHistoryMessage) bool {
	for _, history := range histories {
		if history.Statement == statement && history.Status == dbdriver.Done {
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
		parts := strings.Split(baseName, "##")
		if len(parts) != 2 {
			return nil, errors.Errorf("migration filename %q should include '##'", name)
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
func getMinorMigrationVersions(names []string, currentVersion semver.Version) ([]semver.Version, error) {
	versions, err := getMinorVersions(names)
	if err != nil {
		return nil, err
	}

	// We should still include the version with the same minor version with currentVersion in case we have missed some patches.
	currentVersion.Patch = 0

	var migrateVersions []semver.Version
	for _, version := range versions {
		// If the migration version is less than to the current version, we will skip the migration since it's already applied.
		// We should still double check the current version in case there's any patch needed.
		if version.LT(currentVersion) {
			log.Debug(fmt.Sprintf("Skip migration %s; the current schema version %s is higher.", version, currentVersion))
			continue
		}
		migrateVersions = append(migrateVersions, version)
	}
	return migrateVersions, nil
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
