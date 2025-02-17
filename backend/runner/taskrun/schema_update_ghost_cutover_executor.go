package taskrun

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/github/gh-ost/go/base"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	"github.com/bytebase/bytebase/backend/component/ghost"
	"github.com/bytebase/bytebase/backend/component/state"
	enterprise "github.com/bytebase/bytebase/backend/enterprise/api"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/runner/schemasync"
	"github.com/bytebase/bytebase/backend/store"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// NewSchemaUpdateGhostCutoverExecutor creates a schema update (gh-ost) cutover task executor.
func NewSchemaUpdateGhostCutoverExecutor(store *store.Store, dbFactory *dbfactory.DBFactory, license enterprise.LicenseService, stateCfg *state.State, schemaSyncer *schemasync.Syncer, profile *config.Profile) Executor {
	return &SchemaUpdateGhostCutoverExecutor{
		store:        store,
		dbFactory:    dbFactory,
		license:      license,
		stateCfg:     stateCfg,
		schemaSyncer: schemaSyncer,
		profile:      profile,
	}
}

// SchemaUpdateGhostCutoverExecutor is the schema update (gh-ost) cutover task executor.
type SchemaUpdateGhostCutoverExecutor struct {
	store        *store.Store
	dbFactory    *dbfactory.DBFactory
	license      enterprise.LicenseService
	stateCfg     *state.State
	schemaSyncer *schemasync.Syncer
	profile      *config.Profile
}

// RunOnce will run SchemaUpdateGhostCutover task once.
// TODO: support cancellation.
func (e *SchemaUpdateGhostCutoverExecutor) RunOnce(ctx context.Context, taskContext context.Context, task *store.TaskMessage, taskRunUID int) (bool, *storepb.TaskRunResult, error) {
	if len(task.DependsOn) != 1 {
		return true, nil, errors.Errorf("failed to find task dag for ToTask %v", task.ID)
	}
	syncTaskID := task.DependsOn[0]
	defer e.stateCfg.GhostTaskState.Delete(syncTaskID)

	instance, err := e.store.GetInstanceV2(ctx, &store.FindInstanceMessage{ResourceID: &task.InstanceID})
	if err != nil {
		return true, nil, err
	}
	database, err := e.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{InstanceID: &task.InstanceID, DatabaseName: task.DatabaseName})
	if err != nil {
		return true, nil, err
	}

	syncTask, err := e.store.GetTaskV2ByID(ctx, syncTaskID)
	if err != nil {
		return true, nil, errors.Wrap(err, "failed to get schema update gh-ost sync task for cutover task")
	}
	payload := &storepb.TaskDatabaseUpdatePayload{}
	if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(syncTask.Payload), payload); err != nil {
		return true, nil, errors.Wrap(err, "invalid database schema update gh-ost sync payload")
	}
	sheetID := int(payload.SheetId)
	statement, err := e.store.GetSheetStatementByID(ctx, sheetID)
	if err != nil {
		return true, nil, errors.Wrapf(err, "failed to get sheet statement by id: %d", sheetID)
	}

	postponeFilename := ghost.GetPostponeFlagFilename(syncTaskID)

	value, ok := e.stateCfg.GhostTaskState.Load(syncTaskID)
	if !ok {
		return true, nil, errors.Errorf("failed to get gh-ost state from sync task")
	}
	sharedGhost, ok := value.(sharedGhostState)
	if !ok {
		return true, nil, errors.Errorf("failed to convert shared gh-ost state")
	}

	terminated, result, err := cutover(ctx, taskContext, e.store, e.dbFactory, e.profile, e.schemaSyncer, task, taskRunUID, statement, sheetID, payload.SchemaVersion, postponeFilename, sharedGhost.migrationContext, sharedGhost.errCh)
	if err := e.schemaSyncer.SyncDatabaseSchema(ctx, database, false /* force */); err != nil {
		slog.Error("failed to sync database schema",
			slog.String("instanceName", instance.ResourceID),
			slog.String("databaseName", database.DatabaseName),
			log.BBError(err),
		)
	}

	return terminated, result, err
}

func cutover(ctx context.Context, taskContext context.Context, stores *store.Store, dbFactory *dbfactory.DBFactory, profile *config.Profile, syncer *schemasync.Syncer, task *store.TaskMessage, taskRunUID int, statement string, sheetID int, schemaVersion string, postponeFilename string, migrationContext *base.MigrationContext, errCh <-chan error) (terminated bool, result *storepb.TaskRunResult, err error) {
	statement = strings.TrimSpace(statement)
	// wait for heartbeat lag.
	// try to make the time gap between the migration history insertion and the actual cutover as close as possible.
	cancelled := waitForCutover(ctx, taskContext, migrationContext)
	if cancelled {
		err := errors.Errorf("cutover context cancelled")
		migrationContext.PanicAbort <- err
		return true, nil, err
	}

	mc, err := getMigrationInfo(ctx, stores, profile, syncer, task, db.Migrate, schemaVersion, &sheetID, taskRunUID, dbFactory)
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
	// TODO(p0ny): we may need to defer execFunc to do the cleanup always.
	// And we might want to move the check that determines if the task should be skipped
	// to the sync executor.
	skipped, err := executeMigrationWithFunc(ctx, ctx, stores, mc, statement, execFunc, db.ExecuteOptions{})
	if err != nil {
		return true, nil, err
	}

	return postMigration(ctx, stores, mc, skipped)
}

func waitForCutover(ctx context.Context, taskContext context.Context, migrationContext *base.MigrationContext) bool {
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
		case <-taskContext.Done():
			return true
		}
	}
}
