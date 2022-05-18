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

// issueEventName is the MetricEventName for issue
var issueEventName MetricEventName = "bb.issue"

// NewIssueCollector creates a new instance of issueCollector
func NewIssueCollector(l *zap.Logger, store *store.Store) MetricCollector {
	return &issueCollector{
		l:     l,
		store: store,
	}
}

// Collect will collect the netric for issue
func (c *issueCollector) Collect(ctx context.Context) ([]*Metric, error) {
	var res []*Metric

	issueCountMap, err := c.store.CountIssueGroupByType(ctx, &api.IssueFind{})
	if err != nil {
		return nil, err
	}

	for issueType, count := range issueCountMap {
		res = append(res, &Metric{
			EventName: issueEventName,
			Properties: map[string]interface{}{
				"type":  string(issueType),
				"count": count,
			},
		})
	}

	return res, nil
}
