package server

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/bin/bb/connect"
	"github.com/bytebase/bytebase/bin/bb/dump/mysqldump"
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
func (exec *DatabaseBackupTaskExecutor) RunOnce(ctx context.Context, server *Server, task *api.Task) (terminated bool, detail string, err error) {
	defer func() {
		if r := recover(); r != nil {
			panicErr, ok := r.(error)
			if !ok {
				panicErr = fmt.Errorf("%v", r)
			}
			exec.l.Error("DatabaseBackupTaskExecutor PANIC RECOVER", zap.Error(panicErr))
			terminated = true
			err = fmt.Errorf("encounter internal error when backing database")
		}
	}()
	// Close the pipeline when the backup task is completed regardless its status.
	defer func() {
		status := api.Pipeline_Done
		pipelinePatch := &api.PipelinePatch{
			ID:        task.PipelineId,
			UpdaterId: api.SYSTEM_BOT_ID,
			Status:    &status,
		}
		if _, err := server.PipelineService.PatchPipeline(context.Background(), pipelinePatch); err != nil {
			err = fmt.Errorf("failed to update pipeline status: %w", err)
		}
	}()

	payload := &api.TaskDatabaseBackupPayload{}
	if err := json.Unmarshal([]byte(task.Payload), payload); err != nil {
		return true, "", fmt.Errorf("invalid database backup payload: %w", err)
	}

	if err := server.ComposeTaskRelationship(ctx, task); err != nil {
		return true, "", err
	}

	backup, err := server.BackupService.FindBackup(ctx, &api.BackupFind{ID: &payload.BackupID})
	if err != nil {
		return true, "", fmt.Errorf("failed to find backup: %w", err)
	}
	exec.l.Debug("Start database backup...",
		zap.String("instance", task.Instance.Name),
		zap.String("database", task.Database.Name),
		zap.String("backup", backup.Name),
	)

	backupErr := backupDatabase(task.Instance, task.Database, backup)
	// Update the status of the backup.
	newBackupStatus := string(api.BackupStatusDone)
	if backupErr != nil {
		newBackupStatus = string(api.BackupStatusFailed)
	}
	if _, err = server.BackupService.PatchBackup(ctx, &api.BackupPatch{
		ID:        backup.ID,
		Status:    newBackupStatus,
		UpdaterId: api.SYSTEM_BOT_ID,
	}); err != nil {
		return true, "", fmt.Errorf("failed to patch backup: %w", err)
	}

	if backupErr != nil {
		return true, "", err
	}

	return true, fmt.Sprintf("Backup database '%s'", task.Database.Name), nil
}

// backupDatabase will take a backup of a database.
func backupDatabase(instance *api.Instance, database *api.Database, backup *api.Backup) error {
	conn, err := connect.NewMysql(instance.Username, instance.Password, instance.Host, instance.Port, database.Name, nil /* tlsConfig */)
	if err != nil {
		return fmt.Errorf("connect.NewMysql(%q, %q, %q, %q) got error: %v", instance.Username, instance.Password, instance.Host, instance.Port, err)
	}
	defer conn.Close()
	dp := mysqldump.New(conn)

	f, err := os.Create(backup.Path)
	if err != nil {
		return fmt.Errorf("failed to open backup path: %s", backup.Path)
	}
	defer f.Close()

	if err := dp.Dump(database.Name, f, false /* schemaOnly */, false /* dumpAll */); err != nil {
		return err
	}

	return nil
}
