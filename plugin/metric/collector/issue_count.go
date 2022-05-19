package collector

import (
	"context"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/store"
	"go.uber.org/zap"
)

// issueCollector is the metric data collector for issue.
type issueCollector struct {
	l     *zap.Logger
	store *store.Store
}

// NewIssueCollector creates a new instance of issueCollector
func NewIssueCollector(l *zap.Logger, store *store.Store) api.MetricCollector {
	return &issueCollector{
		l:     l,
		store: store,
	}
}

// Collect will collect the metric for issue
func (c *issueCollector) Collect(ctx context.Context) ([]*api.Metric, error) {
	var res []*api.Metric

	issueCountMetricList, err := c.store.CountIssueGroupByType(ctx)
	if err != nil {
		return nil, err
	}

	for _, issueCountMetric := range issueCountMetricList {
		res = append(res, &api.Metric{
			Name:  api.IssueCountMetricName,
			Value: issueCountMetric.Count,
			Labels: map[string]string{
				"type": string(issueCountMetric.Type),
			},
		})
	}

	return res, nil
}
