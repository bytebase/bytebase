package collector

import (
	"context"

	metricAPI "github.com/bytebase/bytebase/metric"
	"github.com/bytebase/bytebase/plugin/metric"
	"github.com/bytebase/bytebase/store"
)

var _ metric.Collector = (*memberCountCollector)(nil)

// memberCountCollector is the metric data collector for member.
type memberCountCollector struct {
	store *store.Store
}

// NewMemberCountCollector creates a new instance of memberCountCollector
func NewMemberCountCollector(store *store.Store) metric.Collector {
	return &memberCountCollector{
		store: store,
	}
}

// Collect will collect the metric for issue
func (c *memberCountCollector) Collect(ctx context.Context) ([]*metric.Metric, error) {
	var res []*metric.Metric

	memberCountMetricList, err := c.store.CountMemberGroupByRoleAndStatus(ctx)
	if err != nil {
		return nil, err
	}

	for _, memberCountMetric := range memberCountMetricList {
		res = append(res, &metric.Metric{
			Name:  metricAPI.MemberCountMetricName,
			Value: memberCountMetric.Count,
			Labels: map[string]string{
				"role":       memberCountMetric.Role.String(),
				"status":     string(memberCountMetric.Status),
				"row_status": memberCountMetric.RowStatus.String(),
			},
		})
	}

	return res, nil
}
