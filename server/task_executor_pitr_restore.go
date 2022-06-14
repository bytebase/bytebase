package server

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common/log"
	"github.com/bytebase/bytebase/plugin/db"
	pluginmysql "github.com/bytebase/bytebase/plugin/db/mysql"
	restoremysql "github.com/bytebase/bytebase/plugin/restore/mysql"
	"github.com/bytebase/bytebase/resources/mysqlutil"
	"github.com/bytebase/bytebase/store"
	"go.uber.org/zap"
)

// NewPITRRestoreTaskExecutor creates a PITR restore task executor.
func NewPITRRestoreTaskExecutor(instance *mysqlutil.Instance) TaskExecutor {
	return &PITRRestoreTaskExecutor{
		mysqlutil: instance,
	}
}

// PITRRestoreTaskExecutor is the PITR restore task executor.
type PITRRestoreTaskExecutor struct {
	mysqlutil *mysqlutil.Instance
}

// RunOnce will run the PITR restore task executor once.
func (exec *PITRRestoreTaskExecutor) RunOnce(ctx context.Context, server *Server, task *api.Task) (terminated bool, result *api.TaskRunResultPayload, err error) {
	log.Info("Run PITR restore task", zap.String("task", task.Name))

	payload := api.TaskDatabasePITRRestorePayload{}
	if err := json.Unmarshal([]byte(task.Payload), &payload); err != nil {
		return true, nil, fmt.Errorf("invalid PITR restore payload[%s], error[%w]", task.Payload, err)
	}

	driver, err := getAdminDatabaseDriver(ctx, task.Instance, "", "" /* pgInstanceDir */)
	if err != nil {
		return true, nil, err
	}
	defer driver.Close(ctx)

	if err := exec.doPITRRestore(ctx, task, server.store, driver, server.profile.DataDir, payload.PointInTimeTs); err != nil {
		log.Error("Failed to do PITR restore", zap.Error(err))
		return true, nil, err
	}

	log.Info("created PITR database", zap.String("target database", task.Database.Name))

	return true, &api.TaskRunResultPayload{
		Detail: fmt.Sprintf("Created PITR database for target database %q", task.Database.Name),
	}, nil
}

func (exec *PITRRestoreTaskExecutor) doPITRRestore(ctx context.Context, task *api.Task, store *store.Store, driver db.Driver, dataDir string, targetTs int64) error {
	instance := task.Instance
	database := task.Database

	issue, err := getIssueByPipelineID(ctx, store, task.PipelineID)
	if err != nil {
		return err
	}

	backupList, err := store.FindBackup(ctx, &api.BackupFind{DatabaseID: task.DatabaseID})
	if err != nil {
		return err
	}
	log.Debug("Found backup list", zap.Array("backups", api.ZapBackupArray(backupList)))

	connCfg, err := getConnectionConfig(ctx, instance, database.Name)
	if err != nil {
		return err
	}

	binlogDir := getBinlogAbsDir(dataDir, task.Instance.ID)
	if err := createBinlogDir(dataDir, task.Instance.ID); err != nil {
		return err
	}

	mysqlDriver, ok := driver.(*pluginmysql.Driver)
	if !ok {
		log.Error("failed to cast driver to mysql.Driver")
		return fmt.Errorf("[internal] cast driver to mysql.Driver failed")
	}

	mysqlRestore := restoremysql.New(mysqlDriver, exec.mysqlutil, connCfg, binlogDir)

	log.Debug("Download all binlog files up to targetTs", zap.Time("targetTs", time.Unix(targetTs, 0)))
	// TODO(dragonly): Do this on a regular basis.
	if err := mysqlRestore.IncrementalFetchAllBinlogFiles(ctx); err != nil {
		return err
	}

	log.Debug("Getting latest backup before or equal to targetTs...", zap.Time("targetTs", time.Unix(targetTs, 0)))
	backup, err := mysqlRestore.GetLatestBackupBeforeOrEqualTs(ctx, backupList, targetTs)
	if err != nil {
		log.Error("Failed to get backup before or equal to targetTs",
			zap.Time("targetTs", time.Unix(targetTs, 0)),
			zap.Error(err))
		return fmt.Errorf("failed to get latest backup before or equal to targetTs[%d], error[%w]", targetTs, err)
	}
	log.Debug("Got latest backup before or equal to targetTs", zap.String("backup", backup.Name))
	backupFileName := getBackupAbsFilePath(dataDir, backup.DatabaseID, backup.Name)
	backupFile, err := os.OpenFile(backupFileName, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to open backup file[%s], error[%w]", backupFileName, err)
	}
	defer backupFile.Close()
	log.Debug("Successfully opened backup file", zap.String("filename", backupFileName))

	log.Debug("Start creating and restoring PITR database",
		zap.String("instance", instance.Name),
		zap.String("database", database.Name),
	)
	// RestorePITR will create the pitr database.
	// Since it's ephemeral and will be renamed to the original database soon, we will reuse the original
	// database's migration history, and append a new BASELINE migration.
	startBinlogInfo := backup.Payload.BinlogInfo
	if err := mysqlRestore.RestorePITR(ctx, bufio.NewScanner(backupFile), startBinlogInfo, database.Name, issue.CreatedTs, targetTs); err != nil {
		log.Error("failed to perform a PITR restore in the PITR database",
			zap.Int("issueID", issue.ID),
			zap.String("database", database.Name),
			zap.Error(err))
		return fmt.Errorf("failed to perform a PITR restore in the PITR database, error[%w]", err)
	}

	return nil
}

func getIssueByPipelineID(ctx context.Context, store *store.Store, pid int) (*api.Issue, error) {
	issue, err := store.GetIssueByPipelineID(ctx, pid)
	if err != nil {
		log.Error("failed to get issue by PipelineID", zap.Int("PipelineID", pid), zap.Error(err))
		return nil, fmt.Errorf("failed to get issue by PipelineID[%d], error[%w]", pid, err)
	}
	if issue == nil {
		log.Error("issue not found with PipelineID", zap.Int("PipelineID", pid))
		return nil, fmt.Errorf("issue not found with PipelineID[%d]", pid)
	}
	return issue, nil
}
