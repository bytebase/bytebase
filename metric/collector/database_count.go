package collector

import (
	"context"
	"strconv"

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

	databaseCountMetricList, err := c.store.CountDatabaseGroupByBackupScheduleAndEnabled(ctx)
	if err != nil {
		return nil, err
	}

	for _, databaseCountMetric := range databaseCountMetricList {
		labels := map[string]string{
			"backup_schedule": "null",
			"backup_enabled":  "null",
		}
		if v := databaseCountMetric.BackupPlanPolicySchedule; v != nil {
			labels["backup_schedule"] = string(*v)
		}
		if v := databaseCountMetric.BackupSettingEnabled; v != nil {
			labels["backup_enabled"] = strconv.FormatBool(*v)
		}
		res = append(res, &metric.Metric{
			Name:   metricAPI.DatabaseCountMetricName,
			Value:  databaseCountMetric.Count,
			Labels: labels,
		})
	}

	return res, nil
}
