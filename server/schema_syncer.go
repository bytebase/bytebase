package server

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common/log"
)

const (
	schemaSyncInterval = time.Duration(30) * time.Minute
)

var (
	instanceSyncChan chan *api.Instance = make(chan *api.Instance)
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
							err = errors.Errorf("%v", r)
						}
						log.Error("Schema syncer PANIC RECOVER", zap.Error(err), zap.Stack("panic-stack"))
					}
				}()

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
						databaseList, err := s.server.syncInstance(ctx, instance)
						if err != nil {
							log.Debug("Failed to sync instance",
								zap.Int("id", instance.ID),
								zap.String("name", instance.Name),
								zap.String("error", err.Error()))
							return
						}
						for _, databaseName := range databaseList {
							// If we fail to sync a particular database due to permission issue, we will continue to sync the rest of the databases.
							if err := s.server.syncDatabaseSchema(ctx, instance, databaseName); err != nil {
								log.Debug("Failed to sync database schema",
									zap.Int("instanceID", instance.ID),
									zap.String("instanceName", instance.Name),
									zap.String("databaseName", databaseName),
									zap.String("error", err.Error()))
							}
						}
					}(instance)
				}
			}()
		case instance := <-instanceSyncChan:
			go func(instance *api.Instance) {
				defer func() {
					if r := recover(); r != nil {
						err, ok := r.(error)
						if !ok {
							err = errors.Errorf("%v", r)
						}
						log.Error("Schema syncer PANIC RECOVER", zap.Error(err), zap.Stack("panic-stack"))
					}
				}()

				databaseList, err := s.server.store.FindDatabase(ctx, &api.DatabaseFind{InstanceID: &instance.ID})
				if err != nil {
					log.Debug("Failed to find databases for the syncing instance",
						zap.Int("id", instance.ID),
						zap.String("name", instance.Name),
						zap.String("error", err.Error()))
					return
				}
				for _, database := range databaseList {
					if database.SyncStatus != api.OK {
						continue
					}

					// If we fail to sync a particular database due to permission issue, we will continue to sync the rest of the databases.
					if err := s.server.syncDatabaseSchema(ctx, instance, database.Name); err != nil {
						log.Debug("Failed to sync database schema",
							zap.Int("instanceID", instance.ID),
							zap.String("instanceName", instance.Name),
							zap.String("databaseName", database.Name),
							zap.String("error", err.Error()))
					}
				}
			}(instance)
		case <-ctx.Done(): // if cancel() execute
			return
		}
	}
}
