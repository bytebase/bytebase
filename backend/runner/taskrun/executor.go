package taskrun

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	"github.com/bytebase/bytebase/backend/component/state"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/runner/schemasync"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/store/model"
)

// Executor is the task executor.
type Executor interface {
	// RunOnce will be called periodically by the scheduler until terminated is true.
	//
	// NOTE
	//
	// 1. It's possible that err could be non-nil while terminated is false, which
	// usually indicates a transient error and will make scheduler retry later.
	// 2. If err is non-nil, then the detail field will be ignored since info is provided in the err.
	// driverCtx is used by the database driver so that we can cancel the query
	// while have the ability to cleanup migration history etc.
	RunOnce(ctx context.Context, driverCtx context.Context, task *store.TaskMessage, taskRunUID int) (terminated bool, result *storepb.TaskRunResult, err error)
}

type execFuncType func(context.Context, string, db.Driver, db.ExecuteOptions) error

// RunExecutorOnce wraps a TaskExecutor.RunOnce call with panic recovery.
func RunExecutorOnce(ctx context.Context, driverCtx context.Context, exec Executor, task *store.TaskMessage, taskRunUID int) (terminated bool, result *storepb.TaskRunResult, err error) {
	defer func() {
		if r := recover(); r != nil {
			panicErr, ok := r.(error)
			if !ok {
				panicErr = errors.Errorf("%v", r)
			}
			slog.Error("TaskExecutor PANIC RECOVER", log.BBError(panicErr), log.BBStack("panic-stack"))
			terminated = true
			result = nil
			err = errors.Errorf("TaskExecutor PANIC RECOVER, err: %v", panicErr)
		}
	}()

	return exec.RunOnce(ctx, driverCtx, task, taskRunUID)
}

// Pointer fields are not nullable unless mentioned otherwise.
type migrateContext struct {
	syncer    *schemasync.Syncer
	profile   *config.Profile
	dbFactory *dbfactory.DBFactory

	instance *store.InstanceMessage
	database *store.DatabaseMessage
	sheet    *store.SheetMessage

	task        *store.TaskMessage
	taskRunUID  int
	taskRunName string

	version string

	release struct {
		// The release
		// Format: projects/{project}/releases/{release}
		release string
		// The file
		// Format: projects/{project}/releases/{release}/files/{id}
		file string
	}

	// mutable
	// changelog uid
	changelog int64

	// needDump indicates whether schema dump is needed before/after migration.
	// False for pure DML (INSERT, UPDATE, DELETE) since they don't change schema.
	needDump bool
}

func getCreateTaskRunLog(ctx context.Context, taskRunUID int, s *store.Store, profile *config.Profile) func(t time.Time, e *storepb.TaskRunLog) error {
	return func(t time.Time, e *storepb.TaskRunLog) error {
		return s.CreateTaskRunLog(ctx, taskRunUID, t.UTC(), profile.DeployID, e)
	}
}

func runMigration(ctx context.Context, driverCtx context.Context, store *store.Store, dbFactory *dbfactory.DBFactory, stateCfg *state.State, syncer *schemasync.Syncer, profile *config.Profile, task *store.TaskMessage, taskRunUID int, sheet *store.SheetMessage, schemaVersion string) (terminated bool, result *storepb.TaskRunResult, err error) {
	return runMigrationWithFunc(ctx, driverCtx, store, dbFactory, stateCfg, syncer, profile, task, taskRunUID, sheet, schemaVersion, nil /* default */)
}

func runMigrationWithFunc(
	ctx context.Context,
	driverCtx context.Context,
	store *store.Store,
	dbFactory *dbfactory.DBFactory,
	stateCfg *state.State,
	syncer *schemasync.Syncer,
	profile *config.Profile,
	task *store.TaskMessage,
	taskRunUID int,
	sheet *store.SheetMessage,
	schemaVersion string,
	execFunc execFuncType,
) (terminated bool, result *storepb.TaskRunResult, err error) {
	mc, err := getMigrationInfo(ctx, store, profile, syncer, task, schemaVersion, sheet, taskRunUID, dbFactory)
	if err != nil {
		return true, nil, err
	}

	// Pre-compute whether schema dump is needed.
	// Skip dump for pure DML statements (INSERT, UPDATE, DELETE) as they don't change schema.
	mc.needDump = computeNeedDump(task.Type, mc.database.Engine, sheet.Statement)

	skipped, err := doMigrationWithFunc(ctx, driverCtx, store, stateCfg, profile, sheet.Statement, mc, execFunc)
	if err != nil {
		return true, nil, err
	}
	return postMigration(ctx, store, mc, skipped)
}

func getMigrationInfo(ctx context.Context, stores *store.Store, profile *config.Profile, syncer *schemasync.Syncer, task *store.TaskMessage, schemaVersion string, sheet *store.SheetMessage, taskRunUID int, dbFactory *dbfactory.DBFactory) (*migrateContext, error) {
	instance, err := stores.GetInstance(ctx, &store.FindInstanceMessage{ResourceID: &task.InstanceID})
	if err != nil {
		return nil, err
	}
	database, err := stores.GetDatabase(ctx, &store.FindDatabaseMessage{InstanceID: &task.InstanceID, DatabaseName: task.DatabaseName})
	if err != nil {
		return nil, err
	}
	if database == nil {
		return nil, errors.Errorf("database not found")
	}
	plan, err := stores.GetPlan(ctx, &store.FindPlanMessage{UID: &task.PlanID})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get plan")
	}
	if plan == nil {
		return nil, errors.Errorf("plan %v not found", task.PlanID)
	}

	mc := &migrateContext{
		syncer:      syncer,
		profile:     profile,
		dbFactory:   dbFactory,
		instance:    instance,
		database:    database,
		sheet:       sheet,
		task:        task,
		version:     schemaVersion,
		taskRunName: common.FormatTaskRun(plan.ProjectID, int(task.PlanID), task.Environment, task.ID, taskRunUID),
		taskRunUID:  taskRunUID,
	}

	switch task.Type {
	case
		storepb.Task_TASK_TYPE_UNSPECIFIED,
		storepb.Task_DATABASE_EXPORT,
		storepb.Task_DATABASE_CREATE:
		return nil, errors.Errorf("task type %s is unexpected", task.Type)
	case storepb.Task_DATABASE_MIGRATE,
		storepb.Task_DATABASE_SDL:
		// Valid type for migration context
	default:
		return nil, errors.Errorf("task type %s is unexpected", task.Type)
	}

	if isChangeDatabaseTask(task) {
		if f := task.Payload.GetTaskReleaseSource().GetFile(); f != "" {
			project, release, _, err := common.GetProjectReleaseUIDFile(f)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to parse file %s", f)
			}
			mc.release.release = common.FormatReleaseName(project, release)
			mc.release.file = f
		}
	}

	return mc, nil
}

func doMigrationWithFunc(
	ctx context.Context,
	driverCtx context.Context,
	stores *store.Store,
	stateCfg *state.State,
	profile *config.Profile,
	statement string,
	mc *migrateContext,
	execFunc execFuncType,
) (bool, error) {
	instance := mc.instance
	database := mc.database

	project, err := stores.GetProject(ctx, &store.FindProjectMessage{ResourceID: &database.ProjectID})
	if err != nil {
		return false, errors.Wrapf(err, "failed to get project")
	}
	driver, err := mc.dbFactory.GetAdminDatabaseDriver(ctx, instance, database, db.ConnectionContext{
		TenantMode: project.Setting.GetPostgresDatabaseTenantMode(),
	})
	if err != nil {
		return false, errors.Wrapf(err, "failed to get driver connection for instance %q", instance.ResourceID)
	}
	defer driver.Close(ctx)

	statementRecord, _ := common.TruncateString(statement, common.MaxSheetSize)
	slog.Debug("Start migration...",
		slog.String("instance", database.InstanceID),
		slog.String("database", database.DatabaseName),
		slog.String("type", string(mc.task.Type.String())),
		slog.String("statement", statementRecord),
	)

	opts := db.ExecuteOptions{}

	if project != nil && project.Setting != nil {
		opts.MaximumRetries = int(project.Setting.GetExecutionRetryPolicy().GetMaximumRetries())
	}

	opts.SetConnectionID = func(id string) {
		stateCfg.TaskRunConnectionID.Store(mc.taskRunUID, id)
	}
	opts.DeleteConnectionID = func() {
		stateCfg.TaskRunConnectionID.Delete(mc.taskRunUID)
	}
	opts.CreateTaskRunLog = getCreateTaskRunLog(ctx, mc.taskRunUID, stores, profile)

	if execFunc == nil {
		execFunc = func(ctx context.Context, execStatement string, driver db.Driver, opts db.ExecuteOptions) error {
			if _, err := driver.Execute(ctx, execStatement, opts); err != nil {
				return err
			}
			return nil
		}
	}
	return executeMigrationWithFunc(ctx, driverCtx, stores, mc, statement, execFunc, driver, opts)
}

func postMigration(ctx context.Context, stores *store.Store, mc *migrateContext, skipped bool) (bool, *storepb.TaskRunResult, error) {
	if skipped {
		return true, &storepb.TaskRunResult{
			Detail: fmt.Sprintf("Task skipped because version %s has been applied", mc.version),
		}, nil
	}

	instance := mc.instance
	database := mc.database

	slog.Debug("Post migration...",
		slog.String("instance", instance.ResourceID),
		slog.String("database", database.DatabaseName),
	)

	// Remove schema drift.
	if _, err := stores.UpdateDatabase(ctx, &store.UpdateDatabaseMessage{
		InstanceID:   database.InstanceID,
		DatabaseName: database.DatabaseName,
		MetadataUpdates: []func(*storepb.DatabaseMetadata){func(md *storepb.DatabaseMetadata) {
			md.Drifted = false
		}},
	}); err != nil {
		return false, nil, errors.Wrapf(err, "failed to update database %q for instance %q", database.DatabaseName, database.InstanceID)
	}

	var detail string
	if mc.version == "" {
		detail = fmt.Sprintf("Applied migration to database %q.", database.DatabaseName)
	} else {
		detail = fmt.Sprintf("Applied migration version %s to database %q.", mc.version, database.DatabaseName)
	}

	return true, &storepb.TaskRunResult{
		Detail:    detail,
		Changelog: common.FormatChangelog(instance.ResourceID, database.DatabaseName, mc.changelog),
		Version:   mc.version,
	}, nil
}

// executeMigrationWithFunc executes the migration with custom migration function.
func executeMigrationWithFunc(
	ctx context.Context,
	driverCtx context.Context,
	s *store.Store,
	mc *migrateContext,
	statement string,
	execFunc execFuncType,
	driver db.Driver,
	opts db.ExecuteOptions,
) (skipped bool, resErr error) {
	// Phase 1 - Dump before migration.
	// Check if versioned is already applied.
	skipExecution, err := beginMigration(ctx, s, mc, opts)
	if err != nil {
		return false, errors.Wrapf(err, "failed to begin migration")
	}
	if skipExecution {
		return true, nil
	}

	defer func() {
		// Phase 3 - Dump after migration.
		// Insert revision for versioned.
		if err := endMigration(ctx, s, mc, resErr == nil /* isDone */, opts); err != nil {
			slog.Error("failed to end migration",
				log.BBError(err),
			)
		}
	}()

	// Phase 2 - Executing migration.
	if err := execFunc(driverCtx, statement, driver, opts); err != nil {
		return false, err
	}

	return false, nil
}

// beginMigration checks before executing migration and inserts a migration history record with pending status.
func beginMigration(ctx context.Context, stores *store.Store, mc *migrateContext, opts db.ExecuteOptions) (bool, error) {
	// list revisions and see if it has been applied
	// we can do this because
	// versioned migrations are executed one by one
	// so no other migrations can insert revisions
	//
	// users can create revisions though via API
	// however we can warn users not to unless they know
	// what they are doing
	if mc.version != "" {
		if mc.task.Type == storepb.Task_DATABASE_SDL {
			// Declarative case
			list, err := stores.ListRevisions(ctx, &store.FindRevisionMessage{
				InstanceID:   &mc.database.InstanceID,
				DatabaseName: &mc.database.DatabaseName,
				Limit:        common.NewP(1),
				Type:         common.NewP(storepb.SchemaChangeType_DECLARATIVE),
			})
			if err != nil {
				return false, errors.Wrapf(err, "failed to list revisions")
			}
			if len(list) > 0 {
				// If the version is equal or higher than the current version, return error
				latestRevision := list[0]
				latestVersion, err := model.NewVersion(latestRevision.Version)
				if err != nil {
					return false, errors.Wrapf(err, "failed to parse latest revision version %q", latestRevision.Version)
				}
				currentVersion, err := model.NewVersion(mc.version)
				if err != nil {
					return false, errors.Wrapf(err, "failed to parse current version %q", mc.version)
				}
				if currentVersion.LessThanOrEqual(latestVersion) {
					return false, errors.Errorf("cannot apply SDL migration with version %s because an equal or newer version %s already exists", mc.version, latestRevision.Version)
				}
			}
		} else {
			// Versioned case
			list, err := stores.ListRevisions(ctx, &store.FindRevisionMessage{
				InstanceID:   &mc.database.InstanceID,
				DatabaseName: &mc.database.DatabaseName,
				Version:      &mc.version,
				Type:         common.NewP(storepb.SchemaChangeType_VERSIONED),
			})
			if err != nil {
				return false, errors.Wrapf(err, "failed to list revisions")
			}
			if len(list) > 0 {
				// This version has been executed.
				// skip execution.
				return true, nil
			}
		}
	}

	// sync history
	var syncHistoryPrevUID *int64
	if mc.needDump {
		opts.LogDatabaseSyncStart()
		syncHistoryPrev, err := mc.syncer.SyncDatabaseSchemaToHistory(ctx, mc.database)
		if err != nil {
			opts.LogDatabaseSyncEnd(err.Error())
			return false, errors.Wrapf(err, "failed to sync database metadata and schema")
		}
		opts.LogDatabaseSyncEnd("")
		syncHistoryPrevUID = &syncHistoryPrev
	}

	// create pending changelog
	changelogType := convertTaskType(mc.task)
	changelogUID, err := stores.CreateChangelog(ctx, &store.ChangelogMessage{
		InstanceID:         mc.database.InstanceID,
		DatabaseName:       mc.database.DatabaseName,
		Status:             store.ChangelogStatusPending,
		PrevSyncHistoryUID: syncHistoryPrevUID,
		SyncHistoryUID:     nil,
		Payload: &storepb.ChangelogPayload{
			TaskRun:     mc.taskRunName,
			Revision:    0,
			SheetSha256: mc.sheet.Sha256,
			Version:     mc.version,
			Type:        changelogType,
			GitCommit:   mc.profile.GitCommit,
			DumpVersion: schema.GetDumpFormatVersion(mc.instance.Metadata.GetEngine()),
		}})
	if err != nil {
		return false, errors.Wrapf(err, "failed to create changelog")
	}
	mc.changelog = changelogUID

	return false, nil
}

// endMigration updates the migration history record to DONE or FAILED depending on migration is done or not.
func endMigration(ctx context.Context, storeInstance *store.Store, mc *migrateContext, isDone bool, opts db.ExecuteOptions) error {
	update := &store.UpdateChangelogMessage{
		UID: mc.changelog,
	}

	if mc.needDump {
		opts.LogDatabaseSyncStart()
		syncHistory, err := mc.syncer.SyncDatabaseSchemaToHistory(ctx, mc.database)
		if err != nil {
			opts.LogDatabaseSyncEnd(err.Error())
			return errors.Wrapf(err, "failed to sync database metadata and schema")
		}
		opts.LogDatabaseSyncEnd("")
		update.SyncHistoryUID = &syncHistory
	}

	if isDone {
		// if isDone, record in revision
		if mc.version != "" {
			r := &store.RevisionMessage{
				InstanceID:   mc.database.InstanceID,
				DatabaseName: mc.database.DatabaseName,
				Version:      mc.version,
				Payload: &storepb.RevisionPayload{
					Release:     mc.release.release,
					File:        mc.release.file,
					SheetSha256: mc.sheet.Sha256,
					TaskRun:     mc.taskRunName,
					Type:        storepb.SchemaChangeType_VERSIONED,
				},
			}
			if mc.task.Type == storepb.Task_DATABASE_SDL {
				r.Payload.Type = storepb.SchemaChangeType_DECLARATIVE
			}

			revision, err := storeInstance.CreateRevision(ctx, r)
			if err != nil {
				return errors.Wrapf(err, "failed to create revision")
			}
			update.RevisionUID = &revision.UID

			// Update database metadata with the version only if the new version is greater
			if shouldUpdateVersion(mc.database.Metadata.Version, mc.version) {
				if _, err := storeInstance.UpdateDatabase(ctx, &store.UpdateDatabaseMessage{
					InstanceID:   mc.database.InstanceID,
					DatabaseName: mc.database.DatabaseName,
					MetadataUpdates: []func(*storepb.DatabaseMetadata){func(md *storepb.DatabaseMetadata) {
						md.Version = mc.version
					}},
				}); err != nil {
					return errors.Wrapf(err, "failed to update database metadata with version")
				}
			}
		}
		status := store.ChangelogStatusDone
		update.Status = &status
	} else {
		status := store.ChangelogStatusFailed
		update.Status = &status
	}

	if err := storeInstance.UpdateChangelog(ctx, update); err != nil {
		return errors.Wrapf(err, "failed to update changelog")
	}

	return nil
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

func convertTaskType(t *store.TaskMessage) storepb.ChangelogPayload_Type {
	//exhaustive:enforce
	switch t.Type {
	case storepb.Task_DATABASE_MIGRATE:
		return storepb.ChangelogPayload_MIGRATE
	case storepb.Task_DATABASE_SDL:
		return storepb.ChangelogPayload_SDL
	case
		storepb.Task_TASK_TYPE_UNSPECIFIED,
		storepb.Task_DATABASE_CREATE,
		storepb.Task_DATABASE_EXPORT:
		return storepb.ChangelogPayload_TYPE_UNSPECIFIED
	default:
		return storepb.ChangelogPayload_TYPE_UNSPECIFIED
	}
}

// isChangeDatabaseTask returns whether the task involves changing a database.
func isChangeDatabaseTask(task *store.TaskMessage) bool {
	switch task.Type {
	case storepb.Task_DATABASE_MIGRATE:
		return true
	case storepb.Task_DATABASE_SDL:
		return true
	case storepb.Task_DATABASE_CREATE,
		storepb.Task_DATABASE_EXPORT:
		return false
	default:
		return false
	}
}

// computeNeedDump determines if schema dump is needed based on task type and statements.
func computeNeedDump(taskType storepb.Task_Type, engine storepb.Engine, statement string) bool {
	//exhaustive:enforce
	switch taskType {
	case storepb.Task_DATABASE_MIGRATE:
		// For DATABASE_MIGRATE, skip dump if all statements are DML
		// (INSERT, UPDATE, DELETE) since they don't change schema.
		return !base.IsAllDML(engine, statement)
	case storepb.Task_DATABASE_SDL,
		storepb.Task_DATABASE_CREATE:
		return true
	case
		storepb.Task_TASK_TYPE_UNSPECIFIED,
		storepb.Task_DATABASE_EXPORT:
		return false
	default:
		return false
	}
}
