package collector

import (
	"context"
	"strconv"

	"github.com/bytebase/bytebase/api"
	metricAPI "github.com/bytebase/bytebase/metric"
	"github.com/bytebase/bytebase/plugin/metric"
	"github.com/bytebase/bytebase/store"
	"go.uber.org/zap"
)

var _ metric.Collector = (*databaseCountCollector)(nil)

// databaseCountCollector is the metric data collector for database.
type databaseCountCollector struct {
	l     *zap.Logger
	store *store.Store
}

// NewDatabaseCountCollector creates a new instance of databaseCountCollector
func NewDatabaseCountCollector(l *zap.Logger, store *store.Store) metric.Collector {
	return &databaseCountCollector{
		l:     l,
		store: store,
	}
}

// Collect will collect the metric for database
func (c *databaseCountCollector) Collect(ctx context.Context) ([]*metric.Metric, error) {
	var res []*metric.Metric

	databaseList, err := c.store.FindDatabase(ctx, &api.DatabaseFind{})
	if err != nil {
		return nil, err
	}

	group := make(map[api.BackupPlanPolicySchedule]map[bool]int, 3) // map[schedule][enabled]count
	for _, schedule := range []api.BackupPlanPolicySchedule{
		api.BackupPlanPolicyScheduleDaily,
		api.BackupPlanPolicyScheduleWeekly,
		api.BackupPlanPolicyScheduleUnset,
	} {
		group[schedule] = make(map[bool]int, 2)
		group[schedule][true], group[schedule][false] = 0, 0
	}

	for _, database := range databaseList {
		if database.Instance == nil {
			c.l.Debug("failed to get instance by id", zap.Int("id", database.InstanceID))
			continue
		}
		backupPolicy, err := c.store.GetBackupPlanPolicyByEnvID(ctx, database.Instance.EnvironmentID)
		if err != nil {
			c.l.Debug("failed to get policy by environment id", zap.Int("id", database.Instance.EnvironmentID))
			continue
		}
		backupSetting, err := c.store.GetBackupSettingByDatabaseID(ctx, database.ID)
		if err != nil {
			c.l.Debug("failed to get backup setting by id", zap.Int("id", database.ID))
			continue
		}
		enabled := false
		if backupSetting != nil {
			enabled = backupSetting.Enabled
		}
		group[backupPolicy.Schedule][enabled]++
	}

	for schedule, subgroup := range group {
		for enabled, count := range subgroup {
			res = append(res, &metric.Metric{
				Name:  metricAPI.DatabaseCountMetricName,
				Value: count,
				Labels: map[string]string{
					"backup_schedule": string(schedule),
					"backup_enabled":  strconv.FormatBool(enabled),
				},
			})
		}
	}

	return res, nil
}
