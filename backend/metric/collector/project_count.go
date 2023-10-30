package collector

import (
	"context"

	metricapi "github.com/bytebase/bytebase/backend/metric"
	"github.com/bytebase/bytebase/backend/plugin/metric"
	"github.com/bytebase/bytebase/backend/store"
)

var _ metric.Collector = (*projectCountCollector)(nil)

// projectCountCollector is the metric data collector for project.
type projectCountCollector struct {
	store *store.Store
}

// NewProjectCountCollector creates a new instance of projectCollector.
func NewProjectCountCollector(store *store.Store) metric.Collector {
	return &projectCountCollector{
		store: store,
	}
}

// Collect will collect the metric for project.
func (c *projectCountCollector) Collect(ctx context.Context) ([]*metric.Metric, error) {
	var res []*metric.Metric

	projectCountMetricList, err := c.store.CountProjectGroupByTenantModeAndWorkflow(ctx)
	if err != nil {
		return nil, err
	}

	for _, projectCountMetric := range projectCountMetricList {
		res = append(res, &metric.Metric{
			Name:  metricapi.ProjectCountMetricName,
			Value: projectCountMetric.Count,
			Labels: map[string]any{
				"tenant_mode": string(projectCountMetric.TenantMode),
				"workflow":    string(projectCountMetric.WorkflowType),
				"status":      string(projectCountMetric.RowStatus),
			},
		})
	}

	return res, nil
}
