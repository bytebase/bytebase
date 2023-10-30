package collector

import (
	"context"

	metricapi "github.com/bytebase/bytebase/backend/metric"
	"github.com/bytebase/bytebase/backend/plugin/metric"
	"github.com/bytebase/bytebase/backend/store"
)

var _ metric.Collector = (*memberCountCollector)(nil)

// memberCountCollector is the metric data collector for member.
type memberCountCollector struct {
	store *store.Store
}

// NewMemberCountCollector creates a new instance of memberCountCollector.
func NewMemberCountCollector(store *store.Store) metric.Collector {
	return &memberCountCollector{
		store: store,
	}
}

// Collect will collect the metric for issue.
func (c *memberCountCollector) Collect(ctx context.Context) ([]*metric.Metric, error) {
	var res []*metric.Metric

	memberCountMetricList, err := c.store.CountMemberGroupByRoleAndStatus(ctx)
	if err != nil {
		return nil, err
	}

	for _, memberCountMetric := range memberCountMetricList {
		res = append(res, &metric.Metric{
			Name:  metricapi.MemberCountMetricName,
			Value: memberCountMetric.Count,
			Labels: map[string]any{
				"role":       string(memberCountMetric.Role),
				"status":     string(memberCountMetric.Status),
				"row_status": string(memberCountMetric.RowStatus),
				"type":       string(memberCountMetric.Type),
			},
		})
	}

	return res, nil
}
