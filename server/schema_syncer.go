package server

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common/log"
	"go.uber.org/zap"
)

const (
	schemaSyncInterval = time.Duration(30) * time.Minute
)

// NewSchemaSyncer creates a schema syncer.
func NewSchemaSyncer(server *Server) *SchemaSyncer {
	return &SchemaSyncer{
		server: server,
	}
}

// SchemaSyncer is the schema syncer.
type SchemaSyncer struct {
	server *Server
}

// Run will run the schema syncer once.
func (s *SchemaSyncer) Run(ctx context.Context, wg *sync.WaitGroup) {
	ticker := time.NewTicker(schemaSyncInterval)
	defer ticker.Stop()
	defer wg.Done()
	log.Debug(fmt.Sprintf("Schema syncer started and will run every %v", schemaSyncInterval))
	runningTasks := make(map[int]bool)
	mu := sync.RWMutex{}
	for {
		select {
		case <-ticker.C:
			log.Debug("New schema syncer round started...")
			func() {
				defer func() {
					if r := recover(); r != nil {
						err, ok := r.(error)
						if !ok {
							err = fmt.Errorf("%v", r)
						}
						log.Error("Schema syncer PANIC RECOVER", zap.Error(err))
					}
				}()

				ctx := context.Background()

				rowStatus := api.Normal
				instanceFind := &api.InstanceFind{
					RowStatus: &rowStatus,
				}
				instanceList, err := s.server.store.FindInstance(ctx, instanceFind)
				if err != nil {
					log.Error("Failed to retrieve instances", zap.Error(err))
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
						log.Debug("Sync instance schema", zap.String("instance", instance.Name))
						defer func() {
							mu.Lock()
							delete(runningTasks, instance.ID)
							mu.Unlock()
						}()
						resultSet := s.server.syncEngineVersionAndSchema(ctx, instance)
						if resultSet.Error != "" {
							log.Debug("Failed to sync instance",
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
