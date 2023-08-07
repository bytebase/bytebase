// Package slowquerysync is a runner that synchronize slow query logs.
package slowquerysync

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	"github.com/bytebase/bytebase/backend/component/state"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/pg"
	"github.com/bytebase/bytebase/backend/store"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
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
		case instanceResourceID := <-s.stateCfg.InstanceSlowQuerySyncChan:
			log.Debug("Slow query syncer received instance slow query sync request", zap.String("instance", instanceResourceID))
			s.syncSlowQuery(ctx, &instanceResourceID)
		case <-ticker.C:
			log.Debug("Slow query syncer received tick")
			s.syncSlowQuery(ctx, nil)
		}
	}
}

func (s *Syncer) syncSlowQuery(ctx context.Context, instanceResourceID *string) {
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
	if instanceResourceID != nil {
		find.ResourceID = instanceResourceID
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

	switch instance.Engine {
	case db.MySQL:
		return s.syncMySQLSlowQuery(ctx, instance)
	case db.Postgres:
		return s.syncPostgreSQLSlowQuery(ctx, instance)
	default:
		return errors.Errorf("unsupported database engine: %s", instance.Engine)
	}
}

func (s *Syncer) syncPostgreSQLSlowQuery(ctx context.Context, instance *store.InstanceMessage) error {
	today := time.Now().UTC().Truncate(24 * time.Hour)

	earliestDate := today.AddDate(0, 0, -retentionCycle)

	if err := s.store.DeleteOutdatedSlowLog(ctx, instance.UID, earliestDate); err != nil {
		return err
	}

	databases, err := s.store.ListDatabases(ctx, &store.FindDatabaseMessage{
		InstanceID: &instance.ResourceID,
	})
	if err != nil {
		return err
	}

	var enabledDatabases []*store.DatabaseMessage

	for _, database := range databases {
		if database.SyncState != api.OK {
			continue
		}
		if _, exists := pg.ExcludedDatabaseList[database.DatabaseName]; exists {
			continue
		}
		if err := func() error {
			driver, err := s.dbFactory.GetAdminDatabaseDriver(ctx, instance, database)
			if err != nil {
				return err
			}
			defer driver.Close(ctx)
			return driver.CheckSlowQueryLogEnabled(ctx)
		}(); err != nil {
			log.Warn("pg_stat_statements is not enabled",
				zap.String("instance", instance.ResourceID),
				zap.String("database", database.DatabaseName),
				zap.Int("databaseID", database.UID),
				zap.Error(err))
			continue
		}

		enabledDatabases = append(enabledDatabases, database)
	}

	if len(enabledDatabases) == 0 {
		return errors.Errorf("no database is available for slow query sync in instance %s", instance.ResourceID)
	}

	driver, err := s.dbFactory.GetAdminDatabaseDriver(ctx, instance, enabledDatabases[0])
	if err != nil {
		return err
	}
	defer driver.Close(ctx)

	logMap, err := driver.SyncSlowQuery(ctx, time.Now() /* logDateTs is not used for postgresql */)
	if err != nil {
		return err
	}

	latestLogDate := getLatestLogTime(logMap)
	if latestLogDate.IsZero() {
		// Empty log, no need to sync.
		return nil
	}
	latestLogDate = latestLogDate.Truncate(24 * time.Hour)
	nextLogDate := latestLogDate.AddDate(0, 0, 1)

	for _, database := range enabledDatabases {
		statistics, exists := logMap[database.DatabaseName]
		if !exists {
			continue
		}

		logs, err := s.store.ListSlowQuery(ctx, &store.ListSlowQueryMessage{
			InstanceUID:  &instance.UID,
			DatabaseUID:  &database.UID,
			StartLogDate: &latestLogDate,
			EndLogDate:   &nextLogDate,
		})
		if err != nil {
			log.Warn("Failed to list slow query logs",
				zap.String("instance", instance.ResourceID),
				zap.String("database", database.DatabaseName),
				zap.Int("databaseID", database.UID),
				zap.Error(err))
			logs = nil
		}

		if len(logs) != 0 {
			statistics = pgMergeSlowQueryLog(statistics, logs)
		}
		if err := s.store.UpsertSlowLog(ctx, &store.UpsertSlowLogMessage{
			EnvironmentID: &instance.EnvironmentID,
			InstanceID:    &instance.ResourceID,
			DatabaseName:  database.DatabaseName,
			InstanceUID:   instance.UID,
			LogDate:       latestLogDate,
			SlowLog:       statistics,
			UpdaterID:     api.SystemBotID,
		}); err != nil {
			log.Warn("Failed to upsert slow query log",
				zap.String("instance", instance.ResourceID),
				zap.String("database", database.DatabaseName),
				zap.Int("databaseID", database.UID),
				zap.Error(err))
		}
	}

	return nil
}

func pgMergeSlowQueryLog(statistics *storepb.SlowQueryStatistics, logs []*v1pb.SlowQueryLog) *storepb.SlowQueryStatistics {
	status := make(map[string]*storepb.SlowQueryStatisticsItem)

	for _, item := range statistics.Items {
		status[item.SqlFingerprint] = item
	}

	for _, log := range logs {
		value, exists := status[log.Statistics.SqlFingerprint]
		if !exists {
			status[log.Statistics.SqlFingerprint] = &storepb.SlowQueryStatisticsItem{
				SqlFingerprint:   log.Statistics.SqlFingerprint,
				Count:            log.Statistics.Count,
				LatestLogTime:    log.Statistics.LatestLogTime,
				TotalQueryTime:   durationpb.New(log.Statistics.AverageQueryTime.AsDuration() * time.Duration(log.Statistics.Count)),
				MaximumQueryTime: log.Statistics.MaximumQueryTime,
				TotalRowsSent:    log.Statistics.AverageRowsSent * log.Statistics.Count,
			}
		} else {
			value.Count += log.Statistics.Count
			totalQueryTime := log.Statistics.AverageQueryTime.AsDuration() * time.Duration(log.Statistics.Count)
			value.TotalQueryTime = durationpb.New(value.TotalQueryTime.AsDuration() + totalQueryTime)
			if value.MaximumQueryTime.AsDuration() < log.Statistics.MaximumQueryTime.AsDuration() {
				value.MaximumQueryTime = log.Statistics.MaximumQueryTime
			}
			value.TotalRowsSent += log.Statistics.AverageRowsSent * log.Statistics.Count
		}
	}

	var result []*storepb.SlowQueryStatisticsItem
	for _, item := range status {
		result = append(result, item)
	}
	return &storepb.SlowQueryStatistics{Items: result}
}

func getLatestLogTime(logMap map[string]*storepb.SlowQueryStatistics) time.Time {
	for _, log := range logMap {
		for _, item := range log.Items {
			return item.LatestLogTime.AsTime()
		}
	}
	return time.Time{}
}

func (s *Syncer) syncMySQLSlowQuery(ctx context.Context, instance *store.InstanceMessage) error {
	today := time.Now().UTC().Truncate(24 * time.Hour)

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

	driver, err := s.dbFactory.GetAdminDatabaseDriver(ctx, instance, nil /* database */)
	if err != nil {
		return err
	}
	defer driver.Close(ctx)
	if err := driver.CheckSlowQueryLogEnabled(ctx); err != nil {
		return err
	}

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
