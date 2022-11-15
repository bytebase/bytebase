package server

import (
	"context"
	"encoding/json"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common/log"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/plugin/db/mysql"
)

const (
	rollbackSQLChanLen          = 100
	generateRollbackSQLInterval = 10 * time.Second
)

var (
	generateRollbackSQLChan  = make(chan *api.Task, rollbackSQLChanLen)
	hasMissedRollbackSQLTask int32
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
	ticker := time.NewTicker(generateRollbackSQLInterval)
	defer ticker.Stop()
	r.retryGetRollbackSQL(ctx, rollbackSQLChanLen)
	for {
		select {
		case task := <-generateRollbackSQLChan:
			r.getRollbackSQL(ctx, task)
		case <-ticker.C:
			if atomic.LoadInt32(&hasMissedRollbackSQLTask) == 1 && len(generateRollbackSQLChan) == 0 {
				r.retryGetRollbackSQL(ctx, rollbackSQLChanLen-len(generateRollbackSQLChan))
				atomic.StoreInt32(&hasMissedRollbackSQLTask, 0)
			}
		case <-ctx.Done(): // if cancel() execute
			return
		}
	}
}

// retryGetRollbackSQL retries generating rollback SQL for tasks.
// It is currently called when Bytebase server starts and only rerun unfinished generation.
func (r *RollbackRunner) retryGetRollbackSQL(ctx context.Context, limit int) {
	find := &api.TaskFind{
		StatusList: &[]api.TaskStatus{api.TaskRunning},
		TypeList:   &[]api.TaskType{api.TaskDatabaseDataUpdate},
		Payload:    "payload->>'threadID'!='' AND payload->>'rollbackError' IS NULL AND payload->>'rollbackStatement' IS NULL",
		// Limit so that the query result will not block on the channel and result in a deadlock.
		Limit: limit,
	}
	taskList, err := r.server.store.FindTask(ctx, find, true)
	if err != nil {
		log.Error("Failed to get running DML tasks", zap.Error(err))
		return
	}
	for _, task := range taskList {
		// Do not block the runner if the task executor is also sending task to the channel.
		// Otherwise the runner is blocked and rollback SQL generation is in a deadlock.
		select {
		case generateRollbackSQLChan <- task:
		default:
		}
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

	rollbackSQL, err := r.generateRollbackSQL(ctx, task, payload)
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
	if len(list) > 1 {
		return "", errors.WithMessagef(err, "found %d migration history record, expecting one", len(list))
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
