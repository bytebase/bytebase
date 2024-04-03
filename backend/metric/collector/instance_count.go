package collector

import (
	"context"

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
		res = append(res, &metric.Metric{
			Name:  metricapi.InstanceCountMetricName,
			Value: instanceCountMetric.Count,
			Labels: map[string]any{
				"engine":      instanceCountMetric.Engine.String(),
				"environment": instanceCountMetric.EnvironmentID,
				"status":      string(instanceCountMetric.RowStatus),
			},
		})
	}

	return res, nil
}
