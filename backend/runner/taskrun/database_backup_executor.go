package taskrun

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/sys/unix"

	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	"github.com/bytebase/bytebase/backend/component/state"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/db"
	bbs3 "github.com/bytebase/bytebase/backend/plugin/storage/s3"
	"github.com/bytebase/bytebase/backend/runner/backuprun"
	"github.com/bytebase/bytebase/backend/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

const (
	// Do not dump backup file when the available file system space is less than 500MB.
	minAvailableFSBytes = 500 * 1024 * 1024
)

// NewDatabaseBackupExecutor creates a new database backup task executor.
func NewDatabaseBackupExecutor(store *store.Store, dbFactory *dbfactory.DBFactory, s3Client *bbs3.Client, stateCfg *state.State, profile config.Profile) Executor {
	return &DatabaseBackupExecutor{
		store:     store,
		dbFactory: dbFactory,
		s3Client:  s3Client,
		stateCfg:  stateCfg,
		profile:   profile,
	}
}

// DatabaseBackupExecutor is the task executor for database backup.
type DatabaseBackupExecutor struct {
	store     *store.Store
	dbFactory *dbfactory.DBFactory
	s3Client  *bbs3.Client
	stateCfg  *state.State
	profile   config.Profile
}

// RunOnce will run database backup once.
// TODO: support cancellation.
func (exec *DatabaseBackupExecutor) RunOnce(ctx context.Context, _ context.Context, task *store.TaskMessage, taskRunUID int) (terminated bool, result *api.TaskRunResultPayload, err error) {
	exec.stateCfg.TaskRunExecutionStatuses.Store(taskRunUID,
		state.TaskRunExecutionStatus{
			ExecutionStatus: v1pb.TaskRun_PRE_EXECUTING,
			UpdateTime:      time.Now(),
		})

	payload := &api.TaskDatabaseBackupPayload{}
	if err := json.Unmarshal([]byte(task.Payload), payload); err != nil {
		return true, nil, errors.Wrap(err, "invalid database backup payload")
	}

	database, err := exec.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{UID: task.DatabaseID})
	if err != nil {
		return true, nil, err
	}
	instance, err := exec.store.GetInstanceV2(ctx, &store.FindInstanceMessage{
		ResourceID: &database.InstanceID,
	})
	if err != nil {
		return true, nil, err
	}
	backup, err := exec.store.GetBackupByUID(ctx, payload.BackupID)
	if err != nil {
		return true, nil, errors.Wrapf(err, "failed to find backup with ID %d", payload.BackupID)
	}
	if backup == nil {
		return true, nil, errors.Errorf("backup %v not found", payload.BackupID)
	}

	if backup.StorageBackend == api.BackupStorageBackendLocal {
		backupFileDir := filepath.Dir(filepath.Join(exec.profile.DataDir, backup.Path))
		availableBytes, err := getAvailableFSSpace(backupFileDir)
		if err != nil {
			return true, nil, errors.Wrapf(err, "failed to get available file system space, backup file dir is %s", backupFileDir)
		}
		if availableBytes < minAvailableFSBytes {
			return true, nil, errors.Errorf("the available file system space %dMB is less than the minimal threshold %dMB", availableBytes/1024/1024, minAvailableFSBytes/1024/1024)
		}
	}

	exec.stateCfg.TaskRunExecutionStatuses.Store(taskRunUID,
		state.TaskRunExecutionStatus{
			ExecutionStatus: v1pb.TaskRun_EXECUTING,
			UpdateTime:      time.Now(),
		})

	slog.Debug("Start database backup.", slog.String("instance", instance.Title), slog.String("database", database.DatabaseName), slog.String("backup", backup.Name))
	backupPayload, backupErr := exec.backupDatabase(ctx, exec.dbFactory, exec.s3Client, exec.profile, instance, database, backup)

	exec.stateCfg.TaskRunExecutionStatuses.Store(taskRunUID,
		state.TaskRunExecutionStatus{
			ExecutionStatus: v1pb.TaskRun_POST_EXECUTING,
			UpdateTime:      time.Now(),
		})

	backupStatus := string(api.BackupStatusDone)
	comment := ""
	if backupErr != nil {
		backupStatus = string(api.BackupStatusFailed)
		comment = backupErr.Error()
		if err := removeLocalBackupFile(exec.profile.DataDir, backup); err != nil {
			slog.Warn(err.Error())
		}
	}
	backupPatch := store.UpdateBackupMessage{
		UID:       backup.UID,
		Status:    &backupStatus,
		UpdaterID: api.SystemBotID,
		Comment:   &comment,
		Payload:   &backupPayload,
	}

	if _, err := exec.store.UpdateBackupV2(ctx, &backupPatch); err != nil {
		return true, nil, errors.Wrap(err, "failed to patch backup")
	}

	if backupErr != nil {
		return true, nil, backupErr
	}

	return true, &api.TaskRunResultPayload{
		Detail: fmt.Sprintf("Backup database %q", database.DatabaseName),
	}, nil
}

func removeLocalBackupFile(dataDir string, backup *store.BackupMessage) error {
	if backup.StorageBackend != api.BackupStorageBackendLocal {
		return nil
	}
	backupFilePath := backuprun.GetBackupAbsFilePath(dataDir, backup.DatabaseUID, backup.Name)
	if err := os.Remove(backupFilePath); err != nil {
		return errors.Wrapf(err, "failed to delete the local backup file %s", backupFilePath)
	}
	return nil
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

func dumpBackupFile(ctx context.Context, driver db.Driver, backupFilePath string) (string, error) {
	backupFile, err := os.Create(backupFilePath)
	if err != nil {
		return "", errors.Errorf("failed to open backup path %q", backupFilePath)
	}
	defer backupFile.Close()
	payload, err := driver.Dump(ctx, backupFile, false /* schemaOnly */)
	if err != nil {
		return "", errors.Wrapf(err, "failed to dump database to local backup file %q", backupFilePath)
	}
	return payload, nil
}

// backupDatabase will take a backup of a database.
func (*DatabaseBackupExecutor) backupDatabase(ctx context.Context, dbFactory *dbfactory.DBFactory, s3Client *bbs3.Client, profile config.Profile, instance *store.InstanceMessage, database *store.DatabaseMessage, backup *store.BackupMessage) (string, error) {
	driver, err := dbFactory.GetAdminDatabaseDriver(ctx, instance, database, db.ConnectionContext{})
	if err != nil {
		return "", err
	}
	defer driver.Close(ctx)

	backupFilePathLocal := filepath.Join(profile.DataDir, backup.Path)
	payload, err := dumpBackupFile(ctx, driver, backupFilePathLocal)
	if err != nil {
		return "", errors.Wrapf(err, "failed to dump backup file %q", backupFilePathLocal)
	}

	switch backup.StorageBackend {
	case api.BackupStorageBackendLocal:
		return payload, nil
	case api.BackupStorageBackendS3:
		slog.Debug("Uploading backup to s3 bucket.", slog.String("bucket", s3Client.GetBucket()), slog.String("path", backupFilePathLocal))
		bucketFileToUpload, err := os.Open(backupFilePathLocal)
		if err != nil {
			return "", errors.Wrapf(err, "failed to open backup file %q for uploading to s3 bucket", backupFilePathLocal)
		}
		defer bucketFileToUpload.Close()

		if _, err := s3Client.UploadObject(ctx, backup.Path, bucketFileToUpload); err != nil {
			return "", errors.Wrapf(err, "failed to upload backup to AWS S3")
		}
		slog.Debug("Successfully uploaded backup to s3 bucket.")

		if err := os.Remove(backupFilePathLocal); err != nil {
			slog.Warn("Failed to remove the local backup file after uploading to s3 bucket.", slog.String("path", backupFilePathLocal), log.BBError(err))
		} else {
			slog.Debug("Successfully removed the local backup file after uploading to s3 bucket.", slog.String("path", backupFilePathLocal))
		}
		return payload, nil
	default:
		return "", errors.Errorf("backup to %s not implemented yet", backup.StorageBackend)
	}
}
