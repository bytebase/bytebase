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
	"github.com/bytebase/bytebase/store"
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
		return true, nil, fmt.Errorf("invalid PITR restore payload: %s, error: %w", task.Payload, err)
	}

	if (payload.BackupID == nil) == (payload.PointInTimeTs == nil) {
		return true, nil, fmt.Errorf("only one of BackupID and time point can be set")
	}

	if payload.BackupID != nil {
		// TODO(dragonly): backup restore/PITR to new db do not require a cutover task.
		// The current implementation here is actually not working, because it restores the backup directly to the target database, and the later cutover task will fail.
		if payload.DatabaseName == nil {
			return true, nil, fmt.Errorf("unexpected nil database name for backup restore")
		}
		// Restore full backup only
		backup, err := server.store.GetBackupByID(ctx, *payload.BackupID)
		if err != nil {
			return true, nil, fmt.Errorf("failed to find backup with ID %d, error: %w", *payload.BackupID, err)
		}
		if backup == nil {
			return true, nil, fmt.Errorf("backup with ID %d not found", *payload.BackupID)
		}
		sourceDatabase, err := server.store.GetDatabase(ctx, &api.DatabaseFind{ID: &backup.DatabaseID})
		if err != nil {
			return true, nil, fmt.Errorf("failed to find database for the backup: %w", err)
		}
		if sourceDatabase == nil {
			return true, nil, fmt.Errorf("source database ID not found %v", backup.DatabaseID)
		}

		targetInstanceID := task.InstanceID
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
			return true, nil, fmt.Errorf("failed to find target database %q in instance %q: %w", *payload.DatabaseName, task.Instance.Name, err)
		}
		if targetDatabase == nil {
			return true, nil, fmt.Errorf("target database %q not found in instance %q: %w", *payload.DatabaseName, task.Instance.Name, err)
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
			return true, nil, err
		}
		// TODO(zp): This should be done in the same transaction as restoreDatabase to guarantee consistency.
		// For now, we do this after restoreDatabase, since this one is unlikely to fail.
		migrationID, version, err := createBranchMigrationHistory(ctx, server, sourceDatabase, targetDatabase, backup, task)
		if err != nil {
			return true, nil, err
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
			return true, nil, fmt.Errorf("failed to patch database source with ID %d and backup ID %d after restore, error: %w", targetDatabase.ID, backup.ID, err)
		}

		// Sync database schema after restore is completed.
		if err := server.syncDatabaseSchema(ctx, targetDatabase.Instance, targetDatabase.Name); err != nil {
			log.Error("failed to sync database schema",
				zap.String("instance", targetDatabase.Instance.Name),
				zap.String("databaseName", targetDatabase.Name),
			)
		}

		return true, &api.TaskRunResultPayload{
			Detail:      fmt.Sprintf("Restored database %q from backup %q", targetDatabase.Name, backup.Name),
			MigrationID: migrationID,
			Version:     version,
		}, nil
	}

	driver, err := server.getAdminDatabaseDriver(ctx, task.Instance, "")
	if err != nil {
		return true, nil, err
	}
	defer driver.Close(ctx)

	if err := exec.doPITRRestore(ctx, task, server.store, driver, server.profile.DataDir, *payload.PointInTimeTs, server.profile.Mode); err != nil {
		log.Error("Failed to do PITR restore", zap.Error(err))
		return true, nil, err
	}

	log.Info("created PITR database", zap.String("target database", task.Database.Name))

	return true, &api.TaskRunResultPayload{
		Detail: fmt.Sprintf("Created PITR database for target database %q", task.Database.Name),
	}, nil
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

func (exec *PITRRestoreTaskExecutor) doPITRRestore(ctx context.Context, task *api.Task, store *store.Store, driver db.Driver, dataDir string, targetTs int64, mode common.ReleaseMode) error {
	issue, err := getIssueByPipelineID(ctx, store, task.PipelineID)
	if err != nil {
		return err
	}

	backupStatus := api.BackupStatusDone
	backupList, err := store.FindBackup(ctx, &api.BackupFind{DatabaseID: task.DatabaseID, Status: &backupStatus})
	if err != nil {
		return err
	}
	log.Debug("Found backup list", zap.Array("backups", api.ZapBackupArray(backupList)))

	mysqlDriver, ok := driver.(*mysql.Driver)
	if !ok {
		log.Error("Failed to cast driver to mysql.Driver")
		return fmt.Errorf("[internal] cast driver to mysql.Driver failed")
	}

	log.Debug("Downloading all binlog files")
	if err := mysqlDriver.FetchAllBinlogFiles(ctx, true /* downloadLatestBinlogFile */); err != nil {
		return err
	}

	log.Debug("Getting latest backup before or equal to targetTs", zap.Int64("targetTs", targetTs))
	backup, err := mysqlDriver.GetLatestBackupBeforeOrEqualTs(ctx, backupList, targetTs, mode)
	if err != nil {
		targetTsHuman := time.Unix(targetTs, 0).Format(time.RFC822)
		log.Error("Failed to get backup before or equal to time",
			zap.Int64("targetTs", targetTs),
			zap.String("targetTsHuman", targetTsHuman),
			zap.Error(err))
		return fmt.Errorf("failed to get latest backup before or equal to %s, error: %w", targetTsHuman, err)
	}
	log.Debug("Got latest backup before or equal to targetTs", zap.String("backup", backup.Name))
	backupFileName := getBackupAbsFilePath(dataDir, backup.DatabaseID, backup.Name)
	backupFile, err := os.Open(backupFileName)
	if err != nil {
		return fmt.Errorf("failed to open backup file %q, error: %w", backupFileName, err)
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

	binlogDir := getBinlogAbsDir(dataDir, task.Instance.ID)
	if err := exec.updateProgress(ctx, mysqlDriver, backupFile, startBinlogInfo, binlogDir); err != nil {
		return fmt.Errorf("failed to setup progress update process, error: %w", err)
	}

	if err := mysqlDriver.RestoreBackupToPITRDatabase(ctx, backupFile, task.Database.Name, issue.CreatedTs); err != nil {
		log.Error("failed to restore full backup in the PITR database",
			zap.Int("issueID", issue.ID),
			zap.String("databaseName", task.Database.Name),
			zap.Error(err))
		return fmt.Errorf("failed to perform a backup restore in the PITR database, error: %w", err)
	}

	if err := mysqlDriver.ReplayBinlogToPITRDatabase(ctx, task.Database.Name, startBinlogInfo, issue.CreatedTs, targetTs); err != nil {
		log.Error("failed to perform a PITR restore in the PITR database",
			zap.Int("issueID", issue.ID),
			zap.String("databaseName", task.Database.Name),
			zap.Error(err))
		return fmt.Errorf("failed to replay binlog in the PITR database, error: %w", err)
	}

	return nil
}

func (exec *PITRRestoreTaskExecutor) updateProgress(ctx context.Context, driver *mysql.Driver, backupFile *os.File, startBinlogInfo api.BinlogInfo, binlogDir string) error {
	backupFileInfo, err := backupFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to get stat of backup file %q, error: %w", backupFile.Name(), err)
	}
	backupFileBytes := backupFileInfo.Size()
	replayBinlogPaths, err := mysql.GetBinlogReplayList(startBinlogInfo, binlogDir)
	if err != nil {
		return fmt.Errorf("failed to get binlog replay list with startBinlogInfo %+v in binlog directory %q, error: %w", startBinlogInfo, binlogDir, err)
	}
	totalBinlogBytes, err := common.GetFileSizeSum(replayBinlogPaths)
	if err != nil {
		return fmt.Errorf("failed to get file size sum of replay binlog files, error: %w", err)
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
		return nil, fmt.Errorf("failed to get issue by PipelineID: %d, error: %w", pid, err)
	}
	if issue == nil {
		log.Error("issue not found with PipelineID", zap.Int("PipelineID", pid))
		return nil, fmt.Errorf("issue not found with PipelineID: %d", pid)
	}
	return issue, nil
}

// restoreDatabase will restore the database from a backup.
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
