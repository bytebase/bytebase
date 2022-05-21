package collector

import (
	"context"

	metricAPI "github.com/bytebase/bytebase/metric"
	"github.com/bytebase/bytebase/plugin/metric"
	"github.com/bytebase/bytebase/plugin/metric/collector"
	"github.com/bytebase/bytebase/store"
	"go.uber.org/zap"
)

// issueCollector is the metric data collector for issue.
type issueCollector struct {
	l     *zap.Logger
	store *store.Store
}

// NewIssueCollector creates a new instance of issueCollector
func NewIssueCollector(l *zap.Logger, store *store.Store) collector.MetricCollector {
	return &issueCollector{
		l:     l,
		store: store,
	}
}

// Collect will collect the metric for issue
func (c *issueCollector) Collect(ctx context.Context) ([]*metric.Metric, error) {
	var res []*metric.Metric

	issueCountMetricList, err := c.store.CountIssueGroupByType(ctx)
	if err != nil {
		return nil, err
	}

	for _, issueCountMetric := range issueCountMetricList {
		res = append(res, &metric.Metric{
			Name:  metricAPI.IssueCountMetricName,
			Value: issueCountMetric.Count,
			Labels: map[string]string{
				"type": string(issueCountMetric.Type),
			},
		})
	}

	return res, nil
}
