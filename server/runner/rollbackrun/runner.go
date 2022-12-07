// Package rollbackrun is the runner for generating rollback statements for DMLs.
package rollbackrun

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common/log"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/plugin/db/mysql"
	"github.com/bytebase/bytebase/server/component/dbfactory"
	"github.com/bytebase/bytebase/server/component/state"
	"github.com/bytebase/bytebase/store"
)

// NewRunner creates a new rollback runner.
func NewRunner(store *store.Store, dbFactory *dbfactory.DBFactory) *Runner {
	return &Runner{
		store:     store,
		dbFactory: dbFactory,
	}
}

// Runner is the rollback runner generating rollback SQL statements.
type Runner struct {
	store     *store.Store
	dbFactory *dbfactory.DBFactory
}

// Run starts the rollback runner.
func (r *Runner) Run(ctx context.Context, wg *sync.WaitGroup) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	defer wg.Done()
	r.retryGenerateRollbackSQL(ctx)
	for {
		select {
		case <-ticker.C:
			state.RollbackGenerateMap.Range(func(key, value any) bool {
				task := value.(*api.Task)
				log.Debug(fmt.Sprintf("Generating rollback SQL for task %d", task.ID))
				r.generateRollbackSQL(ctx, task)
				state.RollbackGenerateMap.Delete(key)
				return true
			})
		case <-ctx.Done(): // if cancel() execute
			return
		}
	}
}

// retryGenerateRollbackSQL retries generating rollback SQL for tasks.
// It is currently called when Bytebase server starts and only rerun unfinished generation.
func (r *Runner) retryGenerateRollbackSQL(ctx context.Context) {
	find := &api.TaskFind{
		StatusList: &[]api.TaskStatus{api.TaskDone},
		TypeList:   &[]api.TaskType{api.TaskDatabaseDataUpdate},
		Payload:    "payload->>'threadID'!='' AND payload->>'rollbackError' IS NULL AND payload->>'rollbackStatement' IS NULL",
	}
	taskList, err := r.store.FindTask(ctx, find, true)
	if err != nil {
		log.Error("Failed to get running DML tasks", zap.Error(err))
		return
	}
	for _, task := range taskList {
		log.Debug("retry generate rollback SQL for task", zap.Int("ID", task.ID), zap.String("name", task.Name))
		state.RollbackGenerateMap.Store(task.ID, task)
	}
}

func (r *Runner) generateRollbackSQL(ctx context.Context, task *api.Task) {
	defer func() {
		if r := recover(); r != nil {
			err, ok := r.(error)
			if !ok {
				err = errors.Errorf("%v", r)
			}
			log.Error("Rollback runner PANIC RECOVER", zap.Error(err), zap.Stack("panic-stack"))
		}
	}()

	payload := &api.TaskDatabaseDataUpdatePayload{}
	if err := json.Unmarshal([]byte(task.Payload), payload); err != nil {
		log.Error("Invalid database data update payload", zap.Error(err))
		return
	}

	rollbackSQL, err := r.generateRollbackSQLImpl(ctx, task, payload)
	if err != nil {
		log.Error("Failed to generate rollback SQL statement", zap.Error(err))
		payload.RollbackError = err.Error()
	}
	payload.RollbackStatement = rollbackSQL

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		log.Error("Failed to marshal task payload", zap.Error(err))
		return
	}
	payloadString := string(payloadBytes)
	patch := &api.TaskPatch{
		ID:        task.ID,
		UpdaterID: api.SystemBotID,
		Payload:   &payloadString,
	}
	if _, err := r.store.PatchTask(ctx, patch); err != nil {
		log.Error("Failed to patch task with the MySQL thread ID", zap.Int("taskID", task.ID))
		return
	}
	log.Debug("Rollback SQL generation success", zap.Int("taskID", task.ID))
}

func (r *Runner) generateRollbackSQLImpl(ctx context.Context, task *api.Task, payload *api.TaskDatabaseDataUpdatePayload) (string, error) {
	basename, seqStart, err := mysql.ParseBinlogName(payload.BinlogFileStart)
	if err != nil {
		return "", errors.WithMessagef(err, "invalid start binlog file name %s", payload.BinlogFileStart)
	}
	_, seqEnd, err := mysql.ParseBinlogName(payload.BinlogFileEnd)
	if err != nil {
		return "", errors.WithMessagef(err, "Invalid end binlog file name %s", payload.BinlogFileEnd)
	}
	binlogFileNameList := mysql.GenBinlogFileNames(basename, seqStart, seqEnd)

	driver, err := r.dbFactory.GetAdminDatabaseDriver(ctx, task.Instance, "")
	if err != nil {
		return "", errors.WithMessage(err, "failed to get admin database driver")
	}
	list, err := driver.FindMigrationHistoryList(ctx, &db.MigrationHistoryFind{ID: &payload.MigrationID})
	if err != nil {
		return "", errors.WithMessagef(err, "failed to find migration history with ID %d", payload.MigrationID)
	}
	if len(list) == 0 {
		return "", errors.Errorf("migration history with ID %d not found", payload.MigrationID)
	}
	if len(list) > 1 {
		return "", errors.Errorf("found %d migration history record, expecting one", len(list))
	}
	history := list[0]
	tableMap, err := mysql.GetTableColumns(history.Schema)
	if err != nil {
		return "", errors.WithMessage(err, "failed to parse the schema")
	}

	mysqlDriver, ok := driver.(*mysql.Driver)
	if !ok {
		return "", errors.Errorf("failed to cast driver to mysql.Driver")
	}
	rollbackSQL, err := mysqlDriver.GenerateRollbackSQL(ctx, binlogFileNameList, payload.BinlogPosStart, payload.BinlogPosEnd, payload.ThreadID, tableMap)
	if err != nil {
		return "", errors.WithMessage(err, "failed to generate rollback SQL statement")
	}

	return rollbackSQL, nil
}
