package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"sync"
	"time"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/common/log"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/plugin/db/mysql"
	"github.com/bytebase/bytebase/resources/mysqlutil"
	"go.uber.org/zap"
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
			log.Debug("New auto backup round started...")
			func() {
				defer func() {
					if r := recover(); r != nil {
						err, ok := r.(error)
						if !ok {
							err = fmt.Errorf("%v", r)
						}
						log.Error("Auto backup runner PANIC RECOVER", zap.Error(err))
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
		log.Error("Failed to find all the backup settings", zap.Error(err))
		return
	}

	log.Debug("Deleting expired snapshot backup files")
	for _, bs := range backupSettingList {
		if bs.RetentionPeriodTs == api.BackupRetentionPeriodUnset {
			continue
		}
		backupDir := path.Join(r.server.profile.DataDir, getBackupRelativeDir(bs.DatabaseID))
		backupFileInfoList, err := ioutil.ReadDir(backupDir)
		if err != nil {
			log.Error("Failed to read backup directory", zap.String("path", backupDir), zap.Error(err))
			continue
		}
		for _, backupFileInfo := range backupFileInfoList {
			expireTime := backupFileInfo.ModTime().Add(time.Duration(bs.RetentionPeriodTs))
			if time.Now().After(expireTime) {
				backupFilePath := path.Join(backupDir, backupFileInfo.Name())
				if err := os.Remove(backupFilePath); err != nil {
					log.Error("Failed to remove an expired backup file", zap.String("path", backupFilePath), zap.Error(err))
				}
				log.Info("Deleted expired backup file", zap.String("path", backupFilePath))
			}
		}
	}

	log.Debug("Deleting expired MySQL binlog files")
	normal := api.Normal
	instanceList, err := r.server.store.FindInstance(ctx, &api.InstanceFind{RowStatus: &normal})
	if err != nil {
		log.Error("Failed to find non-archived instances", zap.Error(err))
		return
	}

	for _, instance := range instanceList {
		if instance.Engine != db.MySQL {
			continue
		}
		backupSettingList, err := r.server.store.FindBackupSetting(ctx, api.BackupSettingFind{InstanceID: &instance.ID})
		if err != nil {
			log.Error("Failed to find backup settings for instance", zap.String("instance", instance.Name), zap.Error(err))
			continue // next instance
		}
		maxRetentionPeriodForInstance := api.BackupRetentionPeriodUnset
		for _, bs := range backupSettingList {
			if bs.RetentionPeriodTs > maxRetentionPeriodForInstance {
				maxRetentionPeriodForInstance = bs.RetentionPeriodTs
			}
		}
		if maxRetentionPeriodForInstance == api.BackupRetentionPeriodUnset {
			log.Debug("All the databases in instance have unset retention period. Skip deleting binlog files", zap.String("instance", instance.Name))
			continue // next instance
		}

		log.Debug("Deleting old binlog files for MySQL instance", zap.String("instance", instance.Name))
		binlogDir := getBinlogAbsDir(r.server.profile.DataDir, instance.ID)
		binlogFileInfoList, err := ioutil.ReadDir(binlogDir)
		if err != nil {
			log.Error("Failed to read backup directory", zap.String("path", binlogDir), zap.Error(err))
			continue // next instance
		}
		for _, binlogFileInfo := range binlogFileInfoList {
			if _, err := mysql.GetBinlogNameSeq(binlogFileInfo.Name()); err != nil {
				log.Warn("Found an irregular file in binlog directory", zap.String("path", binlogFileInfo.Name()))
				continue // next binlog file
			}
			expireTime := binlogFileInfo.ModTime().Add(time.Duration(maxRetentionPeriodForInstance))
			if time.Now().After(expireTime) {
				binlogFilePath := path.Join(binlogDir, binlogFileInfo.Name())
				if err := os.Remove(binlogFilePath); err != nil {
					log.Error("Failed to remove an expired binlog file", zap.String("path", binlogFilePath), zap.Error(err))
				}
				log.Info("Deleted expired binlog file", zap.String("path", binlogFilePath))
			}
		}
	}
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
			go r.downloadBinlogFilesForInstance(ctx, instance, r.server.profile.DataDir, r.server.mysqlutil)
			r.downloadBinlogWg.Add(1)
		}
	}
}

func (r *BackupRunner) downloadBinlogFilesForInstance(ctx context.Context, instance *api.Instance, dataDir string, mysqlutil mysqlutil.Instance) {
	log.Debug("Downloading binlog files for MySQL instance", zap.String("instance", instance.Name))
	defer func() {
		r.downloadBinlogMu.Lock()
		delete(r.downloadBinlogInstanceIDs, instance.ID)
		r.downloadBinlogMu.Unlock()
		r.downloadBinlogWg.Done()
	}()
	driver, err := getAdminDatabaseDriver(ctx, instance, "", "" /* pgInstanceDir */)
	if err != nil {
		if common.ErrorCode(err) == common.DbConnectionFailure {
			log.Warn("Cannot connect to instance", zap.String("instance", instance.Name), zap.Error(err))
			return
		}
		log.Error("Failed to get driver for MySQL instance when downloading binlog", zap.String("instance", instance.Name), zap.Error(err))
		return
	}
	defer driver.Close(ctx)

	binlogDir := getBinlogAbsDir(dataDir, instance.ID)
	if err := createBinlogDir(dataDir, instance.ID); err != nil {
		log.Error("Failed to create binlog directory", zap.Error(err))
		return
	}
	mysqlDriver, ok := driver.(*mysql.Driver)
	if !ok {
		log.Error("Failed to cast driver to mysql.Driver", zap.String("instance", instance.Name))
		return
	}
	mysqlDriver.SetUpForPITR(mysqlutil, binlogDir)
	if err := mysqlDriver.FetchAllBinlogFiles(ctx, false /* downloadLatestBinlogFile */); err != nil {
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
			if _, err := r.server.scheduleBackupTask(ctx, database, backupName, api.BackupTypeAutomatic, api.BackupStorageBackendLocal, api.SystemBotID); err != nil {
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

func (s *Server) scheduleBackupTask(ctx context.Context, database *api.Database, backupName string, backupType api.BackupType, storageBackend api.BackupStorageBackend, creatorID int) (*api.Backup, error) {
	// Store the migration history version if exists.
	driver, err := getAdminDatabaseDriver(ctx, database.Instance, database.Name, s.pgInstanceDir)
	if err != nil {
		return nil, fmt.Errorf("failed to get admin database driver, error: %w", err)
	}
	defer driver.Close(ctx)

	migrationHistoryVersion, err := getLatestSchemaVersion(ctx, driver, database.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to get migration history for database %q, error: %w", database.Name, err)
	}
	path := getBackupRelativeFilePath(database.ID, backupName)
	if err := createBackupDirectory(s.profile.DataDir, database.ID); err != nil {
		return nil, fmt.Errorf("failed to create backup directory, error: %w", err)
	}
	backupCreate := &api.BackupCreate{
		CreatorID:               creatorID,
		DatabaseID:              database.ID,
		Name:                    backupName,
		StorageBackend:          storageBackend,
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
		return nil, fmt.Errorf("failed to create backup %q, error: %w", backupName, err)
	}

	payload := api.TaskDatabaseBackupPayload{
		BackupID: backupNew.ID,
	}
	bytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to create task payload for backup %q, error: %w", backupName, err)
	}

	createdPipeline, err := s.store.CreatePipeline(ctx, &api.PipelineCreate{
		Name:      fmt.Sprintf("backup-%s", backupName),
		CreatorID: creatorID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create pipeline for backup %q, error: %w", backupName, err)
	}

	createdStage, err := s.store.CreateStage(ctx, &api.StageCreate{
		Name:          fmt.Sprintf("backup-%s", backupName),
		EnvironmentID: database.Instance.EnvironmentID,
		PipelineID:    createdPipeline.ID,
		CreatorID:     creatorID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create stage for backup %q, error: %w", backupName, err)
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
		return nil, fmt.Errorf("failed to create task for backup %q, error: %w", backupName, err)
	}
	return backupNew, nil
}
