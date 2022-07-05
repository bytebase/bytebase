package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/common/log"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/plugin/db/mysql"
	"github.com/bytebase/bytebase/resources/mysqlutil"
	"github.com/bytebase/bytebase/store"
	"go.uber.org/zap"
)

// NewPITRCutoverTaskExecutor creates a PITR cutover task executor.
func NewPITRCutoverTaskExecutor(instance mysqlutil.Instance) TaskExecutor {
	return &PITRCutoverTaskExecutor{
		mysqlutil: instance,
	}
}

// PITRCutoverTaskExecutor is the PITR cutover task executor.
type PITRCutoverTaskExecutor struct {
	mysqlutil mysqlutil.Instance
}

// RunOnce will run the PITR cutover task executor once.
func (exec *PITRCutoverTaskExecutor) RunOnce(ctx context.Context, server *Server, task *api.Task) (terminated bool, result *api.TaskRunResultPayload, err error) {
	log.Info("Run PITR cutover task", zap.String("task", task.Name))

	issue, err := getIssueByPipelineID(ctx, server.store, task.PipelineID)
	if err != nil {
		log.Error("failed to fetch containing issue doing pitr cutover task", zap.Error(err))
		return true, nil, err
	}

	// Currently api.TaskDatabasePITRCutoverPayload is empty, so we do not need to unmarshal from task.Payload.

	terminated, result, err = exec.pitrCutover(ctx, task, server, issue)
	if err != nil {
		return terminated, result, err
	}

	payload, err := json.Marshal(api.ActivityPipelineTaskStatusUpdatePayload{
		TaskID:    task.ID,
		OldStatus: task.Status,
		NewStatus: api.TaskDone,
		IssueName: issue.Name,
		TaskName:  task.Name,
	})
	if err != nil {
		log.Error("failed to marshal activity", zap.Error(err))
		return terminated, result, nil
	}

	activityCreate := &api.ActivityCreate{
		CreatorID:   task.UpdaterID,
		ContainerID: issue.ProjectID,
		Type:        api.ActivityDatabaseRecoveryPITRDone,
		Level:       api.ActivityInfo,
		Payload:     string(payload),
		Comment:     fmt.Sprintf("Restore database %s in instance %s successfully.", task.Database.Name, task.Instance.Name),
	}
	activityMeta := ActivityMeta{}
	activityMeta.issue = issue
	if _, err = server.ActivityManager.CreateActivity(ctx, activityCreate, &activityMeta); err != nil {
		log.Error("cannot create an pitr activity", zap.Error(err))
	}

	return terminated, result, nil
}

// The PITR cutover algorithm is:
// 1. Lock tables in the current database by FLUSH TABLES table1, table2, ... WITH READ LOCK.
// 2. Swap the current and PITR database.
// 3. Get the current binlog coordinate and save to the backup metadata.
// 3. Create a backup with type PITR.
// The backup is dumped asynchronously to prevent performance issues by waiting for the backup process for too long.
// And we must check the possible failed/ongoing PITR backup in the recovery process.
func (exec *PITRCutoverTaskExecutor) pitrCutover(ctx context.Context, task *api.Task, server *Server, issue *api.Issue) (terminated bool, result *api.TaskRunResultPayload, err error) {
	driver, err := getAdminDatabaseDriver(ctx, task.Instance, "", "" /* pgInstanceDir */)
	if err != nil {
		return true, nil, err
	}
	defer driver.Close(ctx)

	driverDB, err := driver.GetDbConnection(ctx, "")
	if err != nil {
		return true, nil, err
	}
	conn, err := driverDB.Conn(ctx)
	if err != nil {
		return true, nil, err
	}
	defer func() {
		if err != nil {
			// The conn is used in an asynchronous goroutine in backupDatabaseAfterPITR(), so we should not close it in a normal deferred function here.
			// But if error happens elsewhere, we should close the connection to return it to the database connection pool.
			// In this case, the backup process failure is expected.
			conn.Close()
		}
	}()
	log.Debug("Acquiring table locks in database", zap.String("database", task.Database.Name))
	if err := mysql.FlushTablesWithReadLock(ctx, conn, task.Database.Name); err != nil {
		return true, nil, err
	}
	log.Debug("Swapping the original and PITR database", zap.String("originalDatabase", task.Database.Name))
	pitrDatabaseName, pitrOldDatabaseName, err := mysql.SwapPITRDatabase(ctx, conn, task.Database.Name, issue.CreatedTs)
	if err != nil {
		log.Error("Failed to swap the original and PITR database", zap.String("originalDatabase", task.Database.Name), zap.String("pitrDatabase", pitrDatabaseName), zap.Error(err))
		return true, nil, fmt.Errorf("failed to swap the original and PITR database, error: %w", err)
	}
	log.Debug("Finished swapping the original and PITR database", zap.String("originalDatabase", task.Database.Name), zap.String("pitrDatabase", pitrDatabaseName), zap.String("oldDatabase", pitrOldDatabaseName))

	binlogInfo, err := mysql.GetBinlogInfo(ctx, conn)
	if err != nil {
		return true, nil, err
	}
	payload := api.BackupPayload{BinlogInfo: binlogInfo}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return true, nil, err
	}
	backupPayload := string(payloadBytes)

	log.Debug("Starting new transaction and unlock tables")
	options := sql.TxOptions{ReadOnly: true}
	// Beginning a transaction in the same session will implicitly release existing table locks.
	// ref: https://dev.mysql.com/doc/refman/8.0/en/lock-tables.html, section "Interaction of Table Locking and Transactions".
	txn, err := conn.BeginTx(ctx, &options)
	if err != nil {
		return true, nil, err
	}

	if err := exec.backupDatabaseAfterPITR(ctx, conn, txn, backupPayload, driver, server.store, task.Database, server.profile.DataDir, issue.CreatedTs); err != nil {
		return true, nil, fmt.Errorf("failed to backup database %q after PITR, error: %w", task.Database.Name, err)
	}

	log.Debug("Appending new migration history record")
	m := &db.MigrationInfo{
		ReleaseVersion: server.profile.Version,
		Version:        common.DefaultMigrationVersion(),
		Namespace:      task.Database.Name,
		Database:       task.Database.Name,
		Environment:    task.Database.Instance.Environment.Name,
		Source:         db.MigrationSource(task.Database.Project.WorkflowType),
		Type:           db.Baseline,
		Status:         db.Done,
		Description:    fmt.Sprintf("PITR: restoring database %s", task.Database.Name),
		Creator:        task.Creator.Name,
		IssueID:        strconv.Itoa(issue.ID),
	}

	if _, _, err := driver.ExecuteMigration(ctx, m, "/* pitr cutover */"); err != nil {
		log.Error("Failed to add migration history record", zap.Error(err))
		return true, nil, fmt.Errorf("failed to add migration history record, error: %w", err)
	}

	// Sync database schema after restore is completed.
	server.syncEngineVersionAndSchema(ctx, task.Database.Instance)

	return true, &api.TaskRunResultPayload{
		Detail: fmt.Sprintf("Swapped PITR database for target database %q", task.Database.Name),
	}, nil
}

func (exec *PITRCutoverTaskExecutor) backupDatabaseAfterPITR(ctx context.Context, conn *sql.Conn, txn *sql.Tx, backupPayload string, driver db.Driver, store *store.Store, database *api.Database, dataDir string, suffixTs int64) error {
	backupName := fmt.Sprintf("%s-%s-pitr_%d", api.ProjectShortSlug(database.Project), api.EnvSlug(database.Instance.Environment), suffixTs)
	if err := createBackupDirectory(dataDir, database.ID); err != nil {
		return fmt.Errorf("failed to create backup directory, error: %w", err)
	}
	path := getBackupRelativeFilePath(database.ID, backupName)
	migrationHistoryVersion, err := getLatestSchemaVersion(ctx, driver, database.Name)
	if err != nil {
		return fmt.Errorf("failed to get migration history for database %q, error: %w", database.Name, err)
	}
	backupCreate := &api.BackupCreate{
		CreatorID:               api.SystemBotID,
		DatabaseID:              database.ID,
		Name:                    backupName,
		Type:                    api.BackupTypePITR,
		MigrationHistoryVersion: migrationHistoryVersion,
		StorageBackend:          api.BackupStorageBackendLocal,
		Path:                    path,
	}
	backup, err := store.CreateBackup(ctx, backupCreate)
	if err != nil {
		if common.ErrorCode(err) == common.Conflict {
			return fmt.Errorf("backup name %q already exists", backupName)
		}
		return fmt.Errorf("failed to create backup %q, error: %w", backupName, err)
	}

	backupFile, err := os.Create(filepath.Join(dataDir, path))
	if err != nil {
		return fmt.Errorf("failed to create backup path %q", path)
	}
	binlogDir := getBinlogAbsDir(dataDir, database.Instance.ID)
	if err := createBinlogDir(dataDir, database.Instance.ID); err != nil {
		return err
	}
	mysqlDriver, ok := driver.(*mysql.Driver)
	if !ok {
		log.Error("failed to cast driver to mysql.Driver")
		return fmt.Errorf("[internal] cast driver to mysql.Driver failed")
	}
	mysqlDriver.SetUpForPITR(exec.mysqlutil, binlogDir)

	// Asynchronously dump the database and update the backup state.
	go func() {
		log.Debug("Taking a backup with type AUTOMATIC_PITR_CUTOVER asynchronously")
		defer conn.Close()
		defer txn.Rollback()
		defer backupFile.Close()
		backupPatch := api.BackupPatch{
			ID:        backup.ID,
			Status:    string(api.BackupStatusDone),
			UpdaterID: api.SystemBotID,
			Comment:   "",
			Payload:   backupPayload,
		}
		backupErr := func() error {
			if err := mysqlDriver.DumpTx(ctx, txn, database.Name, backupFile); err != nil {
				return err
			}
			if err := txn.Commit(); err != nil {
				return err
			}
			return nil
		}()
		if backupErr != nil {
			backupPatch.Status = string(api.BackupStatusFailed)
			backupPatch.Comment = fmt.Sprintf("failed to dump backup for database %q, error: %v", database.Name, backupErr)
		}
		if _, err := store.PatchBackup(ctx, &backupPatch); err != nil {
			log.Error("Failed to patch backup metadata for backup after PITR", zap.Error(err))
		}
		log.Debug("Asynchronous backup process after PITR is finished")
	}()

	return nil
}
