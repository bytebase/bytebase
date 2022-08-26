package server

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync/atomic"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/common/log"
	"github.com/bytebase/bytebase/plugin/db"

	"go.uber.org/zap"
)

// NewDatabaseRestoreTaskExecutor creates a new database restore task executor.
func NewDatabaseRestoreTaskExecutor() TaskExecutor {
	return &DatabaseRestoreTaskExecutor{}
}

// DatabaseRestoreTaskExecutor is the task executor for database restore.
type DatabaseRestoreTaskExecutor struct {
	completed int32
}

// IsCompleted tells the scheduler if the task execution has completed.
func (exec *DatabaseRestoreTaskExecutor) IsCompleted() bool {
	return atomic.LoadInt32(&exec.completed) == 1
}

// GetProgress returns the task progress.
func (*DatabaseRestoreTaskExecutor) GetProgress() api.Progress {
	return api.Progress{}
}

// RunOnce will run database restore once.
func (exec *DatabaseRestoreTaskExecutor) RunOnce(ctx context.Context, server *Server, task *api.Task) (terminated bool, result *api.TaskRunResultPayload, err error) {
	defer atomic.StoreInt32(&exec.completed, 1)
	payload := &api.TaskDatabaseRestorePayload{}
	if err := json.Unmarshal([]byte(task.Payload), payload); err != nil {
		return true, nil, errors.Wrap(err, "invalid database backup payload")
	}

	backup, err := server.store.GetBackupByID(ctx, payload.BackupID)
	if err != nil {
		return true, nil, errors.Wrapf(err, "failed to find backup with ID %d", payload.BackupID)
	}
	if backup == nil {
		return true, nil, errors.Errorf("backup with ID %d not found", payload.BackupID)
	}

	sourceDatabase, err := server.store.GetDatabase(ctx, &api.DatabaseFind{ID: &backup.DatabaseID})
	if err != nil {
		return true, nil, errors.Wrap(err, "failed to find database for the backup")
	}
	if sourceDatabase == nil {
		return true, nil, errors.Errorf("source database ID not found %v", backup.DatabaseID)
	}

	targetDatabaseFind := &api.DatabaseFind{
		InstanceID: &task.InstanceID,
		Name:       &payload.DatabaseName,
	}
	targetDatabase, err := server.store.GetDatabase(ctx, targetDatabaseFind)
	if err != nil {
		return true, nil, errors.Wrapf(err, "failed to find target database %q in instance %q", payload.DatabaseName, task.Instance.Name)
	}
	if targetDatabase == nil {
		return true, nil, errors.Wrapf(err, "target database %q not found in instance %q", payload.DatabaseName, task.Instance.Name)
	}

	log.Debug("Start database restore from backup...",
		zap.String("source_instance", sourceDatabase.Instance.Name),
		zap.String("source_database", sourceDatabase.Name),
		zap.String("target_instance", targetDatabase.Instance.Name),
		zap.String("target_database", targetDatabase.Name),
		zap.String("backup", backup.Name),
	)

	// Restore the database to the target database.
	if err := exec.restoreDatabase(ctx, server, targetDatabase.Instance, targetDatabase.Name, backup); err != nil {
		return true, nil, err
	}

	// TODO(tianzhou): This should be done in the same transaction as restoreDatabase to guarantee consistency.
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
		return true, nil, errors.Wrapf(err, "failed to patch database source with ID %d and backup ID %d after restore", targetDatabase.ID, backup.ID)
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

// restoreDatabase will restore the database from a backup.
func (*DatabaseRestoreTaskExecutor) restoreDatabase(ctx context.Context, server *Server, instance *api.Instance, databaseName string, backup *api.Backup) error {
	driver, err := server.getAdminDatabaseDriver(ctx, instance, databaseName)
	if err != nil {
		return err
	}
	defer driver.Close(ctx)

	backupAbsPathLocal := filepath.Join(server.profile.DataDir, backup.Path)

	if backup.StorageBackend == api.BackupStorageBackendS3 {
		if err := downloadBackupFileFromCloud(ctx, server, backup.Path, backupAbsPathLocal); err != nil {
			return errors.Wrapf(err, "failed to download backup %q from S3", backup.Path)
		}
		defer os.Remove(backupAbsPathLocal)
	}

	backupFileLocal, err := os.Open(backupAbsPathLocal)
	if err != nil {
		return errors.Wrapf(err, "failed to open backup file at %s", backupAbsPathLocal)
	}
	defer backupFileLocal.Close()

	if err := driver.Restore(ctx, backupFileLocal); err != nil {
		return errors.Wrap(err, "failed to restore backup")
	}

	return nil
}

func downloadBackupFileFromCloud(ctx context.Context, server *Server, backupPath, backupAbsPathLocal string) error {
	log.Debug("Downloading backup file from s3 bucket.", zap.String("path", backupPath))
	backupFileDownload, err := os.Create(backupAbsPathLocal)
	if err != nil {
		return errors.Wrapf(err, "failed to create local backup file %q for downloading from s3 bucket", backupAbsPathLocal)
	}
	defer backupFileDownload.Close()
	if _, err := server.s3Client.DownloadObject(ctx, backupPath, backupFileDownload); err != nil {
		return errors.Wrapf(err, "failed to download backup file %q from s3 bucket", backupPath)
	}
	log.Debug("Successfully downloaded backup file from s3 bucket.")
	return nil
}

// createBranchMigrationHistory creates a migration history with "BRANCH" type. We choose NOT to copy over
// all migration history from source database because that might be expensive (e.g. we may use restore to
// create many ephemeral databases from backup for testing purpose)
// Returns migration history id and the version on success.
func createBranchMigrationHistory(ctx context.Context, server *Server, sourceDatabase, targetDatabase *api.Database, backup *api.Backup, task *api.Task) (int64, string, error) {
	targetDriver, err := server.getAdminDatabaseDriver(ctx, targetDatabase.Instance, targetDatabase.Name)
	if err != nil {
		return -1, "", err
	}
	defer targetDriver.Close(ctx)

	issue, err := server.store.GetIssueByPipelineID(ctx, task.PipelineID)
	if err != nil {
		return -1, "", errors.Wrapf(err, "failed to fetch containing issue when creating the migration history: %v", task.Name)
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
	}
	migrationID, _, err := targetDriver.ExecuteMigration(ctx, m, "")
	if err != nil {
		return -1, "", errors.Wrap(err, "failed to create migration history")
	}
	return migrationID, m.Version, nil
}
