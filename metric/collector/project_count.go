package collector

import (
	"context"

	metricAPI "github.com/bytebase/bytebase/metric"
	"github.com/bytebase/bytebase/plugin/metric"
	"github.com/bytebase/bytebase/store"
	"go.uber.org/zap"
)

var _ metric.Collector = (*projectCountCollector)(nil)

// projectCountCollector is the metric data collector for project.
type projectCountCollector struct {
	l     *zap.Logger
	store *store.Store
}

// NewProjectCountCollector creates a new instance of projectCollector
func NewProjectCountCollector(l *zap.Logger, store *store.Store) metric.Collector {
	return &projectCountCollector{
		l:     l,
		store: store,
	}
}

// Collect will collect the metric for project
func (c *projectCountCollector) Collect(ctx context.Context) ([]*metric.Metric, error) {
	var res []*metric.Metric

	projectCountMetricList, err := c.store.CountProjectGroupByTenantModeAndWorkflow(ctx)
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
				"status":      projectCountMetric.RowStatus.String(),
			},
		})
	}

	return res, nil
}
