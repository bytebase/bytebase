package taskrun

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/github/gh-ost/go/base"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/common/log"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/plugin/db/util"
	vcsPlugin "github.com/bytebase/bytebase/plugin/vcs"
	"github.com/bytebase/bytebase/server/component/activity"
	"github.com/bytebase/bytebase/server/component/config"
	"github.com/bytebase/bytebase/server/component/dbfactory"
	"github.com/bytebase/bytebase/server/component/state"
	"github.com/bytebase/bytebase/server/runner/schemasync"
	"github.com/bytebase/bytebase/server/utils"
	"github.com/bytebase/bytebase/store"
)

// NewSchemaUpdateGhostCutoverExecutor creates a schema update (gh-ost) cutover task executor.
func NewSchemaUpdateGhostCutoverExecutor(store *store.Store, dbFactory *dbfactory.DBFactory, activityManager *activity.Manager, stateCfg *state.State, schemaSyncer *schemasync.Syncer, profile config.Profile) Executor {
	return &SchemaUpdateGhostCutoverExecutor{
		store:           store,
		dbFactory:       dbFactory,
		activityManager: activityManager,
		stateCfg:        stateCfg,
		schemaSyncer:    schemaSyncer,
		profile:         profile,
	}
}

// SchemaUpdateGhostCutoverExecutor is the schema update (gh-ost) cutover task executor.
type SchemaUpdateGhostCutoverExecutor struct {
	store           *store.Store
	dbFactory       *dbfactory.DBFactory
	activityManager *activity.Manager
	stateCfg        *state.State
	schemaSyncer    *schemasync.Syncer
	profile         config.Profile
}

// RunOnce will run SchemaUpdateGhostCutover task once.
func (exec *SchemaUpdateGhostCutoverExecutor) RunOnce(ctx context.Context, task *api.Task) (bool, *api.TaskRunResultPayload, error) {
	taskDAG, err := exec.store.GetTaskDAGByToTaskID(ctx, task.ID)
	if err != nil {
		return true, nil, errors.Wrapf(err, "failed to get a single taskDAG for schema update gh-ost cutover task, id: %v", task.ID)
	}
	database, err := exec.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
		EnvironmentID: &task.Database.Instance.Environment.ResourceID,
		InstanceID:    &task.Database.Instance.ResourceID,
		DatabaseName:  &task.Database.Name,
	})
	if err != nil {
		return true, nil, err
	}

	syncTaskID := taskDAG.FromTaskID
	defer exec.stateCfg.GhostTaskState.Delete(syncTaskID)

	syncTask, err := exec.store.GetTaskByID(ctx, syncTaskID)
	if err != nil {
		return true, nil, errors.Wrap(err, "failed to get schema update gh-ost sync task for cutover task")
	}
	payload := &api.TaskDatabaseSchemaUpdateGhostSyncPayload{}
	if err := json.Unmarshal([]byte(syncTask.Payload), payload); err != nil {
		return true, nil, errors.Wrap(err, "invalid database schema update gh-ost sync payload")
	}

	tableName, err := utils.GetTableNameFromStatement(payload.Statement)
	if err != nil {
		return true, nil, errors.Wrap(err, "failed to parse table name from statement")
	}

	postponeFilename := utils.GetPostponeFlagFilename(syncTaskID, task.Database.ID, task.Database.Name, tableName)

	value, ok := exec.stateCfg.GhostTaskState.Load(syncTaskID)
	if !ok {
		return true, nil, errors.Errorf("failed to get gh-ost state from sync task")
	}
	sharedGhost := value.(sharedGhostState)

	terminated, result, err := cutover(ctx, exec.store, exec.dbFactory, exec.activityManager, exec.profile, task, payload.Statement, payload.SchemaVersion, payload.VCSPushEvent, postponeFilename, sharedGhost.migrationContext, sharedGhost.errCh)
	if err := exec.schemaSyncer.SyncDatabaseSchema(ctx, database, true /* force */); err != nil {
		log.Error("failed to sync database schema",
			zap.String("instanceName", task.Instance.Name),
			zap.String("databaseName", task.Database.Name),
			zap.Error(err),
		)
	}

	return terminated, result, err
}

func cutover(ctx context.Context, store *store.Store, dbFactory *dbfactory.DBFactory, activityManager *activity.Manager, profile config.Profile, task *api.Task, statement, schemaVersion string, vcsPushEvent *vcsPlugin.PushEvent, postponeFilename string, migrationContext *base.MigrationContext, errCh <-chan error) (terminated bool, result *api.TaskRunResultPayload, err error) {
	statement = strings.TrimSpace(statement)

	mi, err := preMigration(ctx, store, profile, task, db.Migrate, statement, schemaVersion, vcsPushEvent)
	if err != nil {
		return true, nil, err
	}
	migrationID, schema, err := func() (migrationHistoryID string, updatedSchema string, resErr error) {
		driver, err := dbFactory.GetAdminDatabaseDriver(ctx, task.Instance, task.Database.Name)
		if err != nil {
			return "", "", err
		}
		defer driver.Close(ctx)
		needsSetup, err := driver.NeedsSetupMigration(ctx)
		if err != nil {
			return "", "", errors.Wrapf(err, "failed to check migration setup for instance %q", task.Instance.Name)
		}
		if needsSetup {
			return "", "", common.Errorf(common.MigrationSchemaMissing, "missing migration schema for instance %q", task.Instance.Name)
		}

		executor := driver.(util.MigrationExecutor)

		var prevSchemaBuf bytes.Buffer
		if _, err := driver.Dump(ctx, mi.Database, &prevSchemaBuf, true); err != nil {
			return "", "", err
		}

		// wait for heartbeat lag.
		// try to make the time gap between the migration history insertion and the actual cutover as close as possible.
		cancelled := waitForCutover(ctx, migrationContext)
		if cancelled {
			return "", "", errors.Errorf("cutover poller cancelled")
		}

		insertedID, err := util.BeginMigration(ctx, executor, mi, prevSchemaBuf.String(), statement, db.BytebaseDatabase)
		if err != nil {
			if common.ErrorCode(err) == common.MigrationAlreadyApplied {
				return insertedID, prevSchemaBuf.String(), nil
			}
			return "", "", errors.Wrapf(err, "failed to begin migration for issue %s", mi.IssueID)
		}
		startedNs := time.Now().UnixNano()

		defer func() {
			if err := util.EndMigration(ctx, executor, startedNs, insertedID, updatedSchema, db.BytebaseDatabase, resErr == nil /*isDone*/); err != nil {
				log.Error("failed to update migration history record",
					zap.Error(err),
					zap.String("migration_id", migrationHistoryID),
				)
			}
		}()

		if err := os.Remove(postponeFilename); err != nil {
			return "", "", errors.Wrap(err, "failed to remove postpone flag file")
		}

		if migrationErr := <-errCh; migrationErr != nil {
			return "", "", errors.Wrapf(migrationErr, "failed to run gh-ost migration")
		}

		var afterSchemaBuf bytes.Buffer
		if _, err := executor.Dump(ctx, mi.Database, &afterSchemaBuf, true /*schemaOnly*/); err != nil {
			return "", "", util.FormatError(err)
		}

		return insertedID, afterSchemaBuf.String(), nil
	}()
	if err != nil {
		return true, nil, err
	}

	return postMigration(ctx, store, activityManager, profile, task, vcsPushEvent, mi, migrationID, schema)
}

func waitForCutover(ctx context.Context, migrationContext *base.MigrationContext) bool {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			heartbeatLag := migrationContext.TimeSinceLastHeartbeatOnChangelog()
			maxLagMillisecondsThrottle := time.Duration(atomic.LoadInt64(&migrationContext.MaxLagMillisecondsThrottleThreshold)) * time.Millisecond
			cutOverLockTimeout := time.Duration(migrationContext.CutOverLockTimeoutSeconds) * time.Second
			if heartbeatLag <= maxLagMillisecondsThrottle && heartbeatLag <= cutOverLockTimeout {
				return false
			}
		case <-ctx.Done(): // if cancel() execute
			return true
		}
	}
}
