package server

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/bytebase/bytebase"
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

	// TODO(tianzhou): revisit this, if we want to do this, it should be better handled in the upper layer.
	// // Close the pipeline when the restore task is completed regardless its status.
	// defer func() {
	// 	status := api.Pipeline_Done
	// 	pipelinePatch := &api.PipelinePatch{
	// 		ID:        task.PipelineId,
	// 		UpdaterId: api.SYSTEM_BOT_ID,
	// 		Status:    &status,
	// 	}
	// 	if _, err := server.PipelineService.PatchPipeline(context.Background(), pipelinePatch); err != nil {
	// 		err = fmt.Errorf("failed to update pipeline status: %w", err)
	// 	}
	// }()

	payload := &api.TaskDatabaseRestorePayload{}
	if err := json.Unmarshal([]byte(task.Payload), payload); err != nil {
		return true, "", fmt.Errorf("invalid database backup payload: %w", err)
	}

	if err := server.ComposeTaskRelationship(ctx, task); err != nil {
		return true, "", err
	}

	backup, err := server.BackupService.FindBackup(ctx, &api.BackupFind{ID: &payload.BackupId})
	if err != nil {
		return true, "", fmt.Errorf("failed to find backup: %w", err)
	}

	sourceDatabaseFind := &api.DatabaseFind{
		ID: &backup.DatabaseId,
	}
	sourceDatabase, err := server.ComposeDatabaseByFind(context.Background(), sourceDatabaseFind)
	if err != nil {
		return true, "", fmt.Errorf("failed to find database for the backup: %w", err)
	}

	targetDatabaseFind := &api.DatabaseFind{
		InstanceId: &task.InstanceId,
		Name:       &payload.DatabaseName,
	}
	targetDatabase, err := server.ComposeDatabaseByFind(context.Background(), targetDatabaseFind)
	if err != nil {
		if bytebase.ErrorCode(err) == bytebase.ENOTFOUND {
			return true, "", fmt.Errorf("target database %q not found in instance %q: %w", targetDatabase.Name, task.Instance.Name, err)
		}
		return true, "", fmt.Errorf("failed to find target database %q in instance %q: %w", targetDatabase.Name, task.Instance.Name, err)
	}

	exec.l.Debug("Start database restore from backup...",
		zap.String("source_instance", sourceDatabase.Instance.Name),
		zap.String("source_database", sourceDatabase.Name),
		zap.String("target_instance", targetDatabase.Instance.Name),
		zap.String("target_database", targetDatabase.Name),
		zap.String("backup", backup.Name),
	)

	// Fork migration history.
	if backup.MigrationHistoryVersion != "" {
		if err := forkMigrationHistory(sourceDatabase, targetDatabase, backup, task, exec.l); err != nil {
			return true, "", err
		}
	}

	// Restore the database to the target database.
	if err := restoreDatabase(targetDatabase, backup, server.dataDir); err != nil {
		return true, "", err
	}

	return true, fmt.Sprintf("Restored database %q from backup %q", targetDatabase.Name, backup.Name), nil
}

// restoreDatabase will restore the database from a backup
func restoreDatabase(database *api.Database, backup *api.Backup, dataDir string) error {
	instance := database.Instance
	conn, err := connect.NewMysql(instance.Username, instance.Password, instance.Host, instance.Port, database.Name, nil /* tlsConfig */)
	if err != nil {
		return fmt.Errorf("failed to connect database: %v", err)
	}
	defer conn.Close()

	backupPath := backup.Path
	if !filepath.IsAbs(backupPath) {
		backupPath = filepath.Join(dataDir, backupPath)
	}

	f, err := os.OpenFile(backupPath, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to open backup file at %s: %v", backupPath, err)
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	if err := mysqlrestore.Restore(conn, sc); err != nil {
		return fmt.Errorf("failed to restore backup: %v", err)
	}
	return nil
}

// forkMigrationHistory will fork the migration history from source database to target database based on backup migration history version.
// This is needed only when backup migration history version is not empty.
func forkMigrationHistory(sourceDatabase, targetDatabase *api.Database, backup *api.Backup, task *api.Task, logger *zap.Logger) error {
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
		if history.Version == backup.MigrationHistoryVersion {
			break
		}
	}

	for _, history := range forkList {
		m := &db.MigrationInfo{
			Version:     history.Version,
			Namespace:   targetDatabase.Name,
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

	// Add a branch migration history record.
	// TODO(spinningbot): fill in the new issue ID.
	m := &db.MigrationInfo{
		Version:     strings.Join([]string{time.Now().Format("20060102150405"), strconv.Itoa(task.ID)}, "."),
		Namespace:   targetDatabase.Name,
		Database:    targetDatabase.Name,
		Environment: targetDatabase.Instance.Environment.Name,
		Engine:      db.MigrationEngine(targetDatabase.Project.WorkflowType),
		Type:        db.Branch,
		Description: fmt.Sprintf("Branched from backup %q of database %q.", backup.Name, sourceDatabase.Name),
		Creator:     task.Creator.Name,
		IssueId:     "TODO: newIssueID",
		Payload:     "",
	}
	if err := targetDriver.ExecuteMigration(context.Background(), m, ""); err != nil {
		return fmt.Errorf("failed to create migration history: %w", err)
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
