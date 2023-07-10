// Package backuprun is the runner for backups.
package backuprun

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sync"
	"time"

	"github.com/gosimple/slug"
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
	"github.com/bytebase/bytebase/backend/plugin/storage/s3"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
)

// NewRunner creates a new backup runner.
func NewRunner(store *store.Store, dbFactory *dbfactory.DBFactory, s3Client *s3.Client, stateCfg *state.State, profile *config.Profile) *Runner {
	return &Runner{
		store:                     store,
		dbFactory:                 dbFactory,
		s3Client:                  s3Client,
		stateCfg:                  stateCfg,
		profile:                   profile,
		downloadBinlogInstanceIDs: make(map[int]bool),
	}
}

// Runner is the backup runner scheduling automatic backups.
type Runner struct {
	store                     *store.Store
	dbFactory                 *dbfactory.DBFactory
	s3Client                  *s3.Client
	stateCfg                  *state.State
	profile                   *config.Profile
	downloadBinlogInstanceIDs map[int]bool
	backupWg                  sync.WaitGroup
	downloadBinlogWg          sync.WaitGroup
	downloadBinlogMu          sync.Mutex
}

// Run is the runner for backup runner.
func (r *Runner) Run(ctx context.Context, wg *sync.WaitGroup) {
	ticker := time.NewTicker(r.profile.BackupRunnerInterval)
	defer ticker.Stop()
	defer wg.Done()
	log.Debug("Auto backup runner started", zap.Duration("interval", r.profile.BackupRunnerInterval))
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
				r.startAutoBackups(ctx)
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
func (r *Runner) purgeExpiredBackupData(ctx context.Context) {
	backupSettingList, err := r.store.ListBackupSettingV2(ctx, &store.FindBackupSettingMessage{})
	if err != nil {
		log.Error("Failed to find all the backup settings.", zap.Error(err))
		return
	}

	for _, bs := range backupSettingList {
		if bs.RetentionPeriodTs == api.BackupRetentionPeriodUnset {
			continue // next database
		}
		statusNormal := api.Normal
		backupList, err := r.store.ListBackupV2(ctx, &store.FindBackupMessage{
			DatabaseUID: &bs.DatabaseUID,
			RowStatus:   &statusNormal,
		})
		if err != nil {
			log.Error("Failed to get backups for database.", zap.Int("databaseID", bs.DatabaseUID))
			return
		}
		for _, backup := range backupList {
			backupTime := time.Unix(backup.UpdatedTs, 0)
			expireTime := backupTime.Add(time.Duration(bs.RetentionPeriodTs) * time.Second)
			if time.Now().After(expireTime) {
				log.Debug("Purging expired backup", zap.Int("databaseID", backup.DatabaseUID), zap.String("backup", backup.Name), zap.String("storageBackend", string(backup.StorageBackend)))
				if err := r.purgeBackup(ctx, backup); err != nil {
					log.Error("Failed to purge backup", zap.String("backup", backup.Name), zap.Error(err))
				}
			}
		}
	}

	instanceList, err := r.store.ListInstancesV2(ctx, &store.FindInstanceMessage{})
	if err != nil {
		log.Error("Failed to find non-archived instances.", zap.Error(err))
		return
	}

	for _, instance := range instanceList {
		if instance.Engine != db.MySQL && instance.Engine != db.MariaDB {
			continue
		}
		maxRetentionPeriodTs, err := r.getMaxRetentionPeriodTsForMySQLInstance(ctx, instance)
		if err != nil {
			log.Error("Failed to get max retention period for MySQL instance", zap.String("instance", instance.Title), zap.Error(err))
			continue
		}
		if maxRetentionPeriodTs == math.MaxInt {
			continue
		}
		if err := r.purgeBinlogFiles(ctx, instance.UID, maxRetentionPeriodTs); err != nil {
			log.Error("Failed to purge binlog files for instance", zap.String("instance", instance.Title), zap.Int("retentionPeriodTs", maxRetentionPeriodTs), zap.Error(err))
		}
	}
}

func (r *Runner) getMaxRetentionPeriodTsForMySQLInstance(ctx context.Context, instance *store.InstanceMessage) (int, error) {
	backupSettingList, err := r.store.ListBackupSettingV2(ctx, &store.FindBackupSettingMessage{InstanceUID: &instance.UID})
	if err != nil {
		log.Error("Failed to find backup settings for instance.", zap.String("instance", instance.Title), zap.Error(err))
		return 0, errors.Wrapf(err, "failed to find backup settings for instance %q", instance.Title)
	}
	maxRetentionPeriodTs := math.MaxInt
	for _, bs := range backupSettingList {
		if bs.RetentionPeriodTs != api.BackupRetentionPeriodUnset && bs.RetentionPeriodTs < maxRetentionPeriodTs {
			maxRetentionPeriodTs = bs.RetentionPeriodTs
		}
	}
	return maxRetentionPeriodTs, nil
}

func (r *Runner) purgeBinlogFiles(ctx context.Context, instanceID, retentionPeriodTs int) error {
	binlogDir := common.GetBinlogAbsDir(r.profile.DataDir, instanceID)
	switch r.profile.BackupStorageBackend {
	case api.BackupStorageBackendLocal:
		return r.purgeBinlogFilesLocal(binlogDir, retentionPeriodTs)
	case api.BackupStorageBackendS3:
		return r.purgeBinlogFilesOnCloud(ctx, binlogDir, retentionPeriodTs)
	default:
		return errors.Errorf("purge binlog files not implemented for storage backend %s", r.profile.BackupStorageBackend)
	}
}

func (r *Runner) purgeBinlogFilesOnCloud(ctx context.Context, binlogDir string, retentionPeriodTs int) error {
	binlogDirOnCloud := common.GetBinlogRelativeDir(binlogDir)
	listOutput, err := r.s3Client.ListObjects(ctx, binlogDirOnCloud)
	if err != nil {
		return errors.Wrapf(err, "failed to list binlog dir %q in the cloud storage", binlogDirOnCloud)
	}
	var purgeBinlogPathList []string
	for _, item := range listOutput {
		expireTime := item.LastModified.Add(time.Duration(retentionPeriodTs) * time.Second)
		if time.Now().After(expireTime) {
			purgeBinlogPathList = append(purgeBinlogPathList, *item.Key)
		}
	}
	if len(purgeBinlogPathList) > 0 {
		log.Debug(fmt.Sprintf("Deleting %d expired binlog files from the cloud storage.", len(purgeBinlogPathList)))
		if _, err := r.s3Client.DeleteObjects(ctx, purgeBinlogPathList...); err != nil {
			return errors.Wrapf(err, "failed to delete %d expired binlog files from the cloud storage", len(purgeBinlogPathList))
		}
	}
	return nil
}

// TODO(dragonly): Remove metadata as well.
func (*Runner) purgeBinlogFilesLocal(binlogDir string, retentionPeriodTs int) error {
	binlogFileInfoList, err := os.ReadDir(binlogDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
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
				if !os.IsNotExist(err) {
					log.Warn("Failed to remove an expired binlog file.", zap.String("path", binlogFilePath), zap.Error(err))
				}
				continue
			}
			log.Info("Deleted expired binlog file.", zap.String("path", binlogFilePath))
		}
	}
	return nil
}

func (r *Runner) purgeBackup(ctx context.Context, backup *store.BackupMessage) error {
	archive := api.Archived
	backupPatch := &store.UpdateBackupMessage{
		UID:       backup.UID,
		UpdaterID: api.SystemBotID,
		RowStatus: &archive,
	}
	if _, err := r.store.UpdateBackupV2(ctx, backupPatch); err != nil {
		return errors.Wrapf(err, "failed to update status for deleted backup %q for database with ID %d", backup.Name, backup.DatabaseUID)
	}
	log.Debug("Archived expired backup record", zap.String("name", backup.Name), zap.Int("id", backup.UID))

	switch backup.StorageBackend {
	case api.BackupStorageBackendLocal:
		backupFilePath := GetBackupAbsFilePath(r.profile.DataDir, backup.DatabaseUID, backup.Name)
		if err := os.Remove(backupFilePath); err != nil {
			return errors.Wrapf(err, "failed to delete an expired backup file %q", backupFilePath)
		}
		log.Debug(fmt.Sprintf("Deleted expired local backup file %s", backupFilePath))
	case api.BackupStorageBackendS3:
		backupFilePath := getBackupRelativeFilePath(backup.DatabaseUID, backup.Name)
		if _, err := r.s3Client.DeleteObjects(ctx, backupFilePath); err != nil {
			return errors.Wrapf(err, "failed to delete backup file %s in the cloud storage", backupFilePath)
		}
		log.Debug(fmt.Sprintf("Deleted expired backup file %s in the cloud storage", backupFilePath))
	}

	return nil
}

func (r *Runner) downloadBinlogFiles(ctx context.Context) {
	instances, err := r.store.FindInstanceWithDatabaseBackupEnabled(ctx)
	if err != nil {
		log.Error("Failed to retrieve MySQL instance list with at least one database backup enabled", zap.Error(err))
		return
	}

	r.downloadBinlogMu.Lock()
	defer r.downloadBinlogMu.Unlock()
	for _, instance := range instances {
		if instance.Engine != db.MySQL && instance.Engine != db.MariaDB {
			continue
		}
		if _, ok := r.downloadBinlogInstanceIDs[instance.UID]; !ok {
			r.downloadBinlogInstanceIDs[instance.UID] = true
			go r.downloadBinlogFilesForInstance(ctx, instance)
			r.downloadBinlogWg.Add(1)
		}
	}
}

func (r *Runner) downloadBinlogFilesForInstance(ctx context.Context, instance *store.InstanceMessage) {
	defer func() {
		r.downloadBinlogMu.Lock()
		delete(r.downloadBinlogInstanceIDs, instance.UID)
		r.downloadBinlogMu.Unlock()
		r.downloadBinlogWg.Done()
	}()
	driver, err := r.dbFactory.GetAdminDatabaseDriver(ctx, instance, nil /* database */)
	if err != nil {
		if common.ErrorCode(err) == common.DbConnectionFailure {
			log.Debug("Cannot connect to instance", zap.String("instance", instance.ResourceID), zap.Error(err))
			return
		}
		log.Error("Failed to get driver for MySQL instance when downloading binlog", zap.String("instance", instance.ResourceID), zap.Error(err))
		return
	}
	defer driver.Close(ctx)

	mysqlDriver, ok := driver.(*mysql.Driver)
	if !ok {
		log.Error("Failed to cast driver to mysql.Driver", zap.String("instance", instance.ResourceID))
		return
	}
	if err := mysqlDriver.FetchAllBinlogFiles(ctx, false /* downloadLatestBinlogFile */, r.s3Client); err != nil {
		log.Error("Failed to download all binlog files for instance", zap.String("instance", instance.ResourceID), zap.Error(err))
		return
	}
}

func (r *Runner) startAutoBackups(ctx context.Context) {
	// Find all databases that need a backup in this hour.
	t := time.Now().UTC().Truncate(time.Hour)
	backupSettingList, err := r.store.FindBackupSettingsMatch(ctx, &store.BackupSettingsMatchMessage{
		Hour:      t.Hour(),
		DayOfWeek: int(t.Weekday()),
	})
	if err != nil {
		log.Error("Failed to retrieve backup settings match", zap.Error(err))
		return
	}

	for _, backupSetting := range backupSettingList {
		if _, ok := r.stateCfg.RunningBackupDatabases.Load(backupSetting.DatabaseUID); ok {
			continue
		}
		database, err := r.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{UID: &backupSetting.DatabaseUID})
		if err != nil {
			log.Error("Failed to get database", zap.Error(err))
			return
		}
		if database.DatabaseName == api.AllDatabaseName {
			// Skip backup job for wildcard database `*`.
			continue
		}
		project, err := r.store.GetProjectV2(ctx, &store.FindProjectMessage{ResourceID: &database.ProjectID})
		if err != nil {
			log.Error("Failed to get project", zap.Error(err))
		}
		if project.Deleted {
			continue
		}
		instance, err := r.store.GetInstanceV2(ctx, &store.FindInstanceMessage{ResourceID: &database.InstanceID})
		if err != nil {
			log.Error("Failed to get instance", zap.Error(err))
		}
		if instance.Deleted {
			continue
		}
		// backup for ClickHouse, Snowflake, MongoDB, Spanner, Redis, Oracle is not supported.
		if instance.Engine == db.ClickHouse || instance.Engine == db.Snowflake || instance.Engine == db.MongoDB || instance.Engine == db.Spanner || instance.Engine == db.Redis || instance.Engine == db.Oracle {
			continue
		}
		environment, err := r.store.GetEnvironmentV2(ctx, &store.FindEnvironmentMessage{ResourceID: &instance.EnvironmentID})
		if err != nil {
			log.Error("Failed to get environment", zap.Error(err))
		}
		if environment.Deleted {
			continue
		}
		backupName := fmt.Sprintf("%s-%s-%s-autobackup", slug.Make(project.Title), slug.Make(environment.Title), t.Format("20060102T030405"))
		backupList, err := r.store.ListBackupV2(ctx, &store.FindBackupMessage{
			DatabaseUID: &database.UID,
			Name:        &backupName,
		})
		if err != nil {
			log.Error("Failed to find backup", zap.Error(err))
			continue
		}
		if len(backupList) > 0 {
			continue
		}

		r.stateCfg.RunningBackupDatabases.Store(backupSetting.DatabaseUID, true)
		go func(database *store.DatabaseMessage, backupName string, hookURL string) {
			defer func() {
				r.stateCfg.RunningBackupDatabases.Delete(database.UID)
				r.backupWg.Done()
			}()
			log.Debug("Schedule auto backup",
				zap.String("database", database.DatabaseName),
				zap.String("backup", backupName),
			)
			if _, err := r.ScheduleBackupTask(ctx, database, backupName, api.BackupTypeAutomatic, api.SystemBotID); err != nil {
				log.Error("Failed to create automatic backup for database",
					zap.Int("databaseID", database.UID),
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
					zap.Int("databaseID", database.UID),
					zap.Error(err))
			}
		}(database, backupName, backupSetting.HookURL)
		r.backupWg.Add(1)
	}
}

// ScheduleBackupTask schedules a backup task.
func (r *Runner) ScheduleBackupTask(ctx context.Context, database *store.DatabaseMessage, backupName string, backupType api.BackupType, creatorID int) (*store.BackupMessage, error) {
	instance, err := r.store.GetInstanceV2(ctx, &store.FindInstanceMessage{ResourceID: &database.InstanceID})
	if err != nil {
		return nil, err
	}
	if instance == nil {
		return nil, errors.Errorf("instance %q not found", database.InstanceID)
	}
	if instance.Deleted {
		return nil, errors.Errorf("instance %q deleted", database.InstanceID)
	}
	environment, err := r.store.GetEnvironmentV2(ctx, &store.FindEnvironmentMessage{ResourceID: &instance.EnvironmentID})
	if err != nil {
		return nil, err
	}
	if environment == nil {
		return nil, errors.Errorf("environment %q not found", instance.EnvironmentID)
	}

	driver, err := r.dbFactory.GetAdminDatabaseDriver(ctx, instance, database)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get admin database driver")
	}
	defer driver.Close(ctx)

	migrationHistoryVersion, err := utils.GetLatestSchemaVersion(ctx, r.store, instance.UID, database.UID, database.DatabaseName)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get migration history for database %q", database.DatabaseName)
	}
	path := getBackupRelativeFilePath(database.UID, backupName)
	if err := createBackupDirectory(r.profile.DataDir, database.UID); err != nil {
		return nil, errors.Wrap(err, "failed to create backup directory")
	}

	backupNew, err := r.store.CreateBackupV2(ctx, &store.BackupMessage{
		Name:                    backupName,
		Status:                  api.BackupStatusPendingCreate,
		BackupType:              backupType,
		Comment:                 "",
		StorageBackend:          r.profile.BackupStorageBackend,
		MigrationHistoryVersion: migrationHistoryVersion,
		Path:                    path,
	}, database.UID, creatorID)
	if err != nil {
		if common.ErrorCode(err) == common.Conflict {
			log.Error("Backup already exists for the database", zap.String("backup", backupName), zap.String("database", database.DatabaseName))
			return nil, nil
		}
		return nil, errors.Wrapf(err, "failed to create backup %q", backupName)
	}

	payload := api.TaskDatabaseBackupPayload{
		BackupID: backupNew.UID,
	}
	bytes, err := json.Marshal(payload)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create task payload for backup %q", backupName)
	}

	pipeline, err := r.store.CreatePipelineV2(ctx, &store.PipelineMessage{
		Name: fmt.Sprintf("backup-%s", backupName),
	}, creatorID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create pipeline for backup %q", backupName)
	}

	createdStages, err := r.store.CreateStageV2(ctx, []*store.StageMessage{
		{
			Name:          fmt.Sprintf("backup-%s", backupName),
			EnvironmentID: environment.UID,
			PipelineID:    pipeline.ID,
		},
	}, creatorID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create stage for backup %q", backupName)
	}
	if len(createdStages) != 1 {
		return nil, errors.Errorf("expect to get one returned stage after creating stage, got %d", len(createdStages))
	}

	createdStage := createdStages[0]
	if _, err := r.store.CreateTasksV2(ctx, &store.RolloutTask{
		Name:       fmt.Sprintf("backup-%s", backupName),
		PipelineID: pipeline.ID,
		StageID:    createdStage.ID,
		InstanceID: instance.UID,
		DatabaseID: &database.UID,
		Status:     api.TaskPending,
		Type:       api.TaskDatabaseBackup,
		Payload:    string(bytes),
		CreatorID:  creatorID,
	}); err != nil {
		return nil, errors.Wrapf(err, "failed to create task for backup %q", backupName)
	}
	return backupNew, nil
}

// Get backup dir relative to the data dir.
func getBackupRelativeDir(databaseID int) string {
	return filepath.Join("backup", "db", fmt.Sprintf("%d", databaseID))
}

func getBackupRelativeFilePath(databaseID int, name string) string {
	dir := getBackupRelativeDir(databaseID)
	return filepath.Join(dir, fmt.Sprintf("%s.sql", name))
}

// GetBackupAbsFilePath returns backup absolute file path for a database.
func GetBackupAbsFilePath(dataDir string, databaseID int, name string) string {
	path := getBackupRelativeFilePath(databaseID, name)
	return filepath.Join(dataDir, path)
}

// Create backup directory for database.
func createBackupDirectory(dataDir string, databaseID int) error {
	dir := getBackupRelativeDir(databaseID)
	absDir := filepath.Join(dataDir, dir)
	return os.MkdirAll(absDir, os.ModePerm)
}
