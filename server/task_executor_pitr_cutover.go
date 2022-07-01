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
// 4. dump backup with type AUTOMATIC_PITR_CUTOVER.
// 5. Unlock tables in the current database.
// The step 4 is done asynchronously to prevent performance issues by waiting for the backup process for too long.
// And we must check the possible failed/ongoing AUTOMATIC_PITR_CUTOVER backup in the recovery process.
func (exec *PITRCutoverTaskExecutor) pitrCutover(ctx context.Context, task *api.Task, server *Server, issue *api.Issue) (terminated bool, result *api.TaskRunResultPayload, err error) {
	driver, err := getAdminDatabaseDriver(ctx, task.Instance, "", "" /* pgInstanceDir */)
	if err != nil {
		return true, nil, err
	}
	defer driver.Close(ctx)

	binlogDir := getBinlogAbsDir(server.profile.DataDir, task.Instance.ID)
	if err := createBinlogDir(server.profile.DataDir, task.Instance.ID); err != nil {
		return true, nil, err
	}

	driverDB, err := driver.GetDbConnection(ctx, "")
	if err != nil {
		return true, nil, err
	}
	conn, err := driverDB.Conn(ctx)
	if err != nil {
		return true, nil, err
	}
	defer conn.Close()

	backup, err := createBackupMetadataAfterPITR(ctx, driver, server.store, task.Database, server.profile.DataDir, issue.CreatedTs)
	if err != nil {
		return true, nil, fmt.Errorf("failed to create backup metadata, error: %w", err)
	}

	mysqlDriver, ok := driver.(*mysql.Driver)
	if !ok {
		log.Error("failed to cast driver to mysql.Driver")
		return true, nil, fmt.Errorf("[internal] cast driver to mysql.Driver failed")
	}
	mysqlDriver.SetUpForPITR(exec.mysqlutil, binlogDir)

	log.Debug("Swapping the original and PITR database.", zap.String("instance", task.Instance.Name), zap.String("originalDatabase", task.Database.Name))
	pitrDatabaseName, pitrOldDatabaseName, err := mysql.SwapPITRDatabase(ctx, conn, task.Database.Name, issue.CreatedTs)
	if err := swapAndBackupDatabasesWithLock(ctx, server.store, mysqlDriver, conn, task.Database, backup, issue.CreatedTs, server.profile.DataDir); err != nil {
		log.Error("Failed to swap databases and backup the original database", zap.Error(err))
		return true, nil, fmt.Errorf("failed to swap databases and backup the original database, error: %w", err)
	}
	log.Debug("Finish swapping the original and PITR database", zap.String("original_database", task.Database.Name), zap.String("pitr_database", pitrDatabaseName), zap.String("old_database", pitrOldDatabaseName))

	log.Debug("Appending new migration history record...")
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

func swapAndBackupDatabasesWithLock(ctx context.Context, store *store.Store, mysqlDriver *mysql.Driver, conn *sql.Conn, database *api.Database, backup *api.Backup, suffixTs int64, dataDir string) error {
	if err := mysql.FlushTablesWithReadLock(ctx, conn, database.Name); err != nil {
		return err
	}
	pitrDatabaseName, pitrOldDatabaseName, err := mysql.SwapPITRDatabase(ctx, conn, database.Name, suffixTs)
	if err != nil {
		log.Error("Failed to swap the original and PITR database", zap.String("originalDatabase", database.Name), zap.String("pitrDatabase", pitrDatabaseName), zap.Error(err))
		return fmt.Errorf("failed to swap the original and PITR database, error: %w", err)
	}
	log.Debug("Finish swapping the original and PITR database", zap.String("originalDatabase", database.Name), zap.String("pitrDatabase", pitrDatabaseName), zap.String("oldDatabase", pitrOldDatabaseName))

	binlogInfo, err := mysql.GetBinlogInfo(ctx, conn)
	if err != nil {
		return err
	}
	payload := api.BackupPayload{BinlogInfo: binlogInfo}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	backupPayload := string(payloadBytes)

	options := sql.TxOptions{ReadOnly: true}
	// Beginning a transaction in the same session will implicitly release existing table locks.
	// ref: https://dev.mysql.com/doc/refman/8.0/en/lock-tables.html, section "Interaction of Table Locking and Transactions".
	txn, err := conn.BeginTx(ctx, &options)
	if err != nil {
		return err
	}

	// Asynchronously dump the database and update the backup state.
	go func() {
		backupPatch := api.BackupPatch{
			ID:        backup.ID,
			Status:    string(api.BackupStatusDone),
			UpdaterID: api.SystemBotID,
			Comment:   "",
			Payload:   backupPayload,
		}
		log.Debug("Taking a backup with type AUTOMATIC_PITR_CUTOVER...")
		backupErr := func() error {
			defer txn.Rollback()
			backupFile, err := os.Create(filepath.Join(dataDir, backup.Path))
			if err != nil {
				return fmt.Errorf("failed to create backup path %q", backup.Path)
			}
			defer backupFile.Close()
			if err := mysqlDriver.DumpTx(ctx, txn, database.Name, backupFile); err != nil {
				return err
			}
			if err := txn.Commit(); err != nil {
				return err
			}
			return nil
		}()
		log.Debug("Backup process finish")
		if backupErr != nil {
			backupPatch.Status = string(api.BackupStatusFailed)
			backupPatch.Comment = fmt.Sprintf("failed to dump backup for database %q, error: %v", database.Name, backupErr)
		}
		if _, err := store.PatchBackup(ctx, &backupPatch); err != nil {
			log.Error("Failed to patch backup metadata for backup after PITR", zap.Error(err))
		}
	}()

	return nil
}

func createBackupMetadataAfterPITR(ctx context.Context, driver db.Driver, store *store.Store, database *api.Database, dataDir string, suffixTs int64) (*api.Backup, error) {
	backupName := fmt.Sprintf("%s-%s-pitr_%d", api.ProjectShortSlug(database.Project), api.EnvSlug(database.Instance.Environment), suffixTs)
	if err := createBackupDirectory(dataDir, database.ID); err != nil {
		return nil, fmt.Errorf("failed to create backup directory, error: %w", err)
	}
	path := getBackupRelativeFilePath(database.ID, backupName)
	migrationHistoryVersion, err := getLatestSchemaVersion(ctx, driver, database.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to get migration history for database %q, error: %w", database.Name, err)
	}
	backupCreate := &api.BackupCreate{
		CreatorID:               api.SystemBotID,
		DatabaseID:              database.ID,
		Name:                    backupName,
		Type:                    api.BackupTypeAutomaticPITRCutover,
		MigrationHistoryVersion: migrationHistoryVersion,
		StorageBackend:          api.BackupStorageBackendLocal,
		Path:                    path,
	}
	backup, err := store.CreateBackup(ctx, backupCreate)
	if err != nil {
		if common.ErrorCode(err) == common.Conflict {
			return nil, fmt.Errorf("backup name %q already exists", backupName)
		}
		return nil, fmt.Errorf("failed to create backup %q, error: %w", backupName, err)
	}
	return backup, nil
}
