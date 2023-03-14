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

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/activity"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	"github.com/bytebase/bytebase/backend/component/state"
	enterpriseAPI "github.com/bytebase/bytebase/backend/enterprise/api"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	vcsPlugin "github.com/bytebase/bytebase/backend/plugin/vcs"
	"github.com/bytebase/bytebase/backend/runner/schemasync"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
)

// NewSchemaUpdateGhostCutoverExecutor creates a schema update (gh-ost) cutover task executor.
func NewSchemaUpdateGhostCutoverExecutor(store *store.Store, dbFactory *dbfactory.DBFactory, activityManager *activity.Manager, license enterpriseAPI.LicenseService, stateCfg *state.State, schemaSyncer *schemasync.Syncer, profile config.Profile) Executor {
	return &SchemaUpdateGhostCutoverExecutor{
		store:           store,
		dbFactory:       dbFactory,
		activityManager: activityManager,
		license:         license,
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
	license         enterpriseAPI.LicenseService
	stateCfg        *state.State
	schemaSyncer    *schemasync.Syncer
	profile         config.Profile
}

// RunOnce will run SchemaUpdateGhostCutover task once.
func (exec *SchemaUpdateGhostCutoverExecutor) RunOnce(ctx context.Context, task *store.TaskMessage) (bool, *api.TaskRunResultPayload, error) {
	if len(task.BlockedBy) != 1 {
		return true, nil, errors.Errorf("failed to find task dag for ToTask %v", task.ID)
	}
	syncTaskID := task.BlockedBy[0]
	defer exec.stateCfg.GhostTaskState.Delete(syncTaskID)

	instance, err := exec.store.GetInstanceV2(ctx, &store.FindInstanceMessage{UID: &task.InstanceID})
	if err != nil {
		return true, nil, err
	}
	database, err := exec.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{UID: task.DatabaseID})
	if err != nil {
		return true, nil, err
	}

	syncTask, err := exec.store.GetTaskV2ByID(ctx, syncTaskID)
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

	postponeFilename := utils.GetPostponeFlagFilename(syncTaskID, database.UID, database.DatabaseName, tableName)

	value, ok := exec.stateCfg.GhostTaskState.Load(syncTaskID)
	if !ok {
		return true, nil, errors.Errorf("failed to get gh-ost state from sync task")
	}
	sharedGhost := value.(sharedGhostState)

	terminated, result, err := cutover(ctx, exec.store, exec.dbFactory, exec.activityManager, exec.license, exec.profile, task, payload.Statement, payload.SchemaVersion, payload.VCSPushEvent, postponeFilename, sharedGhost.migrationContext, sharedGhost.errCh)
	if err := exec.schemaSyncer.SyncDatabaseSchema(ctx, database, true /* force */); err != nil {
		log.Error("failed to sync database schema",
			zap.String("instanceName", instance.ResourceID),
			zap.String("databaseName", database.DatabaseName),
			zap.Error(err),
		)
	}

	return terminated, result, err
}

func cutover(ctx context.Context, stores *store.Store, dbFactory *dbfactory.DBFactory, activityManager *activity.Manager, license enterpriseAPI.LicenseService, profile config.Profile, task *store.TaskMessage, statement, schemaVersion string, vcsPushEvent *vcsPlugin.PushEvent, postponeFilename string, migrationContext *base.MigrationContext, errCh <-chan error) (terminated bool, result *api.TaskRunResultPayload, err error) {
	statement = strings.TrimSpace(statement)
	instance, err := stores.GetInstanceV2(ctx, &store.FindInstanceMessage{UID: &task.InstanceID})
	if err != nil {
		return true, nil, err
	}
	database, err := stores.GetDatabaseV2(ctx, &store.FindDatabaseMessage{UID: task.DatabaseID})
	if err != nil {
		return true, nil, err
	}

	mi, err := preMigration(ctx, stores, profile, task, db.Migrate, statement, schemaVersion, vcsPushEvent)
	if err != nil {
		return true, nil, err
	}
	migrationID, schema, err := func() (migrationHistoryID string, updatedSchema string, resErr error) {
		driver, err := dbFactory.GetAdminDatabaseDriver(ctx, instance, database.DatabaseName)
		if err != nil {
			return "", "", err
		}
		defer driver.Close(ctx)

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

		insertedID, err := utils.BeginMigration(ctx, stores, mi, prevSchemaBuf.String(), statement)
		if err != nil {
			if common.ErrorCode(err) == common.MigrationAlreadyApplied {
				return insertedID, prevSchemaBuf.String(), nil
			}
			return "", "", errors.Wrapf(err, "failed to begin migration for issue %s", mi.IssueID)
		}
		startedNs := time.Now().UnixNano()

		defer func() {
			if err := utils.EndMigration(ctx, stores, startedNs, insertedID, updatedSchema, db.BytebaseDatabase, resErr == nil /*isDone*/); err != nil {
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

	return postMigration(ctx, stores, activityManager, license, task, vcsPushEvent, mi, migrationID, schema)
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
