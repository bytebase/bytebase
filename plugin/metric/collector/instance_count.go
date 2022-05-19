package collector

import (
	"context"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/store"
	"go.uber.org/zap"
)

// instanceCollector is the metric data collector for instance.
type instanceCollector struct {
	l     *zap.Logger
	store *store.Store
}

// NewInstanceCollector creates a new instance of instanceCollector
func NewInstanceCollector(l *zap.Logger, store *store.Store) api.MetricCollector {
	return &instanceCollector{
		l:     l,
		store: store,
	}
}

// Collect will collect the netric for instance
func (c *instanceCollector) Collect(ctx context.Context) ([]*api.Metric, error) {
	var res []*api.Metric

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

		res = append(res, &api.Metric{
			Name:  api.InstanceCountMetricName,
			Value: instanceCountMetric.Count,
			Labels: map[string]string{
				"engine":      string(instanceCountMetric.Engine),
				"environment": env.Name,
			},
		})
	}

	return res, nil
}
