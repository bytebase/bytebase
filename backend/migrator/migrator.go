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
	"github.com/bytebase/bytebase/backend/plugin/db/pg"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
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
	// The migration file follows the name pattern of {{version_number}}##{{description}}.
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
	bytebasePgDriver := bytebaseDriver.(*pg.Driver)

	if err := migrateOld(ctx, metadataDriver, bytebasePgDriver, databaseName, serverVersion); err != nil {
		return nil, err
	}

	if err := backfillHistory(ctx, storeInstance, bytebasePgDriver, databaseName); err != nil {
		return nil, err
	}

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

func backfillHistory(ctx context.Context, storeInstance *store.Store, bytebasePgDriver *pg.Driver, databaseName string) error {
	histories, err := storeInstance.ListInstanceChangeHistory(ctx, &store.FindInstanceChangeHistoryMessage{
		// Metadata database has instanceID nil;
		InstanceID: nil,
	})
	if err != nil {
		return err
	}
	// For new database and backfilled database, there should be histories already.
	if len(histories) > 0 {
		return nil
	}

	log.Info("Backfilling Bytebase metadata migration history")
	limit := 10
	offset := 0
	for {
		oldHistories, err := bytebasePgDriver.FindMigrationHistoryList(ctx, &dbdriver.MigrationHistoryFind{
			Database: &databaseName,
			Limit:    &limit,
			Offset:   &offset,
		})
		if err != nil {
			return err
		}
		if len(oldHistories) == 0 {
			break
		}
		offset += limit

		var creates []*store.InstanceChangeHistoryMessage
		for _, h := range oldHistories {
			storedVersion, err := util.ToStoredVersion(h.UseSemanticVersion, h.Version, h.SemanticVersionSuffix)
			if err != nil {
				return err
			}
			changeHistory := store.InstanceChangeHistoryMessage{
				CreatorID:           api.SystemBotID,
				CreatedTs:           h.CreatedTs,
				UpdaterID:           api.SystemBotID,
				UpdatedTs:           h.UpdatedTs,
				InstanceID:          nil,
				DatabaseID:          nil,
				IssueID:             nil,
				ReleaseVersion:      h.ReleaseVersion,
				Sequence:            int64(h.Sequence),
				Source:              h.Source,
				Type:                h.Type,
				Status:              h.Status,
				Version:             storedVersion,
				Description:         h.Description,
				Statement:           h.Statement,
				Schema:              h.Schema,
				SchemaPrev:          h.SchemaPrev,
				ExecutionDurationNs: h.ExecutionDurationNs,
				Payload:             h.Payload,
			}

			creates = append(creates, &changeHistory)
		}

		if _, err := storeInstance.CreateInstanceChangeHistory(ctx, creates...); err != nil {
			return err
		}
	}

	return nil
}

func hasInstanceChangeTable(ctx context.Context, metadataDriver dbdriver.Driver) (bool, error) {
	var exists bool
	if err := metadataDriver.GetDB().QueryRowContext(ctx,
		`SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'instance_change_history')`,
	).Scan(&exists); err != nil {
		return false, err
	}
	return exists, nil
}

func migrateOld(ctx context.Context, metadataDriver dbdriver.Driver, bytebasePgDriver *pg.Driver, databaseName, serverVersion string) error {
	has, err := hasInstanceChangeTable(ctx, metadataDriver)
	if err != nil {
		return err
	}
	if has {
		return nil
	}

	curVer, curSequence, err := getLatestVersionOld(ctx, bytebasePgDriver, databaseName)
	if err != nil {
		return err
	}

	// Apply migrations if needed.
	names, err := fs.Glob(migrationFS, fmt.Sprintf("migration/%s/*", common.ReleaseModeProd))
	if err != nil {
		return err
	}
	minorVersions, _, err := getMinorMigrationVersions(names, *curVer)
	if err != nil {
		return err
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
			log.Info(fmt.Sprintf("Migrating %s...", pv.version))
			storedVersion, err := util.ToStoredVersion(true /* UseSemanticVersion */, pv.version.String(), common.DefaultMigrationVersion())
			if err != nil {
				return err
			}
			curSequence++
			if _, err := metadataDriver.GetDB().ExecContext(ctx, string(buf)); err != nil {
				return err
			}
			const insertHistoryQuery = `
				INSERT INTO migration_history (
					created_by,
					created_ts,
					updated_by,
					updated_ts,
					release_version,
					namespace,
					sequence,
					source,
					type,
					status,
					version,
					description,
					statement,
					` + `"schema",` + `
					schema_prev,
					execution_duration_ns,
					issue_id,
					payload
				)
				VALUES ($1, EXTRACT(epoch from NOW()), $2, EXTRACT(epoch from NOW()), $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, 0, $14, $15)
			`
			if _, err := bytebasePgDriver.GetDB().ExecContext(ctx, insertHistoryQuery,
				"", /* created_by */
				"", /* updated_by */
				serverVersion,
				databaseName,
				curSequence,
				dbdriver.LIBRARY,
				dbdriver.Migrate,
				dbdriver.Done,
				storedVersion,
				fmt.Sprintf("Migrate version %s server version %s with files %s.", pv.version, serverVersion, pv.filename),
				string(buf),
				"", /* schema */
				"", /* schema_prev */
				"", /* issue_id */
				"", /* payload */
			); err != nil {
				return err
			}

			has, err := hasInstanceChangeTable(ctx, metadataDriver)
			if err != nil {
				return err
			}
			if has {
				return nil
			}
		}
	}

	return nil
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
	if _, err := storeInstance.CreateInstanceChangeHistory(ctx, &store.InstanceChangeHistoryMessage{
		CreatorID:      api.SystemBotID,
		InstanceID:     nil,
		DatabaseID:     nil,
		IssueID:        nil,
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
		Payload:             "",
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
	histories, err := storeInstance.ListInstanceChangeHistory(ctx, &store.FindInstanceChangeHistoryMessage{
		// Metadata database has instanceID nil;
		InstanceID: nil,
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

// getLatestVersionOld returns the latest schema version in semantic versioning format.
// We expect our own migration history to use semantic versions.
// If there's no migration history, version will be nil.
func getLatestVersionOld(ctx context.Context, bytebasePgDriver *pg.Driver, database string) (*semver.Version, int, error) {
	// We look back the past migration history records and return the latest successful (DONE) migration version.
	history, err := bytebasePgDriver.FindMigrationHistoryList(ctx, &dbdriver.MigrationHistoryFind{
		Database: &database,
	})
	if err != nil {
		return nil, 0, errors.Wrap(err, "failed to get migration history")
	}
	if len(history) == 0 {
		return nil, 0, nil
	}

	for _, h := range history {
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
		v, err := semver.Make(h.Version)
		if err != nil {
			return nil, 0, errors.Wrapf(err, "invalid version %q", h.Version)
		}
		return &v, h.Sequence, nil
	}

	return nil, 0, errors.Errorf("failed to find a successful migration history to determine the schema version")
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
	major := curVer.Major
	if major != majorSchemaVersion {
		return errors.Errorf("current major schema version %d is different from the major schema version %d this release %s expects", major, majorSchemaVersion, serverVersion)
	}

	var histories []*store.InstanceChangeHistoryMessage
	// Because dev migrations don't use semantic versioning, we have to look at all migration history to
	// figure out whether the migration statement has already been applied.
	if mode == common.ReleaseModeDev {
		h, err := storeInstance.ListInstanceChangeHistory(ctx, &store.FindInstanceChangeHistoryMessage{
			// Metadata database has instanceID nil;
			InstanceID: nil,
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

	minorVersions, messages, err := getMinorMigrationVersions(names, curVer)
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
			if _, _, err := utils.ExecuteMigration(ctx, storeInstance, metadataDriver, mi, string(buf), nil /* executeBeforeCommitTx */); err != nil {
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
		if _, _, err := utils.ExecuteMigration(ctx, storeInstance, metadataDriver, mi, m.statement, nil /* executeBeforeCommitTx */); err != nil {
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
