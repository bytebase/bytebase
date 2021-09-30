package server

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/bytebase/bytebase/api"
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

					go func(instance *api.Instance) {
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
							if backupPlanPolicyMap[instance.EnvironmentId].Schedule != api.BackupPlanPolicyScheduleUnset {
								backupSettingFind := &api.BackupSettingFind{
									DatabaseId: &database.ID,
								}
								backupSetting, err := s.server.BackupService.FindBackupSetting(context.Background(), backupSettingFind)
								if err != nil {
									s.l.Error("Failed to retrieve backup setting",
										zap.String("instance", instance.Name),
										zap.String("database", database.Name),
										zap.Error(err))
									return
								}
								if !backupSetting.Enabled {
									s.l.Info(fmt.Sprintf("Database %s not enabled backup", database.Name))
								}
							}
						}
					}(instance)
				}
			}()

			time.Sleep(ANOMALY_SCAN_INTERVAL)
		}
	}()

	return nil
}
