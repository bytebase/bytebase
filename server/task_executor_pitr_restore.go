package server

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/common/log"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/plugin/db/mysql"
	"github.com/bytebase/bytebase/plugin/db/util"
	"github.com/bytebase/bytebase/store"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

// NewPITRRestoreTaskExecutor creates a PITR restore task executor.
func NewPITRRestoreTaskExecutor() TaskExecutor {
	return &PITRRestoreTaskExecutor{}
}

// PITRRestoreTaskExecutor is the PITR restore task executor.
type PITRRestoreTaskExecutor struct {
	completed int32
	progress  atomic.Value // api.Progress
}

// RunOnce will run the PITR restore task executor once.
func (exec *PITRRestoreTaskExecutor) RunOnce(ctx context.Context, server *Server, task *api.Task) (terminated bool, result *api.TaskRunResultPayload, err error) {
	log.Info("Run PITR restore task", zap.String("task", task.Name))
	defer atomic.StoreInt32(&exec.completed, 1)

	payload := api.TaskDatabasePITRRestorePayload{}
	if err := json.Unmarshal([]byte(task.Payload), &payload); err != nil {
		return true, nil, errors.Wrapf(err, "invalid PITR restore payload: %s", task.Payload)
	}

	if (payload.BackupID == nil) == (payload.PointInTimeTs == nil) {
		return true, nil, fmt.Errorf("only one of BackupID and time point can be set")
	}

	if !((payload.DatabaseName == nil) == (payload.TargetInstanceID == nil)) {
		return true, nil, fmt.Errorf("DatabaseName and TargetInstanceID must be both set or unset")
	}

	// There are 2 * 2 = 4 kinds of task by combination of the following cases:
	// 1. in-place or restore to new database: the latter does not create database with _pitr/_del suffix
	// 2. backup restore or Point-in-Time restore: the former does not apply binlog/wal

	if payload.BackupID != nil {
		// Restore Backup
		resultPayload, err := exec.doBackupRestore(ctx, server, task, payload)
		return true, resultPayload, err
	}

	resultPayload, err := exec.doPITRRestore(ctx, server, task, payload)
	return true, resultPayload, err
}

// IsCompleted tells the scheduler if the task execution has completed.
func (exec *PITRRestoreTaskExecutor) IsCompleted() bool {
	return atomic.LoadInt32(&exec.completed) == 1
}

// GetProgress returns the task progress.
func (exec *PITRRestoreTaskExecutor) GetProgress() api.Progress {
	progress := exec.progress.Load()
	if progress == nil {
		return api.Progress{}
	}
	return progress.(api.Progress)
}

func (exec *PITRRestoreTaskExecutor) doBackupRestore(ctx context.Context, server *Server, task *api.Task, payload api.TaskDatabasePITRRestorePayload) (*api.TaskRunResultPayload, error) {
	backup, err := server.store.GetBackupByID(ctx, *payload.BackupID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find backup with ID %d", *payload.BackupID)
	}
	if backup == nil {
		return nil, fmt.Errorf("backup with ID %d not found", *payload.BackupID)
	}

	// TODO(dragonly): We should let users restore the backup even if the source database is gone.
	sourceDatabase, err := server.store.GetDatabase(ctx, &api.DatabaseFind{ID: &backup.DatabaseID})
	if err != nil {
		return nil, fmt.Errorf("failed to find database for the backup: %w", err)
	}
	if sourceDatabase == nil {
		return nil, fmt.Errorf("source database ID not found %v", backup.DatabaseID)
	}

	if payload.TargetInstanceID == nil {
		// Backup restore in place
		if task.Instance.Engine == db.Postgres {
			issue, err := getIssueByPipelineID(ctx, server.store, task.PipelineID)
			if err != nil {
				return nil, err
			}
			return exec.doRestoreInPlacePostgres(ctx, server, issue, task, payload)
		}
		return nil, fmt.Errorf("we only support backup restore replace for PostgreSQL now")
	}

	targetInstanceID := *payload.TargetInstanceID
	if payload.TargetInstanceID != nil {
		// For now, we just support restore full backup to the same instance with the origin database.
		// But for generality, we will use TargetInstanceID in payload to find the target instance.
		targetInstanceID = *payload.TargetInstanceID
	}

	targetDatabaseFind := &api.DatabaseFind{
		InstanceID: &targetInstanceID,
		Name:       payload.DatabaseName,
	}

	targetDatabase, err := server.store.GetDatabase(ctx, targetDatabaseFind)
	if err != nil {
		return nil, fmt.Errorf("failed to find target database %q in instance %q: %w", *payload.DatabaseName, task.Instance.Name, err)
	}
	if targetDatabase == nil {
		return nil, fmt.Errorf("target database %q not found in instance %q: %w", *payload.DatabaseName, task.Instance.Name, err)
	}
	log.Debug("Start database restore from backup...",
		zap.String("source_instance", sourceDatabase.Instance.Name),
		zap.String("source_database", sourceDatabase.Name),
		zap.String("target_instance", targetDatabase.Instance.Name),
		zap.String("target_database", targetDatabase.Name),
		zap.String("backup", backup.Name),
	)

	// Restore the database to the target database.
	if err := exec.restoreDatabase(ctx, server, targetDatabase.Instance, targetDatabase.Name, backup, server.profile.DataDir); err != nil {
		return nil, err
	}
	// TODO(zp): This should be done in the same transaction as restoreDatabase to guarantee consistency.
	// For now, we do this after restoreDatabase, since this one is unlikely to fail.
	migrationID, version, err := createBranchMigrationHistory(ctx, server, sourceDatabase, targetDatabase, backup, task)
	if err != nil {
		return nil, err
	}

	// Patch the backup id after we successfully restore the database using the backup.
	// restoringDatabase is changing the customer database instance, while here we are changing our own meta db,
	// and since we can't guarantee cross database transaction consistency, there is always a chance to have
	// inconsistent data. We choose to do Patch afterwards since this one is unlikely to fail.
	databasePatch := &api.DatabasePatch{
		ID:             targetDatabase.ID,
		UpdaterID:      api.SystemBotID,
		SourceBackupID: &backup.ID,
	}
	if _, err = server.store.PatchDatabase(ctx, databasePatch); err != nil {
		return nil, errors.Wrapf(err, "failed to patch database source with ID %d and backup ID %d after restore", targetDatabase.ID, backup.ID)
	}

	// Sync database schema after restore is completed.
	if err := server.syncDatabaseSchema(ctx, targetDatabase.Instance, targetDatabase.Name); err != nil {
		log.Error("failed to sync database schema",
			zap.String("instance", targetDatabase.Instance.Name),
			zap.String("databaseName", targetDatabase.Name),
		)
	}

	return &api.TaskRunResultPayload{
		Detail:      fmt.Sprintf("Restored database %q from backup %q", targetDatabase.Name, backup.Name),
		MigrationID: migrationID,
		Version:     version,
	}, nil
}

func (exec *PITRRestoreTaskExecutor) doPITRRestore(ctx context.Context, server *Server, task *api.Task, payload api.TaskDatabasePITRRestorePayload) (*api.TaskRunResultPayload, error) {
	sourceDriver, err := server.getAdminDatabaseDriver(ctx, task.Instance, "")
	if err != nil {
		return nil, err
	}
	defer sourceDriver.Close(ctx)

	targetDriver := sourceDriver
	if payload.TargetInstanceID != nil {
		targetDriver.Close(ctx)
	}
	// DB.Close is idempotent, so we can feel free to assign sourceDriver to targetDriver first.
	defer targetDriver.Close(ctx)

	issue, err := getIssueByPipelineID(ctx, server.store, task.PipelineID)
	if err != nil {
		return nil, err
	}

	backupStatus := api.BackupStatusDone
	backupList, err := server.store.FindBackup(ctx, &api.BackupFind{DatabaseID: task.DatabaseID, Status: &backupStatus})
	if err != nil {
		return nil, err
	}
	log.Debug("Found backup list", zap.Array("backups", api.ZapBackupArray(backupList)))

	mysqlSourceDriver, sourceOk := sourceDriver.(*mysql.Driver)
	mysqlTargetDriver, targetOk := targetDriver.(*mysql.Driver)
	if (!sourceOk) || (!targetOk) {
		log.Error("Failed to cast driver to mysql.Driver")
		return nil, fmt.Errorf("[internal] cast driver to mysql.Driver failed")
	}

	log.Debug("Downloading all binlog files")
	if err := mysqlSourceDriver.FetchAllBinlogFiles(ctx, true /* downloadLatestBinlogFile */); err != nil {
		return nil, err
	}

	targetTs := *payload.PointInTimeTs
	log.Debug("Getting latest backup before or equal to targetTs", zap.Int64("targetTs", targetTs))
	backup, err := mysqlSourceDriver.GetLatestBackupBeforeOrEqualTs(ctx, backupList, targetTs, server.profile.Mode)
	if err != nil {
		targetTsHuman := time.Unix(targetTs, 0).Format(time.RFC822)
		log.Error("Failed to get backup before or equal to time",
			zap.Int64("targetTs", targetTs),
			zap.String("targetTsHuman", targetTsHuman),
			zap.Error(err))
		return nil, errors.Wrapf(err, "failed to get latest backup before or equal to %s", targetTsHuman)
	}
	log.Debug("Got latest backup before or equal to targetTs", zap.String("backup", backup.Name))
	backupFileName := getBackupAbsFilePath(server.profile.DataDir, backup.DatabaseID, backup.Name)
	backupFile, err := os.Open(backupFileName)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to open backup file %q", backupFileName)
	}
	defer backupFile.Close()
	log.Debug("Successfully opened backup file", zap.String("filename", backupFileName))

	log.Debug("Start creating and restoring PITR database",
		zap.String("instance", task.Instance.Name),
		zap.String("database", task.Database.Name),
	)
	// RestorePITR will create the pitr database.
	// Since it's ephemeral and will be renamed to the original database soon, we will reuse the original
	// database's migration history, and append a new BRANCH migration.
	startBinlogInfo := backup.Payload.BinlogInfo
	binlogDir := getBinlogAbsDir(server.profile.DataDir, task.Instance.ID)

	if err := exec.updateProgress(ctx, mysqlTargetDriver, backupFile, startBinlogInfo, binlogDir); err != nil {
		return nil, fmt.Errorf("failed to setup progress update process, error: %w", err)
	}

	if payload.DatabaseName != nil {
		if err := mysqlTargetDriver.RestoreBackupToDatabase(ctx, backupFile, *payload.DatabaseName); err != nil {
			log.Error("failed to restore full backup in the new database",
				zap.Int("issueID", issue.ID),
				zap.String("databaseName", *payload.DatabaseName),
				zap.Error(err))
			return nil, fmt.Errorf("failed to restore full backup in the new database, error: %w", err)
		}
		if err := mysqlTargetDriver.ReplayBinlogToDatabase(ctx, task.Database.Name, *payload.DatabaseName, startBinlogInfo, targetTs); err != nil {
			log.Error("failed to perform a PITR restore in the new database",
				zap.Int("issueID", issue.ID),
				zap.String("databaseName", *payload.DatabaseName),
				zap.Error(err))
			return nil, fmt.Errorf("failed to perform a PITR restore in the new database, error: %w", err)
		}
	} else {
		if err := mysqlTargetDriver.RestoreBackupToPITRDatabase(ctx, backupFile, task.Database.Name, issue.CreatedTs); err != nil {
			log.Error("failed to restore full backup in the PITR database",
				zap.Int("issueID", issue.ID),
				zap.String("databaseName", task.Database.Name),
				zap.Error(err))
			return nil, fmt.Errorf("failed to perform a backup restore in the PITR database, error: %w", err)
		}
		if err := mysqlTargetDriver.ReplayBinlogToPITRDatabase(ctx, task.Database.Name, startBinlogInfo, issue.CreatedTs, targetTs); err != nil {
			log.Error("failed to perform a PITR restore in the PITR database",
				zap.Int("issueID", issue.ID),
				zap.String("databaseName", task.Database.Name),
				zap.Error(err))
			return nil, fmt.Errorf("failed to replay binlog in the PITR database, error: %w", err)
		}
	}

	targetDatabaseName := task.Database.Name
	if payload.DatabaseName != nil {
		targetDatabaseName = *payload.DatabaseName
	}
	log.Info("PITR restore success", zap.String("target database", targetDatabaseName))
	return &api.TaskRunResultPayload{
		Detail: fmt.Sprintf("PITR restore success for target database %q", targetDatabaseName),
	}, nil
}

func (*PITRRestoreTaskExecutor) doRestoreInPlacePostgres(ctx context.Context, server *Server, issue *api.Issue, task *api.Task, payload api.TaskDatabasePITRRestorePayload) (*api.TaskRunResultPayload, error) {
	if payload.BackupID == nil {
		return nil, fmt.Errorf("PITR for Postgres is not implemented")
	}

	backup, err := server.store.GetBackupByID(ctx, *payload.BackupID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find backup with ID %d", *payload.BackupID)
	}
	if backup == nil {
		return nil, fmt.Errorf("backup with ID %d not found", *payload.BackupID)
	}
	backupFileName := getBackupAbsFilePath(server.profile.DataDir, backup.DatabaseID, backup.Name)
	backupFile, err := os.Open(backupFileName)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to open backup file %q", backupFileName)
	}
	defer backupFile.Close()

	driver, err := server.getAdminDatabaseDriver(ctx, task.Instance, "")
	if err != nil {
		return nil, err
	}
	defer driver.Close(ctx)

	db, err := driver.GetDBConnection(ctx, db.BytebaseDatabase)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get connection for PostgreSQL")
	}
	pitrDatabaseName := util.GetPITRDatabaseName(task.Database.Name, issue.CreatedTs)
	// If there's already a PITR database, it means there's a failed trial before this task execution.
	// We need to clean up the dirty state and start clean for idempotent task execution.
	if _, err := db.ExecContext(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS %s;", pitrDatabaseName)); err != nil {
		return nil, errors.Wrapf(err, "failed to drop the dirty PITR database %q left from a former task execution", pitrDatabaseName)
	}
	if _, err := db.ExecContext(ctx, fmt.Sprintf("CREATE DATABASE %s;", pitrDatabaseName)); err != nil {
		return nil, errors.Wrapf(err, "failed to create the PITR database %q", pitrDatabaseName)
	}
	// Switch to the PITR database.
	// TODO(dragonly): This is a trick, needs refactor.
	if _, err := driver.GetDBConnection(ctx, pitrDatabaseName); err != nil {
		return nil, errors.Wrapf(err, "failed to switch connection to database %q", pitrDatabaseName)
	}
	if err := driver.Restore(ctx, backupFile); err != nil {
		return nil, errors.Wrapf(err, "failed to restore backup to the PITR database %q", pitrDatabaseName)
	}
	return &api.TaskRunResultPayload{
		Detail: fmt.Sprintf("Restored backup %q to the temporary PITR database %q", backup.Name, pitrDatabaseName),
	}, nil
}

func (exec *PITRRestoreTaskExecutor) updateProgress(ctx context.Context, driver *mysql.Driver, backupFile *os.File, startBinlogInfo api.BinlogInfo, binlogDir string) error {
	backupFileInfo, err := backupFile.Stat()
	if err != nil {
		return errors.Wrapf(err, "failed to get stat of backup file %q", backupFile.Name())
	}
	backupFileBytes := backupFileInfo.Size()
	replayBinlogPaths, err := mysql.GetBinlogReplayList(startBinlogInfo, binlogDir)
	if err != nil {
		return errors.Wrapf(err, "failed to get binlog replay list with startBinlogInfo %+v in binlog directory %q", startBinlogInfo, binlogDir)
	}
	totalBinlogBytes, err := common.GetFileSizeSum(replayBinlogPaths)
	if err != nil {
		return errors.Wrap(err, "failed to get file size sum of replay binlog files")
	}

	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		createdTs := time.Now().Unix()
		exec.progress.Store(api.Progress{
			TotalUnit:     backupFileBytes + totalBinlogBytes,
			CompletedUnit: 0,
			CreatedTs:     createdTs,
			UpdatedTs:     createdTs,
		})
		for {
			select {
			case <-ticker.C:
				progressPrev := exec.progress.Load().(api.Progress)
				// TODO(dragonly): Calculate restored backup bytes when using mysqldump.
				restoredBackupFileBytes := backupFileBytes
				replayedBinlogBytes := driver.GetReplayedBinlogBytes()
				exec.progress.Store(api.Progress{
					TotalUnit:     progressPrev.TotalUnit,
					CompletedUnit: restoredBackupFileBytes + replayedBinlogBytes,
					CreatedTs:     progressPrev.CreatedTs,
					UpdatedTs:     time.Now().Unix(),
				})
			case <-ctx.Done():
				return
			}
		}
	}()

	return nil
}

func getIssueByPipelineID(ctx context.Context, store *store.Store, pid int) (*api.Issue, error) {
	issue, err := store.GetIssueByPipelineID(ctx, pid)
	if err != nil {
		log.Error("failed to get issue by PipelineID", zap.Int("PipelineID", pid), zap.Error(err))
		return nil, errors.Wrapf(err, "failed to get issue by PipelineID: %d", pid)
	}
	if issue == nil {
		log.Error("issue not found with PipelineID", zap.Int("PipelineID", pid))
		return nil, fmt.Errorf("issue not found with PipelineID: %d", pid)
	}
	return issue, nil
}

// restoreDatabase will restore the database to the instance from the backup.
func (*PITRRestoreTaskExecutor) restoreDatabase(ctx context.Context, server *Server, instance *api.Instance, databaseName string, backup *api.Backup, dataDir string) error {
	driver, err := server.getAdminDatabaseDriver(ctx, instance, databaseName)
	if err != nil {
		return err
	}
	defer driver.Close(ctx)

	backupPath := backup.Path
	if !filepath.IsAbs(backupPath) {
		backupPath = filepath.Join(dataDir, backupPath)
	}

	f, err := os.Open(backupPath)
	if err != nil {
		return fmt.Errorf("failed to open backup file at %s: %w", backupPath, err)
	}
	defer f.Close()

	if err := driver.Restore(ctx, f); err != nil {
		return fmt.Errorf("failed to restore backup: %w", err)
	}
	return nil
}
