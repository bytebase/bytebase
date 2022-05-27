package collector

import (
	"context"

	"github.com/bytebase/bytebase/api"
	metricAPI "github.com/bytebase/bytebase/metric"
	"github.com/bytebase/bytebase/plugin/metric"
	"github.com/bytebase/bytebase/store"
	"go.uber.org/zap"
)

var _ metric.Collector = (*instanceCountCollector)(nil)

// instanceCountCollector is the metric data collector for instance.
type instanceCountCollector struct {
	l     *zap.Logger
	store *store.Store
}

// NewInstanceCountCollector creates a new instance of instanceCollector
func NewInstanceCountCollector(l *zap.Logger, store *store.Store) metric.Collector {
	return &instanceCountCollector{
		l:     l,
		store: store,
	}
}

// Collect will collect the metric for instance
func (c *instanceCountCollector) Collect(ctx context.Context) ([]*metric.Metric, error) {
	var res []*metric.Metric

	instanceCountMetricList, err := c.store.CountInstanceGroupByEngineAndEnvironmentID(ctx, api.Normal)
	if err != nil {
		return nil, err
	}

	for _, instanceCountMetric := range instanceCountMetricList {
		env, err := c.store.GetEnvironmentByID(ctx, instanceCountMetric.EnvironmentID)
		if err != nil {
			c.l.Debug("failed to get environment by id", zap.Int("id", instanceCountMetric.EnvironmentID))
			continue
		}

		res = append(res, &metric.Metric{
			Name:  metricAPI.InstanceCountMetricName,
			Value: instanceCountMetric.Count,
			Labels: map[string]string{
				"engine":      string(instanceCountMetric.Engine),
				"environment": env.Name,
			},
		})
	}

	return res, nil
}
