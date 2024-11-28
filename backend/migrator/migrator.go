// Package migrator handles store schema migration.
package migrator

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/blang/semver/v4"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/state"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	dbdriver "github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/store/model"
	"github.com/bytebase/bytebase/backend/utils"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

//go:embed migration
var migrationFS embed.FS

// MigrateSchema migrates the schema for metadata database.
func MigrateSchema(ctx context.Context, storeDB *store.DB, storeInstance *store.Store, pgBinDir, serverVersion string, mode common.ReleaseMode) (*semver.Version, error) {
	metadataDriver, err := dbdriver.Open(
		ctx,
		storepb.Engine_POSTGRES,
		dbdriver.DriverConfig{DbBinDir: pgBinDir},
		storeDB.ConnCfg,
	)
	if err != nil {
		return nil, err
	}
	defer metadataDriver.Close(ctx)

	if err := backfillSchemaObjectOwner(ctx, metadataDriver); err != nil {
		return nil, err
	}

	// Calculate prod cutoffSchemaVersion.
	cutoffSchemaVersion, err := getProdCutoffVersion()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get cutoff version")
	}
	slog.Info(fmt.Sprintf("The prod cutoff schema version: %s", cutoffSchemaVersion))
	if err := initializeSchema(ctx, storeInstance, metadataDriver, cutoffSchemaVersion, serverVersion); err != nil {
		return nil, err
	}
	if _, err := metadataDriver.GetDB().ExecContext(ctx, "ALTER TABLE instance_change_history ADD COLUMN IF NOT EXISTS project_id INTEGER REFERENCES project (id);"); err != nil {
		return nil, err
	}

	verBefore, err := getLatestVersion(ctx, storeInstance)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get current schema version")
	}

	migrated, err := migrate(ctx, storeInstance, metadataDriver, cutoffSchemaVersion, verBefore, mode, serverVersion, storeDB.ConnCfg.Database)
	if err != nil {
		return nil, errors.Wrap(err, "failed to migrate")
	}
	if migrated {
		if err := backfillOracleSchema(ctx, storeInstance); err != nil {
			return nil, errors.Wrap(err, "failed to backfill oracle schema")
		}
	}

	verAfter, err := getLatestVersion(ctx, storeInstance)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get current schema version")
	}
	slog.Info(fmt.Sprintf("Current schema version after migration: %s", verAfter))

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
	slog.Info("The database schema has not been setup.")

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

	version := model.Version{Semantic: true, Version: cutoffSchemaVersion.String(), Suffix: time.Now().Format("20060102150405")}
	// Set role to database owner so that the schema owner and database owner are consistent.
	owner, err := getCurrentDatabaseOwner(ctx, metadataDriver)
	if err != nil {
		return err
	}
	conn, err := metadataDriver.GetDB().Conn(ctx)
	if err != nil {
		return err
	}
	if _, err := conn.ExecContext(ctx, fmt.Sprintf("SET ROLE '%s'", owner)); err != nil {
		return err
	}
	if _, err := conn.ExecContext(ctx, stmt); err != nil {
		return err
	}
	if err := storeInstance.CreateInstanceChangeHistoryForMigrator(ctx, &store.InstanceChangeHistoryMessage{
		CreatorID:      api.SystemBotID,
		InstanceUID:    nil,
		DatabaseUID:    nil,
		ProjectUID:     nil,
		IssueUID:       nil,
		ReleaseVersion: serverVersion,
		// Sequence starts from 1.
		Sequence:            1,
		Source:              dbdriver.LIBRARY,
		Type:                dbdriver.Migrate,
		Status:              dbdriver.Done,
		Version:             version,
		Description:         fmt.Sprintf("Initial migration version %s server version %s with file %s.", cutoffSchemaVersion, serverVersion, latestSchemaPath),
		Statement:           stmt,
		Schema:              stmt,
		SchemaPrev:          "",
		ExecutionDurationNs: 0,
		Payload:             nil,
	}); err != nil {
		return err
	}
	slog.Info(fmt.Sprintf("Completed database initial migration with version %s.", cutoffSchemaVersion))
	return nil
}

// getCurrentDatabaseOwner gets the role of the current database.
func getCurrentDatabaseOwner(ctx context.Context, metadataDriver dbdriver.Driver) (string, error) {
	const query = `
		SELECT
			u.rolname
		FROM
			pg_roles AS u JOIN pg_database AS d ON (d.datdba = u.oid)
		WHERE
			d.datname = current_database();
		`
	var owner string
	if err := metadataDriver.GetDB().QueryRowContext(ctx, query).Scan(&owner); err != nil {
		return "", err
	}
	return owner, nil
}

func getCurrentUser(ctx context.Context, metadataDriver dbdriver.Driver) (string, error) {
	row := metadataDriver.GetDB().QueryRowContext(ctx, "SELECT current_user;")
	var user string
	if err := row.Scan(&user); err != nil {
		return "", err
	}
	return user, nil
}

func backfillSchemaObjectOwner(ctx context.Context, metadataDriver dbdriver.Driver) error {
	currentUser, err := getCurrentUser(ctx, metadataDriver)
	if err != nil {
		return err
	}
	databaseOwner, err := getCurrentDatabaseOwner(ctx, metadataDriver)
	if err != nil {
		return err
	}
	if currentUser == databaseOwner {
		return nil
	}
	if _, err := metadataDriver.GetDB().ExecContext(ctx, fmt.Sprintf("reassign owned by %s to %s;", currentUser, databaseOwner)); err != nil {
		return err
	}
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
			slog.Warn(fmt.Sprintf("Found %s migration history", h.Status),
				slog.String("type", string(h.Type)),
				slog.String("version", h.Version.Version),
				slog.String("description", h.Description),
				slog.String("statement", stmt),
			)
			continue
		}
		v, err := semver.Make(h.Version.Version)
		if err != nil {
			return semver.Version{}, errors.Wrapf(err, "invalid version %q", h.Version.Version)
		}
		return v, nil
	}

	return semver.Version{}, errors.Errorf("failed to find a successful migration history to determine the schema version")
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
func migrate(ctx context.Context, storeInstance *store.Store, metadataDriver dbdriver.Driver, cutoffSchemaVersion, curVer semver.Version, mode common.ReleaseMode, serverVersion, databaseName string) (bool, error) {
	slog.Info("Apply database migration if needed...")
	slog.Info(fmt.Sprintf("Current schema version before migration: %s", curVer))

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
			return false, err
		}
		histories = h
	}

	// Apply migrations if needed.
	retVersion := curVer
	names, err := fs.Glob(migrationFS, fmt.Sprintf("migration/%s/*", common.ReleaseModeProd))
	if err != nil {
		return false, err
	}

	minorVersions, err := getMinorMigrationVersions(names, curVer)
	if err != nil {
		return false, err
	}

	for _, minorVersion := range minorVersions {
		slog.Info(fmt.Sprintf("Starting minor version migration cycle from %s ...", minorVersion))
		names, err := fs.Glob(migrationFS, fmt.Sprintf("migration/%s/%d.%d/*.sql", common.ReleaseModeProd, minorVersion.Major, minorVersion.Minor))
		if err != nil {
			return false, err
		}
		patchVersions, err := getPatchVersions(minorVersion, curVer, names)
		if err != nil {
			return false, err
		}

		for _, pv := range patchVersions {
			buf, err := fs.ReadFile(migrationFS, pv.filename)
			if err != nil {
				return false, errors.Wrapf(err, "failed to read migration file %q", pv.filename)
			}
			// This happens when a migration file is moved from dev to release and we should not reapply the migration.
			// For example,
			//   before - prod: 1.2; dev: 123.sql, something else.
			//   after - prod 1.3 with 123.sql; dev: something else.
			// When dev starts, it will try to apply version 1.3 including 123.sql. If we don't skip, the same statement will be re-applied and most likely to fail.
			if mode == common.ReleaseModeDev && migrationExists(string(buf), histories) {
				slog.Info(fmt.Sprintf("Skip migrating migration file %s that's already migrated.", pv.filename))
				continue
			}
			slog.Info(fmt.Sprintf("Migrating %s...", pv.version))
			mi := &dbdriver.MigrationInfo{
				InstanceID:     nil,
				CreatorID:      api.SystemBotID,
				ReleaseVersion: serverVersion,
				Version:        model.Version{Semantic: true, Version: pv.version.String(), Suffix: time.Now().Format("20060102150405")},
				Namespace:      databaseName,
				Database:       databaseName,
				Environment:    "", /* unused in execute migration */
				Source:         dbdriver.LIBRARY,
				Type:           dbdriver.Migrate,
				Description:    fmt.Sprintf("Migrate version %s server version %s with files %s.", pv.version, serverVersion, pv.filename),
			}
			if _, _, err := executeMigrationDefault(ctx, ctx, storeInstance, nil, 0, metadataDriver, mi, string(buf), nil, dbdriver.ExecuteOptions{}); err != nil {
				return false, err
			}
			retVersion = pv.version
		}
		if retVersion.EQ(curVer) {
			slog.Info(fmt.Sprintf("Database schema is at version %s; nothing to migrate.", curVer))
		} else {
			slog.Info(fmt.Sprintf("Completed database migration from version %s to %s.", curVer, retVersion))
		}
	}

	if mode == common.ReleaseModeDev {
		if err := migrateDev(ctx, storeInstance, metadataDriver, serverVersion, databaseName, cutoffSchemaVersion, histories); err != nil {
			return false, errors.Wrapf(err, "failed to migrate dev schema")
		}
	}
	return len(minorVersions) > 0, nil
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
			slog.Info(fmt.Sprintf("Skip migrating dev migration file %s that's already migrated.", m.filename))
		} else {
			migrations = append(migrations, m)
		}
	}
	if len(migrations) == 0 {
		slog.Info("Skip dev mode migration; no new version.")
		return nil
	}

	for _, m := range migrations {
		slog.Info(fmt.Sprintf("Migrating dev %s...", m.filename))
		// We expect to use semantic versioning for dev environment too because getLatestVersion() always expect to get the latest version in semantic format.
		mi := &dbdriver.MigrationInfo{
			InstanceID:     nil,
			CreatorID:      api.SystemBotID,
			ReleaseVersion: serverVersion,
			Version:        model.Version{Semantic: true, Version: cutoffSchemaVersion.String(), Suffix: fmt.Sprintf("dev%s", m.version)},
			Namespace:      databaseName,
			Database:       databaseName,
			Environment:    "", /* unused in execute migration */
			Source:         dbdriver.LIBRARY,
			Type:           dbdriver.Migrate,
			Description:    fmt.Sprintf("Migrate version %s server version %s with files %s.", m.version, serverVersion, m.filename),
		}
		if _, _, err := executeMigrationDefault(ctx, ctx, storeInstance, nil, 0, metadataDriver, mi, m.statement, nil, dbdriver.ExecuteOptions{}); err != nil {
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
			slog.Debug(fmt.Sprintf("Skip migration %s; the current schema version %s is higher.", version, currentVersion))
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

func backfillOracleSchema(ctx context.Context, stores *store.Store) error {
	engine := storepb.Engine_ORACLE
	databases, err := stores.ListDatabases(ctx, &store.FindDatabaseMessage{Engine: &engine})
	if err != nil {
		return err
	}
	for _, database := range databases {
		dbSchema, err := stores.GetDBSchema(ctx, database.UID)
		if err != nil {
			return err
		}
		if dbSchema == nil {
			continue
		}
		dbMetadata := dbSchema.GetMetadata()
		if dbMetadata == nil {
			continue
		}
		for _, schema := range dbMetadata.Schemas {
			schema.Name = ""
		}
		if err := stores.UpsertDBSchema(ctx, database.UID, dbSchema, api.SystemBotID); err != nil {
			return err
		}
	}
	return nil
}

// executeMigrationDefault executes migration.
func executeMigrationDefault(ctx context.Context, driverCtx context.Context, store *store.Store, _ *state.State, taskRunUID int, driver dbdriver.Driver, mi *dbdriver.MigrationInfo, statement string, sheetID *int, opts dbdriver.ExecuteOptions) (migrationHistoryID string, updatedSchema string, resErr error) {
	execFunc := func(ctx context.Context, execStatement string) error {
		if _, err := driver.Execute(ctx, execStatement, opts); err != nil {
			return err
		}
		return nil
	}
	return ExecuteMigrationWithFunc(ctx, driverCtx, store, taskRunUID, driver, mi, statement, sheetID, execFunc, opts)
}

// ExecuteMigrationWithFunc executes the migration with custom migration function.
func ExecuteMigrationWithFunc(ctx context.Context, driverCtx context.Context, s *store.Store, _ int, driver dbdriver.Driver, m *dbdriver.MigrationInfo, statement string, sheetID *int, execFunc func(ctx context.Context, execStatement string) error, opts dbdriver.ExecuteOptions) (migrationHistoryID string, updatedSchema string, resErr error) {
	var prevSchemaBuf bytes.Buffer
	if m.Type.NeedDump() {
		opts.LogSchemaDumpStart()
		// Don't record schema if the database hasn't existed yet or is schemaless, e.g. MongoDB.
		// For baseline migration, we also record the live schema to detect the schema drift.
		// See https://bytebase.com/blog/what-is-database-schema-drift
		if err := driver.Dump(ctx, &prevSchemaBuf, nil); err != nil {
			opts.LogSchemaDumpEnd(err.Error())
			return "", "", err
		}
		opts.LogSchemaDumpEnd("")
	}

	insertedID, err := BeginMigration(ctx, s, m, prevSchemaBuf.String(), statement, sheetID)
	if err != nil {
		return "", "", errors.Wrapf(err, "failed to begin migration")
	}

	startedNs := time.Now().UnixNano()

	defer func() {
		if err := EndMigration(ctx, s, startedNs, insertedID, updatedSchema, prevSchemaBuf.String(), sheetID, resErr == nil /* isDone */); err != nil {
			slog.Error("Failed to update migration history record",
				log.BBError(err),
				slog.String("migration_id", migrationHistoryID),
			)
		}
	}()

	// Phase 3 - Executing migration
	// Branch migration type always has empty sql.
	// Baseline migration type could has non-empty sql but will not execute.
	// https://github.com/bytebase/bytebase/issues/394
	doMigrate := true
	if statement == "" || m.Type == dbdriver.Baseline {
		doMigrate = false
	}
	if doMigrate {
		renderedStatement := statement
		// The m.DatabaseID is nil means the migration is a instance level migration
		if m.DatabaseID != nil {
			database, err := s.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
				UID: m.DatabaseID,
			})
			if err != nil {
				return "", "", err
			}
			if database == nil {
				return "", "", errors.Errorf("database %d not found", *m.DatabaseID)
			}
			materials := utils.GetSecretMapFromDatabaseMessage(database)
			// To avoid leak the rendered statement, the error message should use the original statement and not the rendered statement.
			renderedStatement = utils.RenderStatement(statement, materials)
		}

		if err := execFunc(driverCtx, renderedStatement); err != nil {
			return "", "", err
		}
	}

	// Phase 4 - Dump the schema after migration
	var afterSchemaBuf bytes.Buffer
	if m.Type.NeedDump() {
		opts.LogSchemaDumpStart()
		if err := driver.Dump(ctx, &afterSchemaBuf, nil); err != nil {
			// We will ignore the dump error if the database is dropped.
			if strings.Contains(err.Error(), "not found") {
				return insertedID, "", nil
			}
			opts.LogSchemaDumpEnd(err.Error())
			return "", "", err
		}
		opts.LogSchemaDumpEnd("")
	}

	return insertedID, afterSchemaBuf.String(), nil
}

// BeginMigration checks before executing migration and inserts a migration history record with pending status.
func BeginMigration(ctx context.Context, stores *store.Store, m *dbdriver.MigrationInfo, prevSchema, statement string, sheetID *int) (string, error) {
	// Phase 1 - Pre-check before executing migration
	// Check if the same migration version has already been applied.
	if list, err := stores.ListInstanceChangeHistoryForMigrator(ctx, &store.FindInstanceChangeHistoryMessage{
		InstanceID: m.InstanceID,
		DatabaseID: m.DatabaseID,
		Version:    &m.Version,
	}); err != nil {
		return "", errors.Wrap(err, "failed to check duplicate version")
	} else if len(list) > 0 {
		migrationHistory := list[0]
		switch migrationHistory.Status {
		case dbdriver.Done:
			err := common.Errorf(common.MigrationAlreadyApplied, "database %q has already applied version %s, hint: the version might be duplicate, please check the version", m.Database, m.Version.Version)
			slog.Debug(err.Error())
			// Force migration
			return migrationHistory.UID, nil
		case dbdriver.Pending:
			err := errors.Errorf("database %q version %s migration is already in progress", m.Database, m.Version.Version)
			slog.Debug(err.Error())
			// For force migration, we will ignore the existing migration history and continue to migration.
			return migrationHistory.UID, nil
		case dbdriver.Failed:
			err := errors.Errorf("database %q version %s migration has failed, please check your database to make sure things are fine and then start a new migration using a new version", m.Database, m.Version.Version)
			slog.Debug(err.Error())
			// For force migration, we will ignore the existing migration history and continue to migration.
			return migrationHistory.UID, nil
		}
	}

	// Phase 2 - Record migration history as PENDING.
	statementRecord, _ := common.TruncateString(statement, common.MaxSheetSize)
	insertedID, err := stores.CreatePendingInstanceChangeHistory(ctx, prevSchema, m, statementRecord, sheetID)
	if err != nil {
		return "", err
	}

	return insertedID, nil
}

// EndMigration updates the migration history record to DONE or FAILED depending on migration is done or not.
func EndMigration(ctx context.Context, storeInstance *store.Store, startedNs int64, insertedID string, updatedSchema, schemaPrev string, sheetID *int, isDone bool) error {
	migrationDurationNs := time.Now().UnixNano() - startedNs
	update := &store.UpdateInstanceChangeHistoryMessage{
		ID:                  insertedID,
		ExecutionDurationNs: &migrationDurationNs,
		// Update the sheet ID just in case it has been updated.
		Sheet: sheetID,
		// Update schemaPrev because we might be re-using a previous change history entry.
		SchemaPrev: &schemaPrev,
	}
	if isDone {
		// Upon success, update the migration history as 'DONE', execution_duration_ns, updated schema.
		status := dbdriver.Done
		update.Status = &status
		update.Schema = &updatedSchema
	} else {
		// Otherwise, update the migration history as 'FAILED', execution_duration.
		status := dbdriver.Failed
		update.Status = &status
	}
	return storeInstance.UpdateInstanceChangeHistory(ctx, update)
}
