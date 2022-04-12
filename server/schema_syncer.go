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
	schemaSyncInterval = time.Duration(30) * time.Minute
)

// NewSchemaSyncer creates a schema syncer.
func NewSchemaSyncer(logger *zap.Logger, server *Server) *SchemaSyncer {
	return &SchemaSyncer{
		l:      logger,
		server: server,
	}
}

// SchemaSyncer is the schema syncer.
type SchemaSyncer struct {
	l      *zap.Logger
	server *Server
}

// Run will run the schema syncer once.
func (s *SchemaSyncer) Run(ctx context.Context, wg *sync.WaitGroup) {
	ticker := time.NewTicker(schemaSyncInterval)
	defer ticker.Stop()
	defer wg.Done()
	s.l.Debug(fmt.Sprintf("Schema syncer started and will run every %v", schemaSyncInterval))
	runningTasks := make(map[int]bool)
	mu := sync.RWMutex{}
	for {
		select {
		case <-ticker.C:
			s.l.Debug("New schema syncer round started...")
			func() {
				defer func() {
					if r := recover(); r != nil {
						err, ok := r.(error)
						if !ok {
							err = fmt.Errorf("%v", r)
						}
						s.l.Error("Schema syncer PANIC RECOVER", zap.Error(err), zap.Stack("stack"))
					}
				}()

				ctx := context.Background()

				rowStatus := api.Normal
				instanceFind := &api.InstanceFind{
					RowStatus: &rowStatus,
				}
				instanceList, err := s.server.store.FindInstance(ctx, instanceFind)
				if err != nil {
					s.l.Error("Failed to retrieve instances", zap.Error(err))
					return
				}

				for _, instance := range instanceList {
					mu.Lock()
					if _, ok := runningTasks[instance.ID]; ok {
						mu.Unlock()
						continue
					}
					runningTasks[instance.ID] = true
					mu.Unlock()

					go func(instance *api.Instance) {
						s.l.Debug("Sync instance schema", zap.String("instance", instance.Name))
						defer func() {
							mu.Lock()
							delete(runningTasks, instance.ID)
							mu.Unlock()
						}()
						resultSet := s.server.syncEngineVersionAndSchema(ctx, instance)
						if resultSet.Error != "" {
							s.l.Debug("Failed to sync instance",
								zap.Int("id", instance.ID),
								zap.String("name", instance.Name),
								zap.String("error", resultSet.Error))
						}
					}(instance)
				}
			}()
		case <-ctx.Done(): // if cancel() execute
			return
		}
	}
}
