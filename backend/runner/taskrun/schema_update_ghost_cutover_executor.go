package taskrun

import (
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
// TODO: support cancellation.
func (e *SchemaUpdateGhostCutoverExecutor) RunOnce(ctx context.Context, _ context.Context, task *store.TaskMessage) (bool, *api.TaskRunResultPayload, error) {
	if len(task.BlockedBy) != 1 {
		return true, nil, errors.Errorf("failed to find task dag for ToTask %v", task.ID)
	}
	syncTaskID := task.BlockedBy[0]
	defer e.stateCfg.GhostTaskState.Delete(syncTaskID)

	instance, err := e.store.GetInstanceV2(ctx, &store.FindInstanceMessage{UID: &task.InstanceID})
	if err != nil {
		return true, nil, err
	}
	database, err := e.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{UID: task.DatabaseID})
	if err != nil {
		return true, nil, err
	}

	syncTask, err := e.store.GetTaskV2ByID(ctx, syncTaskID)
	if err != nil {
		return true, nil, errors.Wrap(err, "failed to get schema update gh-ost sync task for cutover task")
	}
	payload := &api.TaskDatabaseSchemaUpdateGhostSyncPayload{}
	if err := json.Unmarshal([]byte(syncTask.Payload), payload); err != nil {
		return true, nil, errors.Wrap(err, "invalid database schema update gh-ost sync payload")
	}
	statement, err := e.store.GetSheetStatementByID(ctx, payload.SheetID)
	if err != nil {
		return true, nil, errors.Wrapf(err, "failed to get sheet statement by id: %d", payload.SheetID)
	}
	materials := utils.GetSecretMapFromDatabaseMessage(database)
	// To avoid leaking the rendered statement, the error message should use the original statement and not the rendered statement.
	renderedStatement := utils.RenderStatement(statement, materials)

	tableName, err := utils.GetTableNameFromStatement(renderedStatement)
	if err != nil {
		return true, nil, common.Wrapf(err, common.Internal, "failed to parse table name from statement, statement: %v", statement)
	}

	postponeFilename := utils.GetPostponeFlagFilename(syncTaskID, database.UID, database.DatabaseName, tableName)

	value, ok := e.stateCfg.GhostTaskState.Load(syncTaskID)
	if !ok {
		return true, nil, errors.Errorf("failed to get gh-ost state from sync task")
	}
	sharedGhost := value.(sharedGhostState)

	// not using the rendered statement here because we want to avoid leaking the rendered statement
	terminated, result, err := cutover(ctx, e.store, e.dbFactory, e.activityManager, e.license, e.profile, task, statement, payload.SheetID, payload.SchemaVersion, payload.VCSPushEvent, postponeFilename, sharedGhost.migrationContext, sharedGhost.errCh)
	if err := e.schemaSyncer.SyncDatabaseSchema(ctx, database, true /* force */); err != nil {
		log.Error("failed to sync database schema",
			zap.String("instanceName", instance.ResourceID),
			zap.String("databaseName", database.DatabaseName),
			zap.Error(err),
		)
	}

	return terminated, result, err
}

func cutover(ctx context.Context, stores *store.Store, dbFactory *dbfactory.DBFactory, activityManager *activity.Manager, license enterpriseAPI.LicenseService, profile config.Profile, task *store.TaskMessage, statement string, sheetID int, schemaVersion string, vcsPushEvent *vcsPlugin.PushEvent, postponeFilename string, migrationContext *base.MigrationContext, errCh <-chan error) (terminated bool, result *api.TaskRunResultPayload, err error) {
	statement = strings.TrimSpace(statement)
	instance, err := stores.GetInstanceV2(ctx, &store.FindInstanceMessage{UID: &task.InstanceID})
	if err != nil {
		return true, nil, err
	}
	database, err := stores.GetDatabaseV2(ctx, &store.FindDatabaseMessage{UID: task.DatabaseID})
	if err != nil {
		return true, nil, err
	}
	// wait for heartbeat lag.
	// try to make the time gap between the migration history insertion and the actual cutover as close as possible.
	cancelled := waitForCutover(ctx, migrationContext)
	if cancelled {
		return true, nil, errors.Errorf("cutover poller cancelled")
	}

	mi, err := getMigrationInfo(ctx, stores, profile, task, db.Migrate, statement, schemaVersion, vcsPushEvent)
	if err != nil {
		return true, nil, err
	}

	execFunc := func(_ context.Context, _ string) error {
		if err := os.Remove(postponeFilename); err != nil {
			return errors.Wrap(err, "failed to remove postpone flag file")
		}
		if migrationErr := <-errCh; migrationErr != nil {
			return errors.Wrapf(migrationErr, "failed to run gh-ost migration")
		}
		return nil
	}
	driver, err := dbFactory.GetAdminDatabaseDriver(ctx, instance, database)
	if err != nil {
		return true, nil, err
	}
	defer driver.Close(ctx)
	migrationID, schema, err := utils.ExecuteMigrationWithFunc(ctx, ctx, stores, driver, mi, statement, &sheetID, execFunc)
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
