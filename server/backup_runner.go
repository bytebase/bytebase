package server

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os"
	"path"
	"sync"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/common/log"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/plugin/db/mysql"
)

// NewBackupRunner creates a new backup runner.
func NewBackupRunner(server *Server, backupRunnerInterval time.Duration) *BackupRunner {
	return &BackupRunner{
		server:                    server,
		backupRunnerInterval:      backupRunnerInterval,
		downloadBinlogInstanceIDs: make(map[int]bool),
	}
}

// BackupRunner is the backup runner scheduling automatic backups.
type BackupRunner struct {
	server                    *Server
	backupRunnerInterval      time.Duration
	downloadBinlogInstanceIDs map[int]bool
	backupWg                  sync.WaitGroup
	downloadBinlogWg          sync.WaitGroup
	downloadBinlogMu          sync.Mutex
}

// Run is the runner for backup runner.
func (r *BackupRunner) Run(ctx context.Context, wg *sync.WaitGroup) {
	ticker := time.NewTicker(r.backupRunnerInterval)
	defer ticker.Stop()
	defer wg.Done()
	log.Debug("Auto backup runner started", zap.Duration("interval", r.backupRunnerInterval))
	runningTasks := make(map[int]bool)
	var mu sync.RWMutex
	for {
		select {
		case <-ticker.C:
			func() {
				defer func() {
					if r := recover(); r != nil {
						err, ok := r.(error)
						if !ok {
							err = errors.Errorf("%v", r)
						}
						log.Error("Auto backup runner PANIC RECOVER", zap.Error(err), zap.Stack("panic-stack"))
					}
				}()
				r.startAutoBackups(ctx, runningTasks, &mu)
				r.downloadBinlogFiles(ctx)
				r.purgeExpiredBackupData(ctx)
			}()
		case <-ctx.Done(): // if cancel() execute
			r.backupWg.Wait()
			r.downloadBinlogWg.Wait()
			return
		}
	}
}

// TODO(dragonly): Make best effort to assure that users could recover to at least RetentionPeriodTs ago.
// This may require pending deleting expired backup files and binlog files.
func (r *BackupRunner) purgeExpiredBackupData(ctx context.Context) {
	backupSettingList, err := r.server.store.FindBackupSetting(ctx, api.BackupSettingFind{})
	if err != nil {
		log.Error("Failed to find all the backup settings.", zap.Error(err))
		return
	}

	for _, bs := range backupSettingList {
		if bs.RetentionPeriodTs == api.BackupRetentionPeriodUnset {
			continue // next database
		}
		backupList, err := r.server.store.FindBackup(ctx, &api.BackupFind{DatabaseID: &bs.DatabaseID})
		if err != nil {
			log.Error("Failed to get backups for database.", zap.Int("databaseID", bs.DatabaseID), zap.String("database", bs.Database.Name))
			return
		}
		for _, backup := range backupList {
			backupTime := time.Unix(backup.UpdatedTs, 0)
			expireTime := backupTime.Add(time.Duration(bs.RetentionPeriodTs) * time.Second)
			if time.Now().After(expireTime) {
				log.Debug("Purging expired backup", zap.Int("databaseID", backup.DatabaseID), zap.String("backup", backup.Name), zap.String("storageBackend", string(backup.StorageBackend)))
				if err := r.purgeBackup(ctx, backup); err != nil {
					log.Error("Failed to purge backup", zap.String("backup", backup.Name), zap.Error(err))
				}
			}
		}
	}

	instanceList, err := r.server.store.FindInstance(ctx, &api.InstanceFind{})
	if err != nil {
		log.Error("Failed to find non-archived instances.", zap.Error(err))
		return
	}

	for _, instance := range instanceList {
		if instance.Engine != db.MySQL {
			continue
		}
		maxRetentionPeriodTs, err := r.getMaxRetentionPeriodTsForMySQLInstance(ctx, instance)
		if err != nil {
			log.Error("Failed to get max retention period for MySQL instance", zap.String("instance", instance.Name), zap.Error(err))
			continue
		}
		if maxRetentionPeriodTs == math.MaxInt {
			continue
		}
		if err := r.purgeBinlogFiles(ctx, instance.ID, maxRetentionPeriodTs); err != nil {
			log.Error("Failed to purge binlog files for instance", zap.String("instance", instance.Name), zap.Int("retentionPeriodTs", maxRetentionPeriodTs), zap.Error(err))
		}
	}
}

func (r *BackupRunner) getMaxRetentionPeriodTsForMySQLInstance(ctx context.Context, instance *api.Instance) (int, error) {
	backupSettingList, err := r.server.store.FindBackupSetting(ctx, api.BackupSettingFind{InstanceID: &instance.ID})
	if err != nil {
		log.Error("Failed to find backup settings for instance.", zap.String("instance", instance.Name), zap.Error(err))
		return 0, errors.Wrapf(err, "failed to find backup settings for instance %q", instance.Name)
	}
	maxRetentionPeriodTs := math.MaxInt
	for _, bs := range backupSettingList {
		if bs.RetentionPeriodTs != api.BackupRetentionPeriodUnset && bs.RetentionPeriodTs < maxRetentionPeriodTs {
			maxRetentionPeriodTs = bs.RetentionPeriodTs
		}
	}
	return maxRetentionPeriodTs, nil
}

func (r *BackupRunner) purgeBinlogFiles(ctx context.Context, instanceID, retentionPeriodTs int) error {
	binlogDir := getBinlogAbsDir(r.server.profile.DataDir, instanceID)
	switch r.server.profile.BackupStorageBackend {
	case api.BackupStorageBackendLocal:
		return r.purgeBinlogFilesLocal(binlogDir, retentionPeriodTs)
	case api.BackupStorageBackendS3:
		return r.purgeBinlogFilesOnCloud(ctx, binlogDir, retentionPeriodTs)
	default:
		return errors.Errorf("purge binlog files not implemented for storage backend %s", r.server.profile.BackupStorageBackend)
	}
}

func (r *BackupRunner) purgeBinlogFilesOnCloud(ctx context.Context, binlogDir string, retentionPeriodTs int) error {
	binlogDirOnCloud := common.GetBinlogRelativeDir(binlogDir)
	listOutput, err := r.server.s3Client.ListObjects(ctx, binlogDirOnCloud)
	if err != nil {
		return errors.Wrapf(err, "failed to list binlog dir %q in the cloud storage", binlogDirOnCloud)
	}
	var purgeBinlogPathList []string
	for _, item := range listOutput.Contents {
		expireTime := item.LastModified.Add(time.Duration(retentionPeriodTs) * time.Second)
		if time.Now().After(expireTime) {
			purgeBinlogPathList = append(purgeBinlogPathList, *item.Key)
		}
	}
	if len(purgeBinlogPathList) > 0 {
		log.Debug(fmt.Sprintf("Deleting %d expired binlog files from the cloud storage.", len(purgeBinlogPathList)))
		if _, err := r.server.s3Client.DeleteObjects(ctx, purgeBinlogPathList...); err != nil {
			return errors.Wrapf(err, "failed to delete %d expired binlog files from the cloud storage", len(purgeBinlogPathList))
		}
	}
	return nil
}

// TODO(dragonly): Remove metadata as well.
func (*BackupRunner) purgeBinlogFilesLocal(binlogDir string, retentionPeriodTs int) error {
	binlogFileInfoList, err := os.ReadDir(binlogDir)
	if err != nil {
		return errors.Wrapf(err, "failed to read backup directory %q", binlogDir)
	}
	for _, binlogFileInfo := range binlogFileInfoList {
		// We use modification time of local binlog files which is later than the modification time of that on the MySQL server,
		// which in turn is later than the last event timestamp of the binlog file.
		// This is not accurate and gives about 10 minutes (backup runner interval) more retention time to the binlog files, which is acceptable.
		fileInfo, err := binlogFileInfo.Info()
		if err != nil {
			log.Warn("Failed to get file info.", zap.String("path", binlogFileInfo.Name()), zap.Error(err))
			continue
		}
		expireTime := fileInfo.ModTime().Add(time.Duration(retentionPeriodTs) * time.Second)
		if time.Now().After(expireTime) {
			binlogFilePath := path.Join(binlogDir, binlogFileInfo.Name())
			log.Debug("Deleting expired local binlog file for MySQL instance.", zap.String("path", binlogFilePath))
			if err := os.Remove(binlogFilePath); err != nil {
				log.Warn("Failed to remove an expired binlog file.", zap.String("path", binlogFilePath), zap.Error(err))
				continue
			}
			log.Info("Deleted expired binlog file.", zap.String("path", binlogFilePath))
		}
	}
	return nil
}

func (r *BackupRunner) purgeBackup(ctx context.Context, backup *api.Backup) error {
	archive := api.Archived
	backupPatch := api.BackupPatch{
		ID:        backup.ID,
		UpdaterID: api.SystemBotID,
		RowStatus: &archive,
	}
	if _, err := r.server.store.PatchBackup(ctx, &backupPatch); err != nil {
		return errors.Wrapf(err, "failed to update status for deleted backup %q for database with ID %d", backup.Name, backup.DatabaseID)
	}

	switch backup.StorageBackend {
	case api.BackupStorageBackendLocal:
		backupFilePath := getBackupAbsFilePath(r.server.profile.DataDir, backup.DatabaseID, backup.Name)
		if err := os.Remove(backupFilePath); err != nil {
			return errors.Wrapf(err, "failed to delete an expired backup file %q", backupFilePath)
		}
		log.Info(fmt.Sprintf("Deleted expired local backup file %s", backupFilePath))
	case api.BackupStorageBackendS3:
		backupFilePath := getBackupRelativeFilePath(backup.DatabaseID, backup.Name)
		if _, err := r.server.s3Client.DeleteObjects(ctx, backupFilePath); err != nil {
			return errors.Wrapf(err, "failed to delete backup file %s in the cloud storage", backupFilePath)
		}
		log.Info(fmt.Sprintf("Deleted expired backup file %s in the cloud storage", backupFilePath))
	}

	return nil
}

func (r *BackupRunner) downloadBinlogFiles(ctx context.Context) {
	instanceList, err := r.server.store.FindInstanceWithDatabaseBackupEnabled(ctx, db.MySQL)
	if err != nil {
		log.Error("Failed to retrieve MySQL instance list with at least one database backup enabled", zap.Error(err))
		return
	}

	r.downloadBinlogMu.Lock()
	defer r.downloadBinlogMu.Unlock()
	for _, instance := range instanceList {
		if _, ok := r.downloadBinlogInstanceIDs[instance.ID]; !ok {
			r.downloadBinlogInstanceIDs[instance.ID] = true
			go r.downloadBinlogFilesForInstance(ctx, instance)
			r.downloadBinlogWg.Add(1)
		}
	}
}

func (r *BackupRunner) downloadBinlogFilesForInstance(ctx context.Context, instance *api.Instance) {
	defer func() {
		r.downloadBinlogMu.Lock()
		delete(r.downloadBinlogInstanceIDs, instance.ID)
		r.downloadBinlogMu.Unlock()
		r.downloadBinlogWg.Done()
	}()
	driver, err := r.server.getAdminDatabaseDriver(ctx, instance, "" /* databaseName */)
	if err != nil {
		if common.ErrorCode(err) == common.DbConnectionFailure {
			log.Debug("Cannot connect to instance", zap.String("instance", instance.Name), zap.Error(err))
			return
		}
		log.Error("Failed to get driver for MySQL instance when downloading binlog", zap.String("instance", instance.Name), zap.Error(err))
		return
	}
	defer driver.Close(ctx)

	mysqlDriver, ok := driver.(*mysql.Driver)
	if !ok {
		log.Error("Failed to cast driver to mysql.Driver", zap.String("instance", instance.Name))
		return
	}
	if err := mysqlDriver.FetchAllBinlogFiles(ctx, false /* downloadLatestBinlogFile */, r.server.s3Client); err != nil {
		log.Error("Failed to download all binlog files for instance", zap.String("instance", instance.Name), zap.Error(err))
		return
	}
}

func (r *BackupRunner) startAutoBackups(ctx context.Context, runningTasks map[int]bool, mu *sync.RWMutex) {
	// Find all databases that need a backup in this hour.
	t := time.Now().UTC().Truncate(time.Hour)
	match := &api.BackupSettingsMatch{
		Hour:      t.Hour(),
		DayOfWeek: int(t.Weekday()),
	}
	backupSettingList, err := r.server.store.FindBackupSettingsMatch(ctx, match)
	if err != nil {
		log.Error("Failed to retrieve backup settings match", zap.Error(err))
		return
	}

	for _, backupSetting := range backupSettingList {
		mu.Lock()
		if _, ok := runningTasks[backupSetting.ID]; ok {
			mu.Unlock()
			continue
		}
		runningTasks[backupSetting.ID] = true
		mu.Unlock()

		db := backupSetting.Database
		if db.Name == api.AllDatabaseName {
			// Skip backup job for wildcard database `*`.
			continue
		}
		backupName := fmt.Sprintf("%s-%s-%s-autobackup", api.ProjectShortSlug(db.Project), api.EnvSlug(db.Instance.Environment), t.Format("20060102T030405"))
		go func(database *api.Database, backupSettingID int, backupName string, hookURL string) {
			defer func() {
				mu.Lock()
				delete(runningTasks, backupSettingID)
				mu.Unlock()
				r.backupWg.Done()
			}()
			log.Debug("Schedule auto backup",
				zap.String("database", database.Name),
				zap.String("backup", backupName),
			)
			if _, err := r.server.scheduleBackupTask(ctx, database, backupName, api.BackupTypeAutomatic, api.SystemBotID); err != nil {
				log.Error("Failed to create automatic backup for database",
					zap.Int("databaseID", database.ID),
					zap.Error(err))
				return
			}
			// Backup succeeded. POST hook URL.
			if hookURL == "" {
				return
			}
			if _, err := http.PostForm(hookURL, nil); err != nil {
				log.Warn("Failed to POST hook URL",
					zap.String("hookURL", hookURL),
					zap.Int("databaseID", database.ID),
					zap.Error(err))
			}
		}(db, backupSetting.ID, backupName, backupSetting.HookURL)
		r.backupWg.Add(1)
	}
}

func (s *Server) scheduleBackupTask(ctx context.Context, database *api.Database, backupName string, backupType api.BackupType, creatorID int) (*api.Backup, error) {
	// Store the migration history version if exists.
	driver, err := s.getAdminDatabaseDriver(ctx, database.Instance, database.Name)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get admin database driver")
	}
	defer driver.Close(ctx)

	migrationHistoryVersion, err := getLatestSchemaVersion(ctx, driver, database.Name)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get migration history for database %q", database.Name)
	}
	path := getBackupRelativeFilePath(database.ID, backupName)
	if err := createBackupDirectory(s.profile.DataDir, database.ID); err != nil {
		return nil, errors.Wrap(err, "failed to create backup directory")
	}
	backupCreate := &api.BackupCreate{
		CreatorID:               creatorID,
		DatabaseID:              database.ID,
		Name:                    backupName,
		StorageBackend:          s.profile.BackupStorageBackend,
		Type:                    backupType,
		Path:                    path,
		MigrationHistoryVersion: migrationHistoryVersion,
	}

	backupNew, err := s.store.CreateBackup(ctx, backupCreate)
	if err != nil {
		if common.ErrorCode(err) == common.Conflict {
			log.Debug("Backup already exists for the database", zap.String("backup", backupName), zap.String("database", database.Name))
			return nil, nil
		}
		return nil, errors.Wrapf(err, "failed to create backup %q", backupName)
	}

	payload := api.TaskDatabaseBackupPayload{
		BackupID: backupNew.ID,
	}
	bytes, err := json.Marshal(payload)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create task payload for backup %q", backupName)
	}

	createdPipeline, err := s.store.CreatePipeline(ctx, &api.PipelineCreate{
		Name:      fmt.Sprintf("backup-%s", backupName),
		CreatorID: creatorID,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create pipeline for backup %q", backupName)
	}

	createdStage, err := s.store.CreateStage(ctx, &api.StageCreate{
		Name:          fmt.Sprintf("backup-%s", backupName),
		EnvironmentID: database.Instance.EnvironmentID,
		PipelineID:    createdPipeline.ID,
		CreatorID:     creatorID,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create stage for backup %q", backupName)
	}

	_, err = s.store.CreateTask(ctx, &api.TaskCreate{
		Name:       fmt.Sprintf("backup-%s", backupName),
		PipelineID: createdPipeline.ID,
		StageID:    createdStage.ID,
		InstanceID: database.InstanceID,
		DatabaseID: &database.ID,
		Status:     api.TaskPending,
		Type:       api.TaskDatabaseBackup,
		Payload:    string(bytes),
		CreatorID:  creatorID,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create task for backup %q", backupName)
	}
	return backupNew, nil
}
