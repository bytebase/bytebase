package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/common/log"
	"go.uber.org/zap"
)

const (
	// TODO(dragonly): Design the user configuration in pitr-mysql.md, and implement it.
	backupExpireDuration  = 7 * 24 * time.Hour
	backupCleanupInterval = 30 * time.Minute
)

// NewBackupRunner creates a new backup runner.
func NewBackupRunner(server *Server, backupRunnerInterval time.Duration) *BackupRunner {
	return &BackupRunner{
		server:               server,
		backupRunnerInterval: backupRunnerInterval,
		tasks: runningTasks{
			running: make(map[int]bool),
		},
	}
}

type runningTasks struct {
	running map[int]bool // task id set
	mu      sync.RWMutex
}

// BackupRunner is the backup runner scheduling automatic backups.
type BackupRunner struct {
	server               *Server
	backupRunnerInterval time.Duration
	tasks                runningTasks
}

// Run is the runner for backup runner.
func (r *BackupRunner) Run(ctx context.Context, wg *sync.WaitGroup) {
	tickerAutoBackup := time.NewTicker(r.backupRunnerInterval)
	tickerBackupCleaner := time.NewTicker(backupCleanupInterval)
	defer tickerAutoBackup.Stop()
	defer tickerBackupCleaner.Stop()
	defer wg.Done()
	log.Debug("Auto backup runner started", zap.Duration("interval", r.backupRunnerInterval))

	for {
		select {
		case <-tickerAutoBackup.C:
			log.Debug("New auto backup round started...")
			r.run(ctx)
		case <-tickerBackupCleaner.C:
			log.Debug("New backup clean up round started...")
			r.cleanupBackups(ctx)
		case <-ctx.Done(): // if cancel() execute
			return
		}
	}
}

func (r *BackupRunner) run(ctx context.Context) {
	defer recoverBackupRunnerPanic()

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
		r.tasks.mu.Lock()
		if _, ok := r.tasks.running[backupSetting.ID]; ok {
			r.tasks.mu.Unlock()
			log.Debug("Backup task is already running, skip.",
				zap.Int("backupSettingID", backupSetting.ID))
			continue
		}
		r.tasks.running[backupSetting.ID] = true
		r.tasks.mu.Unlock()

		db := backupSetting.Database
		backupName := fmt.Sprintf("%s-%s-%s-autobackup", api.ProjectShortSlug(db.Project), api.EnvSlug(db.Instance.Environment), t.Format("20060102T030405"))
		go func(database *api.Database, backupSettingID int, backupName string, hookURL string) {
			log.Debug("Schedule auto backup",
				zap.String("database", database.Name),
				zap.String("backup", backupName),
			)
			defer func() {
				r.tasks.mu.Lock()
				delete(r.tasks.running, backupSettingID)
				r.tasks.mu.Unlock()
			}()
			err := r.scheduleBackupTask(ctx, database, backupName)
			if err != nil {
				log.Error("Failed to create automatic backup for database",
					zap.Int("databaseID", database.ID),
					zap.Error(err))
				return
			}
			// Backup succeeded. POST hook URL.
			if hookURL == "" {
				return
			}
			_, err = http.PostForm(hookURL, nil)
			if err != nil {
				log.Warn("Failed to POST hook URL",
					zap.String("hookURL", hookURL),
					zap.Int("databaseID", database.ID),
					zap.Error(err))
			}
		}(db, backupSetting.ID, backupName, backupSetting.HookURL)
	}
}

// Delete backup/binlog/WAL if it is older than 1 week.
func (r *BackupRunner) cleanupBackups(ctx context.Context) {
	defer recoverBackupRunnerPanic()

	backupList, err := r.server.store.FindBackup(ctx, &api.BackupFind{})
	if err != nil {
		log.Error("failed to get all backups", zap.Error(err))
		return
	}

	expireTime := time.Now().Add(-backupExpireDuration)
	for _, backup := range backupList {
		backupTime := time.Unix(backup.UpdatedTs, 0)
		if backupTime.Before(expireTime) {
			if err := r.deleteBackupFile(backup); err != nil {
				log.Error("Failed to delete backup file", zap.Error(err))
			}
		}
	}

	// TODO(dragonly): Delete binlog for MySQL databases after MySQL binlog replay is done.
}

// delete the backup file corresponds to the api.Backup.
func (r *BackupRunner) deleteBackupFile(backup *api.Backup) error {
	path := getBackupPath(r.server.profile.DataDir, backup.DatabaseID, backup.Name)
	return os.Remove(path)
}

func recoverBackupRunnerPanic() {
	if r := recover(); r != nil {
		err, ok := r.(error)
		if !ok {
			err = fmt.Errorf("%v", r)
		}
		log.Error("Auto backup runner PANIC RECOVER", zap.Error(err), zap.Stack("stack"))
	}
}

func (r *BackupRunner) scheduleBackupTask(ctx context.Context, database *api.Database, backupName string) error {
	path, err := getAndCreateBackupPath(r.server.profile.DataDir, database.ID, backupName)
	if err != nil {
		return err
	}

	// Store the migration history version if exists.
	driver, err := getAdminDatabaseDriver(ctx, database.Instance, database.Name, r.server.pgInstanceDir)
	if err != nil {
		return err
	}
	defer driver.Close(ctx)
	migrationHistoryVersion, err := getLatestSchemaVersion(ctx, driver, database.Name)
	if err != nil {
		return fmt.Errorf("failed to get migration history for database[%s], error[%w]", database.Name, err)
	}

	// Return early if the backupOld already exists.
	backupOld, err := r.server.store.FindBackup(ctx, &api.BackupFind{Name: &backupName})
	if err != nil {
		return fmt.Errorf("failed to find backup[%s], error[%w]", backupName, err)
	}
	if backupOld != nil {
		log.Debug("Backup already exist, skip", zap.String("backup", backupName))
		return nil
	}

	backupCreate := &api.BackupCreate{
		CreatorID:               api.SystemBotID,
		DatabaseID:              database.ID,
		Name:                    backupName,
		Type:                    api.BackupTypeAutomatic,
		MigrationHistoryVersion: migrationHistoryVersion,
		StorageBackend:          api.BackupStorageBackendLocal,
		Path:                    path,
	}
	backupNew, err := r.server.store.CreateBackup(ctx, backupCreate)
	if err != nil {
		if common.ErrorCode(err) == common.Conflict {
			log.Debug("Backup already exist, skip", zap.String("backup", backupName))
			return nil
		}
		return fmt.Errorf("failed to create backup: %w", err)
	}

	payload := api.TaskDatabaseBackupPayload{
		BackupID: backupNew.ID,
	}
	bytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to create task payload: %w", err)
	}

	createdPipeline, err := r.server.store.CreatePipeline(ctx, &api.PipelineCreate{
		Name:      backupName,
		CreatorID: backupCreate.CreatorID,
	})
	if err != nil {
		return fmt.Errorf("failed to create pipeline: %w", err)
	}

	createdStage, err := r.server.store.CreateStage(ctx, &api.StageCreate{
		Name:          backupName,
		EnvironmentID: database.Instance.EnvironmentID,
		PipelineID:    createdPipeline.ID,
		CreatorID:     backupCreate.CreatorID,
	})
	if err != nil {
		return fmt.Errorf("failed to create stage: %w", err)
	}

	_, err = r.server.store.CreateTask(ctx, &api.TaskCreate{
		Name:       backupName,
		PipelineID: createdPipeline.ID,
		StageID:    createdStage.ID,
		InstanceID: database.InstanceID,
		DatabaseID: &database.ID,
		Status:     api.TaskPending,
		Type:       api.TaskDatabaseBackup,
		Payload:    string(bytes),
		CreatorID:  backupCreate.CreatorID,
	})
	if err != nil {
		return fmt.Errorf("failed to create task: %w", err)
	}
	return nil
}
