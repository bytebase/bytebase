package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/db"
	"go.uber.org/zap"
)

const (
	// The chosen interval is a balance between anomaly staleness tolerance and background load.
	anomalyScanInterval = time.Duration(10) * time.Minute
)

// NewAnomalyScanner creates a anomaly scanner
func NewAnomalyScanner(logger *zap.Logger, server *Server) *AnomalyScanner {
	return &AnomalyScanner{
		l:      logger,
		server: server,
	}
}

// AnomalyScanner is the anomaly scanner.
type AnomalyScanner struct {
	l      *zap.Logger
	server *Server
}

// Run will run the anomaly scanner once.
func (s *AnomalyScanner) Run(ctx context.Context, wg *sync.WaitGroup) {
	ticker := time.NewTicker(anomalyScanInterval)
	defer ticker.Stop()
	defer wg.Done()
	s.l.Debug(fmt.Sprintf("Anomaly scanner started and will run every %v", anomalyScanInterval))
	runningTasks := make(map[int]bool)
	mu := sync.RWMutex{}
	for {
		select {
		case <-ticker.C:
			s.l.Debug("New anomaly scanner round started...")
			func() {
				defer func() {
					if r := recover(); r != nil {
						err, ok := r.(error)
						if !ok {
							err = fmt.Errorf("%v", r)
						}
						s.l.Error("Anomaly scanner PANIC RECOVER", zap.Error(err))
					}
				}()

				ctx := context.Background()

				envList, err := s.server.store.FindEnvironment(ctx, &api.EnvironmentFind{})
				if err != nil {
					s.l.Error("Failed to retrieve instance list", zap.Error(err))
					return
				}

				backupPlanPolicyMap := make(map[int]*api.BackupPlanPolicy)
				for _, env := range envList {
					policy, err := s.server.store.GetBackupPlanPolicyByEnvID(ctx, env.ID)
					if err != nil {
						s.l.Error("Failed to retrieve backup policy",
							zap.String("environment", env.Name),
							zap.Error(err))
						return
					}
					backupPlanPolicyMap[env.ID] = policy
				}

				rowStatus := api.Normal
				instanceFind := &api.InstanceFind{
					RowStatus: &rowStatus,
				}
				instanceList, err := s.server.store.FindInstance(ctx, instanceFind)
				if err != nil {
					s.l.Error("Failed to retrieve instance list", zap.Error(err))
					return
				}

				for _, instance := range instanceList {
					foundEnv := false
					for _, env := range envList {
						if env.ID == instance.EnvironmentID {
							if env.RowStatus == api.Normal {
								instance.Environment = env
							}
							foundEnv = true
							break
						}
					}

					if !foundEnv {
						continue
					}

					mu.Lock()
					if _, ok := runningTasks[instance.ID]; ok {
						mu.Unlock()
						continue
					}
					runningTasks[instance.ID] = true
					mu.Unlock()

					// Do NOT use go-routine otherwise would cause "database locked" in underlying SQLite
					func(instance *api.Instance) {
						s.l.Debug("Scan instance anomaly", zap.String("instance", instance.Name))
						defer func() {
							mu.Lock()
							delete(runningTasks, instance.ID)
							mu.Unlock()
						}()

						s.checkInstanceAnomaly(ctx, instance)

						databaseFind := &api.DatabaseFind{
							InstanceID: &instance.ID,
						}
						dbList, err := s.server.store.FindDatabase(ctx, databaseFind)
						if err != nil {
							s.l.Error("Failed to retrieve database list",
								zap.String("instance", instance.Name),
								zap.Error(err))
							return
						}
						for _, database := range dbList {
							s.checkDatabaseAnomaly(ctx, instance, database)
							s.checkBackupAnomaly(ctx, instance, database, backupPlanPolicyMap)
						}
					}(instance)

					// Sleep 1 second after finishing scanning each instance to avoid database lock error in SQLITE
					time.Sleep(1 * time.Second)
				}
			}()
		case <-ctx.Done(): // if cancel() execute
			return
		}
	}
}

func (s *AnomalyScanner) checkInstanceAnomaly(ctx context.Context, instance *api.Instance) {
	driver, err := getAdminDatabaseDriver(ctx, instance, "", s.server.pgInstance.BaseDir, s.l)

	// Check connection
	if err != nil {
		anomalyPayload := api.AnomalyInstanceConnectionPayload{
			Detail: err.Error(),
		}
		payload, err := json.Marshal(anomalyPayload)
		if err != nil {
			s.l.Error("Failed to marshal anomaly payload",
				zap.String("instance", instance.Name),
				zap.String("type", string(api.AnomalyInstanceConnection)),
				zap.Error(err))
		} else {
			if _, err = s.server.store.UpsertActiveAnomaly(ctx, &api.AnomalyUpsert{
				CreatorID:  api.SystemBotID,
				InstanceID: instance.ID,
				Type:       api.AnomalyInstanceConnection,
				Payload:    string(payload),
			}); err != nil {
				s.l.Error("Failed to create anomaly",
					zap.String("instance", instance.Name),
					zap.String("type", string(api.AnomalyInstanceConnection)),
					zap.Error(err))
			}
		}
		return
	}

	defer driver.Close(ctx)
	err = s.server.store.ArchiveAnomaly(ctx, &api.AnomalyArchive{
		InstanceID: &instance.ID,
		Type:       api.AnomalyInstanceConnection,
	})
	if err != nil && common.ErrorCode(err) != common.NotFound {
		s.l.Error("Failed to close anomaly",
			zap.String("instance", instance.Name),
			zap.String("type", string(api.AnomalyInstanceConnection)),
			zap.Error(err))
	}

	// Check migration schema
	{
		setup, err := driver.NeedsSetupMigration(ctx)
		if err != nil {
			s.l.Error("Failed to check migration schema",
				zap.String("instance", instance.Name),
				zap.String("type", string(api.AnomalyInstanceMigrationSchema)),
				zap.Error(err))
		} else {
			if setup {
				if _, err = s.server.store.UpsertActiveAnomaly(ctx, &api.AnomalyUpsert{
					CreatorID:  api.SystemBotID,
					InstanceID: instance.ID,
					Type:       api.AnomalyInstanceMigrationSchema,
				}); err != nil {
					s.l.Error("Failed to create anomaly",
						zap.String("instance", instance.Name),
						zap.String("type", string(api.AnomalyInstanceMigrationSchema)),
						zap.Error(err))
				}
			} else {
				err := s.server.store.ArchiveAnomaly(ctx, &api.AnomalyArchive{
					InstanceID: &instance.ID,
					Type:       api.AnomalyInstanceMigrationSchema,
				})
				if err != nil && common.ErrorCode(err) != common.NotFound {
					s.l.Error("Failed to close anomaly",
						zap.String("instance", instance.Name),
						zap.String("type", string(api.AnomalyInstanceMigrationSchema)),
						zap.Error(err))
				}
			}
		}
	}
}

func (s *AnomalyScanner) checkDatabaseAnomaly(ctx context.Context, instance *api.Instance, database *api.Database) {
	driver, err := getAdminDatabaseDriver(ctx, instance, database.Name, s.server.pgInstance.BaseDir, s.l)

	// Check connection
	if err != nil {
		anomalyPayload := api.AnomalyDatabaseConnectionPayload{
			Detail: err.Error(),
		}
		payload, err := json.Marshal(anomalyPayload)
		if err != nil {
			s.l.Error("Failed to marshal anomaly payload",
				zap.String("instance", instance.Name),
				zap.String("database", database.Name),
				zap.String("type", string(api.AnomalyDatabaseConnection)),
				zap.Error(err))
		} else {
			if _, err = s.server.store.UpsertActiveAnomaly(ctx, &api.AnomalyUpsert{
				CreatorID:  api.SystemBotID,
				InstanceID: instance.ID,
				DatabaseID: &database.ID,
				Type:       api.AnomalyDatabaseConnection,
				Payload:    string(payload),
			}); err != nil {
				s.l.Error("Failed to create anomaly",
					zap.String("instance", instance.Name),
					zap.String("database", database.Name),
					zap.String("type", string(api.AnomalyDatabaseConnection)),
					zap.Error(err))
			}
		}
		return
	}
	defer driver.Close(ctx)
	err = s.server.store.ArchiveAnomaly(ctx, &api.AnomalyArchive{
		DatabaseID: &database.ID,
		Type:       api.AnomalyDatabaseConnection,
	})
	if err != nil && common.ErrorCode(err) != common.NotFound {
		s.l.Error("Failed to close anomaly",
			zap.String("instance", instance.Name),
			zap.String("database", database.Name),
			zap.String("type", string(api.AnomalyDatabaseConnection)),
			zap.Error(err))
	}

	// Check schema drift
	if s.server.feature(api.FeatureSchemaDrift) {
		setup, err := driver.NeedsSetupMigration(ctx)
		if err != nil {
			s.l.Debug("Failed to check anomaly",
				zap.String("instance", instance.Name),
				zap.String("database", database.Name),
				zap.String("type", string(api.AnomalyDatabaseSchemaDrift)),
				zap.Error(err))
			goto SchemaDriftEnd
		}
		// Skip drift check if migration schema is not ready (we have instance anomaly to cover that)
		if setup {
			goto SchemaDriftEnd
		}
		var schemaBuf bytes.Buffer
		if _, err := driver.Dump(ctx, database.Name, &schemaBuf, true /*schemaOnly*/); err != nil {
			if common.ErrorCode(err) == common.NotFound {
				s.l.Debug("Failed to check anomaly",
					zap.String("instance", instance.Name),
					zap.String("database", database.Name),
					zap.String("type", string(api.AnomalyDatabaseSchemaDrift)),
					zap.Error(err))
			} else {
				s.l.Error("Failed to check anomaly",
					zap.String("instance", instance.Name),
					zap.String("database", database.Name),
					zap.String("type", string(api.AnomalyDatabaseSchemaDrift)),
					zap.Error(err))
			}
			goto SchemaDriftEnd
		}
		limit := 1
		list, err := driver.FindMigrationHistoryList(ctx, &db.MigrationHistoryFind{
			Database: &database.Name,
			Limit:    &limit,
		})
		if err != nil {
			s.l.Error("Failed to check anomaly",
				zap.String("instance", instance.Name),
				zap.String("database", database.Name),
				zap.String("type", string(api.AnomalyDatabaseSchemaDrift)),
				zap.Error(err))
			goto SchemaDriftEnd
		}
		if len(list) > 0 {
			if list[0].Schema != schemaBuf.String() {
				anomalyPayload := api.AnomalyDatabaseSchemaDriftPayload{
					Version: list[0].Version,
					Expect:  list[0].Schema,
					Actual:  schemaBuf.String(),
				}
				payload, err := json.Marshal(anomalyPayload)
				if err != nil {
					s.l.Error("Failed to marshal anomaly payload",
						zap.String("instance", instance.Name),
						zap.String("database", database.Name),
						zap.String("type", string(api.AnomalyDatabaseSchemaDrift)),
						zap.Error(err))
				} else {
					if _, err = s.server.store.UpsertActiveAnomaly(ctx, &api.AnomalyUpsert{
						CreatorID:  api.SystemBotID,
						InstanceID: instance.ID,
						DatabaseID: &database.ID,
						Type:       api.AnomalyDatabaseSchemaDrift,
						Payload:    string(payload),
					}); err != nil {
						s.l.Error("Failed to create anomaly",
							zap.String("instance", instance.Name),
							zap.String("database", database.Name),
							zap.String("type", string(api.AnomalyDatabaseSchemaDrift)),
							zap.Error(err))
					}
				}
			} else {
				err := s.server.store.ArchiveAnomaly(ctx, &api.AnomalyArchive{
					DatabaseID: &database.ID,
					Type:       api.AnomalyDatabaseConnection,
				})
				if err != nil && common.ErrorCode(err) != common.NotFound {
					s.l.Error("Failed to close anomaly",
						zap.String("instance", instance.Name),
						zap.String("database", database.Name),
						zap.String("type", string(api.AnomalyDatabaseSchemaDrift)),
						zap.Error(err))
				}
			}
		}
	}
SchemaDriftEnd:
}

func (s *AnomalyScanner) checkBackupAnomaly(ctx context.Context, instance *api.Instance, database *api.Database, policyMap map[int]*api.BackupPlanPolicy) {
	schedule := api.BackupPlanPolicyScheduleUnset
	backupSetting, err := s.server.store.GetBackupSettingByDatabaseID(ctx, database.ID)
	if err != nil {
		s.l.Error("Failed to retrieve backup setting",
			zap.String("instance", instance.Name),
			zap.String("database", database.Name),
			zap.Error(err))
		return
	}

	if backupSetting != nil && backupSetting.Enabled && backupSetting.Hour != -1 {
		if backupSetting.DayOfWeek == -1 {
			schedule = api.BackupPlanPolicyScheduleDaily
		} else {
			schedule = api.BackupPlanPolicyScheduleWeekly
		}
	}

	// Check backup policy violation
	{
		var backupPolicyAnomalyPayload *api.AnomalyDatabaseBackupPolicyViolationPayload
		if policyMap[instance.EnvironmentID].Schedule != api.BackupPlanPolicyScheduleUnset {
			if policyMap[instance.EnvironmentID].Schedule == api.BackupPlanPolicyScheduleDaily &&
				schedule != api.BackupPlanPolicyScheduleDaily {
				backupPolicyAnomalyPayload = &api.AnomalyDatabaseBackupPolicyViolationPayload{
					EnvironmentID:          instance.EnvironmentID,
					ExpectedBackupSchedule: policyMap[instance.EnvironmentID].Schedule,
					ActualBackupSchedule:   schedule,
				}
			} else if policyMap[instance.EnvironmentID].Schedule == api.BackupPlanPolicyScheduleWeekly &&
				schedule == api.BackupPlanPolicyScheduleUnset {
				backupPolicyAnomalyPayload = &api.AnomalyDatabaseBackupPolicyViolationPayload{
					EnvironmentID:          instance.EnvironmentID,
					ExpectedBackupSchedule: policyMap[instance.EnvironmentID].Schedule,
					ActualBackupSchedule:   schedule,
				}
			}
		}

		if backupPolicyAnomalyPayload != nil {
			payload, err := json.Marshal(*backupPolicyAnomalyPayload)
			if err != nil {
				s.l.Error("Failed to marshal anomaly payload",
					zap.String("instance", instance.Name),
					zap.String("database", database.Name),
					zap.String("type", string(api.AnomalyDatabaseBackupPolicyViolation)),
					zap.Error(err))
			} else {
				if _, err = s.server.store.UpsertActiveAnomaly(ctx, &api.AnomalyUpsert{
					CreatorID:  api.SystemBotID,
					InstanceID: instance.ID,
					DatabaseID: &database.ID,
					Type:       api.AnomalyDatabaseBackupPolicyViolation,
					Payload:    string(payload),
				}); err != nil {
					s.l.Error("Failed to create anomaly",
						zap.String("instance", instance.Name),
						zap.String("database", database.Name),
						zap.String("type", string(api.AnomalyDatabaseBackupPolicyViolation)),
						zap.Error(err))
				}
			}
		} else {
			err := s.server.store.ArchiveAnomaly(ctx, &api.AnomalyArchive{
				DatabaseID: &database.ID,
				Type:       api.AnomalyDatabaseBackupPolicyViolation,
			})
			if err != nil && common.ErrorCode(err) != common.NotFound {
				s.l.Error("Failed to close anomaly",
					zap.String("instance", instance.Name),
					zap.String("database", database.Name),
					zap.String("type", string(api.AnomalyDatabaseBackupPolicyViolation)),
					zap.Error(err))
			}
		}
	}

	// Check backup missing
	{
		var backupMissingAnomalyPayload *api.AnomalyDatabaseBackupMissingPayload
		// The anomaly fires if backup is enabled, however no successful backup has been taken during the period.
		if backupSetting != nil && backupSetting.Enabled {
			expectedSchedule := api.BackupPlanPolicyScheduleWeekly
			backupMaxAge := time.Duration(7*24) * time.Hour
			if backupSetting.DayOfWeek == -1 {
				expectedSchedule = api.BackupPlanPolicyScheduleDaily
				backupMaxAge = time.Duration(24) * time.Hour
			}

			// Ignore if backup setting has been changed after the max age.
			if backupSetting.UpdatedTs < time.Now().Add(-backupMaxAge).Unix() {
				status := api.BackupStatusDone
				backupFind := &api.BackupFind{
					DatabaseID: &database.ID,
					Status:     &status,
				}
				backupList, err := s.server.store.FindBackup(ctx, backupFind)
				if err != nil {
					s.l.Error("Failed to retrieve backup list",
						zap.String("instance", instance.Name),
						zap.String("database", database.Name),
						zap.Error(err))
				}

				hasValidBackup := false
				if len(backupList) > 0 {
					if backupList[0].UpdatedTs >= time.Now().Add(-backupMaxAge).Unix() {
						hasValidBackup = true
					}
				}

				if !hasValidBackup {
					backupMissingAnomalyPayload = &api.AnomalyDatabaseBackupMissingPayload{
						ExpectedBackupSchedule: expectedSchedule,
					}
					if len(backupList) > 0 {
						backupMissingAnomalyPayload.LastBackupTs = backupList[0].UpdatedTs
					}
				}
			}
		}

		if backupMissingAnomalyPayload != nil {
			payload, err := json.Marshal(*backupMissingAnomalyPayload)
			if err != nil {
				s.l.Error("Failed to marshal anomaly payload",
					zap.String("instance", instance.Name),
					zap.String("database", database.Name),
					zap.String("type", string(api.AnomalyDatabaseBackupMissing)),
					zap.Error(err))
			} else {
				if _, err = s.server.store.UpsertActiveAnomaly(ctx, &api.AnomalyUpsert{
					CreatorID:  api.SystemBotID,
					InstanceID: instance.ID,
					DatabaseID: &database.ID,
					Type:       api.AnomalyDatabaseBackupMissing,
					Payload:    string(payload),
				}); err != nil {
					s.l.Error("Failed to create anomaly",
						zap.String("instance", instance.Name),
						zap.String("database", database.Name),
						zap.String("type", string(api.AnomalyDatabaseBackupMissing)),
						zap.Error(err))
				}
			}
		} else {
			err := s.server.store.ArchiveAnomaly(ctx, &api.AnomalyArchive{
				DatabaseID: &database.ID,
				Type:       api.AnomalyDatabaseBackupMissing,
			})
			if err != nil && common.ErrorCode(err) != common.NotFound {
				s.l.Error("Failed to close anomaly",
					zap.String("instance", instance.Name),
					zap.String("database", database.Name),
					zap.String("type", string(api.AnomalyDatabaseBackupMissing)),
					zap.Error(err))
			}
		}
	}
}
