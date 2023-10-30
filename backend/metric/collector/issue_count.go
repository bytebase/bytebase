package collector

import (
	"context"

	metricapi "github.com/bytebase/bytebase/backend/metric"
	"github.com/bytebase/bytebase/backend/plugin/metric"
	"github.com/bytebase/bytebase/backend/store"
)

var _ metric.Collector = (*issueCountCollector)(nil)

// issueCountCollector is the metric data collector for issue.
type issueCountCollector struct {
	store *store.Store
}

// NewIssueCountCollector creates a new instance of issueCollector.
func NewIssueCountCollector(store *store.Store) metric.Collector {
	return &issueCountCollector{
		store: store,
	}
}

// Collect will collect the metric for issue.
func (c *issueCountCollector) Collect(ctx context.Context) ([]*metric.Metric, error) {
	var res []*metric.Metric

	issueCountMetricList, err := c.store.CountIssueGroupByTypeAndStatus(ctx)
	if err != nil {
		return nil, err
	}

	for _, issueCountMetric := range issueCountMetricList {
		res = append(res, &metric.Metric{
			Name:  metricapi.IssueCountMetricName,
			Value: issueCountMetric.Count,
			Labels: map[string]any{
				"type":   string(issueCountMetric.Type),
				"status": string(issueCountMetric.Status),
			},
		})
	}

	return res, nil
}
