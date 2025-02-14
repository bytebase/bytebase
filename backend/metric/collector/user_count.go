package collector

import (
	"context"

	metricapi "github.com/bytebase/bytebase/backend/metric"
	"github.com/bytebase/bytebase/backend/plugin/metric"
	"github.com/bytebase/bytebase/backend/store"
)

var _ metric.Collector = (*userCountCollector)(nil)

// userCountCollector is the metric data collector for member.
type userCountCollector struct {
	store *store.Store
}

// NewUserCountCollector creates a new instance of memberCountCollector.
func NewUserCountCollector(store *store.Store) metric.Collector {
	return &userCountCollector{
		store: store,
	}
}

// Collect will collect the metric for issue.
func (c *userCountCollector) Collect(ctx context.Context) ([]*metric.Metric, error) {
	count, err := c.store.CountActiveUsers(ctx)
	if err != nil {
		return nil, err
	}

	return []*metric.Metric{
		{
			Name:  metricapi.UserCountMetricName,
			Value: count,
		},
	}, nil
}
