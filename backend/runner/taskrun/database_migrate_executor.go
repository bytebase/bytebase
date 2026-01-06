package taskrun

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/antlr4-go/antlr/v4"
	"github.com/bytebase/parser/postgresql"
	"github.com/github/gh-ost/go/logic"
	gomysql "github.com/go-sql-driver/mysql"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/bus"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	"github.com/bytebase/bytebase/backend/component/ghost"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/oracle"
	parserbase "github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/plugin/parser/pg"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/runner/schemasync"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/store/model"
	"github.com/bytebase/bytebase/backend/utils"
)

// NewDatabaseMigrateExecutor creates a database migration task executor.
func NewDatabaseMigrateExecutor(store *store.Store, dbFactory *dbfactory.DBFactory, bus *bus.Bus, schemaSyncer *schemasync.Syncer, profile *config.Profile) Executor {
	return &DatabaseMigrateExecutor{
		store:        store,
		dbFactory:    dbFactory,
		bus:          bus,
		schemaSyncer: schemaSyncer,
		profile:      profile,
	}
}

// DatabaseMigrateExecutor is the database migration task executor.
type DatabaseMigrateExecutor struct {
	store        *store.Store
	dbFactory    *dbfactory.DBFactory
	bus          *bus.Bus
	schemaSyncer *schemasync.Syncer
	profile      *config.Profile
}

// RunOnce will run the database migration task executor once.
func (exec *DatabaseMigrateExecutor) RunOnce(ctx context.Context, driverCtx context.Context, task *store.TaskMessage, taskRunUID int) (*storepb.TaskRunResult, error) {
	// Fetch instance, database, and project (common to all migration types)
	instance, err := exec.store.GetInstance(ctx, &store.FindInstanceMessage{ResourceID: &task.InstanceID})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get instance")
	}
	if instance == nil {
		return nil, errors.Errorf("instance not found for task %v", task.ID)
	}

	database, err := exec.store.GetDatabase(ctx, &store.FindDatabaseMessage{InstanceID: &task.InstanceID, DatabaseName: task.DatabaseName})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get database")
	}
	if database == nil {
		return nil, errors.Errorf("database not found for task %v", task.ID)
	}

	project, err := exec.store.GetProject(ctx, &store.FindProjectMessage{ResourceID: &database.ProjectID})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get project")
	}

	// Ensure baseline changelog exists before running any migration
	if err := exec.ensureBaselineChangelog(ctx, database, instance); err != nil {
		return nil, errors.Wrap(err, "failed to ensure baseline changelog")
	}

	// Mark database as not drifted since we're about to sync it
	if _, err := exec.store.UpdateDatabase(ctx, &store.UpdateDatabaseMessage{
		InstanceID:   database.InstanceID,
		DatabaseName: database.DatabaseName,
		MetadataUpdates: []func(*storepb.DatabaseMetadata){func(md *storepb.DatabaseMetadata) {
			md.Drifted = false
		}},
	}); err != nil {
		return nil, errors.Wrapf(err, "failed to update database %q for instance %q", database.DatabaseName, database.InstanceID)
	}

	// Execute migration based on task type
	if releaseName := task.Payload.GetRelease(); releaseName != "" {
		// Parse release name to get project ID and release UID
		_, releaseUID, err := common.GetProjectReleaseUID(releaseName)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse release name %q", releaseName)
		}

		// Fetch the release
		release, err := exec.store.GetReleaseByUID(ctx, releaseUID)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get release %d", releaseUID)
		}
		if release == nil {
			return nil, errors.Errorf("release %d not found", releaseUID)
		}

		// Switch based on release type
		switch release.Payload.Type {
		case storepb.SchemaChangeType_VERSIONED:
			return exec.runVersionedRelease(ctx, driverCtx, task, taskRunUID, release, instance, database, project)
		case storepb.SchemaChangeType_DECLARATIVE:
			return exec.runDeclarativeRelease(ctx, driverCtx, task, taskRunUID, release, instance, database, project)
		default:
			return nil, errors.Errorf("unsupported release type %q", release.Payload.Type)
		}
	}

	// Fetch sheet for non-release tasks
	sheet, err := exec.store.GetSheetFull(ctx, task.Payload.GetSheetSha256())
	if err != nil {
		return nil, err
	}
	if sheet == nil {
		return nil, errors.Errorf("sheet not found: %s", task.Payload.GetSheetSha256())
	}

	if task.Payload.GetEnableGhost() {
		return exec.runGhostMigration(ctx, driverCtx, task, taskRunUID, sheet, instance, database, project)
	}
	return exec.runStandardMigration(ctx, driverCtx, task, taskRunUID, sheet, instance, database, project)
}

// ensureBaselineChangelog creates a baseline changelog if this is the first migration for the database.
func (exec *DatabaseMigrateExecutor) ensureBaselineChangelog(ctx context.Context, database *store.DatabaseMessage, instance *store.InstanceMessage) error {
	// Check if this database has any existing changelogs
	limit := 1
	existingChangelogs, err := exec.store.ListChangelogs(ctx, &store.FindChangelogMessage{
		InstanceID:   &database.InstanceID,
		DatabaseName: &database.DatabaseName,
		Limit:        &limit,
	})
	if err != nil {
		return errors.Wrapf(err, "failed to check for existing changelogs")
	}

	// If no changelogs exist, create a baseline with the current schema
	if len(existingChangelogs) == 0 {
		baselineSyncHistoryUID, err := exec.schemaSyncer.SyncDatabaseSchemaToHistory(ctx, database)
		if err != nil {
			return errors.Wrapf(err, "failed to sync database schema for baseline")
		}

		_, err = exec.store.CreateChangelog(ctx, &store.ChangelogMessage{
			InstanceID:     database.InstanceID,
			DatabaseName:   database.DatabaseName,
			Status:         store.ChangelogStatusDone,
			SyncHistoryUID: &baselineSyncHistoryUID,
			Payload: &storepb.ChangelogPayload{
				Type:        storepb.ChangelogPayload_BASELINE,
				GitCommit:   exec.profile.GitCommit,
				DumpVersion: schema.GetDumpFormatVersion(instance.Metadata.GetEngine()),
			},
		})
		if err != nil {
			return errors.Wrapf(err, "failed to create baseline changelog")
		}
	}

	return nil
}

func (exec *DatabaseMigrateExecutor) runStandardMigration(ctx context.Context, driverCtx context.Context, task *store.TaskMessage, taskRunUID int, sheet *store.SheetMessage, instance *store.InstanceMessage, database *store.DatabaseMessage, project *store.ProjectMessage) (*storepb.TaskRunResult, error) {
	// Handle prior backup if enabled.
	// TransformDMLToSelect will automatically filter out DDL statements,
	// so this works correctly for mixed DDL+DML statements.
	var priorBackupDetail *storepb.PriorBackupDetail
	if task.Payload.GetEnablePriorBackup() {
		exec.store.CreateTaskRunLogS(ctx, taskRunUID, time.Now(), exec.profile.DeployID, &storepb.TaskRunLog{
			Type:             storepb.TaskRunLog_PRIOR_BACKUP_START,
			PriorBackupStart: &storepb.TaskRunLog_PriorBackupStart{},
		})

		// Check if we should skip backup or not.
		if common.EngineSupportPriorBackup(database.Engine) {
			var backupErr error
			priorBackupDetail, backupErr = exec.backupData(ctx, driverCtx, sheet.Statement, task.Payload, task, instance, database)
			if backupErr != nil {
				exec.store.CreateTaskRunLogS(ctx, taskRunUID, time.Now(), exec.profile.DeployID, &storepb.TaskRunLog{
					Type: storepb.TaskRunLog_PRIOR_BACKUP_END,
					PriorBackupEnd: &storepb.TaskRunLog_PriorBackupEnd{
						Error: backupErr.Error(),
					},
				})

				// Check if we should skip backup error and continue to run migration.
				if project != nil && project.Setting != nil && !project.Setting.SkipBackupErrors {
					return nil, backupErr
				}
			} else {
				exec.store.CreateTaskRunLogS(ctx, taskRunUID, time.Now(), exec.profile.DeployID, &storepb.TaskRunLog{
					Type: storepb.TaskRunLog_PRIOR_BACKUP_END,
					PriorBackupEnd: &storepb.TaskRunLog_PriorBackupEnd{
						PriorBackupDetail: priorBackupDetail,
					},
				})
			}
		}
	}

	needDump := computeNeedDump(task.Type, database.Engine, sheet.Statement)

	// Get database driver
	driver, err := exec.dbFactory.GetAdminDatabaseDriver(ctx, instance, database, db.ConnectionContext{
		TenantMode: project.Setting.GetPostgresDatabaseTenantMode(),
		TaskRunUID: &taskRunUID,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get driver connection for instance %q", instance.ResourceID)
	}
	defer driver.Close(ctx)

	slog.Debug("Start migration...",
		slog.String("instance", database.InstanceID),
		slog.String("database", database.DatabaseName),
		slog.String("type", task.Type.String()),
		slog.String("sheetSha256", sheet.Sha256),
	)

	// Set up execute options
	opts := db.ExecuteOptions{}
	if project != nil && project.Setting != nil {
		opts.MaximumRetries = int(project.Setting.GetExecutionRetryPolicy().GetMaximumRetries())
	}
	opts.CreateTaskRunLog = func(t time.Time, e *storepb.TaskRunLog) error {
		return exec.store.CreateTaskRunLog(ctx, taskRunUID, t.UTC(), exec.profile.DeployID, e)
	}

	// Begin migration - create pending changelog
	changelogUID, err := exec.store.CreateChangelog(ctx, &store.ChangelogMessage{
		InstanceID:     database.InstanceID,
		DatabaseName:   database.DatabaseName,
		Status:         store.ChangelogStatusPending,
		SyncHistoryUID: nil,
		Payload: &storepb.ChangelogPayload{
			TaskRun:     common.FormatTaskRun(database.ProjectID, task.PlanID, task.Environment, task.ID, taskRunUID),
			Type:        storepb.ChangelogPayload_MIGRATE,
			GitCommit:   exec.profile.GitCommit,
			DumpVersion: schema.GetDumpFormatVersion(instance.Metadata.GetEngine()),
		},
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create changelog")
	}

	// Execute the SQL
	_, migrationErr := driver.Execute(driverCtx, sheet.Statement, opts)

	// Dump after migration and update changelog
	update := &store.UpdateChangelogMessage{
		UID: changelogUID,
	}
	if needDump {
		opts.LogDatabaseSyncStart()
		syncHistory, err := exec.schemaSyncer.SyncDatabaseSchemaToHistory(ctx, database)
		if err != nil {
			opts.LogDatabaseSyncEnd(err.Error())
			slog.Error("failed to sync database schema", log.BBError(err))
		} else {
			opts.LogDatabaseSyncEnd("")
			update.SyncHistoryUID = &syncHistory
		}
	}
	if migrationErr == nil {
		status := store.ChangelogStatusDone
		update.Status = &status
	} else {
		status := store.ChangelogStatusFailed
		update.Status = &status
	}
	if err := exec.store.UpdateChangelog(ctx, update); err != nil {
		slog.Error("failed to update changelog", log.BBError(err))
	}

	if migrationErr != nil {
		return nil, migrationErr
	}

	return &storepb.TaskRunResult{
		HasPriorBackup: priorBackupDetail != nil && len(priorBackupDetail.Items) > 0,
	}, nil
}

func (exec *DatabaseMigrateExecutor) runGhostMigration(ctx context.Context, driverCtx context.Context, task *store.TaskMessage, taskRunUID int, sheet *store.SheetMessage, instance *store.InstanceMessage, database *store.DatabaseMessage, project *store.ProjectMessage) (*storepb.TaskRunResult, error) {
	flags := task.Payload.GetFlags()

	// Get database driver
	driver, err := exec.dbFactory.GetAdminDatabaseDriver(ctx, instance, database, db.ConnectionContext{
		TenantMode: project.Setting.GetPostgresDatabaseTenantMode(),
		TaskRunUID: &taskRunUID,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get driver connection for instance %q", instance.ResourceID)
	}
	defer driver.Close(ctx)

	slog.Debug("Start migration...",
		slog.String("instance", database.InstanceID),
		slog.String("database", database.DatabaseName),
		slog.String("type", task.Type.String()),
		slog.String("sheetSha256", sheet.Sha256),
	)

	// Set up execute options
	opts := db.ExecuteOptions{}
	if project != nil && project.Setting != nil {
		opts.MaximumRetries = int(project.Setting.GetExecutionRetryPolicy().GetMaximumRetries())
	}
	opts.CreateTaskRunLog = func(t time.Time, e *storepb.TaskRunLog) error {
		return exec.store.CreateTaskRunLog(ctx, taskRunUID, t.UTC(), exec.profile.DeployID, e)
	}

	// Prepare gh-ost migration context before beginning migration
	statement := strings.TrimSpace(sheet.Statement)
	// Trim trailing semicolons.
	statement = strings.TrimRight(statement, ";")

	tableName, err := ghost.GetTableNameFromStatement(statement)
	if err != nil {
		return nil, err
	}

	adminDataSource := utils.DataSourceFromInstanceWithType(instance, storepb.DataSourceType_ADMIN)
	if adminDataSource == nil {
		return nil, common.Errorf(common.Internal, "admin data source not found for instance %s", instance.ResourceID)
	}

	migrationContext, err := ghost.NewMigrationContext(ctx, task.ID, database, adminDataSource, tableName, fmt.Sprintf("_%d", time.Now().Unix()), sheet.Statement, false, flags, 10000000)
	if err != nil {
		return nil, errors.Wrap(err, "failed to init migrationContext for gh-ost")
	}
	defer func() {
		// Use migrationContext.Uuid as the tls_config_key by convention.
		// We need to deregister it when gh-ost exits.
		// https://github.com/bytebase/gh-ost2/pull/4
		gomysql.DeregisterTLSConfig(migrationContext.Uuid)
	}()

	// Begin migration - create pending changelog
	changelogUID, err := exec.store.CreateChangelog(ctx, &store.ChangelogMessage{
		InstanceID:     database.InstanceID,
		DatabaseName:   database.DatabaseName,
		Status:         store.ChangelogStatusPending,
		SyncHistoryUID: nil,
		Payload: &storepb.ChangelogPayload{
			TaskRun:     common.FormatTaskRun(database.ProjectID, task.PlanID, task.Environment, task.ID, taskRunUID),
			Type:        storepb.ChangelogPayload_MIGRATE,
			GitCommit:   exec.profile.GitCommit,
			DumpVersion: schema.GetDumpFormatVersion(instance.Metadata.GetEngine()),
		},
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create changelog")
	}

	// Execute gh-ost migration
	// set buffer size to 1 to unblock the sender because there is no listener if the task is canceled.
	migrationError := make(chan error, 1)

	migrator := logic.NewMigrator(migrationContext, "bb")

	defer func() {
		cleanupCtx := context.Background()

		// Use the backup database name of MySQL as the ghost database name.
		ghostDBName := common.BackupDatabaseNameOfEngine(storepb.Engine_MYSQL)
		sql := fmt.Sprintf("DROP TABLE IF EXISTS `%s`.`%s`; DROP TABLE IF EXISTS `%s`.`%s`;",
			ghostDBName,
			migrationContext.GetGhostTableName(),
			ghostDBName,
			migrationContext.GetChangelogTableName(),
		)

		if _, err := driver.GetDB().ExecContext(cleanupCtx, sql); err != nil {
			slog.Warn("failed to cleanup gh-ost temp tables", log.BBError(err))
		}
	}()

	go func() {
		if err := migrator.Migrate(); err != nil {
			slog.Error("failed to run gh-ost migration", log.BBError(err))
			migrationError <- err
			return
		}
		migrationError <- nil
	}()

	var migrationErr error
	select {
	case err := <-migrationError:
		migrationErr = err
	case <-driverCtx.Done():
		migrationContext.PanicAbort <- errors.New("task canceled")
		migrationErr = errors.New("task canceled")
	}

	// Dump after migration and update changelog
	update := &store.UpdateChangelogMessage{
		UID: changelogUID,
	}
	opts.LogDatabaseSyncStart()
	syncHistory, err := exec.schemaSyncer.SyncDatabaseSchemaToHistory(ctx, database)
	if err != nil {
		opts.LogDatabaseSyncEnd(err.Error())
		slog.Error("failed to sync database schema", log.BBError(err))
	} else {
		opts.LogDatabaseSyncEnd("")
		update.SyncHistoryUID = &syncHistory
	}
	if migrationErr == nil {
		status := store.ChangelogStatusDone
		update.Status = &status
	} else {
		status := store.ChangelogStatusFailed
		update.Status = &status
	}
	if err := exec.store.UpdateChangelog(ctx, update); err != nil {
		slog.Error("failed to update changelog", log.BBError(err))
	}

	if migrationErr != nil {
		return nil, migrationErr
	}

	return &storepb.TaskRunResult{}, nil
}

func (exec *DatabaseMigrateExecutor) runVersionedRelease(ctx context.Context, driverCtx context.Context, task *store.TaskMessage, taskRunUID int, release *store.ReleaseMessage, instance *store.InstanceMessage, database *store.DatabaseMessage, project *store.ProjectMessage) (*storepb.TaskRunResult, error) {
	// Get existing revisions for this database
	revisions, err := exec.store.ListRevisions(ctx, &store.FindRevisionMessage{
		InstanceID:   &task.InstanceID,
		DatabaseName: task.DatabaseName,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list revisions for database %q", *task.DatabaseName)
	}

	// Build map of applied versions
	appliedVersions := make(map[string]bool)
	for _, revision := range revisions {
		if revision.Payload.Type == storepb.SchemaChangeType_VERSIONED {
			appliedVersions[revision.Version] = true
		}
	}

	taskRunName := common.FormatTaskRun(database.ProjectID, task.PlanID, task.Environment, task.ID, taskRunUID)

	// Create pending changelog for the entire release
	changelogUID, err := exec.store.CreateChangelog(ctx, &store.ChangelogMessage{
		InstanceID:     database.InstanceID,
		DatabaseName:   database.DatabaseName,
		Status:         store.ChangelogStatusPending,
		SyncHistoryUID: nil,
		Payload: &storepb.ChangelogPayload{
			TaskRun:     taskRunName,
			Type:        storepb.ChangelogPayload_MIGRATE,
			GitCommit:   exec.profile.GitCommit,
			DumpVersion: schema.GetDumpFormatVersion(instance.Metadata.GetEngine()),
		},
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create changelog")
	}

	// Set up execute options
	opts := db.ExecuteOptions{}
	if project != nil && project.Setting != nil {
		opts.MaximumRetries = int(project.Setting.GetExecutionRetryPolicy().GetMaximumRetries())
	}
	opts.CreateTaskRunLog = func(t time.Time, e *storepb.TaskRunLog) error {
		return exec.store.CreateTaskRunLog(ctx, taskRunUID, t.UTC(), exec.profile.DeployID, e)
	}

	// Get database driver once for all files
	driver, err := exec.dbFactory.GetAdminDatabaseDriver(ctx, instance, database, db.ConnectionContext{
		TenantMode: project.Setting.GetPostgresDatabaseTenantMode(),
		TaskRunUID: &taskRunUID,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get driver connection for instance %q", instance.ResourceID)
	}
	defer driver.Close(ctx)

	var migrationErr error
	var lastAppliedVersion string

	// Execute unapplied files in order
	for _, file := range release.Payload.Files {
		// Skip if already applied
		if appliedVersions[file.Version] {
			slog.Info("skipping already applied version",
				slog.String("version", file.Version),
				slog.String("database", *task.DatabaseName))
			continue
		}

		sheet, err := exec.store.GetSheetFull(ctx, file.SheetSha256)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get sheet %s for version %s", file.SheetSha256, file.Version)
		}
		if sheet == nil {
			return nil, errors.Errorf("sheet not found: %s", file.SheetSha256)
		}

		slog.Info("executing release file",
			slog.String("version", file.Version),
			slog.String("database", *task.DatabaseName),
			slog.String("file", file.Path))

		// Log release file execution
		exec.store.CreateTaskRunLogS(ctx, taskRunUID, time.Now(), exec.profile.DeployID, &storepb.TaskRunLog{
			Type: storepb.TaskRunLog_RELEASE_FILE_EXECUTE,
			ReleaseFileExecute: &storepb.TaskRunLog_ReleaseFileExecute{
				Version:  file.Version,
				FilePath: file.Path,
			},
		})

		slog.Debug("Start migration...",
			slog.String("instance", database.InstanceID),
			slog.String("database", database.DatabaseName),
			slog.String("type", task.Type.String()),
		)

		// Execute the SQL
		_, err = driver.Execute(driverCtx, sheet.Statement, opts)
		if err != nil {
			migrationErr = errors.Wrapf(err, "failed to execute release file %s (version %s)", file.Path, file.Version)
			break
		}

		// Create revision for this file
		r := &store.RevisionMessage{
			InstanceID:   database.InstanceID,
			DatabaseName: database.DatabaseName,
			Version:      file.Version,
			Payload: &storepb.RevisionPayload{
				Release:     task.Payload.GetRelease(),
				File:        file.Path,
				SheetSha256: file.SheetSha256,
				TaskRun:     taskRunName,
				Type:        storepb.SchemaChangeType_VERSIONED,
			},
		}

		_, err = exec.store.CreateRevision(ctx, r)
		if err != nil {
			migrationErr = errors.Wrapf(err, "failed to create revision for version %s", file.Version)
			break
		}

		// Track the last successfully applied version
		lastAppliedVersion = file.Version
	}

	// Update changelog after all files are processed
	update := &store.UpdateChangelogMessage{
		UID: changelogUID,
	}
	opts.LogDatabaseSyncStart()
	syncHistory, err := exec.schemaSyncer.SyncDatabaseSchemaToHistory(ctx, database)
	if err != nil {
		opts.LogDatabaseSyncEnd(err.Error())
		slog.Error("failed to sync database schema", log.BBError(err))
	} else {
		opts.LogDatabaseSyncEnd("")
		update.SyncHistoryUID = &syncHistory
	}
	if migrationErr == nil {
		status := store.ChangelogStatusDone
		update.Status = &status
	} else {
		status := store.ChangelogStatusFailed
		update.Status = &status
	}
	if err := exec.store.UpdateChangelog(ctx, update); err != nil {
		slog.Error("failed to update changelog", log.BBError(err))
	}

	if migrationErr != nil {
		return nil, migrationErr
	}

	// Update database version to the last successfully applied version
	if lastAppliedVersion != "" && shouldUpdateVersion(database.Metadata.Version, lastAppliedVersion) {
		if _, err := exec.store.UpdateDatabase(ctx, &store.UpdateDatabaseMessage{
			InstanceID:   database.InstanceID,
			DatabaseName: database.DatabaseName,
			MetadataUpdates: []func(*storepb.DatabaseMetadata){func(md *storepb.DatabaseMetadata) {
				md.Version = lastAppliedVersion
			}},
		}); err != nil {
			return nil, errors.Wrapf(err, "failed to update database version to %s", lastAppliedVersion)
		}
	}

	return &storepb.TaskRunResult{}, nil
}

func (exec *DatabaseMigrateExecutor) runDeclarativeRelease(ctx context.Context, driverCtx context.Context, task *store.TaskMessage, taskRunUID int, release *store.ReleaseMessage, instance *store.InstanceMessage, database *store.DatabaseMessage, project *store.ProjectMessage) (*storepb.TaskRunResult, error) {
	// Declarative releases should have exactly one file
	if len(release.Payload.Files) == 0 {
		return nil, errors.Errorf("no files found in declarative release")
	}
	if len(release.Payload.Files) > 1 {
		return nil, errors.Errorf("declarative release should have exactly one file, found %d", len(release.Payload.Files))
	}

	file := release.Payload.Files[0]

	// Fetch the schema file
	sheet, err := exec.store.GetSheetFull(ctx, file.SheetSha256)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get sheet %s for version %s", file.SheetSha256, file.Version)
	}
	if sheet == nil {
		return nil, errors.Errorf("sheet not found: %s", file.SheetSha256)
	}

	slog.Info("executing declarative release",
		slog.String("version", file.Version),
		slog.String("database", *task.DatabaseName),
		slog.String("file", file.Path))

	// Log release file execution
	exec.store.CreateTaskRunLogS(ctx, taskRunUID, time.Now(), exec.profile.DeployID, &storepb.TaskRunLog{
		Type: storepb.TaskRunLog_RELEASE_FILE_EXECUTE,
		ReleaseFileExecute: &storepb.TaskRunLog_ReleaseFileExecute{
			Version: file.Version,
			// FilePath is omitted because it's artificial for declarative releases
		},
	})

	// Get database driver
	driver, err := exec.dbFactory.GetAdminDatabaseDriver(ctx, instance, database, db.ConnectionContext{
		TenantMode: project.Setting.GetPostgresDatabaseTenantMode(),
		TaskRunUID: &taskRunUID,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get driver connection for instance %q", instance.ResourceID)
	}
	defer driver.Close(ctx)

	slog.Debug("Start migration...",
		slog.String("instance", database.InstanceID),
		slog.String("database", database.DatabaseName),
		slog.String("type", task.Type.String()),
		slog.String("sheetSha256", sheet.Sha256),
	)

	// Set up execute options
	opts := db.ExecuteOptions{}
	if project != nil && project.Setting != nil {
		opts.MaximumRetries = int(project.Setting.GetExecutionRetryPolicy().GetMaximumRetries())
	}
	opts.CreateTaskRunLog = func(t time.Time, e *storepb.TaskRunLog) error {
		return exec.store.CreateTaskRunLog(ctx, taskRunUID, t.UTC(), exec.profile.DeployID, e)
	}

	// Compute SDL diff before beginning migration
	opts.LogComputeDiffStart()
	migrationSQL, err := diff(ctx, exec.store, instance, database, sheet.Statement)
	if err != nil {
		opts.LogComputeDiffEnd(err.Error())
		return nil, errors.Wrapf(err, "failed to diff database schema")
	}
	opts.LogComputeDiffEnd("")

	// Begin migration - create pending changelog
	changelogUID, err := exec.store.CreateChangelog(ctx, &store.ChangelogMessage{
		InstanceID:     database.InstanceID,
		DatabaseName:   database.DatabaseName,
		Status:         store.ChangelogStatusPending,
		SyncHistoryUID: nil,
		Payload: &storepb.ChangelogPayload{
			TaskRun:     common.FormatTaskRun(database.ProjectID, task.PlanID, task.Environment, task.ID, taskRunUID),
			Type:        storepb.ChangelogPayload_SDL,
			GitCommit:   exec.profile.GitCommit,
			DumpVersion: schema.GetDumpFormatVersion(instance.Metadata.GetEngine()),
		},
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create changelog")
	}

	// Execute SDL migration
	// Log statement string.
	opts.LogCommandStatement = true
	_, migrationErr := driver.Execute(driverCtx, migrationSQL, opts)

	// Dump after migration and update changelog
	update := &store.UpdateChangelogMessage{
		UID: changelogUID,
	}
	opts.LogDatabaseSyncStart()
	syncHistory, err := exec.schemaSyncer.SyncDatabaseSchemaToHistory(ctx, database)
	if err != nil {
		opts.LogDatabaseSyncEnd(err.Error())
		slog.Error("failed to sync database schema", log.BBError(err))
	} else {
		opts.LogDatabaseSyncEnd("")
		update.SyncHistoryUID = &syncHistory
	}
	if migrationErr == nil {
		status := store.ChangelogStatusDone
		update.Status = &status
	} else {
		status := store.ChangelogStatusFailed
		update.Status = &status
	}
	if err := exec.store.UpdateChangelog(ctx, update); err != nil {
		slog.Error("failed to update changelog", log.BBError(err))
	}

	if migrationErr != nil {
		return nil, errors.Wrap(migrationErr, "failed to execute declarative release")
	}

	// Post migration - update database schema version
	// Note: Declarative releases do NOT create revisions (they are version-tracked through the database schema itself)
	if _, err := exec.store.UpdateDatabase(ctx, &store.UpdateDatabaseMessage{
		InstanceID:   database.InstanceID,
		DatabaseName: database.DatabaseName,
		MetadataUpdates: []func(*storepb.DatabaseMetadata){func(md *storepb.DatabaseMetadata) {
			md.Version = file.Version
		}},
	}); err != nil {
		return nil, errors.Wrapf(err, "failed to update database version for %q", database.DatabaseName)
	}

	return &storepb.TaskRunResult{}, nil
}

func (exec *DatabaseMigrateExecutor) backupData(
	ctx context.Context,
	driverCtx context.Context,
	originStatement string,
	payload *storepb.Task,
	task *store.TaskMessage,
	instance *store.InstanceMessage,
	database *store.DatabaseMessage,
) (*storepb.PriorBackupDetail, error) {
	if !payload.GetEnablePriorBackup() {
		return nil, nil
	}

	sourceDatabaseName := common.FormatDatabase(database.InstanceID, database.DatabaseName)
	// Format: instances/{instance}/databases/{database}
	backupDBName := common.BackupDatabaseNameOfEngine(database.Engine)
	targetDatabaseName := common.FormatDatabase(database.InstanceID, backupDBName)
	var backupDatabase *store.DatabaseMessage
	var backupDriver db.Driver

	backupInstanceID, backupDatabaseName, err := common.GetInstanceDatabaseID(targetDatabaseName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse backup database")
	}

	if database.Engine != storepb.Engine_POSTGRES {
		backupDatabase, err = exec.store.GetDatabase(ctx, &store.FindDatabaseMessage{InstanceID: &backupInstanceID, DatabaseName: &backupDatabaseName})
		if err != nil {
			return nil, errors.Wrap(err, "failed to get backup database")
		}
		if backupDatabase == nil {
			return nil, errors.Errorf("backup database %q not found", targetDatabaseName)
		}
		backupDriver, err = exec.dbFactory.GetAdminDatabaseDriver(driverCtx, instance, backupDatabase, db.ConnectionContext{})
		if err != nil {
			return nil, errors.Wrap(err, "failed to get backup database driver")
		}
		defer backupDriver.Close(driverCtx)
	}

	project, err := exec.store.GetProject(ctx, &store.FindProjectMessage{ResourceID: &database.ProjectID})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get project")
	}
	driver, err := exec.dbFactory.GetAdminDatabaseDriver(driverCtx, instance, database, db.ConnectionContext{
		TenantMode: project.Setting.GetPostgresDatabaseTenantMode(),
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get database driver")
	}
	defer driver.Close(driverCtx)

	tc := parserbase.TransformContext{
		InstanceID:              instance.ResourceID,
		GetDatabaseMetadataFunc: buildGetDatabaseMetadataFunc(exec.store),
		ListDatabaseNamesFunc:   buildListDatabaseNamesFunc(exec.store),
		IsCaseSensitive:         store.IsObjectCaseSensitive(instance),
		DatabaseName:            database.DatabaseName,
	}
	if database.Engine == storepb.Engine_ORACLE {
		oracleDriver, ok := driver.(*oracle.Driver)
		if ok {
			if version, err := oracleDriver.GetVersion(); err == nil {
				tc.Version = version
			}
		}
	}

	if len(originStatement) > common.MaxSheetCheckSize {
		return nil, errors.Errorf("statement size %d exceeds the limit %d, please disable data backup", len(originStatement), common.MaxSheetCheckSize)
	}

	prefix := "_" + time.Now().Format("20060102150405")
	statements, err := parserbase.TransformDMLToSelect(ctx, database.Engine, tc, originStatement, database.DatabaseName, backupDatabaseName, prefix)
	if err != nil {
		return nil, errors.Wrap(err, "failed to transform DML to select")
	}

	prependStatements, err := getPrependStatements(database.Engine, originStatement)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get prepend statements")
	}

	priorBackupDetail := &storepb.PriorBackupDetail{}
	bbSource := fmt.Sprintf("task %d", task.ID)
	for _, statement := range statements {
		backupStatement := statement.Statement
		if prependStatements != "" {
			backupStatement = prependStatements + backupStatement
		}
		if _, err := driver.Execute(driverCtx, backupStatement, db.ExecuteOptions{}); err != nil {
			return nil, errors.Wrapf(err, "failed to execute backup statement %q", backupStatement)
		}
		switch instance.Metadata.GetEngine() {
		case storepb.Engine_TIDB:
			if _, err := driver.Execute(driverCtx, fmt.Sprintf("ALTER TABLE `%s`.`%s` COMMENT = '%s, source table (%s, %s)'", backupDatabaseName, statement.TargetTableName, bbSource, database.DatabaseName, statement.SourceTableName), db.ExecuteOptions{}); err != nil {
				return nil, errors.Wrap(err, "failed to set table comment")
			}
		case storepb.Engine_MYSQL:
			if _, err := driver.Execute(driverCtx, fmt.Sprintf("ALTER TABLE `%s`.`%s` COMMENT = '%s, source table (%s, %s)'", backupDatabaseName, statement.TargetTableName, bbSource, database.DatabaseName, statement.SourceTableName), db.ExecuteOptions{}); err != nil {
				return nil, errors.Wrap(err, "failed to set table comment")
			}
		case storepb.Engine_MSSQL:
			schemaName := statement.SourceSchema
			if schemaName == "" {
				schemaName = "dbo"
			}
			if _, err := backupDriver.Execute(driverCtx, fmt.Sprintf("EXEC sp_addextendedproperty 'MS_Description', '%s, source table (%s, %s, %s)', 'SCHEMA', 'dbo', 'TABLE', '%s'", bbSource, database.DatabaseName, schemaName, statement.SourceTableName, statement.TargetTableName), db.ExecuteOptions{}); err != nil {
				return nil, errors.Wrap(err, "failed to set table comment")
			}
		case storepb.Engine_POSTGRES:
			schemaName := statement.SourceSchema
			if schemaName == "" {
				schemaName = "public"
			}
			if _, err := driver.Execute(driverCtx, fmt.Sprintf(`COMMENT ON TABLE "%s"."%s" IS '%s, source table (%s, %s)'`, backupDatabaseName, statement.TargetTableName, bbSource, schemaName, statement.SourceTableName), db.ExecuteOptions{}); err != nil {
				return nil, errors.Wrap(err, "failed to set table comment")
			}
		case storepb.Engine_ORACLE:
			if _, err := driver.Execute(driverCtx, fmt.Sprintf(`COMMENT ON TABLE "%s"."%s" IS '%s, source table (%s, %s)'`, backupDatabaseName, statement.TargetTableName, bbSource, database.DatabaseName, statement.SourceTableName), db.ExecuteOptions{}); err != nil {
				return nil, errors.Wrap(err, "failed to set table comment")
			}
		default:
			// No action needed for other database engines
		}

		item := &storepb.PriorBackupDetail_Item{
			SourceTable: &storepb.PriorBackupDetail_Item_Table{
				Database: sourceDatabaseName,
				Schema:   statement.SourceSchema,
				Table:    statement.SourceTableName,
			},
			TargetTable: &storepb.PriorBackupDetail_Item_Table{
				Database: targetDatabaseName,
				Schema:   "",
				Table:    statement.TargetTableName,
			},
			StartPosition: statement.StartPosition,
			EndPosition:   statement.EndPosition,
		}
		if database.Engine == storepb.Engine_POSTGRES {
			item.TargetTable = &storepb.PriorBackupDetail_Item_Table{
				Database: sourceDatabaseName,
				// postgres uses schema as the backup database name currently.
				Schema: backupDatabaseName,
				Table:  statement.TargetTableName,
			}
		}
		priorBackupDetail.Items = append(priorBackupDetail.Items, item)
	}

	if database.Engine != storepb.Engine_POSTGRES {
		if err := exec.schemaSyncer.SyncDatabaseSchema(ctx, backupDatabase); err != nil {
			slog.Error("failed to sync backup database schema",
				slog.String("database", targetDatabaseName),
				log.BBError(err),
			)
		}
	} else {
		if err := exec.schemaSyncer.SyncDatabaseSchema(ctx, database); err != nil {
			slog.Error("failed to sync backup database schema",
				slog.String("database", fmt.Sprintf("/instances/%s/databases/%s", instance.ResourceID, database.DatabaseName)),
				log.BBError(err),
			)
		}
	}

	return priorBackupDetail, nil
}

func buildGetDatabaseMetadataFunc(storeInstance *store.Store) parserbase.GetDatabaseMetadataFunc {
	return func(ctx context.Context, instanceID, databaseName string) (string, *model.DatabaseMetadata, error) {
		database, err := storeInstance.GetDatabase(ctx, &store.FindDatabaseMessage{
			InstanceID:   &instanceID,
			DatabaseName: &databaseName,
		})
		if err != nil {
			return "", nil, err
		}
		if database == nil {
			return "", nil, nil
		}
		databaseMetadata, err := storeInstance.GetDBSchema(ctx, &store.FindDBSchemaMessage{
			InstanceID:   instanceID,
			DatabaseName: databaseName,
		})
		if err != nil {
			return "", nil, err
		}
		if databaseMetadata == nil {
			return "", nil, nil
		}
		return databaseName, databaseMetadata, nil
	}
}

func buildListDatabaseNamesFunc(storeInstance *store.Store) parserbase.ListDatabaseNamesFunc {
	return func(ctx context.Context, instanceID string) ([]string, error) {
		databases, err := storeInstance.ListDatabases(ctx, &store.FindDatabaseMessage{
			InstanceID: &instanceID,
		})
		if err != nil {
			return nil, err
		}
		names := make([]string, 0, len(databases))
		for _, database := range databases {
			names = append(names, database.DatabaseName)
		}
		return names, nil
	}
}

func getPrependStatements(engine storepb.Engine, statement string) (string, error) {
	if engine != storepb.Engine_POSTGRES {
		return "", nil
	}

	parseResults, err := pg.ParsePostgreSQL(statement)
	if err != nil {
		return "", errors.Wrap(err, "failed to parse statement")
	}

	visitor := &prependStatementsVisitor{
		statement: statement,
	}

	// Walk through all statements to find the first SET role/search_path statement
	// The visitor will stop after finding the first one due to its internal check
	for _, result := range parseResults {
		antlr.ParseTreeWalkerDefault.Walk(visitor, result.Tree)
		// If we found a result, stop walking remaining statements
		if visitor.result != "" {
			break
		}
	}

	return visitor.result, nil
}

// prependStatementsVisitor extracts SET role and search_path statements
type prependStatementsVisitor struct {
	*postgresql.BasePostgreSQLParserListener
	statement string
	result    string
}

func (v *prependStatementsVisitor) EnterVariablesetstmt(ctx *postgresql.VariablesetstmtContext) {
	// If we already found a result, don't process more statements
	if v.result != "" {
		return
	}

	setRest := ctx.Set_rest()
	if setRest == nil {
		return
	}
	setRestMore := setRest.Set_rest_more()
	if setRestMore == nil {
		return
	}
	genericSet := setRestMore.Generic_set()
	if genericSet == nil {
		return
	}
	varName := genericSet.Var_name()
	if varName == nil {
		return
	}
	if len(varName.AllColid()) != 1 {
		return
	}

	name := pg.NormalizePostgreSQLColid(varName.Colid(0))
	if name == "role" || name == "search_path" {
		// Extract the text for this SET statement
		v.result = v.extractStatementText(ctx)
	}
}

// extractStatementText extracts the original text for a SET statement context
// This matches pg_query_go behavior: trim leading/trailing whitespace, preserve internal whitespace
func (v *prependStatementsVisitor) extractStatementText(ctx *postgresql.VariablesetstmtContext) string {
	// Extract text from the original statement
	start := ctx.GetStart().GetStart()
	stop := ctx.GetStop().GetStop()

	// Handle potential edge cases with token positions
	if start < 0 || stop < 0 || start >= len(v.statement) {
		return ""
	}

	// Find the semicolon that ends this statement by looking ahead from the stop token
	endPos := stop + 1
	stmtLen := len(v.statement)
	for endPos < stmtLen {
		char := v.statement[endPos]
		if char == ';' {
			// Include the semicolon and any whitespace before it
			stop = endPos
			break
		}
		if char != ' ' && char != '\t' && char != '\n' && char != '\r' {
			// Hit non-whitespace, non-semicolon character, stop looking
			break
		}
		endPos++
	}

	// Ensure stop doesn't exceed statement length
	if stop >= stmtLen {
		stop = stmtLen - 1
	}

	// Extract the raw text
	text := v.statement[start : stop+1]

	// Match pg_query_go behavior: trim leading and trailing whitespace but preserve internal whitespace
	text = strings.TrimSpace(text)

	// Add semicolon if not present (to match pg_query_go behavior)
	if !strings.HasSuffix(text, ";") {
		text += ";"
	}

	return text
}

func diff(ctx context.Context, s *store.Store, instance *store.InstanceMessage, database *store.DatabaseMessage, sheetContent string) (string, error) {
	pengine, err := common.ConvertToParserEngine(instance.Metadata.GetEngine())
	if err != nil {
		return "", errors.Wrapf(err, "failed to convert %q to parser engine", instance.Metadata.GetEngine())
	}

	dbMetadata, err := s.GetDBSchema(ctx, &store.FindDBSchemaMessage{
		InstanceID:   database.InstanceID,
		DatabaseName: database.DatabaseName,
	})
	if err != nil {
		return "", errors.Wrapf(err, "failed to get database schema for database %q", database.DatabaseName)
	}
	if dbMetadata == nil {
		return "", errors.Errorf("database schema %q not found", database.DatabaseName)
	}

	// Try to get the previous successful SDL text and schema from task history
	previousUserSDLText, previousSchema, err := getPreviousSuccessfulSDLAndSchema(ctx, s, database.InstanceID, database.DatabaseName)
	if err != nil {
		return "", errors.Wrapf(err, "failed to get previous SDL text and schema for database %q", database.DatabaseName)
	}

	// Use GetSDLDiff with previous SDL text and schema
	// - engine: the database engine
	// - currentSDLText: user's target SDL input
	// - previousUserSDLText: previous SDL text (empty triggers initialization scenario)
	// - currentSchema: current database schema (used as baseline in initialization)
	// - previousSchema: previous database schema from changelog
	schemaDiff, err := schema.GetSDLDiff(pengine, sheetContent, previousUserSDLText, dbMetadata, previousSchema)
	if err != nil {
		return "", errors.Wrap(err, "failed to compute SDL schema diff")
	}

	// Filter out bbdataarchive schema changes for Postgres
	if instance.Metadata.GetEngine() == storepb.Engine_POSTGRES {
		schemaDiff = schema.FilterPostgresArchiveSchema(schemaDiff)
	}

	migrationSQL, err := schema.GenerateMigration(pengine, schemaDiff)
	if err != nil {
		return "", errors.Wrapf(err, "failed to generate migration SQL")
	}

	return migrationSQL, nil
}

// getPreviousSuccessfulSDLAndSchema gets both the SDL text and database schema from the most recent successful SDL changelog.
// It retrieves the SDL text by following the reference chain: changelog -> task_run -> task -> release -> file -> sheet.
// Returns empty string for SDL text if no previous successful SDL changelog is found or if the reference chain is broken.
func getPreviousSuccessfulSDLAndSchema(ctx context.Context, s *store.Store, instanceID string, databaseName string) (string, *model.DatabaseMetadata, error) {
	// Find the most recent successful SDL changelog for this database
	// We only want MIGRATE_SDL type changelogs that are completed (DONE status)
	doneStatus := store.ChangelogStatusDone
	limit := 1 // We only need the most recent one

	changelogs, err := s.ListChangelogs(ctx, &store.FindChangelogMessage{
		InstanceID:   &instanceID,
		DatabaseName: &databaseName,
		TypeList:     []string{storepb.ChangelogPayload_SDL.String()}, // Only SDL migrations
		Status:       &doneStatus,
		Limit:        &limit, // Get only the most recent one
		ShowFull:     true,   // Get full schema
	})
	if err != nil {
		return "", nil, errors.Wrapf(err, "failed to list previous SDL changelogs for database %s", databaseName)
	}

	if len(changelogs) == 0 {
		// No previous SDL changelogs found - this is fine, we'll use initialization scenario
		return "", nil, nil
	}

	mostRecentChangelog := changelogs[0] // ListChangelogs should return them in descending order by creation time

	// Retrieve the previous SDL text by following the reference chain:
	// changelog -> task_run -> task -> release -> file -> sheet
	var previousUserSDLText string
	if taskRunName := mostRecentChangelog.Payload.GetTaskRun(); taskRunName != "" {
		// Parse task run name to get task ID
		_, _, _, taskID, _, err := common.GetProjectIDPlanIDStageIDTaskIDTaskRunID(taskRunName)
		if err == nil {
			// Get the task
			task, err := s.GetTaskByID(ctx, taskID)
			if err == nil && task != nil {
				// Get the release from the task
				if releaseName := task.Payload.GetRelease(); releaseName != "" {
					_, releaseUID, err := common.GetProjectReleaseUID(releaseName)
					if err == nil {
						release, err := s.GetReleaseByUID(ctx, releaseUID)
						if err == nil && release != nil {
							// For SDL/declarative releases, there should be exactly one file
							if len(release.Payload.Files) == 1 {
								file := release.Payload.Files[0]
								sheet, err := s.GetSheetFull(ctx, file.SheetSha256)
								if err == nil && sheet != nil {
									previousUserSDLText = sheet.Statement
								}
							}
						}
					}
				}
			}
		}
	}

	// Get the previous schema from sync history
	// Use SyncHistoryUID (after applying the SDL) instead of PrevSyncHistoryUID (before applying)
	// This represents the database schema state after the previous SDL was successfully applied
	var previousSchema *model.DatabaseMetadata
	if mostRecentChangelog.SyncHistoryUID != nil {
		// Get the sync history record to obtain the schema metadata
		syncHistory, err := s.GetSyncHistoryByUID(ctx, *mostRecentChangelog.SyncHistoryUID)
		if err != nil {
			return "", nil, errors.Wrapf(err, "failed to get sync history by UID %d", *mostRecentChangelog.SyncHistoryUID)
		}

		if syncHistory != nil && syncHistory.Metadata != nil {
			// Get instance to determine engine and case sensitivity
			instance, err := s.GetInstance(ctx, &store.FindInstanceMessage{ResourceID: &instanceID})
			if err != nil {
				return "", nil, errors.Wrapf(err, "failed to get instance %s", instanceID)
			}
			if instance == nil {
				return "", nil, errors.Errorf("instance %s not found", instanceID)
			}

			// Create a DatabaseSchema wrapper using the metadata from sync history
			previousSchema = model.NewDatabaseMetadata(
				syncHistory.Metadata,
				[]byte(syncHistory.Schema), // Use the schema content from sync history
				&storepb.DatabaseConfig{},  // Empty config
				instance.Metadata.GetEngine(),
				store.IsObjectCaseSensitive(instance),
			)
		}
	}

	return previousUserSDLText, previousSchema, nil
}

// shouldUpdateVersion checks if newVersion is greater than currentVersion.
// Returns true if:
// - currentVersion is empty
// - currentVersion is invalid
// - newVersion is greater than currentVersion
func shouldUpdateVersion(currentVersion, newVersion string) bool {
	if currentVersion == "" {
		// If no current version, always update
		return true
	}
	current, err := model.NewVersion(currentVersion)
	if err != nil {
		// If current version is invalid, update with new version
		return true
	}

	nv, err := model.NewVersion(newVersion)
	if err != nil {
		// If new version is invalid, don't update
		return false
	}
	return current.LessThan(nv)
}

// computeNeedDump determines if schema dump is needed based on task type and statements.
func computeNeedDump(taskType storepb.Task_Type, engine storepb.Engine, statement string) bool {
	//exhaustive:enforce
	switch taskType {
	case storepb.Task_DATABASE_MIGRATE:
		// For DATABASE_MIGRATE, skip dump if all statements are DML
		// (INSERT, UPDATE, DELETE) since they don't change schema.
		return !parserbase.IsAllDML(engine, statement)
	case storepb.Task_DATABASE_CREATE:
		return true
	case
		storepb.Task_TASK_TYPE_UNSPECIFIED,
		storepb.Task_DATABASE_EXPORT:
		return false
	default:
		return false
	}
}
