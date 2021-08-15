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
	"github.com/bytebase/bytebase/db"
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
	exec.l.Debug("Start database restore...",
		zap.String("instance", task.Instance.Name),
		zap.String("database", task.Database.Name),
		zap.String("backup", backup.Name),
	)
	databaseFind := &api.DatabaseFind{
		ID: &backup.DatabaseId,
	}
	backup.Database, err = server.ComposeDatabaseByFind(context.Background(), databaseFind)
	if err != nil {
		return true, "", err
	}

	// Fork migration history.
	if backup.MigrationHistoryVersion != "" {
		if err := forkMigrationHistory(backup.Database, task.Database, backup.MigrationHistoryVersion, exec.l); err != nil {
			return true, "", err
		}
	}

	// Restore the database to the target database.
	if err := restoreDatabase(task.Database, backup); err != nil {
		return true, "", err
	}

	return true, fmt.Sprintf("Restore database '%s'", task.Database.Name), nil
}

// restoreDatabase will restore the database from a backup
func restoreDatabase(database *api.Database, backup *api.Backup) error {
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

// forkMigrationHistory will fork the migration history from source database to target database based on backup migration history version.
// This is needed only when backup migration history version is not empty.
func forkMigrationHistory(sourceDatabase, targetDatabase *api.Database, migrationHistoryVersion string, logger *zap.Logger) error {
	sourceDriver, err := getDatabaseDriver(sourceDatabase, logger)
	if err != nil {
		return err
	}
	defer sourceDriver.Close(context.Background())

	targetDriver, err := getDatabaseDriver(targetDatabase, logger)
	if err != nil {
		return err
	}
	defer targetDriver.Close(context.Background())

	find := &db.MigrationHistoryFind{
		Database: &sourceDatabase.Name,
	}
	list, err := sourceDriver.FindMigrationHistoryList(context.Background(), find)
	if err != nil {
		return fmt.Errorf("failed to fetch migration history list: %v", err)
	}

	var forkList []*db.MigrationHistory
	for i := len(list) - 1; i >= 0; i-- {
		history := list[i]
		history.Namespace = targetDatabase.Name
		forkList = append(forkList, history)
		// Fork the history up to the backup version.
		if history.Version == migrationHistoryVersion {
			break
		}
	}
	// TODO(spinningbot): add a new BRANCH migration history.

	for _, history := range forkList {
		m := &db.MigrationInfo{
			Version:     history.Version,
			Namespace:   history.Namespace,
			Database:    targetDatabase.Name,
			Environment: targetDatabase.Instance.Environment.Name,
			Engine:      history.Engine,
			Type:        history.Type,
			Description: history.Description,
			Creator:     history.Creator,
			IssueId:     history.IssueId,
			Payload:     history.Payload,
		}
		if err := targetDriver.ExecuteMigration(context.Background(), m, history.Statement); err != nil {
			return err
		}
	}

	return nil
}

func getDatabaseDriver(database *api.Database, logger *zap.Logger) (db.Driver, error) {
	instance := database.Instance
	driver, err := db.Open(
		instance.Engine,
		db.DriverConfig{Logger: logger},
		db.ConnectionConfig{
			Username: instance.Username,
			Password: instance.Password,
			Host:     instance.Host,
			Port:     instance.Port,
			Database: database.Name,
		},
		db.ConnectionContext{
			EnvironmentName: instance.Environment.Name,
			InstanceName:    instance.Name,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect instance: %v with user: %v. %w", instance.Name, instance.Username, err)
	}
	return driver, nil
}
