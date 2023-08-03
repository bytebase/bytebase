package taskrun

import (
	"context"
	"encoding/json"
	"strings"
	"sync/atomic"
	"time"

	"github.com/github/gh-ost/go/base"
	"github.com/github/gh-ost/go/logic"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/state"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
)

// NewSchemaUpdateGhostSyncExecutor creates a schema update (gh-ost) sync task executor.
func NewSchemaUpdateGhostSyncExecutor(store *store.Store, stateCfg *state.State, secret string) Executor {
	return &SchemaUpdateGhostSyncExecutor{
		store:    store,
		stateCfg: stateCfg,
		secret:   secret,
	}
}

// SchemaUpdateGhostSyncExecutor is the schema update (gh-ost) sync task executor.
type SchemaUpdateGhostSyncExecutor struct {
	store    *store.Store
	stateCfg *state.State
	secret   string
}

// RunOnce will run SchemaUpdateGhostSync task once.
// TODO: support cancellation.
func (exec *SchemaUpdateGhostSyncExecutor) RunOnce(ctx context.Context, _ context.Context, task *store.TaskMessage) (terminated bool, result *api.TaskRunResultPayload, err error) {
	payload := &api.TaskDatabaseSchemaUpdateGhostSyncPayload{}
	if err := json.Unmarshal([]byte(task.Payload), payload); err != nil {
		return true, nil, errors.Wrap(err, "invalid database schema update gh-ost sync payload")
	}
	statement, err := exec.store.GetSheetStatementByID(ctx, payload.SheetID)
	if err != nil {
		return true, nil, err
	}

	return exec.runGhostMigration(ctx, exec.store, task, statement)
}

type sharedGhostState struct {
	migrationContext *base.MigrationContext
	errCh            <-chan error
}

func (exec *SchemaUpdateGhostSyncExecutor) runGhostMigration(ctx context.Context, stores *store.Store, task *store.TaskMessage, statement string) (terminated bool, result *api.TaskRunResultPayload, err error) {
	syncDone := make(chan struct{})
	// set buffer size to 1 to unblock the sender because there is no listner if the task is canceled.
	// see PR #2919.
	migrationError := make(chan error, 1)

	statement = strings.TrimSpace(statement)

	tableName, err := utils.GetTableNameFromStatement(statement)
	if err != nil {
		return true, nil, err
	}

	instance, err := exec.store.GetInstanceV2(ctx, &store.FindInstanceMessage{UID: &task.InstanceID})
	if err != nil {
		return true, nil, err
	}
	if instance == nil {
		return true, nil, errors.Errorf("instance %d not found", task.InstanceID)
	}
	adminDataSource := utils.DataSourceFromInstanceWithType(instance, api.Admin)
	if adminDataSource == nil {
		return true, nil, common.Errorf(common.Internal, "admin data source not found for instance %d", instance.UID)
	}

	database, err := exec.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{UID: task.DatabaseID})
	if err != nil {
		return true, nil, err
	}
	if database == nil {
		return true, nil, errors.Errorf("database not found")
	}

	instanceUsers, err := stores.ListInstanceUsers(ctx, &store.FindInstanceUserMessage{InstanceUID: task.InstanceID})
	if err != nil {
		return true, nil, common.Errorf(common.Internal, "failed to find instance user by instanceID %d", task.InstanceID)
	}

	materials := utils.GetSecretMapFromDatabaseMessage(database)
	// To avoid leaking the rendered statement, the error message should use the original statement and not the rendered statement.
	renderedStatement := utils.RenderStatement(statement, materials)

	config, err := utils.GetGhostConfig(task.ID, database, adminDataSource, exec.secret, instanceUsers, tableName, renderedStatement, false, 10000000)
	if err != nil {
		return true, nil, err
	}

	migrationContext, err := utils.NewMigrationContext(config)
	if err != nil {
		return true, nil, errors.Wrap(err, "failed to init migrationContext for gh-ost")
	}

	migrator := logic.NewMigrator(migrationContext, "bb")

	childCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	go func(childCtx context.Context) {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		createdTs := time.Now().Unix()
		for {
			select {
			case <-ticker.C:
				var (
					totalUnit     = atomic.LoadInt64(&migrationContext.RowsEstimate) + atomic.LoadInt64(&migrationContext.RowsDeltaEstimate)
					completedUnit = migrationContext.GetTotalRowsCopied()
					updatedTs     = time.Now().Unix()
				)
				exec.stateCfg.TaskProgress.Store(task.ID, api.Progress{
					TotalUnit:     totalUnit,
					CompletedUnit: completedUnit,
					CreatedTs:     createdTs,
					UpdatedTs:     updatedTs,
				})
				// Since we are using postpone flag file to postpone cutover, it's gh-ost mechanism to set migrationContext.IsPostponingCutOver to 1 after synced and before postpone flag file is removed. We utilize this mechanism here to check if synced.
				if atomic.LoadInt64(&migrationContext.IsPostponingCutOver) > 0 {
					close(syncDone)
					return
				}
			case <-childCtx.Done():
				return
			}
		}
	}(childCtx)

	go func() {
		if err := migrator.Migrate(); err != nil {
			log.Error("failed to run gh-ost migration", zap.Error(err))
			migrationError <- err
			return
		}
		migrationError <- nil
		// we send to migrationError channel anyway because:
		// 1. before syncDone, the gh-ost sync task will receive it.
		// 2. after syncDone, the gh-ost cutover task will receive it.
	}()

	select {
	case <-syncDone:
		exec.stateCfg.GhostTaskState.Store(task.ID, sharedGhostState{migrationContext: migrationContext, errCh: migrationError})
		return true, &api.TaskRunResultPayload{Detail: "sync done"}, nil
	case err := <-migrationError:
		return true, nil, err
	case <-ctx.Done():
		migrationContext.PanicAbort <- errors.New("task canceled")
		return true, nil, errors.New("task canceled")
	}
}
