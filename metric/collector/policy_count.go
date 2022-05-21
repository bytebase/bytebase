package collector

import (
	"context"

	"github.com/bytebase/bytebase/api"
	metricAPI "github.com/bytebase/bytebase/metric"
	"github.com/bytebase/bytebase/plugin/metric"
	"github.com/bytebase/bytebase/plugin/metric/collector"
	"github.com/bytebase/bytebase/store"
	"go.uber.org/zap"
)

// policyCollector is the metric data collector for policy.
type policyCollector struct {
	l     *zap.Logger
	store *store.Store
}

// NewPolicyCollector creates a new instance of policyCollector
func NewPolicyCollector(l *zap.Logger, store *store.Store) collector.MetricCollector {
	return &policyCollector{
		l:     l,
		store: store,
	}
}

// Collect will collect the metric for policy
func (c *policyCollector) Collect(ctx context.Context) ([]*metric.Metric, error) {
	var res []*metric.Metric

	policyCountMetricList, err := c.store.CountPolicyGroupByTypeAndEnvironmentID(ctx, api.Normal)
	if err != nil {
		return nil, err
	}

	for _, policyCountMetric := range policyCountMetricList {
		env, err := c.store.GetEnvironmentByID(ctx, policyCountMetric.EnvironmentID)
		if err != nil {
			c.l.Debug("failed to get environment by id", zap.Int("id", policyCountMetric.EnvironmentID))
			continue
		}

		res = append(res, &metric.Metric{
			Name:  metricAPI.PolicyCountMetricName,
			Value: policyCountMetric.Count,
			Labels: map[string]string{
				"type":        string(policyCountMetric.Type),
				"environment": env.Name,
			},
		})
	}

	return res, nil
}
