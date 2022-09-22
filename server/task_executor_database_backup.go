package server

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"

	"github.com/pkg/errors"
	"go.uber.org/zap"
	"golang.org/x/sys/unix"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common/log"
	"github.com/bytebase/bytebase/plugin/db"
)

const (
	// Do not dump backup file when the available file system space is less than 500MB.
	minAvailableFSBytes = 500 * 1024 * 1024
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
		return true, nil, errors.Errorf("backup %v not found", payload.BackupID)
	}

	if backup.StorageBackend == api.BackupStorageBackendLocal {
		backupFileDir := filepath.Dir(filepath.Join(server.profile.DataDir, backup.Path))
		availableBytes, err := getAvailableFSSpace(backupFileDir)
		if err != nil {
			return true, nil, errors.Wrapf(err, "failed to get available file system space, backup file dir is %s", backupFileDir)
		}
		if availableBytes < minAvailableFSBytes {
			return true, nil, errors.Errorf("the available file system space %dMB is less than the minimal threshold %dMB", availableBytes/1024/1024, minAvailableFSBytes/1024/1024)
		}
	}

	log.Debug("Start database backup.", zap.String("instance", task.Instance.Name), zap.String("database", task.Database.Name), zap.String("backup", backup.Name))
	backupPayload, backupErr := exec.backupDatabase(ctx, server, task.Instance, task.Database.Name, backup)
	backupStatus := string(api.BackupStatusDone)
	comment := ""
	if backupErr != nil {
		backupStatus = string(api.BackupStatusFailed)
		comment = backupErr.Error()
	}
	backupPatch := api.BackupPatch{
		ID:        backup.ID,
		Status:    &backupStatus,
		UpdaterID: api.SystemBotID,
		Comment:   &comment,
		Payload:   &backupPayload,
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

// getAvailableFSSpace gets the free space of the mounted filesystem.
// path is the pathname of any file within the mounted filesystem.
// It calls syscall statfs under the hood.
// Returns available space in bytes.
func getAvailableFSSpace(path string) (uint64, error) {
	var stat unix.Statfs_t
	if err := unix.Statfs(path, &stat); err != nil {
		return 0, errors.Wrap(err, "failed to call syscall statfs")
	}
	// Ref: https://man7.org/linux/man-pages/man2/statfs.2.html
	// Bavail: Free blocks available to unprivileged user.
	// Bsize: Optimal transfer block size.
	return stat.Bavail * uint64(stat.Bsize), nil
}

func dumpBackupFile(ctx context.Context, driver db.Driver, databaseName, backupFilePath string) (string, error) {
	backupFile, err := os.Create(backupFilePath)
	if err != nil {
		return "", errors.Errorf("failed to open backup path %q", backupFilePath)
	}
	defer backupFile.Close()
	payload, err := driver.Dump(ctx, databaseName, backupFile, false /* schemaOnly */)
	if err != nil {
		return "", errors.Wrapf(err, "failed to dump database %q to local backup file %q", databaseName, backupFilePath)
	}
	return payload, nil
}

// backupDatabase will take a backup of a database.
func (*DatabaseBackupTaskExecutor) backupDatabase(ctx context.Context, server *Server, instance *api.Instance, databaseName string, backup *api.Backup) (string, error) {
	driver, err := server.getAdminDatabaseDriver(ctx, instance, databaseName)
	if err != nil {
		return "", err
	}
	defer driver.Close(ctx)

	backupFilePathLocal := filepath.Join(server.profile.DataDir, backup.Path)
	payload, err := dumpBackupFile(ctx, driver, databaseName, backupFilePathLocal)
	if err != nil {
		return "", errors.Wrapf(err, "failed to dump backup file %q", backupFilePathLocal)
	}

	switch backup.StorageBackend {
	case api.BackupStorageBackendLocal:
		return payload, nil
	case api.BackupStorageBackendS3:
		log.Debug("Uploading backup to s3 bucket.", zap.String("bucket", server.s3Client.GetBucket()), zap.String("path", backupFilePathLocal))
		bucketFileToUpload, err := os.Open(backupFilePathLocal)
		if err != nil {
			return "", errors.Wrapf(err, "failed to open backup file %q for uploading to s3 bucket", backupFilePathLocal)
		}
		defer bucketFileToUpload.Close()

		if _, err := server.s3Client.UploadObject(ctx, backup.Path, bucketFileToUpload); err != nil {
			return "", errors.Wrapf(err, "failed to upload backup to AWS S3")
		}
		log.Debug("Successfully uploaded backup to s3 bucket.")

		if err := os.Remove(backupFilePathLocal); err != nil {
			log.Warn("Failed to remove the local backup file after uploading to s3 bucket.", zap.String("path", backupFilePathLocal), zap.Error(err))
		} else {
			log.Debug("Successfully removed the local backup file after uploading to s3 bucket.", zap.String("path", backupFilePathLocal))
		}
		return payload, nil
	default:
		return "", errors.Errorf("backup to %s not implemented yet", backup.StorageBackend)
	}
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
