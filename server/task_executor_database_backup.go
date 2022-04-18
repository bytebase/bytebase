package server

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/bytebase/bytebase/api"
	"go.uber.org/zap"
)

// NewDatabaseBackupTaskExecutor creates a new database backup task executor.
func NewDatabaseBackupTaskExecutor(logger *zap.Logger) TaskExecutor {
	return &DatabaseBackupTaskExecutor{
		l: logger,
	}
}

// DatabaseBackupTaskExecutor is the task executor for database backup.
type DatabaseBackupTaskExecutor struct {
	l *zap.Logger
}

// RunOnce will run database backup once.
func (exec *DatabaseBackupTaskExecutor) RunOnce(ctx context.Context, server *Server, task *api.Task) (terminated bool, result *api.TaskRunResultPayload, err error) {
	defer func() {
		if r := recover(); r != nil {
			panicErr, ok := r.(error)
			if !ok {
				panicErr = fmt.Errorf("%v", r)
			}
			exec.l.Error("DatabaseBackupTaskExecutor PANIC RECOVER", zap.Error(panicErr), zap.Stack("stack"))
			terminated = true
			err = fmt.Errorf("encounter internal error when backing database")
		}
	}()

	payload := &api.TaskDatabaseBackupPayload{}
	if err := json.Unmarshal([]byte(task.Payload), payload); err != nil {
		return true, nil, fmt.Errorf("invalid database backup payload: %w", err)
	}

	backup, err := server.store.GetBackupByID(ctx, payload.BackupID)
	if err != nil {
		return true, nil, fmt.Errorf("failed to find backup with ID[%d], error[%w]", payload.BackupID, err)
	}
	if backup == nil {
		return true, nil, fmt.Errorf("backup %v not found", payload.BackupID)
	}
	exec.l.Debug("Start database backup...",
		zap.String("instance", task.Instance.Name),
		zap.String("database", task.Database.Name),
		zap.String("backup", backup.Name),
	)

	backupErr := exec.backupDatabase(ctx, task.Instance, task.Database.Name, backup, server.dataDir)
	// Update the status of the backup.
	newBackupStatus := string(api.BackupStatusDone)
	comment := ""
	if backupErr != nil {
		newBackupStatus = string(api.BackupStatusFailed)
		comment = backupErr.Error()
	}
	if _, err := server.store.PatchBackup(ctx, &api.BackupPatch{
		ID:        backup.ID,
		Status:    newBackupStatus,
		UpdaterID: api.SystemBotID,
		Comment:   comment,
	}); err != nil {
		return true, nil, fmt.Errorf("failed to patch backup: %w", err)
	}

	if backupErr != nil {
		return true, nil, backupErr
	}

	return true, &api.TaskRunResultPayload{
		Detail: fmt.Sprintf("Backup database %q", task.Database.Name),
	}, nil
}

// backupDatabase will take a backup of a database.
func (exec *DatabaseBackupTaskExecutor) backupDatabase(ctx context.Context, instance *api.Instance, databaseName string, backup *api.Backup, dataDir string) error {
	driver, err := getAdminDatabaseDriver(ctx, instance, databaseName, exec.l)
	if err != nil {
		return err
	}
	defer driver.Close(ctx)

	f, err := os.Create(filepath.Join(dataDir, backup.Path))
	if err != nil {
		return fmt.Errorf("failed to open backup path: %s", backup.Path)
	}
	defer f.Close()

	if err := driver.Dump(ctx, databaseName, f, false /* schemaOnly */); err != nil {
		return err
	}

	return nil
}

// getAndCreateBackupDirectory returns the path of a database backup.
func getAndCreateBackupDirectory(dataDir string, database *api.Database) (string, error) {
	dir := filepath.Join("backup", "db", fmt.Sprintf("%d", database.ID))
	absDir := filepath.Join(dataDir, dir)
	if err := os.MkdirAll(absDir, os.ModePerm); err != nil {
		return "", nil
	}

	return dir, nil
}

// getAndCreateBackupPath returns the path of a database backup.
func getAndCreateBackupPath(dataDir string, database *api.Database, name string) (string, error) {
	dir, err := getAndCreateBackupDirectory(dataDir, database)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, fmt.Sprintf("%s.sql", name)), nil
}
