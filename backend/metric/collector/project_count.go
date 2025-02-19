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
	count, err := c.store.CountProjects(ctx)
	if err != nil {
		return nil, err
	}
	return []*metric.Metric{
		{
			Name:  metricapi.ProjectCountMetricName,
			Value: count,
		},
	}, nil
}
