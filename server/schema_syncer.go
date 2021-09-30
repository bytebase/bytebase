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
	SCHEMA_SYNC_INTERVAL = time.Duration(30) * time.Minute
)

func NewSchemaSyncer(logger *zap.Logger, server *Server) *SchemaSyncer {
	return &SchemaSyncer{
		l:      logger,
		server: server,
	}
}

type SchemaSyncer struct {
	l      *zap.Logger
	server *Server
}

func (s *SchemaSyncer) Run() error {
	go func() {
		s.l.Debug(fmt.Sprintf("Schema syncer started and will run every %v", SCHEMA_SYNC_INTERVAL))
		runningTasks := make(map[int]bool)
		mu := sync.RWMutex{}
		for {
			s.l.Debug("New schema syncer round started...")
			func() {
				defer func() {
					if r := recover(); r != nil {
						err, ok := r.(error)
						if !ok {
							err = fmt.Errorf("%v", r)
						}
						s.l.Error("Schema syncer PANIC RECOVER", zap.Error(err))
					}
				}()

				rowStatus := api.Normal
				instanceFind := &api.InstanceFind{
					RowStatus: &rowStatus,
				}
				list, err := s.server.InstanceService.FindInstanceList(context.Background(), instanceFind)
				if err != nil {
					s.l.Error("Failed to retrieve instances", zap.Error(err))
				}

				for _, instance := range list {
					mu.Lock()
					if _, ok := runningTasks[instance.ID]; ok {
						mu.Unlock()
						continue
					}
					runningTasks[instance.ID] = true
					mu.Unlock()

					if err := s.server.ComposeInstanceRelationship(context.Background(), instance); err != nil {
						s.l.Error("Failed to sync instance",
							zap.Int("id", instance.ID),
							zap.String("name", instance.Name),
							zap.String("error", err.Error()))
						continue
					}
					go func(instance *api.Instance) {
						s.l.Debug("Sync instance schema", zap.String("instance", instance.Name))
						defer func() {
							mu.Lock()
							delete(runningTasks, instance.ID)
							mu.Unlock()
						}()
						resultSet := s.server.SyncSchema(instance)
						if resultSet.Error != "" {
							s.l.Debug("Failed to sync instance",
								zap.Int("id", instance.ID),
								zap.String("name", instance.Name),
								zap.String("error", resultSet.Error))
						}
					}(instance)
				}
			}()

			time.Sleep(SCHEMA_SYNC_INTERVAL)
		}
	}()

	return nil
}
