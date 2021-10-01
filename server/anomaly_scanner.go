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

const (
	ANOMALY_SCAN_INTERVAL = time.Duration(1) * time.Second
)

func NewAnomalyScanner(logger *zap.Logger, server *Server) *AnomalyScanner {
	return &AnomalyScanner{
		l:      logger,
		server: server,
	}
}

type AnomalyScanner struct {
	l      *zap.Logger
	server *Server
}

func (s *AnomalyScanner) Run() error {
	go func() {
		s.l.Debug(fmt.Sprintf("Anomaly scanner started and will run every %v", ANOMALY_SCAN_INTERVAL))
		runningTasks := make(map[int]bool)
		mu := sync.RWMutex{}
		for {
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

				environmentFind := &api.EnvironmentFind{}
				environmentList, err := s.server.EnvironmentService.FindEnvironmentList(context.Background(), environmentFind)
				if err != nil {
					s.l.Error("Failed to retrieve instance list", zap.Error(err))
					return
				}

				backupPlanPolicyMap := make(map[int]*api.BackupPlanPolicy)
				for _, env := range environmentList {
					policy, err := s.server.PolicyService.GetBackupPlanPolicy(context.Background(), env.ID)
					if err != nil {
						s.l.Error("Failed to retrieve backup policy",
							zap.String("environment", env.Name),
							zap.Error(err))
						return
					}
					backupPlanPolicyMap[env.ID] = policy
				}

				instanceFind := &api.InstanceFind{}
				instanceList, err := s.server.InstanceService.FindInstanceList(context.Background(), instanceFind)
				if err != nil {
					s.l.Error("Failed to retrieve instance list", zap.Error(err))
					return
				}

				for _, instance := range instanceList {
					if instance.RowStatus != api.Normal {
						continue
					}

					for _, env := range environmentList {
						if env.ID == instance.ID {
							if env.RowStatus == api.Normal {
								instance.Environment = env
							}
							break
						}
					}

					if instance.Environment != nil {
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

						databaseFind := &api.DatabaseFind{
							InstanceId: &instance.ID,
						}
						dbList, err := s.server.DatabaseService.FindDatabaseList(context.Background(), databaseFind)
						if err != nil {
							s.l.Error("Failed to retrieve database list",
								zap.String("instance", instance.Name),
								zap.Error(err))
							return
						}
						for _, database := range dbList {
							s.checkBackupAnomaly(instance, database, backupPlanPolicyMap)
						}
					}(instance)
				}
			}()

			time.Sleep(ANOMALY_SCAN_INTERVAL)
		}
	}()

	return nil
}

func (s *AnomalyScanner) checkBackupAnomaly(instance *api.Instance, database *api.Database, policyMap map[int]*api.BackupPlanPolicy) {
	schedule := api.BackupPlanPolicyScheduleUnset
	backupSettingFind := &api.BackupSettingFind{
		DatabaseId: &database.ID,
	}
	backupSetting, err := s.server.BackupService.FindBackupSetting(context.Background(), backupSettingFind)
	if err != nil {
		if common.ErrorCode(err) != common.NotFound {
			s.l.Error("Failed to retrieve backup setting",
				zap.String("instance", instance.Name),
				zap.String("database", database.Name),
				zap.Error(err))
			return
		}
	} else {
		if backupSetting.Enabled && backupSetting.Hour != -1 {
			if backupSetting.DayOfWeek == -1 {
				schedule = api.BackupPlanPolicyScheduleDaily
			} else {
				schedule = api.BackupPlanPolicyScheduleWeekly
			}
		}
	}

	// Check backup policy violation
	{
		var backupPolicyAnomalyPayload *api.AnomalyBackupPolicyViolationPayload
		if policyMap[instance.EnvironmentId].Schedule != api.BackupPlanPolicyScheduleUnset {
			if policyMap[instance.EnvironmentId].Schedule == api.BackupPlanPolicyScheduleDaily &&
				schedule != api.BackupPlanPolicyScheduleDaily {
				backupPolicyAnomalyPayload = &api.AnomalyBackupPolicyViolationPayload{
					EnvironmentId:          instance.EnvironmentId,
					ExpectedBackupSchedule: policyMap[instance.EnvironmentId].Schedule,
					ActualBackupSchedule:   schedule,
				}
			} else if policyMap[instance.EnvironmentId].Schedule == api.BackupPlanPolicyScheduleWeekly &&
				schedule == api.BackupPlanPolicyScheduleUnset {
				backupPolicyAnomalyPayload = &api.AnomalyBackupPolicyViolationPayload{
					EnvironmentId:          instance.EnvironmentId,
					ExpectedBackupSchedule: policyMap[instance.EnvironmentId].Schedule,
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
					zap.String("type", string(api.AnomalyBackupPolicyViolation)),
					zap.Error(err))
			} else {
				_, err = s.server.AnomalyService.UpsertAnomaly(context.Background(), &api.AnomalyUpsert{
					CreatorId:  api.SYSTEM_BOT_ID,
					InstanceId: instance.ID,
					DatabaseId: database.ID,
					Type:       api.AnomalyBackupPolicyViolation,
					Payload:    string(payload),
				})
				if err != nil {
					s.l.Error("Failed to create anomaly",
						zap.String("instance", instance.Name),
						zap.String("database", database.Name),
						zap.String("type", string(api.AnomalyBackupPolicyViolation)),
						zap.Error(err))
				}
			}
		} else {
			err := s.server.AnomalyService.ArchiveAnomaly(context.Background(), &api.AnomalyArchive{
				DatabaseId: database.ID,
				Type:       api.AnomalyBackupPolicyViolation,
			})
			if err != nil && common.ErrorCode(err) != common.NotFound {
				s.l.Error("Failed to close anomaly",
					zap.String("instance", instance.Name),
					zap.String("database", database.Name),
					zap.String("type", string(api.AnomalyBackupPolicyViolation)),
					zap.Error(err))
			}
		}
	}

	// Check backup missing
	// The anomaly fires if backup is enabled, however no succesful backup has been taken during the period.
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
				DatabaseId: &database.ID,
				Status:     &status,
			}
			backupList, err := s.server.BackupService.FindBackupList(context.Background(), backupFind)
			if err != nil {
				s.l.Error("Failed to retrieve backup list",
					zap.String("instance", instance.Name),
					zap.String("database", database.Name),
					zap.Error(err))
			}

			hasValidBackup := false
			if len(backupList) > 0 {
				if backupList[0].CreatedTs >= time.Now().Add(-backupMaxAge).Unix() {
					hasValidBackup = true
				}
			}

			if !hasValidBackup {
				backupMissingAnomalyPayload := &api.AnomalyBackupMissingPayload{
					ExpectedBackupSchedule: expectedSchedule,
				}
				if len(backupList) > 0 {
					backupMissingAnomalyPayload.LastBackupTs = backupList[0].CreatedTs
				}
				payload, err := json.Marshal(*backupMissingAnomalyPayload)
				if err != nil {
					s.l.Error("Failed to marshal anomaly payload",
						zap.String("instance", instance.Name),
						zap.String("database", database.Name),
						zap.String("type", string(api.AnomalyBackupMissing)),
						zap.Error(err))
				} else {
					_, err = s.server.AnomalyService.UpsertAnomaly(context.Background(), &api.AnomalyUpsert{
						CreatorId:  api.SYSTEM_BOT_ID,
						InstanceId: instance.ID,
						DatabaseId: database.ID,
						Type:       api.AnomalyBackupMissing,
						Payload:    string(payload),
					})
					if err != nil {
						s.l.Error("Failed to create anomaly",
							zap.String("instance", instance.Name),
							zap.String("database", database.Name),
							zap.String("type", string(api.AnomalyBackupMissing)),
							zap.Error(err))
					}
				}
			}
		}
	}
}
