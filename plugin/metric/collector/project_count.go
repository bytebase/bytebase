package collector

import (
	"context"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/store"
	"go.uber.org/zap"
)

// projectCollector is the metric data collector for project.
type projectCollector struct {
	l     *zap.Logger
	store *store.Store
}

// NewProjectCollector creates a new instance of projectCollector
func NewProjectCollector(l *zap.Logger, store *store.Store) MetricCollector {
	return &projectCollector{
		l:     l,
		store: store,
	}
}

// Collect will collect the netric for project
func (c *projectCollector) Collect(ctx context.Context) ([]*Metric, error) {
	var res []*Metric

	projectCountMetricList, err := c.store.CountProjectGroupByTenantModeAndWorkflow(ctx, api.Normal)
	if err != nil {
		return nil, err
	}

	for _, projectCountMetric := range projectCountMetricList {
		res = append(res, &Metric{
			Name:  projectCountMetricName,
			Value: projectCountMetric.Count,
			Labels: map[string]string{
				"tenant_mode": string(projectCountMetric.TenantMode),
				"workflow":    projectCountMetric.WorkflowType.String(),
			},
		})
	}

	return res, nil
}
