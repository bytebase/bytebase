package server

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
)

// NewSchemaUpdateGhostCutoverTaskExecutor creates a schema update (gh-ost) cutover task executor.
func NewSchemaUpdateGhostCutoverTaskExecutor() TaskExecutor {
	return &SchemaUpdateGhostCutoverTaskExecutor{}
}

// SchemaUpdateGhostCutoverTaskExecutor is the schema update (gh-ost) cutover task executor.
type SchemaUpdateGhostCutoverTaskExecutor struct {
	completed int32
}

// RunOnce will run SchemaUpdateGhostCutover task once.
func (exec *SchemaUpdateGhostCutoverTaskExecutor) RunOnce(ctx context.Context, server *Server, task *api.Task) (terminated bool, result *api.TaskRunResultPayload, err error) {
	defer atomic.StoreInt32(&exec.completed, 1)

	taskDAG, err := server.store.GetTaskDAGByToTaskID(ctx, task.ID)
	if err != nil {
		return true, nil, errors.Wrapf(err, "failed to get a single taskDAG for schema update gh-ost cutover task, id: %v", task.ID)
	}

	syncTaskID := taskDAG.FromTaskID
	defer server.TaskScheduler.sharedTaskState.Delete(syncTaskID)

	syncTask, err := server.store.GetTaskByID(ctx, syncTaskID)
	if err != nil {
		return true, nil, errors.Wrap(err, "failed to get schema update gh-ost sync task for cutover task")
	}
	payload := &api.TaskDatabaseSchemaUpdateGhostSyncPayload{}
	if err := json.Unmarshal([]byte(syncTask.Payload), payload); err != nil {
		return true, nil, errors.Wrap(err, "invalid database schema update gh-ost sync payload")
	}

	tableName, err := getTableNameFromStatement(payload.Statement)
	if err != nil {
		return true, nil, errors.Wrap(err, "failed to parse table name from statement")
	}

	postponeFilename := getPostponeFlagFilename(syncTaskID, task.Database.ID, task.Database.Name, tableName)

	value, ok := server.TaskScheduler.sharedTaskState.Load(syncTaskID)
	if !ok {
		return true, nil, errors.Errorf("failed to get gh-ost state from sync task")
	}
	sharedGhost := value.(sharedGhostState)

	return cutover(ctx, server, task, payload.Statement, payload.SchemaVersion, payload.VCSPushEvent, postponeFilename, sharedGhost.migrationContext, sharedGhost.errCh)
}

func cutover(ctx context.Context, server *Server, task *api.Task, statement, schemaVersion string, vcsPushEvent *vcsPlugin.PushEvent, postponeFilename string, migrationContext *base.MigrationContext, errCh <-chan error) (terminated bool, result *api.TaskRunResultPayload, err error) {
	statement = strings.TrimSpace(statement)

	mi, err := preMigration(ctx, server.store, server.profile, task, db.Migrate, statement, schemaVersion, vcsPushEvent)
	if err != nil {
		return true, nil, err
	}
	migrationID, schema, err := func() (migrationHistoryID int64, updatedSchema string, resErr error) {
		driver, err := server.dbFactory.GetAdminDatabaseDriver(ctx, task.Instance, task.Database.Name)
		if err != nil {
			return -1, "", err
		}
		defer driver.Close(ctx)
		needsSetup, err := driver.NeedsSetupMigration(ctx)
		if err != nil {
			return -1, "", errors.Wrapf(err, "failed to check migration setup for instance %q", task.Instance.Name)
		}
		if needsSetup {
			return -1, "", common.Errorf(common.MigrationSchemaMissing, "missing migration schema for instance %q", task.Instance.Name)
		}

		executor := driver.(util.MigrationExecutor)

		var prevSchemaBuf bytes.Buffer
		if _, err := driver.Dump(ctx, mi.Database, &prevSchemaBuf, true); err != nil {
			return -1, "", err
		}

		// wait for heartbeat lag.
		// try to make the time gap between the migration history insertion and the actual cutover as close as possible.
		cancelled := waitForCutover(ctx, migrationContext)
		if cancelled {
			return -1, "", errors.Errorf("cutover poller cancelled")
		}

		insertedID, err := util.BeginMigration(ctx, executor, mi, prevSchemaBuf.String(), statement, db.BytebaseDatabase)
		if err != nil {
			if common.ErrorCode(err) == common.MigrationAlreadyApplied {
				return insertedID, prevSchemaBuf.String(), nil
			}
			return -1, "", errors.Wrapf(err, "failed to begin migration for issue %s", mi.IssueID)
		}
		startedNs := time.Now().UnixNano()

		defer func() {
			if err := util.EndMigration(ctx, executor, startedNs, insertedID, updatedSchema, db.BytebaseDatabase, resErr == nil /*isDone*/); err != nil {
				log.Error("failed to update migration history record",
					zap.Error(err),
					zap.Int64("migration_id", migrationHistoryID),
				)
			}
		}()

		if err := os.Remove(postponeFilename); err != nil {
			return -1, "", errors.Wrap(err, "failed to remove postpone flag file")
		}

		if migrationErr := <-errCh; migrationErr != nil {
			return -1, "", errors.Wrapf(migrationErr, "failed to run gh-ost migration")
		}

		var afterSchemaBuf bytes.Buffer
		if _, err := executor.Dump(ctx, mi.Database, &afterSchemaBuf, true /*schemaOnly*/); err != nil {
			return -1, "", util.FormatError(err)
		}

		return insertedID, afterSchemaBuf.String(), nil
	}()
	if err != nil {
		return true, nil, err
	}

	return postMigration(ctx, server.store, server.ActivityManager, server.profile, task, vcsPushEvent, mi, migrationID, schema)
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

// IsCompleted tells the scheduler if the task execution has completed.
func (exec *SchemaUpdateGhostCutoverTaskExecutor) IsCompleted() bool {
	return atomic.LoadInt32(&exec.completed) == 1
}

// GetProgress returns the task progress.
func (*SchemaUpdateGhostCutoverTaskExecutor) GetProgress() api.Progress {
	return api.Progress{}
}
