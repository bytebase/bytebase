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

// projectCollector is the metric data collector for project.
type projectCollector struct {
	l     *zap.Logger
	store *store.Store
}

// NewProjectCollector creates a new instance of projectCollector
func NewProjectCollector(l *zap.Logger, store *store.Store) collector.MetricCollector {
	return &projectCollector{
		l:     l,
		store: store,
	}
}

// Collect will collect the netric for project
func (c *projectCollector) Collect(ctx context.Context) ([]*metric.Metric, error) {
	var res []*metric.Metric

	projectCountMetricList, err := c.store.CountProjectGroupByTenantModeAndWorkflow(ctx, api.Normal)
	if err != nil {
		return nil, err
	}

	for _, projectCountMetric := range projectCountMetricList {
		res = append(res, &metric.Metric{
			Name:  metricAPI.ProjectCountMetricName,
			Value: projectCountMetric.Count,
			Labels: map[string]string{
				"tenant_mode": string(projectCountMetric.TenantMode),
				"workflow":    projectCountMetric.WorkflowType.String(),
			},
		})
	}

	return res, nil
}
