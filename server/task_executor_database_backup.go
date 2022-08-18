package server

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common/log"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

// NewDatabaseBackupTaskExecutor creates a new database backup task executor.
func NewDatabaseBackupTaskExecutor() TaskExecutor {
	return &DatabaseBackupTaskExecutor{}
}

// DatabaseBackupTaskExecutor is the task executor for database backup.
type DatabaseBackupTaskExecutor struct {
	completed int32
}

// IsCompleted tells the scheduler if the task execution has completed.
func (exec *DatabaseBackupTaskExecutor) IsCompleted() bool {
	return atomic.LoadInt32(&exec.completed) == 1
}

// GetProgress returns the task progress.
func (*DatabaseBackupTaskExecutor) GetProgress() api.Progress {
	return api.Progress{}
}

// RunOnce will run database backup once.
func (exec *DatabaseBackupTaskExecutor) RunOnce(ctx context.Context, server *Server, task *api.Task) (terminated bool, result *api.TaskRunResultPayload, err error) {
	defer atomic.StoreInt32(&exec.completed, 1)
	payload := &api.TaskDatabaseBackupPayload{}
	if err := json.Unmarshal([]byte(task.Payload), payload); err != nil {
		return true, nil, errors.Wrap(err, "invalid database backup payload")
	}

	backup, err := server.store.GetBackupByID(ctx, payload.BackupID)
	if err != nil {
		return true, nil, errors.Wrapf(err, "failed to find backup with ID %d", payload.BackupID)
	}
	if backup == nil {
		return true, nil, fmt.Errorf("backup %v not found", payload.BackupID)
	}
	log.Debug("Start database backup...",
		zap.String("instance", task.Instance.Name),
		zap.String("database", task.Database.Name),
		zap.String("backup", backup.Name),
	)

	backupPayload, backupErr := exec.backupDatabase(ctx, server, task.Instance, task.Database.Name, backup)
	backupPatch := api.BackupPatch{
		ID:        backup.ID,
		Status:    string(api.BackupStatusDone),
		UpdaterID: api.SystemBotID,
		Comment:   "",
		Payload:   backupPayload,
	}
	if backupErr != nil {
		backupPatch.Status = string(api.BackupStatusFailed)
		backupPatch.Comment = backupErr.Error()
	}
	if _, err := server.store.PatchBackup(ctx, &backupPatch); err != nil {
		return true, nil, errors.Wrap(err, "failed to patch backup")
	}

	if backupErr != nil {
		return true, nil, backupErr
	}

	return true, &api.TaskRunResultPayload{
		Detail: fmt.Sprintf("Backup database %q", task.Database.Name),
	}, nil
}

// backupDatabase will take a backup of a database.
func (*DatabaseBackupTaskExecutor) backupDatabase(ctx context.Context, server *Server, instance *api.Instance, databaseName string, backup *api.Backup) (string, error) {
	driver, err := server.getAdminDatabaseDriver(ctx, instance, databaseName)
	if err != nil {
		return "", err
	}
	defer driver.Close(ctx)

	f, err := os.Create(filepath.Join(server.profile.DataDir, backup.Path))
	if err != nil {
		return "", fmt.Errorf("failed to open backup path: %s", backup.Path)
	}
	defer f.Close()

	payload, err := driver.Dump(ctx, databaseName, f, false /* schemaOnly */)
	if err != nil {
		return "", err
	}

	return payload, nil
}

// Get backup dir relative to the data dir.
func getBackupRelativeDir(databaseID int) string {
	return filepath.Join("backup", "db", fmt.Sprintf("%d", databaseID))
}

func getBackupRelativeFilePath(databaseID int, name string) string {
	dir := getBackupRelativeDir(databaseID)
	return filepath.Join(dir, fmt.Sprintf("%s.sql", name))
}

func getBackupAbsFilePath(dataDir string, databaseID int, name string) string {
	path := getBackupRelativeFilePath(databaseID, name)
	return filepath.Join(dataDir, path)
}

// Create backup directory for database.
func createBackupDirectory(dataDir string, databaseID int) error {
	dir := getBackupRelativeDir(databaseID)
	absDir := filepath.Join(dataDir, dir)
	return os.MkdirAll(absDir, os.ModePerm)
}

func getBinlogAbsDir(dataDir string, instanceID int) string {
	return filepath.Join(dataDir, "backup", "instance", fmt.Sprintf("%d", instanceID))
}
