package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"go.uber.org/zap"
)

// NewBackupRunner creates a new backup runner.
func NewBackupRunner(logger *zap.Logger, server *Server, backupRunnerInterval time.Duration) *BackupRunner {
	return &BackupRunner{
		l:                    logger,
		server:               server,
		backupRunnerInterval: backupRunnerInterval,
	}
}

// BackupRunner is the backup runner scheduling automatic backups.
type BackupRunner struct {
	l                    *zap.Logger
	server               *Server
	backupRunnerInterval time.Duration
}

// Run is the runner for backup runner.
func (s *BackupRunner) Run(ctx context.Context, wg *sync.WaitGroup) {
	ticker := time.NewTicker(s.backupRunnerInterval)
	defer ticker.Stop()
	defer wg.Done()
	s.l.Debug("Auto backup runner started", zap.Duration("interval", s.backupRunnerInterval))
	runningTasks := make(map[int]bool)
	var mu sync.RWMutex
	for {
		select {
		case <-ticker.C:
			s.l.Debug("New auto backup round started...")
			func() {
				defer func() {
					if r := recover(); r != nil {
						err, ok := r.(error)
						if !ok {
							err = fmt.Errorf("%v", r)
						}
						s.l.Error("Auto backup runner PANIC RECOVER", zap.Error(err), zap.Stack("stack"))
					}
				}()

				// Find all databases that need a backup in this hour.
				t := time.Now().UTC().Truncate(time.Hour)
				match := &api.BackupSettingsMatch{
					Hour:      t.Hour(),
					DayOfWeek: int(t.Weekday()),
				}
				backupSettingList, err := s.server.store.FindBackupSettingsMatch(ctx, match)
				if err != nil {
					s.l.Error("Failed to retrieve backup settings match", zap.Error(err))
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

					db, err := s.server.store.GetDatabase(ctx, &api.DatabaseFind{ID: &backupSetting.DatabaseID})
					if err != nil {
						s.l.Error("Failed to get database for backup setting",
							zap.Int("id", backupSetting.ID),
							zap.Int("databaseID", backupSetting.DatabaseID),
							zap.Error(err))
						continue
					}
					if db == nil {
						err := fmt.Errorf("failed to get database for backup setting, database ID not found %v", backupSetting.DatabaseID)
						s.l.Error(err.Error(),
							zap.Int("id", backupSetting.ID),
							zap.Int("databaseID", backupSetting.DatabaseID),
							zap.Error(err))
						continue
					}
					// TODO(dragonly): what's the purpose of this assignment?
					backupSetting.Database = db

					backupName := fmt.Sprintf("%s-%s-%s-autobackup", api.ProjectShortSlug(db.Project), api.EnvSlug(db.Instance.Environment), t.Format("20060102T030405"))
					go func(database *api.Database, backupSettingID int, backupName string, hookURL string) {
						s.l.Debug("Schedule auto backup",
							zap.String("database", database.Name),
							zap.String("backup", backupName),
						)
						defer func() {
							mu.Lock()
							delete(runningTasks, backupSettingID)
							mu.Unlock()
						}()
						err := s.scheduleBackupTask(ctx, database, backupName)
						if err != nil {
							s.l.Error("Failed to create automatic backup for database",
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
							s.l.Warn("Failed to POST hook URL",
								zap.String("hookURL", hookURL),
								zap.Int("databaseID", database.ID),
								zap.Error(err))
						}
					}(db, backupSetting.ID, backupName, backupSetting.HookURL)
				}
			}()
		case <-ctx.Done(): // if cancel() execute
			return
		}
	}
}

func (s *BackupRunner) scheduleBackupTask(ctx context.Context, database *api.Database, backupName string) error {
	path, err := getAndCreateBackupPath(s.server.dataDir, database, backupName)
	if err != nil {
		return err
	}

	// Store the migration history version if exists.
	driver, err := getAdminDatabaseDriver(ctx, database.Instance, database.Name, s.l)
	if err != nil {
		return err
	}
	defer driver.Close(ctx)
	migrationHistoryVersion, err := getLatestSchemaVersion(ctx, driver, database.Name)
	if err != nil {
		return fmt.Errorf("failed to get migration history for database %q: %w", database.Name, err)
	}

	// Return early if the backupOld already exists.
	backupOld, err := s.server.store.FindBackup(ctx, &api.BackupFind{Name: &backupName})
	if err != nil {
		return fmt.Errorf("failed to find backup %q, error %v", backupName, err)
	}
	if backupOld != nil {
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
	backupNew, err := s.server.store.CreateBackup(ctx, backupCreate)
	if err != nil {
		if common.ErrorCode(err) == common.Conflict {
			// Automatic backup already exists.
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

	createdPipeline, err := s.server.PipelineService.CreatePipeline(ctx, &api.PipelineCreate{
		Name:      backupName,
		CreatorID: backupCreate.CreatorID,
	})
	if err != nil {
		return fmt.Errorf("failed to create pipeline: %w", err)
	}

	createdStage, err := s.server.store.CreateStage(ctx, &api.StageCreate{
		Name:          backupName,
		EnvironmentID: database.Instance.EnvironmentID,
		PipelineID:    createdPipeline.ID,
		CreatorID:     backupCreate.CreatorID,
	})
	if err != nil {
		return fmt.Errorf("failed to create stage: %w", err)
	}

	_, err = s.server.store.CreateTask(ctx, &api.TaskCreate{
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
