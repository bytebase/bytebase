package taskrun

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	"github.com/bytebase/bytebase/backend/component/state"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/mysql"
	"github.com/bytebase/bytebase/backend/plugin/db/pg"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	bbs3 "github.com/bytebase/bytebase/backend/plugin/storage/s3"
	"github.com/bytebase/bytebase/backend/runner/backuprun"
	"github.com/bytebase/bytebase/backend/runner/schemasync"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
)

// NewPITRRestoreExecutor creates a PITR restore task executor.
func NewPITRRestoreExecutor(store *store.Store, dbFactory *dbfactory.DBFactory, s3Client *bbs3.Client, schemaSyncer *schemasync.Syncer, stateCfg *state.State, profile config.Profile) Executor {
	return &PITRRestoreExecutor{
		store:        store,
		dbFactory:    dbFactory,
		s3Client:     s3Client,
		schemaSyncer: schemaSyncer,
		stateCfg:     stateCfg,
		profile:      profile,
	}
}

// PITRRestoreExecutor is the PITR restore task executor.
type PITRRestoreExecutor struct {
	store        *store.Store
	dbFactory    *dbfactory.DBFactory
	s3Client     *bbs3.Client
	schemaSyncer *schemasync.Syncer
	stateCfg     *state.State
	profile      config.Profile
}

// RunOnce will run the PITR restore task executor once.
func (exec *PITRRestoreExecutor) RunOnce(ctx context.Context, task *store.TaskMessage) (terminated bool, result *api.TaskRunResultPayload, err error) {
	log.Info("Run PITR restore task", zap.String("task", task.Name))
	payload := api.TaskDatabasePITRRestorePayload{}
	if err := json.Unmarshal([]byte(task.Payload), &payload); err != nil {
		return true, nil, errors.Wrapf(err, "invalid PITR restore payload: %s", task.Payload)
	}

	if (payload.BackupID == nil) == (payload.PointInTimeTs == nil) {
		return true, nil, errors.Errorf("only one of BackupID and time point can be set")
	}

	if !((payload.DatabaseName == nil) == (payload.TargetInstanceID == nil)) {
		return true, nil, errors.Errorf("DatabaseName and TargetInstanceID must be both set or unset")
	}

	// There are 2 * 2 = 4 kinds of task by combination of the following cases:
	// 1. in-place or restore to new database: the latter does not create database with _pitr/_del suffix
	// 2. backup restore or Point-in-Time restore: the former does not apply binlog/wal

	if payload.BackupID != nil {
		// Restore Backup
		resultPayload, err := exec.doBackupRestore(ctx, exec.store, exec.dbFactory, exec.s3Client, exec.schemaSyncer, exec.profile, task, payload)
		return true, resultPayload, err
	}

	resultPayload, err := exec.doPITRRestore(ctx, exec.dbFactory, exec.s3Client, exec.profile, task, payload)
	return true, resultPayload, err
}

func (exec *PITRRestoreExecutor) doBackupRestore(ctx context.Context, stores *store.Store, dbFactory *dbfactory.DBFactory, s3Client *bbs3.Client, schemaSyncer *schemasync.Syncer, profile config.Profile, task *store.TaskMessage, payload api.TaskDatabasePITRRestorePayload) (*api.TaskRunResultPayload, error) {
	instance, err := stores.GetInstanceV2(ctx, &store.FindInstanceMessage{UID: &task.InstanceID})
	if err != nil {
		return nil, errors.Wrap(err, "failed to find database for the backup")
	}
	backup, err := stores.GetBackupByUID(ctx, *payload.BackupID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find backup with ID %d", *payload.BackupID)
	}
	if backup == nil {
		return nil, errors.Errorf("backup with ID %d not found", *payload.BackupID)
	}

	// TODO(dragonly): We should let users restore the backup even if the source database is gone.
	sourceDatabase, err := stores.GetDatabaseV2(ctx, &store.FindDatabaseMessage{UID: &backup.DatabaseUID})
	if err != nil {
		return nil, errors.Wrap(err, "failed to find database for the backup")
	}
	if sourceDatabase == nil {
		return nil, errors.Errorf("source database ID not found %v", backup.DatabaseUID)
	}

	if payload.TargetInstanceID == nil {
		// Backup restore in place
		if instance.Engine == db.Postgres {
			issue, err := stores.GetIssueV2(ctx, &store.FindIssueMessage{PipelineID: &task.PipelineID})
			if err != nil {
				return nil, err
			}
			return exec.doRestoreInPlacePostgres(ctx, stores, dbFactory, profile, issue, task, payload)
		}
		return nil, errors.Errorf("we only support backup restore replace for PostgreSQL now")
	}

	targetInstance, err := exec.store.GetInstanceV2(ctx, &store.FindInstanceMessage{UID: payload.TargetInstanceID})
	if err != nil {
		return nil, err
	}
	targetDatabase, err := exec.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{InstanceID: &targetInstance.ResourceID, DatabaseName: payload.DatabaseName})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find target database %q in instance %q", *payload.DatabaseName, instance.Title)
	}
	if targetDatabase == nil {
		return nil, errors.Wrapf(err, "target database %q not found in instance %q", *payload.DatabaseName, instance.Title)
	}
	if err != nil {
		return nil, err
	}
	log.Debug("Start database restore from backup...",
		zap.String("source_instance", sourceDatabase.InstanceID),
		zap.String("source_database", sourceDatabase.DatabaseName),
		zap.String("target_instance", targetInstance.ResourceID),
		zap.String("target_database", targetDatabase.DatabaseName),
		zap.String("backup", backup.Name),
	)

	// Restore the database to the target database.
	if err := exec.restoreDatabase(ctx, dbFactory, s3Client, profile, targetInstance, targetDatabase, backup); err != nil {
		return nil, err
	}
	// TODO(zp): This should be done in the same transaction as restoreDatabase to guarantee consistency.
	// For now, we do this after restoreDatabase, since this one is unlikely to fail.
	migrationID, version, err := createBranchMigrationHistory(ctx, stores, dbFactory, profile, targetInstance, sourceDatabase, targetDatabase, backup, task)
	if err != nil {
		return nil, err
	}

	// Patch the backup id after we successfully restore the database using the backup.
	// restoringDatabase is changing the customer database instance, while here we are changing our own meta db,
	// and since we can't guarantee cross database transaction consistency, there is always a chance to have
	// inconsistent data. We choose to do Patch afterwards since this one is unlikely to fail.
	if _, err := stores.UpdateDatabase(ctx, &store.UpdateDatabaseMessage{
		InstanceID:     targetDatabase.InstanceID,
		DatabaseName:   targetDatabase.DatabaseName,
		SourceBackupID: &backup.UID,
	}, api.SystemBotID); err != nil {
		return nil, errors.Wrapf(err, "failed to update database %d backup source ID %d after restore", targetDatabase.UID, backup.UID)
	}

	// Sync database schema after restore is completed.
	if err := schemaSyncer.SyncDatabaseSchema(ctx, targetDatabase, true /* force */); err != nil {
		log.Error("failed to sync database schema",
			zap.String("instanceName", targetDatabase.InstanceID),
			zap.String("databaseName", targetDatabase.DatabaseName),
			zap.Error(err),
		)
	}

	return &api.TaskRunResultPayload{
		Detail:        fmt.Sprintf("Restored database %q from backup %q", targetDatabase.DatabaseName, backup.Name),
		MigrationID:   migrationID,
		ChangeHistory: fmt.Sprintf("instances/%s/databases/%s/migrations/%s", instance.ResourceID, targetDatabase.DatabaseName, migrationID),
		Version:       version,
	}, nil
}

func (exec *PITRRestoreExecutor) doPITRRestore(ctx context.Context, dbFactory *dbfactory.DBFactory, s3Client *bbs3.Client, profile config.Profile, task *store.TaskMessage, payload api.TaskDatabasePITRRestorePayload) (*api.TaskRunResultPayload, error) {
	instance, err := exec.store.GetInstanceV2(ctx, &store.FindInstanceMessage{UID: &task.InstanceID})
	if err != nil {
		return nil, err
	}
	database, err := exec.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{UID: task.DatabaseID})
	if err != nil {
		return nil, err
	}

	sourceDriver, err := dbFactory.GetAdminDatabaseDriver(ctx, instance, nil /* database */)
	if err != nil {
		return nil, err
	}
	defer sourceDriver.Close(ctx)

	targetDriver := sourceDriver
	if payload.TargetInstanceID != nil {
		targetInstance, err := exec.store.GetInstanceV2(ctx, &store.FindInstanceMessage{UID: payload.TargetInstanceID})
		if err != nil {
			return nil, err
		}
		if targetDriver, err = dbFactory.GetAdminDatabaseDriver(ctx, targetInstance, nil /* database */); err != nil {
			return nil, err
		}
	}
	// DB.Close is idempotent, so we can feel free to assign sourceDriver to targetDriver first.
	defer targetDriver.Close(ctx)

	issue, err := exec.store.GetIssueV2(ctx, &store.FindIssueMessage{PipelineID: &task.PipelineID})
	if err != nil {
		return nil, err
	}
	if issue == nil {
		return nil, errors.Errorf("issue not found for pipeline %v", task.PipelineID)
	}

	backupStatus := api.BackupStatusDone
	backupList, err := exec.store.ListBackupV2(ctx, &store.FindBackupMessage{DatabaseUID: task.DatabaseID, Status: &backupStatus})
	if err != nil {
		return nil, err
	}
	log.Debug("Found backup list", zap.Array("backups", store.ZapBackupArray(backupList)))

	mysqlSourceDriver, sourceOk := sourceDriver.(*mysql.Driver)
	mysqlTargetDriver, targetOk := targetDriver.(*mysql.Driver)
	if (!sourceOk) || (!targetOk) {
		log.Error("Failed to cast driver to mysql.Driver")
		return nil, errors.Errorf("[internal] cast driver to mysql.Driver failed")
	}

	log.Debug("Downloading all binlog files")
	if err := mysqlSourceDriver.FetchAllBinlogFiles(ctx, true /* downloadLatestBinlogFile */, s3Client); err != nil {
		return nil, err
	}

	targetTs := *payload.PointInTimeTs
	log.Debug("Getting latest backup before or equal to targetTs", zap.Int64("targetTs", targetTs))
	backup, targetBinlogInfo, err := mysqlSourceDriver.GetLatestBackupBeforeOrEqualTs(ctx, backupList, targetTs, s3Client)
	if err != nil {
		targetTsHuman := time.Unix(targetTs, 0).Format(time.RFC822)
		log.Error("Failed to get backup before or equal to time",
			zap.Int64("targetTs", targetTs),
			zap.String("targetTsHuman", targetTsHuman),
			zap.Error(err))
		return nil, errors.Wrapf(err, "failed to get latest backup before or equal to %s", targetTsHuman)
	}
	startBinlogInfo := backup.Payload.BinlogInfo
	binlogDir := common.GetBinlogAbsDir(profile.DataDir, instance.UID)
	log.Debug("Got latest backup before or equal to targetTs", zap.String("backup", backup.Name))

	backupAbsPathLocal := backuprun.GetBackupAbsFilePath(profile.DataDir, backup.DatabaseUID, backup.Name)
	if backup.StorageBackend == api.BackupStorageBackendS3 {
		if err := downloadBackupFileFromCloud(ctx, s3Client, backup.Path, backupAbsPathLocal); err != nil {
			return nil, errors.Wrapf(err, "failed to download backup %q from S3", backup.Path)
		}
		defer os.Remove(backupAbsPathLocal)
		replayBinlogPathList, err := downloadBinlogFilesFromCloud(ctx, s3Client, startBinlogInfo, *targetBinlogInfo, binlogDir)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to download binlog files from %s to %s from S3", startBinlogInfo.FileName, targetBinlogInfo.FileName)
		}
		defer func() {
			for _, binlogPath := range replayBinlogPathList {
				if err := os.Remove(binlogPath); err != nil {
					log.Warn("Failed to remove downloaded local binlog file after PITR", zap.String("path", binlogPath))
				}
			}
		}()
	}

	backupFile, err := os.Open(backupAbsPathLocal)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to open backup file %q", backupAbsPathLocal)
	}
	defer backupFile.Close()
	log.Debug("Successfully opened backup file", zap.String("filename", backupAbsPathLocal))

	log.Debug("Start creating and restoring PITR database",
		zap.String("instance", instance.ResourceID),
		zap.String("database", database.DatabaseName),
	)

	if err := exec.updateProgress(ctx, mysqlTargetDriver, task.ID, backupFile, startBinlogInfo, *targetBinlogInfo, binlogDir); err != nil {
		return nil, errors.Wrap(err, "failed to setup progress update process")
	}

	if payload.DatabaseName != nil {
		// case 1: PITR to a new database.
		if err := mysqlTargetDriver.RestoreBackupToDatabase(ctx, backupFile, *payload.DatabaseName); err != nil {
			log.Error("failed to restore full backup in the new database",
				zap.Int("issueID", issue.UID),
				zap.String("databaseName", *payload.DatabaseName),
				zap.Error(err))
			return nil, errors.Wrap(err, "failed to restore full backup in the new database")
		}
		if err := mysqlTargetDriver.ReplayBinlogToDatabase(ctx, database.DatabaseName, *payload.DatabaseName, startBinlogInfo, *targetBinlogInfo, targetTs, mysqlSourceDriver.GetBinlogDir()); err != nil {
			log.Error("failed to perform a PITR restore in the new database",
				zap.Int("issueID", issue.UID),
				zap.String("databaseName", *payload.DatabaseName),
				zap.Error(err))
			return nil, errors.Wrap(err, "failed to perform a PITR restore in the new database")
		}
	} else {
		// case 2: in-place PITR.
		if err := mysqlTargetDriver.RestoreBackupToPITRDatabase(ctx, backupFile, database.DatabaseName, issue.CreatedTime.Unix()); err != nil {
			log.Error("failed to restore full backup in the PITR database",
				zap.Int("issueID", issue.UID),
				zap.String("databaseName", database.DatabaseName),
				zap.Error(err))
			return nil, errors.Wrap(err, "failed to perform a backup restore in the PITR database")
		}
		if err := mysqlTargetDriver.ReplayBinlogToPITRDatabase(ctx, database.DatabaseName, startBinlogInfo, *targetBinlogInfo, issue.CreatedTime.Unix(), targetTs); err != nil {
			log.Error("failed to perform a PITR restore in the PITR database",
				zap.Int("issueID", issue.UID),
				zap.String("databaseName", database.DatabaseName),
				zap.Error(err))
			return nil, errors.Wrap(err, "failed to replay binlog in the PITR database")
		}
	}

	targetDatabaseName := database.DatabaseName
	if payload.DatabaseName != nil {
		targetDatabaseName = *payload.DatabaseName
	}
	log.Info("PITR restore success", zap.String("target database", targetDatabaseName))
	return &api.TaskRunResultPayload{
		Detail: fmt.Sprintf("PITR restore success for target database %q", targetDatabaseName),
	}, nil
}

func downloadBinlogFilesFromCloud(ctx context.Context, client *bbs3.Client, startBinlogInfo, targetBinlogInfo api.BinlogInfo, binlogDir string) ([]string, error) {
	replayBinlogPathList, err := mysql.GetBinlogReplayList(startBinlogInfo, targetBinlogInfo, binlogDir)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get binlog replay list in directory %s", binlogDir)
	}
	for _, binlogFilePath := range replayBinlogPathList {
		// Use path.Join to compose a path on cloud which always uses / as the separator.
		filePathOnCloud := path.Join(common.GetBinlogRelativeDir(binlogDir), filepath.Base(binlogFilePath))
		if err := client.DownloadFileFromCloud(ctx, binlogFilePath, filePathOnCloud); err != nil {
			return nil, errors.Wrapf(err, "failed to download binlog file %s from the cloud storage", binlogFilePath)
		}
	}
	return replayBinlogPathList, nil
}

func (*PITRRestoreExecutor) doRestoreInPlacePostgres(ctx context.Context, stores *store.Store, dbFactory *dbfactory.DBFactory, profile config.Profile, issue *store.IssueMessage, task *store.TaskMessage, payload api.TaskDatabasePITRRestorePayload) (*api.TaskRunResultPayload, error) {
	if payload.BackupID == nil {
		return nil, errors.Errorf("PITR for Postgres is not implemented")
	}

	backup, err := stores.GetBackupByUID(ctx, *payload.BackupID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find backup with ID %d", *payload.BackupID)
	}
	if backup == nil {
		return nil, errors.Errorf("backup with ID %d not found", *payload.BackupID)
	}
	backupFileName := backuprun.GetBackupAbsFilePath(profile.DataDir, backup.DatabaseUID, backup.Name)
	backupFile, err := os.Open(backupFileName)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to open backup file %q", backupFileName)
	}
	defer backupFile.Close()

	instance, err := stores.GetInstanceV2(ctx, &store.FindInstanceMessage{UID: &task.InstanceID})
	if err != nil {
		return nil, err
	}
	database, err := stores.GetDatabaseV2(ctx, &store.FindDatabaseMessage{UID: task.DatabaseID})
	if err != nil {
		return nil, err
	}
	driver, err := dbFactory.GetAdminDatabaseDriver(ctx, instance, database)
	if err != nil {
		return nil, err
	}
	defer driver.Close(ctx)

	pgDriver, ok := driver.(*pg.Driver)
	if !ok {
		log.Error("Failed to cast driver to pg.Driver")
		return nil, errors.Errorf("[internal] cast driver to pg.Driver failed")
	}
	originalOwner, err := pgDriver.GetCurrentDatabaseOwner()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get the OWNER of database %q", database.DatabaseName)
	}

	defaultDBDriver, err := dbFactory.GetAdminDatabaseDriver(ctx, instance, nil /* database */)
	if err != nil {
		return nil, err
	}
	defer defaultDBDriver.Close(ctx)
	db := defaultDBDriver.GetDB()
	pitrDatabaseName := util.GetPITRDatabaseName(database.DatabaseName, issue.CreatedTime.Unix())
	// If there's already a PITR database, it means there's a failed trial before this task execution.
	// We need to clean up the dirty state and start clean for idempotent task execution.
	if _, err := db.ExecContext(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS %s;", pitrDatabaseName)); err != nil {
		return nil, errors.Wrapf(err, "failed to drop the dirty PITR database %q left from a former task execution", pitrDatabaseName)
	}
	if _, err := db.ExecContext(ctx, fmt.Sprintf("CREATE DATABASE %s WITH OWNER %s;", pitrDatabaseName, originalOwner)); err != nil {
		return nil, errors.Wrapf(err, "failed to create the PITR database %q", pitrDatabaseName)
	}

	pitrDBDriver, err := dbFactory.GetAdminDatabaseDriver(ctx, instance, nil /* database */)
	if err != nil {
		return nil, err
	}
	defer pitrDBDriver.Close(ctx)
	if err := pitrDBDriver.Restore(ctx, backupFile); err != nil {
		return nil, errors.Wrapf(err, "failed to restore backup to the PITR database %q", pitrDatabaseName)
	}
	return &api.TaskRunResultPayload{
		Detail: fmt.Sprintf("Restored backup %q to the temporary PITR database %q", backup.Name, pitrDatabaseName),
	}, nil
}

func (exec *PITRRestoreExecutor) updateProgress(ctx context.Context, driver *mysql.Driver, taskID int, backupFile *os.File, startBinlogInfo, targetBinlogInfo api.BinlogInfo, binlogDir string) error {
	backupFileInfo, err := backupFile.Stat()
	if err != nil {
		return errors.Wrapf(err, "failed to get stat of backup file %q", backupFile.Name())
	}
	backupFileBytes := backupFileInfo.Size()
	replayBinlogPaths, err := mysql.GetBinlogReplayList(startBinlogInfo, targetBinlogInfo, binlogDir)
	if err != nil {
		return errors.Wrapf(err, "failed to get binlog replay list from %s to %s in binlog directory %q", startBinlogInfo.FileName, targetBinlogInfo.FileName, binlogDir)
	}
	totalBinlogBytes, err := common.GetFileSizeSum(replayBinlogPaths)
	if err != nil {
		return errors.Wrap(err, "failed to get file size sum of replay binlog files")
	}

	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		createdTs := time.Now().Unix()
		totalUnit := backupFileBytes + totalBinlogBytes
		exec.stateCfg.TaskProgress.Store(taskID, api.Progress{
			TotalUnit:     totalUnit,
			CompletedUnit: 0,
			CreatedTs:     createdTs,
			UpdatedTs:     createdTs,
		})
		for {
			select {
			case <-ticker.C:
				exec.stateCfg.TaskProgress.Store(taskID, api.Progress{
					TotalUnit:     totalUnit,
					CompletedUnit: driver.GetRestoredBackupBytes() + driver.GetReplayedBinlogBytes(),
					CreatedTs:     createdTs,
					UpdatedTs:     time.Now().Unix(),
				})
			case <-ctx.Done():
				return
			}
		}
	}()

	return nil
}

// restoreDatabase will restore the database to the instance from the backup.
func (*PITRRestoreExecutor) restoreDatabase(ctx context.Context, dbFactory *dbfactory.DBFactory, s3Client *bbs3.Client, profile config.Profile, instance *store.InstanceMessage, database *store.DatabaseMessage, backup *store.BackupMessage) error {
	driver, err := dbFactory.GetAdminDatabaseDriver(ctx, instance, database)
	if err != nil {
		return err
	}
	defer driver.Close(ctx)

	backupAbsPathLocal := filepath.Join(profile.DataDir, backup.Path)

	if backup.StorageBackend == api.BackupStorageBackendS3 {
		if err := downloadBackupFileFromCloud(ctx, s3Client, backup.Path, backupAbsPathLocal); err != nil {
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

func downloadBackupFileFromCloud(ctx context.Context, s3Client *bbs3.Client, backupPath, backupAbsPathLocal string) error {
	log.Debug("Downloading backup file from s3 bucket.", zap.String("path", backupPath))
	backupFileDownload, err := os.Create(backupAbsPathLocal)
	if err != nil {
		return errors.Wrapf(err, "failed to create local backup file %q for downloading from s3 bucket", backupAbsPathLocal)
	}
	defer backupFileDownload.Close()
	if _, err := s3Client.DownloadObject(ctx, backupPath, backupFileDownload); err != nil {
		return errors.Wrapf(err, "failed to download backup file %q from s3 bucket", backupPath)
	}
	log.Debug("Successfully downloaded backup file from s3 bucket.")
	return nil
}

// createBranchMigrationHistory creates a migration history with "BRANCH" type. We choose NOT to copy over
// all migration history from source database because that might be expensive (e.g. we may use restore to
// create many ephemeral databases from backup for testing purpose)
// Returns migration history id and the version on success.
func createBranchMigrationHistory(ctx context.Context, stores *store.Store, dbFactory *dbfactory.DBFactory, profile config.Profile, targetInstance *store.InstanceMessage, sourceDatabase, targetDatabase *store.DatabaseMessage, backup *store.BackupMessage, task *store.TaskMessage) (string, string, error) {
	targetInstanceEnvironment, err := stores.GetEnvironmentV2(ctx, &store.FindEnvironmentMessage{ResourceID: &targetInstance.EnvironmentID})
	if err != nil {
		return "", "", err
	}
	targetDatabaseProject, err := stores.GetProjectV2(ctx, &store.FindProjectMessage{ResourceID: &targetDatabase.ProjectID})
	if err != nil {
		return "", "", err
	}
	issue, err := stores.GetIssueV2(ctx, &store.FindIssueMessage{PipelineID: &task.PipelineID})
	if err != nil {
		return "", "", errors.Wrapf(err, "failed to fetch containing issue when creating the migration history: %v", task.Name)
	}
	// Add a branch migration history record.
	issueID := ""
	if issue != nil {
		issueID = strconv.Itoa(issue.UID)
	}
	description := fmt.Sprintf("Restored from backup %q of database %q.", backup.Name, sourceDatabase.DatabaseName)
	if sourceDatabase.InstanceID != targetDatabase.InstanceID {
		description = fmt.Sprintf("Restored from backup %q of database %q in instance %q.", backup.Name, sourceDatabase.DatabaseName, sourceDatabase.InstanceID)
	}
	creator, err := stores.GetUserByID(ctx, task.CreatorID)
	if err != nil {
		return "", "", err
	}

	// TODO(d): support semantic versioning.
	m := &db.MigrationInfo{
		InstanceID:     &task.InstanceID,
		ReleaseVersion: profile.Version,
		Version:        common.DefaultMigrationVersion(),
		Namespace:      targetDatabase.DatabaseName,
		Database:       targetDatabase.DatabaseName,
		DatabaseID:     &targetDatabase.UID,
		Environment:    targetInstanceEnvironment.ResourceID,
		Source:         db.MigrationSource(targetDatabaseProject.Workflow),
		Type:           db.Branch,
		Description:    description,
		Creator:        creator.Name,
		CreatorID:      creator.ID,
		IssueID:        issueID,
	}
	targetDriver, err := dbFactory.GetAdminDatabaseDriver(ctx, targetInstance, targetDatabase)
	if err != nil {
		return "", "", err
	}
	defer targetDriver.Close(ctx)

	migrationID, _, err := utils.ExecuteMigrationDefault(ctx, stores, targetDriver, m, "", nil, db.ExecuteOptions{})
	if err != nil {
		return "", "", errors.Wrap(err, "failed to create migration history")
	}
	return migrationID, m.Version, nil
}
