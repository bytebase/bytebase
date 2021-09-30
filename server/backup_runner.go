package server

import (
	"context"
	"encoding/json"
	"fmt"
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
func (s *BackupRunner) Run() error {
	go func() {
		s.l.Debug(fmt.Sprintf("Auto backup runner started and will run every %v", s.backupRunnerInterval))
		runningTasks := make(map[int]bool)
		mu := sync.RWMutex{}
		for {
			s.l.Debug("New auto backup round started...")
			func() {
				defer func() {
					if r := recover(); r != nil {
						err, ok := r.(error)
						if !ok {
							err = fmt.Errorf("%v", r)
						}
						s.l.Error("Auto backup runner PANIC RECOVER", zap.Error(err))
					}
				}()
				// Find all databases that need a backup in this hour.
				t := time.Now().UTC().Truncate(time.Hour)

				match := &api.BackupSettingsMatch{
					Hour:      t.Hour(),
					DayOfWeek: int(t.Weekday()),
				}
				list, err := s.server.BackupService.FindBackupSettingsMatch(context.Background(), match)
				if err != nil {
					s.l.Error("Failed to retrieve backup settings match", zap.Error(err))
					return
				}

				for _, backupSetting := range list {
					mu.Lock()
					if _, ok := runningTasks[backupSetting.ID]; ok {
						mu.Unlock()
						continue
					}
					runningTasks[backupSetting.ID] = true
					mu.Unlock()

					databaseFind := &api.DatabaseFind{
						ID: &backupSetting.DatabaseId,
					}
					database, err := s.server.ComposeDatabaseByFind(context.Background(), databaseFind)
					if err != nil {
						s.l.Error("Failed to get database for backup setting",
							zap.Int("id", backupSetting.ID),
							zap.String("databaseID", fmt.Sprintf("%v", backupSetting.DatabaseId)),
							zap.String("error", err.Error()))
						continue
					}
					backupSetting.Database = database

					backupName := fmt.Sprintf("%s-%s-%s-autobackup", api.ProjectShortSlug(database.Project), api.EnvSlug(database.Instance.Environment), t.Format("20060102T030405"))
					go func(database *api.Database, backupSettingId int, backupName string) {
						s.l.Debug("Schedule auto backup",
							zap.String("database", database.Name),
							zap.String("backup", backupName),
						)
						defer func() {
							mu.Lock()
							delete(runningTasks, backupSettingId)
							mu.Unlock()
						}()
						if err := s.scheduleBackupTask(database, backupName); err != nil {
							s.l.Error("Failed to create automatic backup for database",
								zap.Int("databaseID", database.ID),
								zap.String("error", err.Error()))
						}
					}(database, backupSetting.ID, backupName)
				}
			}()

			time.Sleep(s.backupRunnerInterval)
		}
	}()

	return nil
}

func (s *BackupRunner) scheduleBackupTask(database *api.Database, backupName string) error {
	path, err := getAndCreateBackupPath(s.server.dataDir, database, backupName)
	if err != nil {
		return err
	}

	// Store the migration history version if exists.
	migrationHistoryVersion, err := getMigrationVersion(database, s.l)
	if err != nil {
		return fmt.Errorf("failed to get migration history for database %q: %w", database.Name, err)
	}

	backupCreate := &api.BackupCreate{
		CreatorId:               api.SYSTEM_BOT_ID,
		DatabaseId:              database.ID,
		Name:                    backupName,
		Status:                  api.BackupStatusPendingCreate,
		Type:                    api.BackupTypeAutomatic,
		MigrationHistoryVersion: migrationHistoryVersion,
		StorageBackend:          api.BackupStorageBackendLocal,
		Path:                    path,
	}
	backup, err := s.server.BackupService.CreateBackup(context.Background(), backupCreate)
	if err != nil {
		if common.ErrorCode(err) == common.Conflict {
			// Automatic backup already exists.
			return nil
		}
		return fmt.Errorf("failed to create backup: %w", err)
	}

	payload := api.TaskDatabaseBackupPayload{
		BackupId: backup.ID,
	}
	bytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to create task payload: %w", err)
	}

	createdPipeline, err := s.server.PipelineService.CreatePipeline(context.Background(), &api.PipelineCreate{
		Name:      backupName,
		CreatorId: backupCreate.CreatorId,
	})
	if err != nil {
		return fmt.Errorf("failed to create pipeline: %w", err)
	}

	createdStage, err := s.server.StageService.CreateStage(context.Background(), &api.StageCreate{
		Name:          backupName,
		EnvironmentId: database.Instance.EnvironmentId,
		PipelineId:    createdPipeline.ID,
		CreatorId:     backupCreate.CreatorId,
	})
	if err != nil {
		return fmt.Errorf("failed to create stage: %w", err)
	}

	_, err = s.server.TaskService.CreateTask(context.Background(), &api.TaskCreate{
		Name:       backupName,
		PipelineId: createdPipeline.ID,
		StageId:    createdStage.ID,
		InstanceId: database.InstanceId,
		DatabaseId: &database.ID,
		Status:     api.TaskPending,
		Type:       api.TaskDatabaseBackup,
		Payload:    string(bytes),
		CreatorId:  backupCreate.CreatorId,
	})
	if err != nil {
		return fmt.Errorf("failed to create task: %w", err)
	}
	return nil
}
