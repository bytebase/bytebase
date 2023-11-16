// Package rollbackrun is the runner for generating rollback statements for DMLs.
package rollbackrun

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	"github.com/bytebase/bytebase/backend/component/state"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/db/mysql"
	"github.com/bytebase/bytebase/backend/store"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// NewRunner creates a new rollback runner.
func NewRunner(profile *config.Profile, store *store.Store, dbFactory *dbfactory.DBFactory, stateCfg *state.State) *Runner {
	return &Runner{
		profile:   profile,
		store:     store,
		dbFactory: dbFactory,
		stateCfg:  stateCfg,
	}
}

// Runner is the rollback runner generating rollback SQL statements.
type Runner struct {
	profile   *config.Profile
	store     *store.Store
	dbFactory *dbfactory.DBFactory
	stateCfg  *state.State
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
			r.stateCfg.RollbackGenerate.Range(func(key, value any) bool {
				task, ok := value.(*store.TaskMessage)
				if !ok {
					return true
				}
				slog.Debug(fmt.Sprintf("Generating rollback SQL for task %d", task.ID))
				ctx, cancel := context.WithCancel(ctx)
				r.stateCfg.RollbackCancel.Store(task.ID, cancel)
				r.generateRollbackSQL(ctx, task)
				cancel()
				r.stateCfg.RollbackGenerate.Delete(key)
				r.stateCfg.RollbackCancel.Delete(task.ID)
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
	taskList, err := r.store.ListTasks(ctx, &api.TaskFind{
		LatestTaskRunStatusList: &[]api.TaskRunStatus{api.TaskRunDone},
		TypeList:                &[]api.TaskType{api.TaskDatabaseDataUpdate},
		Payload:                 "(task.payload->>'rollbackEnabled')::BOOLEAN IS TRUE AND (task.payload->>'threadId'!='' OR task.payload->>'transactionId' != '') AND task.payload->>'rollbackSqlStatus'='PENDING'",
	})
	if err != nil {
		slog.Error("Failed to get running DML tasks", log.BBError(err))
		return
	}
	for _, task := range taskList {
		slog.Debug("retry generate rollback SQL for task", slog.Int("ID", task.ID), slog.String("name", task.Name))
		r.stateCfg.RollbackGenerate.Store(task.ID, task)
	}
}

func (r *Runner) generateRollbackSQL(ctx context.Context, task *store.TaskMessage) {
	defer func() {
		if r := recover(); r != nil {
			err, ok := r.(error)
			if !ok {
				err = errors.Errorf("%v", r)
			}
			slog.Error("Rollback runner PANIC RECOVER", log.BBError(err), log.BBStack("panic-stack"))
		}
	}()

	payload := &api.TaskDatabaseDataUpdatePayload{}
	if err := json.Unmarshal([]byte(task.Payload), payload); err != nil {
		slog.Error("Invalid database data update payload", log.BBError(err))
		return
	}
	instance, err := r.store.GetInstanceV2(ctx, &store.FindInstanceMessage{UID: &task.InstanceID})
	if err != nil {
		slog.Error("Failed to find instance", log.BBError(err))
		return
	}
	database, err := r.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{UID: task.DatabaseID})
	if err != nil {
		slog.Error("Failed to find database", log.BBError(err))
		return
	}
	project, err := r.store.GetProjectV2(ctx, &store.FindProjectMessage{ResourceID: &database.ProjectID})
	if err != nil {
		slog.Error("Failed to find project", log.BBError(err))
		return
	}

	switch instance.Engine {
	case storepb.Engine_MYSQL:
		// TODO(d): support MariaDB.
		r.generateMySQLRollbackSQL(ctx, task, payload, instance, project)
	case storepb.Engine_ORACLE:
		r.generateOracleRollbackSQL(ctx, task, payload, instance, project)
	}
}

func (r *Runner) generateOracleRollbackSQL(ctx context.Context, task *store.TaskMessage, payload *api.TaskDatabaseDataUpdatePayload, instance *store.InstanceMessage, project *store.ProjectMessage) {
	var rollbackSQLStatus api.RollbackSQLStatus
	var rollbackStatement, rollbackError string

	statementsBuffer, err := r.generateOracleRollbackSQLImpl(ctx, payload, instance)
	if err != nil {
		slog.Error("Failed to generate rollback SQL statement", log.BBError(err))
		rollbackSQLStatus = api.RollbackSQLStatusFailed
		rollbackError = err.Error()
	} else {
		rollbackSQLStatus = api.RollbackSQLStatusDone
		rollbackStatement = statementsBuffer.String()
	}

	sheet, err := r.store.CreateSheet(ctx, &store.SheetMessage{
		CreatorID:  api.SystemBotID,
		ProjectUID: project.UID,
		Title:      fmt.Sprintf("Sheet for rolling back task %v", task.ID),
		Statement:  rollbackStatement,
		Visibility: store.ProjectSheet,
		Source:     store.SheetFromBytebaseArtifact,
		Type:       store.SheetForSQL,
	})
	if err != nil {
		slog.Error("failed to create database creation sheet", log.BBError(err))
		return
	}

	patch := &api.TaskPatch{
		ID:                task.ID,
		UpdaterID:         api.SystemBotID,
		RollbackSQLStatus: &rollbackSQLStatus,
		RollbackSheetID:   &sheet.UID,
		RollbackError:     &rollbackError,
	}
	if _, err := r.store.UpdateTaskV2(ctx, patch); err != nil {
		slog.Error("Failed to patch task with the Oracle payload", slog.Int("taskID", task.ID))
		return
	}
	slog.Debug("Rollback SQL generation success", slog.Int("taskID", task.ID))
}

func (r *Runner) generateOracleRollbackSQLImpl(ctx context.Context, payload *api.TaskDatabaseDataUpdatePayload, instance *store.InstanceMessage) (*bytes.Buffer, error) {
	if payload.TransactionID == "" {
		return nil, errors.New("missing transaction ID, may be there is no data change in the transaction")
	}
	driver, err := r.dbFactory.GetAdminDatabaseDriver(ctx, instance, nil /* database */)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get admin database driver")
	}
	defer driver.Close(ctx)

	db := driver.GetDB()
	// Get the undo SQL from the undo log.
	var statements bytes.Buffer
	rows, err := db.QueryContext(ctx, "SELECT undo_sql FROM flashback_transaction_query WHERE xid=HEXTORAW(:1)", payload.TransactionID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to query undo SQL")
	}
	defer rows.Close()

	var undoSQL sql.NullString
	for rows.Next() {
		err := rows.Scan(&undoSQL)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to scan undo SQL")
		}
		if !undoSQL.Valid {
			continue
		}
		if _, err := statements.WriteString(undoSQL.String); err != nil {
			return nil, errors.Wrapf(err, "failed to write undo SQL to buffer")
		}
	}
	if err := rows.Err(); err != nil {
		return nil, errors.Wrapf(err, "failed to iterate undo SQL")
	}
	return &statements, nil
}

func (r *Runner) generateMySQLRollbackSQL(ctx context.Context, task *store.TaskMessage, payload *api.TaskDatabaseDataUpdatePayload, instance *store.InstanceMessage, project *store.ProjectMessage) {
	var rollbackSQLStatus api.RollbackSQLStatus
	var rollbackStatement, rollbackError string

	const binlogSizeLimit = 8 * 1024 * 1024
	rollbackSQL, err := r.generateMySQLRollbackSQLImpl(ctx, payload, binlogSizeLimit, instance)
	if err != nil {
		rollbackSQLStatus = api.RollbackSQLStatusFailed
		if mysql.IsErrExceedSizeLimit(err) {
			rollbackError = fmt.Sprintf("Failed to generate rollback SQL statement. Abort because read more than %vKB from binlog stream.", binlogSizeLimit/1024)
		} else if mysql.IsErrParseBinlogName(err) {
			rollbackError = "Failed to generate rollback SQL statement. Please check if binlog is enabled."
		} else {
			slog.Error("Failed to generate rollback SQL statement", log.BBError(err))
			rollbackError = err.Error()
		}
	} else {
		rollbackSQLStatus = api.RollbackSQLStatusDone
		rollbackStatement = rollbackSQL
	}

	sheet, err := r.store.CreateSheet(ctx, &store.SheetMessage{
		CreatorID:  api.SystemBotID,
		ProjectUID: project.UID,
		Title:      fmt.Sprintf("Sheet for rolling back task %d", task.ID),
		Statement:  rollbackStatement,
		Visibility: store.ProjectSheet,
		Source:     store.SheetFromBytebaseArtifact,
		Type:       store.SheetForSQL,
	})
	if err != nil {
		slog.Error("failed to create database creation sheet", log.BBError(err))
		return
	}
	patch := &api.TaskPatch{
		ID:                task.ID,
		UpdaterID:         api.SystemBotID,
		RollbackSQLStatus: &rollbackSQLStatus,
		RollbackSheetID:   &sheet.UID,
		RollbackError:     &rollbackError,
	}
	if _, err := r.store.UpdateTaskV2(ctx, patch); err != nil {
		slog.Error("Failed to patch task with the MySQL thread ID", slog.Int("taskID", task.ID))
		return
	}
	slog.Debug("Rollback SQL generation success", slog.Int("taskID", task.ID))
}

func (r *Runner) generateMySQLRollbackSQLImpl(ctx context.Context, payload *api.TaskDatabaseDataUpdatePayload, binlogSizeLimit int, instance *store.InstanceMessage) (string, error) {
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	// We cannot support rollback SQL generation for sheets because it can take lots of resources.
	sheet, err := r.store.GetSheet(ctx, &store.FindSheetMessage{UID: &payload.SheetID}, api.SystemBotID)
	if err != nil {
		return "", errors.Wrapf(err, "failed to get sheet %d", payload.SheetID)
	}
	if sheet == nil {
		return "", errors.Errorf("sheet %d not found", payload.SheetID)
	}
	if sheet.Size > common.MaxSheetSizeForRollback {
		return "", errors.Errorf("rollback SQL isn't supported for large sheet")
	}
	basename, seqStart, err := mysql.ParseBinlogName(payload.BinlogFileStart)
	if err != nil {
		return "", errors.WithMessagef(err, "invalid start binlog file name %s", payload.BinlogFileStart)
	}
	_, seqEnd, err := mysql.ParseBinlogName(payload.BinlogFileEnd)
	if err != nil {
		return "", errors.WithMessagef(err, "Invalid end binlog file name %s", payload.BinlogFileEnd)
	}
	binlogFileNameList := mysql.GenBinlogFileNames(basename, seqStart, seqEnd)

	driver, err := r.dbFactory.GetAdminDatabaseDriver(ctx, instance, nil /* database */)
	if err != nil {
		return "", errors.WithMessage(err, "failed to get admin database driver")
	}
	defer driver.Close(ctx)
	list, err := r.store.ListInstanceChangeHistory(ctx, &store.FindInstanceChangeHistoryMessage{
		InstanceID: &instance.UID,
		ID:         &payload.MigrationID,
		ShowFull:   true,
	})
	if err != nil {
		return "", errors.WithMessagef(err, "failed to find migration history with ID %s", payload.MigrationID)
	}
	if len(list) == 0 {
		return "", errors.Errorf("migration history with ID %s not found", payload.MigrationID)
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
	if err := mysqlDriver.CheckBinlogEnabled(ctx); err != nil {
		return "", err
	}
	if err := mysqlDriver.CheckBinlogRowFormat(ctx); err != nil {
		return "", err
	}
	rollbackSQL, err := mysqlDriver.GenerateRollbackSQL(ctx, binlogSizeLimit, binlogFileNameList, payload.BinlogPosStart, payload.BinlogPosEnd, payload.ThreadID, tableMap)
	if err != nil {
		return "", errors.WithMessage(err, "failed to generate rollback SQL statement")
	}

	return rollbackSQL, nil
}
