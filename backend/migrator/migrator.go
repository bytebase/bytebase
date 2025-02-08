// Package migrator handles store schema migration.
package migrator

import (
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
func MigrateSchema(ctx context.Context, storeDB *store.DB, storeInstance *store.Store, serverVersion string) (*semver.Version, error) {
	metadataDriver, err := dbdriver.Open(
		ctx,
		storepb.Engine_POSTGRES,
		dbdriver.DriverConfig{},
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

	if _, err := migrate(ctx, storeInstance, metadataDriver, verBefore, serverVersion, storeDB.ConnCfg.Database); err != nil {
		return nil, errors.Wrap(err, "failed to migrate")
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

	latestSchemaPath := fmt.Sprintf("migration/%s", latestSchemaFile)
	buf, err := migrationFS.ReadFile(latestSchemaPath)
	if err != nil {
		return errors.Wrapf(err, "failed to read latest schema %q", latestSchemaPath)
	}
	stmt := string(buf)

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
func migrate(ctx context.Context, storeInstance *store.Store, metadataDriver dbdriver.Driver, curVer semver.Version, serverVersion, databaseName string) (bool, error) {
	slog.Info("Apply database migration if needed...")
	slog.Info(fmt.Sprintf("Current schema version before migration: %s", curVer))

	// Apply migrations if needed.
	retVersion := curVer
	names, err := fs.Glob(migrationFS, "migration/*")
	if err != nil {
		return false, err
	}

	minorVersions, err := getMinorMigrationVersions(names, curVer)
	if err != nil {
		return false, err
	}

	for _, minorVersion := range minorVersions {
		slog.Info(fmt.Sprintf("Starting minor version migration cycle from %s ...", minorVersion))
		names, err := fs.Glob(migrationFS, fmt.Sprintf("migration/%d.%d/*.sql", minorVersion.Major, minorVersion.Minor))
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
			slog.Info(fmt.Sprintf("Migrating %s...", pv.version))
			version := model.Version{Semantic: true, Version: pv.version.String(), Suffix: time.Now().Format("20060102150405")}
			mi := &dbdriver.MigrationInfo{
				InstanceID:     nil,
				CreatorID:      api.SystemBotID,
				ReleaseVersion: serverVersion,
				Namespace:      databaseName,
				Database:       databaseName,
				Environment:    "", /* unused in execute migration */
				Source:         dbdriver.LIBRARY,
				Type:           dbdriver.Migrate,
				Description:    fmt.Sprintf("Migrate version %s server version %s with files %s.", pv.version, serverVersion, pv.filename),
			}
			if _, _, err := executeMigrationDefault(ctx, ctx, storeInstance, metadataDriver, mi, string(buf), version, dbdriver.ExecuteOptions{}); err != nil {
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

	return len(minorVersions) > 0, nil
}

func getProdCutoffVersion() (semver.Version, error) {
	minorPathPrefix := "migration/*"
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

	patchPathPrefix := fmt.Sprintf("migration/%d.%d", minorVersion.Major, minorVersion.Minor)
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
		if baseName == latestSchemaFile {
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

// executeMigrationDefault executes migration.
func executeMigrationDefault(ctx context.Context, driverCtx context.Context, store *store.Store, driver dbdriver.Driver, mi *dbdriver.MigrationInfo, statement string, version model.Version, opts dbdriver.ExecuteOptions) (migrationHistoryID string, updatedSchema string, resErr error) {
	execFunc := func(ctx context.Context, execStatement string) error {
		if _, err := driver.Execute(ctx, execStatement, opts); err != nil {
			return err
		}
		return nil
	}
	return ExecuteMigrationWithFunc(ctx, driverCtx, store, mi, statement, version, execFunc, opts)
}

// ExecuteMigrationWithFunc executes the migration with custom migration function.
func ExecuteMigrationWithFunc(ctx context.Context, driverCtx context.Context, s *store.Store, m *dbdriver.MigrationInfo, statement string, version model.Version, execFunc func(ctx context.Context, execStatement string) error, _ dbdriver.ExecuteOptions) (migrationHistoryID string, updatedSchema string, resErr error) {
	insertedID, err := BeginMigration(ctx, s, m, statement, version)
	if err != nil {
		return "", "", errors.Wrapf(err, "failed to begin migration")
	}

	startedNs := time.Now().UnixNano()

	defer func() {
		if err := EndMigration(ctx, s, startedNs, insertedID, "" /* updatedSchema  */, "" /* prevSchema */, nil /* sheetID */, resErr == nil /* isDone */); err != nil {
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

	return insertedID, "", nil
}

// BeginMigration checks before executing migration and inserts a migration history record with pending status.
func BeginMigration(ctx context.Context, stores *store.Store, m *dbdriver.MigrationInfo, statement string, version model.Version) (string, error) {
	// Phase 1 - Pre-check before executing migration
	// Check if the same migration version has already been applied.
	if list, err := stores.ListInstanceChangeHistoryForMigrator(ctx, &store.FindInstanceChangeHistoryMessage{
		InstanceID: m.InstanceID,
		DatabaseID: m.DatabaseID,
		Version:    &version,
	}); err != nil {
		return "", errors.Wrap(err, "failed to check duplicate version")
	} else if len(list) > 0 {
		migrationHistory := list[0]
		switch migrationHistory.Status {
		case dbdriver.Done:
			err := common.Errorf(common.MigrationAlreadyApplied, "database %q has already applied version %s, hint: the version might be duplicate, please check the version", m.Database, version.Version)
			slog.Debug(err.Error())
			// Force migration
			return migrationHistory.UID, nil
		case dbdriver.Pending:
			err := errors.Errorf("database %q version %s migration is already in progress", m.Database, version.Version)
			slog.Debug(err.Error())
			// For force migration, we will ignore the existing migration history and continue to migration.
			return migrationHistory.UID, nil
		case dbdriver.Failed:
			err := errors.Errorf("database %q version %s migration has failed, please check your database to make sure things are fine and then start a new migration using a new version", m.Database, version.Version)
			slog.Debug(err.Error())
			// For force migration, we will ignore the existing migration history and continue to migration.
			return migrationHistory.UID, nil
		}
	}

	// Phase 2 - Record migration history as PENDING.
	statementRecord, _ := common.TruncateString(statement, common.MaxSheetSize)
	insertedID, err := stores.CreatePendingInstanceChangeHistoryForMigrator(ctx, "" /* prevSchema */, m, statementRecord, nil /* sheetID */, version)
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
