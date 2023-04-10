// Package slowquerysync is a runner that synchronize slow query logs.
package slowquerysync

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	"github.com/bytebase/bytebase/backend/component/state"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/store"
)

const (
	slowQuerySyncInterval = 12 * time.Hour
	// retentionCycle is the number of days to keep slow query logs.
	retentionCycle = 30
)

// NewSyncer creates a new slow query syncer.
func NewSyncer(store *store.Store, dbFactory *dbfactory.DBFactory, stateCfg *state.State, profile config.Profile) *Syncer {
	return &Syncer{
		store:     store,
		dbFactory: dbFactory,
		stateCfg:  stateCfg,
		profile:   profile,
	}
}

// Syncer is the slow query syncer.
type Syncer struct {
	store     *store.Store
	dbFactory *dbfactory.DBFactory
	stateCfg  *state.State
	profile   config.Profile
}

// Run will run the slow query syncer.
func (s *Syncer) Run(ctx context.Context, wg *sync.WaitGroup) {
	ticker := time.NewTicker(slowQuerySyncInterval)
	defer ticker.Stop()
	defer wg.Done()
	log.Debug(fmt.Sprintf("Slow query syncer started and will run every %s", slowQuerySyncInterval.String()))
	for {
		select {
		case <-ctx.Done():
			log.Debug("Slow query syncer received context cancellation")
			return
		case instance := <-s.stateCfg.InstanceSlowQuerySyncChan:
			log.Debug("Slow query syncer received instance slow query sync request", zap.String("instance", instance.ResourceID))
			s.syncSlowQuery(ctx, instance)
		case <-ticker.C:
			log.Debug("Slow query syncer received tick")
			s.syncSlowQuery(ctx, nil)
		}
	}
}

func (s *Syncer) syncSlowQuery(ctx context.Context, instance *api.Instance) {
	defer func() {
		if r := recover(); r != nil {
			err, ok := r.(error)
			if !ok {
				err = errors.Errorf("%v", r)
			}
			log.Error("slow query syncer PANIC RECOVER", zap.Error(err), zap.Stack("panic-stack"))
		}
	}()

	find := &store.FindInstanceMessage{}
	if instance != nil {
		find.UID = &instance.ID
	}
	instances, err := s.store.ListInstancesV2(ctx, find)
	if err != nil {
		log.Error("Failed to list instances", zap.Error(err))
		return
	}

	var instanceWG sync.WaitGroup
	for _, instance := range instances {
		if instance.Deleted {
			continue
		}
		instanceWG.Add(1)
		go func(instance *store.InstanceMessage) {
			defer instanceWG.Done()
			if err := s.syncInstanceSlowQuery(ctx, instance); err != nil {
				log.Debug("Failed to sync instance slow query",
					zap.String("instance", instance.ResourceID),
					zap.Error(err))
			}
		}(instance)
	}
	instanceWG.Wait()
}

func (s *Syncer) syncInstanceSlowQuery(ctx context.Context, instance *store.InstanceMessage) error {
	slowQueryPolicy, err := s.store.GetSlowQueryPolicy(ctx, api.PolicyResourceTypeInstance, instance.UID)
	if err != nil {
		return err
	}
	if slowQueryPolicy == nil || !slowQueryPolicy.Active {
		return nil
	}

	today := time.Now().Truncate(24 * time.Hour)

	earliestDate := today.AddDate(0, 0, -retentionCycle)

	if err := s.store.DeleteOutdatedSlowLog(ctx, instance.UID, earliestDate); err != nil {
		return err
	}

	latestSlowLogDate, err := s.store.GetLatestSlowLogDate(ctx, instance.UID)
	if err != nil {
		return err
	}

	if latestSlowLogDate == nil || latestSlowLogDate.AddDate(0, 0, retentionCycle).Before(today) {
		latestSlowLogDate = &earliestDate
	}

	driver, err := s.dbFactory.GetAdminDatabaseDriver(ctx, instance, "")
	if err != nil {
		return err
	}
	defer driver.Close(ctx)

	for date := latestSlowLogDate.Truncate(24 * time.Hour); !date.After(today); date = date.AddDate(0, 0, 1) {
		logs, err := driver.SyncSlowQuery(ctx, date)
		if err != nil {
			return err
		}

		for dbName, slowLog := range logs {
			if err := s.store.UpsertSlowLog(ctx, &store.UpsertSlowLogMessage{
				EnvironmentID: &instance.EnvironmentID,
				InstanceID:    &instance.ResourceID,
				DatabaseName:  dbName,
				InstanceUID:   instance.UID,
				LogDate:       date,
				SlowLog:       slowLog,
				UpdaterID:     api.SystemBotID,
			}); err != nil {
				return err
			}
		}
	}

	return nil
}
