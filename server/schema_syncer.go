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
	"github.com/bytebase/bytebase/store"
)

const (
	schemaSyncInterval = 30 * time.Minute
)

var (
	instanceDatabaseSyncChan = make(chan *api.Instance, 100)
)

// NewSchemaSyncer creates a schema syncer.
func NewSchemaSyncer(server *Server, store *store.Store) *SchemaSyncer {
	return &SchemaSyncer{
		server: server,
		store:  store,
	}
}

// SchemaSyncer is the schema syncer.
type SchemaSyncer struct {
	server *Server
	store  *store.Store
}

// Run will run the schema syncer once.
func (s *SchemaSyncer) Run(ctx context.Context, wg *sync.WaitGroup) {
	ticker := time.NewTicker(schemaSyncInterval)
	defer ticker.Stop()
	defer wg.Done()
	log.Debug(fmt.Sprintf("Schema syncer started and will run every %v", schemaSyncInterval))
	for {
		select {
		case <-ticker.C:
			s.syncAllInstances(ctx)
			// Sync all databases for all instances.
			s.syncAllDatabases(ctx, nil /* instanceID */)
		case instance := <-instanceDatabaseSyncChan:
			// Sync all databases for instance.
			s.syncAllDatabases(ctx, &instance.ID)
		case <-ctx.Done(): // if cancel() execute
			return
		}
	}
}

func (s *SchemaSyncer) syncAllInstances(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			err, ok := r.(error)
			if !ok {
				err = errors.Errorf("%v", r)
			}
			log.Error("Instance syncer PANIC RECOVER", zap.Error(err), zap.Stack("panic-stack"))
		}
	}()

	rowStatus := api.Normal
	instanceFind := &api.InstanceFind{
		RowStatus: &rowStatus,
	}
	instanceList, err := s.store.FindInstance(ctx, instanceFind)
	if err != nil {
		log.Error("Failed to retrieve instances", zap.Error(err))
		return
	}

	var instanceWG sync.WaitGroup
	for _, instance := range instanceList {
		instanceWG.Add(1)
		go func(instance *api.Instance) {
			defer instanceWG.Done()
			log.Debug("Sync instance schema", zap.String("instance", instance.Name))
			if _, err := s.server.syncInstance(ctx, instance); err != nil {
				log.Debug("Failed to sync instance",
					zap.Int("id", instance.ID),
					zap.String("name", instance.Name),
					zap.String("error", err.Error()))
				return
			}
		}(instance)
	}
	instanceWG.Wait()
}

func (s *SchemaSyncer) syncAllDatabases(ctx context.Context, instanceID *int) {
	defer func() {
		if r := recover(); r != nil {
			err, ok := r.(error)
			if !ok {
				err = errors.Errorf("%v", r)
			}
			log.Error("Database syncer PANIC RECOVER", zap.Error(err), zap.Stack("panic-stack"))
		}
	}()

	okSyncStatus := api.OK
	databaseList, err := s.store.FindDatabase(ctx, &api.DatabaseFind{
		InstanceID: instanceID,
		SyncStatus: &okSyncStatus,
	})
	if err != nil {
		log.Debug("Failed to find databases to sync",
			zap.String("error", err.Error()))
		return
	}

	instanceMap := make(map[int][]*api.Database)
	for _, database := range databaseList {
		instanceMap[database.InstanceID] = append(instanceMap[database.InstanceID], database)
	}

	var instanceWG sync.WaitGroup
	for _, databaseList := range instanceMap {
		instanceWG.Add(1)
		go func(databaseList []*api.Database) {
			defer instanceWG.Done()

			if len(databaseList) == 0 {
				return
			}
			instance := databaseList[0].Instance
			for _, database := range databaseList {
				log.Debug("Sync database schema",
					zap.String("instance", instance.Name),
					zap.String("database", database.Name),
					zap.Int64("lastSuccessfulSyncTs", database.LastSuccessfulSyncTs),
				)
				// If we fail to sync a particular database due to permission issue, we will continue to sync the rest of the databases.
				if err := s.server.syncDatabaseSchema(ctx, instance, database.Name); err != nil {
					log.Debug("Failed to sync database schema",
						zap.Int("instanceID", instance.ID),
						zap.String("instanceName", instance.Name),
						zap.String("databaseName", database.Name),
						zap.Error(err))
				}
			}
		}(databaseList)
	}
	instanceWG.Wait()
}
