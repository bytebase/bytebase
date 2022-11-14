package server

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common/log"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/plugin/db/mysql"
)

var (
	generateRollbackSQLChan = make(chan *api.Task, 100)
)

// NewRollbackRunner creates a new rollback runner.
func NewRollbackRunner(server *Server) *RollbackRunner {
	return &RollbackRunner{
		server: server,
	}
}

// RollbackRunner is the rollback runner generating rollback SQL statements.
type RollbackRunner struct {
	server *Server
}

// Run starts the rollback runner.
func (r *RollbackRunner) Run(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	r.retryGetRollbackSQL(ctx)
	for {
		select {
		case task := <-generateRollbackSQLChan:
			r.getRollbackSQL(ctx, task)
		case <-ctx.Done(): // if cancel() execute
			return
		}
	}
}

// retryGetRollbackSQL retries generating rollback SQL for tasks.
// It is currently called when Bytebase server starts and only rerun unfinished generation.
func (r *RollbackRunner) retryGetRollbackSQL(ctx context.Context) {
	find := &api.TaskFind{
		StatusList: &[]api.TaskStatus{api.TaskRunning},
		TypeList:   &[]api.TaskType{api.TaskDatabaseDataUpdate},
		Payload:    fmt.Sprintf("payload->>'rollbackTaskState'='' OR payload->>'rollbackTaskState'='%s'", api.RollbackTaskRunning),
	}
	taskList, err := r.server.store.FindTask(ctx, find, true)
	if err != nil {
		log.Error("Failed to get running DML tasks", zap.Error(err))
		return
	}
	for _, task := range taskList {
		if task.Instance.Engine != db.MySQL {
			continue
		}
		generateRollbackSQLChan <- task
	}
}

func (r *RollbackRunner) getRollbackSQL(ctx context.Context, task *api.Task) {
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
	if payload.ThreadID == "" ||
		payload.BinlogFileStart == "" ||
		payload.BinlogPosStart == 0 ||
		payload.BinlogFileEnd == "" ||
		payload.BinlogPosEnd == 0 {
		log.Error("Cannot generate rollback SQL statement for the data update task with invalid payload", zap.Any("payload", *payload))
		return
	}

	rollbackSQL, err := r.generateRollbackSQL(ctx, task, payload)
	if err != nil {
		log.Error("Failed to generate rollback SQL statement", zap.Error(err))
		payload.RollbackTaskState = api.RollbackTaskFail
		payload.RollbackError = err.Error()
	}
	payload.RollbackTaskState = api.RollbackTaskSuccess
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
	if _, err := r.server.store.PatchTask(ctx, patch); err != nil {
		log.Error("Failed to patch task with the MySQL thread ID", zap.Int("taskID", task.ID))
		return
	}
	log.Debug("Rollback SQL generation success", zap.Int("taskID", task.ID))
}

func (r *RollbackRunner) generateRollbackSQL(ctx context.Context, task *api.Task, payload *api.TaskDatabaseDataUpdatePayload) (string, error) {
	basename, seqStart, err := mysql.ParseBinlogName(payload.BinlogFileStart)
	if err != nil {
		return "", errors.WithMessagef(err, "invalid start binlog file name %s", payload.BinlogFileStart)
	}
	_, seqEnd, err := mysql.ParseBinlogName(payload.BinlogFileEnd)
	if err != nil {
		return "", errors.WithMessagef(err, "Invalid end binlog file name %s", payload.BinlogFileEnd)
	}
	binlogFileNameList := mysql.GenBinlogFileNames(basename, seqStart, seqEnd)

	driver, err := r.server.getAdminDatabaseDriver(ctx, task.Instance, "")
	if err != nil {
		return "", errors.WithMessage(err, "failed to get admin database driver")
	}
	list, err := driver.FindMigrationHistoryList(ctx, &db.MigrationHistoryFind{ID: &payload.MigrationID})
	if err != nil {
		return "", errors.WithMessagef(err, "failed to find migration history with ID %d", payload.MigrationID)
	}
	if len(list) == 0 {
		return "", errors.WithMessagef(err, "migration history with ID %d not found", payload.MigrationID)
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
