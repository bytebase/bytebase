package collector

import (
	"context"
	"log/slog"

	metricapi "github.com/bytebase/bytebase/backend/metric"
	"github.com/bytebase/bytebase/backend/plugin/metric"
	"github.com/bytebase/bytebase/backend/store"
)

var _ metric.Collector = (*instanceCountCollector)(nil)

// instanceCountCollector is the metric data collector for instance.
type instanceCountCollector struct {
	store *store.Store
}

// NewInstanceCountCollector creates a new instance of instanceCollector.
func NewInstanceCountCollector(store *store.Store) metric.Collector {
	return &instanceCountCollector{
		store: store,
	}
}

// Collect will collect the metric for instance.
func (c *instanceCountCollector) Collect(ctx context.Context) ([]*metric.Metric, error) {
	var res []*metric.Metric

	instanceCountMetricList, err := c.store.CountInstanceGroupByEngineAndEnvironmentID(ctx)
	if err != nil {
		return nil, err
	}

	for _, instanceCountMetric := range instanceCountMetricList {
		env, err := c.store.GetEnvironmentV2(ctx, &store.FindEnvironmentMessage{
			UID: &instanceCountMetric.EnvironmentID,
		})
		if err != nil {
			slog.Debug("failed to get environment by id", slog.Int("id", instanceCountMetric.EnvironmentID))
			continue
		}

		res = append(res, &metric.Metric{
			Name:  metricapi.InstanceCountMetricName,
			Value: instanceCountMetric.Count,
			Labels: map[string]any{
				"engine":      instanceCountMetric.Engine.String(),
				"environment": env.Title,
				"status":      string(instanceCountMetric.RowStatus),
			},
		})
	}

	return res, nil
}
