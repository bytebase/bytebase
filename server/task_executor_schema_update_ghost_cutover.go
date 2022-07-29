package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"time"

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
		return true, nil, fmt.Errorf("failed to get a single taskDAG for schema update gh-ost cutover task, id: %v, error: %w", task.ID, err)
	}

	syncTaskID := taskDAG.FromTaskID
	defer server.TaskScheduler.sharedTaskState.Delete(syncTaskID)

	syncTask, err := server.store.GetTaskByID(ctx, syncTaskID)
	if err != nil {
		return true, nil, fmt.Errorf("failed to get schema update gh-ost sync task for cutover task, error: %w", err)
	}
	payload := &api.TaskDatabaseSchemaUpdateGhostSyncPayload{}
	if err := json.Unmarshal([]byte(syncTask.Payload), payload); err != nil {
		return true, nil, fmt.Errorf("invalid database schema update gh-ost sync payload: %w", err)
	}

	tableName, err := getTableNameFromStatement(payload.Statement)
	if err != nil {
		return true, nil, fmt.Errorf("failed to parse table name from statement, error: %w", err)
	}

	postponeFilename := getPostponeFlagFilename(syncTaskID, task.Database.ID, task.Database.Name, tableName)

	value, ok := server.TaskScheduler.sharedTaskState.Load(syncTaskID)
	if !ok {
		return true, nil, fmt.Errorf("failed to get gh-ost state from sync task")
	}
	sharedGhost := value.(sharedGhostState)

	return cutover(ctx, server, task, payload.Statement, payload.SchemaVersion, payload.VCSPushEvent, postponeFilename, sharedGhost.errCh)
}

func cutover(ctx context.Context, server *Server, task *api.Task, statement, schemaVersion string, vcsPushEvent *vcsPlugin.PushEvent, postponeFilename string, errCh <-chan error) (terminated bool, result *api.TaskRunResultPayload, err error) {
	statement = strings.TrimSpace(statement)

	mi, err := preMigration(ctx, server, task, db.Migrate, statement, schemaVersion, vcsPushEvent)
	if err != nil {
		return true, nil, err
	}
	migrationID, schema, err := func() (migrationHistoryID int64, updatedSchema string, resErr error) {
		driver, err := getAdminDatabaseDriver(ctx, task.Instance, task.Database.Name, "" /* pgInstanceDir */)
		if err != nil {
			return -1, "", err
		}
		defer driver.Close(ctx)
		needsSetup, err := driver.NeedsSetupMigration(ctx)
		if err != nil {
			return -1, "", fmt.Errorf("failed to check migration setup for instance %q: %w", task.Instance.Name, err)
		}
		if needsSetup {
			return -1, "", common.Errorf(common.MigrationSchemaMissing, "missing migration schema for instance %q", task.Instance.Name)
		}

		executor := driver.(util.MigrationExecutor)

		var prevSchemaBuf bytes.Buffer
		if _, err := driver.Dump(ctx, mi.Database, &prevSchemaBuf, true); err != nil {
			return -1, "", err
		}

		insertedID, err := util.BeginMigration(ctx, executor, mi, prevSchemaBuf.String(), statement, db.BytebaseDatabase)
		if err != nil {
			if common.ErrorCode(err) == common.MigrationAlreadyApplied {
				return insertedID, prevSchemaBuf.String(), nil
			}
			return -1, "", err
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
			return -1, "", fmt.Errorf("failed to remove postpone flag file, error: %w", err)
		}

		if migrationErr := <-errCh; migrationErr != nil {
			return -1, "", fmt.Errorf("failed to run gh-ost migration, err: %w", migrationErr)
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

	return postMigration(ctx, server, task, vcsPushEvent, mi, migrationID, schema)
}

// IsCompleted tells the scheduler if the task execution has completed.
func (exec *SchemaUpdateGhostCutoverTaskExecutor) IsCompleted() bool {
	return atomic.LoadInt32(&exec.completed) == 1
}

// GetProgress returns the task progress.
func (*SchemaUpdateGhostCutoverTaskExecutor) GetProgress() api.Progress {
	return api.Progress{}
}
