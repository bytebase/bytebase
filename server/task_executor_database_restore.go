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
func (exec *DatabaseRestoreTaskExecutor) RunOnce(ctx context.Context, server *Server, task *api.Task) (terminated bool, result *api.TaskRunResultPayload, err error) {
	defer func() {
		if r := recover(); r != nil {
			panicErr, ok := r.(error)
			if !ok {
				panicErr = fmt.Errorf("%v", r)
			}
			exec.l.Error("DatabaseRestoreTaskExecutor PANIC RECOVER", zap.Error(panicErr), zap.Stack("stack"))
			terminated = true
			err = fmt.Errorf("encounter internal error when restoring the database")
		}
	}()

	payload := &api.TaskDatabaseRestorePayload{}
	if err := json.Unmarshal([]byte(task.Payload), payload); err != nil {
		return true, nil, fmt.Errorf("invalid database backup payload: %w", err)
	}

	backup, err := server.store.GetBackupByID(ctx, payload.BackupID)
	if err != nil {
		return true, nil, fmt.Errorf("failed to find backup with ID[%d], error[%w]", payload.BackupID, err)
	}
	if backup == nil {
		return true, nil, fmt.Errorf("backup with ID[%d] not found", payload.BackupID)
	}

	sourceDatabase, err := server.store.GetDatabase(ctx, &api.DatabaseFind{ID: &backup.DatabaseID})
	if err != nil {
		return true, nil, fmt.Errorf("failed to find database for the backup: %w", err)
	}
	if sourceDatabase == nil {
		return true, nil, fmt.Errorf("source database ID not found %v", backup.DatabaseID)
	}

	targetDatabaseFind := &api.DatabaseFind{
		InstanceID: &task.InstanceID,
		Name:       &payload.DatabaseName,
	}
	targetDatabase, err := server.store.GetDatabase(ctx, targetDatabaseFind)
	if err != nil {
		return true, nil, fmt.Errorf("failed to find target database %q in instance %q: %w", payload.DatabaseName, task.Instance.Name, err)
	}
	if targetDatabase == nil {
		return true, nil, fmt.Errorf("target database %q not found in instance %q: %w", payload.DatabaseName, task.Instance.Name, err)
	}

	exec.l.Debug("Start database restore from backup...",
		zap.String("source_instance", sourceDatabase.Instance.Name),
		zap.String("source_database", sourceDatabase.Name),
		zap.String("target_instance", targetDatabase.Instance.Name),
		zap.String("target_database", targetDatabase.Name),
		zap.String("backup", backup.Name),
	)

	// Restore the database to the target database.
	if err := exec.restoreDatabase(ctx, targetDatabase.Instance, targetDatabase.Name, backup, server.profile.DataDir); err != nil {
		return true, nil, err
	}

	// TODO(tianzhou): This should be done in the same transaction as restoreDatabase to guarantee consistency.
	// For now, we do this after restoreDatabase, since this one is unlikely to fail.
	migrationID, version, err := createBranchMigrationHistory(ctx, server, sourceDatabase, targetDatabase, backup, task, exec.l)
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
		return true, nil, fmt.Errorf("failed to patch database source with ID[%d] and backup ID[%d] after restore, error[%w]", targetDatabase.ID, backup.ID, err)
	}

	// Sync database schema after restore is completed.
	server.syncEngineVersionAndSchema(ctx, targetDatabase.Instance)

	return true, &api.TaskRunResultPayload{
		Detail:      fmt.Sprintf("Restored database %q from backup %q", targetDatabase.Name, backup.Name),
		MigrationID: migrationID,
		Version:     version,
	}, nil
}

// restoreDatabase will restore the database from a backup
func (exec *DatabaseRestoreTaskExecutor) restoreDatabase(ctx context.Context, instance *api.Instance, databaseName string, backup *api.Backup, dataDir string) error {
	driver, err := getAdminDatabaseDriver(ctx, instance, databaseName, exec.l)
	if err != nil {
		return err
	}
	defer driver.Close(ctx)

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

	if err := driver.Restore(ctx, sc); err != nil {
		return fmt.Errorf("failed to restore backup: %w", err)
	}
	return nil
}

// createBranchMigrationHistory creates a migration history with "BRANCH" type. We choose NOT to copy over
// all migrationhistory from source database because that might be expensive (e.g. we may use restore to
// create many ephemeral databases from backup for testing purpose)
// Returns migration history id and the version on success
func createBranchMigrationHistory(ctx context.Context, server *Server, sourceDatabase, targetDatabase *api.Database, backup *api.Backup, task *api.Task, logger *zap.Logger) (int64, string, error) {
	targetDriver, err := getAdminDatabaseDriver(ctx, targetDatabase.Instance, targetDatabase.Name, logger)
	if err != nil {
		return -1, "", err
	}
	defer targetDriver.Close(ctx)

	issue, err := server.store.GetIssueByPipelineID(ctx, task.PipelineID)
	if err != nil {
		return -1, "", fmt.Errorf("failed to fetch containing issue when creating the migration history: %v, err: %w", task.Name, err)
	}

	// Add a branch migration history record.
	issueID := ""
	if issue != nil {
		issueID = strconv.Itoa(issue.ID)
	}
	description := fmt.Sprintf("Restored from backup %q of database %q.", backup.Name, sourceDatabase.Name)
	if sourceDatabase.InstanceID != targetDatabase.InstanceID {
		description = fmt.Sprintf("Restored from backup %q of database %q in instance %q.", backup.Name, sourceDatabase.Name, sourceDatabase.Instance.Name)
	}
	// TODO(d): support semantic versioning.
	m := &db.MigrationInfo{
		ReleaseVersion: server.profile.Version,
		Version:        common.DefaultMigrationVersion(),
		Namespace:      targetDatabase.Name,
		Database:       targetDatabase.Name,
		Environment:    targetDatabase.Instance.Environment.Name,
		Source:         db.MigrationSource(targetDatabase.Project.WorkflowType),
		Type:           db.Branch,
		Description:    description,
		Creator:        task.Creator.Name,
		IssueID:        issueID,
		Payload:        "",
	}
	migrationID, _, err := targetDriver.ExecuteMigration(ctx, m, "")
	if err != nil {
		return -1, "", fmt.Errorf("failed to create migration history: %w", err)
	}
	return migrationID, m.Version, nil
}
