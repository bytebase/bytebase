package server

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/bin/bb/connect"
	"github.com/bytebase/bytebase/bin/bb/restore/mysqlrestore"
	"go.uber.org/zap"
)

// NewDatabaseRestoreTaskExecutor creates a new database restore task executor.
func NewDatabaseRestoreTaskExecutor(logger *zap.Logger) TaskExecutor {
	return &DatabaseRestoreTaskExecutor{
		l: logger,
	}
}

// DatabaseRestoreTaskExecutor is the task executor for database restore.
type DatabaseRestoreTaskExecutor struct {
	l *zap.Logger
}

// RunOnce will run database restore once.
func (exec *DatabaseRestoreTaskExecutor) RunOnce(ctx context.Context, server *Server, task *api.Task) (terminated bool, detail string, err error) {
	defer func() {
		if r := recover(); r != nil {
			panicErr, ok := r.(error)
			if !ok {
				panicErr = fmt.Errorf("%v", r)
			}
			exec.l.Error("DatabaseRestoreTaskExecutor PANIC RECOVER", zap.Error(panicErr))
			terminated = true
			err = fmt.Errorf("encounter internal error when restoring the database")
		}
	}()
	// Close the pipeline when the restore task is completed regardless its status.
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

	payload := &api.TaskDatabaseRestorePayload{}
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
	backup.Database = task.Database

	if err := restoreDatabase(backup); err != nil {
		return true, "", err
	}

	return true, fmt.Sprintf("Restore database '%s'", task.Database.Name), nil
}

// restoreDatabase will restore the database from a backup
func restoreDatabase(backup *api.Backup) error {
	database := backup.Database
	instance := database.Instance
	conn, err := connect.NewMysql(instance.Username, instance.Password, instance.Host, instance.Port, database.Name, nil /* tlsConfig */)
	if err != nil {
		return fmt.Errorf("connect.NewMysql(%q, %q, %q, %q) got error: %v", instance.Username, instance.Password, instance.Host, instance.Port, err)
	}
	defer conn.Close()
	f, err := os.OpenFile(backup.Path, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return fmt.Errorf("os.OpenFile(%q) error: %v", backup.Path, err)
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	if err := mysqlrestore.Restore(conn, sc); err != nil {
		return fmt.Errorf("mysqlrestore.Restore() got error: %v", err)
	}
	return nil
}
