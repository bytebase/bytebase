package server

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/bin/bb/connect"
	"github.com/bytebase/bytebase/bin/bb/restore/mysqlrestore"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/db"
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
		if common.ErrorCode(err) == common.ENOTFOUND {
			return true, "", fmt.Errorf("target database %q not found in instance %q: %w", payload.DatabaseName, task.Instance.Name, err)
		}
		return true, "", fmt.Errorf("failed to find target database %q in instance %q: %w", payload.DatabaseName, task.Instance.Name, err)
	}

	exec.l.Debug("Start database restore from backup...",
		zap.String("source_instance", sourceDatabase.Instance.Name),
		zap.String("source_database", sourceDatabase.Name),
		zap.String("target_instance", targetDatabase.Instance.Name),
		zap.String("target_database", targetDatabase.Name),
		zap.String("backup", backup.Name),
	)

	// Restore the database to the target database.
	if err := restoreDatabase(targetDatabase, backup, server.dataDir); err != nil {
		return true, "", err
	}

	// TODO(tianzhou): This should be done in the same transaction as restoreDatabase to guarantee consistency.
	// For now, we do this after restoreDatabase, since this one is unlikely to fail.
	if err := createBranchMigrationHistory(ctx, server, sourceDatabase, targetDatabase, backup, task, exec.l); err != nil {
		return true, "", err
	}

	// Patch the backup id after we successfully restore the database using the backup.
	// restoringDatabase is changing the customer database instance, while here we are changing our own meta db,
	// and since we can't guarantee cross database transaction consistency, there is always a chance to have
	// inconsistent data. We choose to do Patch afterwards since this one is unlikely to fail.
	databasePatch := &api.DatabasePatch{
		ID:             targetDatabase.ID,
		UpdaterId:      api.SYSTEM_BOT_ID,
		SourceBackupId: &backup.ID,
	}
	if _, err = server.DatabaseService.PatchDatabase(context.Background(), databasePatch); err != nil {
		return true, "", fmt.Errorf("failed to patch database source backup ID after restore: %w", err)
	}

	return true, fmt.Sprintf("Restored database %q from backup %q", targetDatabase.Name, backup.Name), nil
}

// restoreDatabase will restore the database from a backup
func restoreDatabase(database *api.Database, backup *api.Backup, dataDir string) error {
	instance := database.Instance
	conn, err := connect.NewMysql(instance.Username, instance.Password, instance.Host, instance.Port, database.Name, nil /* tlsConfig */)
	if err != nil {
		return fmt.Errorf("failed to connect database: %w", err)
	}
	defer conn.Close()

	backupPath := backup.Path
	if !filepath.IsAbs(backupPath) {
		backupPath = filepath.Join(dataDir, backupPath)
	}

	f, err := os.OpenFile(backupPath, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to open backup file at %s: %w", backupPath, err)
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	if err := mysqlrestore.Restore(conn, sc); err != nil {
		return fmt.Errorf("failed to restore backup: %w", err)
	}
	return nil
}

// createBranchMigrationHistory creates a migration history with "BRANCH" type. We choose NOT to copy over
// all migrationhistory from source database because that might be expensive (e.g. we may use restore to
// create many ephemeral databases from backup for testing purpose)
func createBranchMigrationHistory(ctx context.Context, server *Server, sourceDatabase, targetDatabase *api.Database, backup *api.Backup, task *api.Task, logger *zap.Logger) error {
	targetDriver, err := GetDatabaseDriver(targetDatabase.Instance, targetDatabase.Name, logger)
	if err != nil {
		return err
	}
	defer targetDriver.Close(ctx)

	issueFind := &api.IssueFind{
		PipelineId: &task.PipelineId,
	}
	issue, err := server.IssueService.FindIssue(ctx, issueFind)
	if err != nil {
		// Not all pipelines belong to an issue, so it's OK if ENOTFOUND
		if common.ErrorCode(err) != common.ENOTFOUND {
			return fmt.Errorf("failed to fetch containing issue when creating the migration history: %v, err: %w", task.Name, err)
		}
	}

	// Add a branch migration history record.
	issueId := ""
	if issue != nil {
		issueId = strconv.Itoa(issue.ID)
	}
	description := fmt.Sprintf("Restored from backup %q of database %q.", backup.Name, sourceDatabase.Name)
	if sourceDatabase.InstanceId != targetDatabase.InstanceId {
		description = fmt.Sprintf("Restored from backup %q of database %q in instance %q.", backup.Name, sourceDatabase.Name, sourceDatabase.Instance.Name)
	}
	m := &db.MigrationInfo{
		Version:     defaultMigrationVersionFromTaskId(task.ID),
		Namespace:   targetDatabase.Name,
		Database:    targetDatabase.Name,
		Environment: targetDatabase.Instance.Environment.Name,
		Engine:      db.MigrationEngine(targetDatabase.Project.WorkflowType),
		Type:        db.Branch,
		Description: description,
		Creator:     task.Creator.Name,
		IssueId:     issueId,
		Payload:     "",
	}
	if err := targetDriver.ExecuteMigration(ctx, m, ""); err != nil {
		return fmt.Errorf("failed to create migration history: %w", err)
	}
	return nil
}
