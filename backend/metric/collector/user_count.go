package collector

import (
	"context"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	metricapi "github.com/bytebase/bytebase/backend/metric"
	"github.com/bytebase/bytebase/backend/plugin/metric"
	"github.com/bytebase/bytebase/backend/store"
)

var _ metric.Collector = (*userCountCollector)(nil)

// userCountCollector is the metric data collector for user.
type userCountCollector struct {
	store *store.Store
}

// NewUserCountCollector creates a new instance of userCountCollector.
func NewUserCountCollector(store *store.Store) metric.Collector {
	return &userCountCollector{
		store: store,
	}
}

// Collect will collect the metric for user.
func (c *userCountCollector) Collect(ctx context.Context) ([]*metric.Metric, error) {
	userStat, err := c.store.StatUsers(ctx)
	if err != nil {
		return nil, err
	}

	metrics := []*metric.Metric{}
	for _, stat := range userStat {
		if stat.Type == storepb.PrincipalType_END_USER {
			metrics = append(metrics, &metric.Metric{
				Name:  metricapi.UserCountMetricName,
				Value: stat.Count,
			})
		}
		if stat.Type == storepb.PrincipalType_SERVICE_ACCOUNT {
			metrics = append(metrics, &metric.Metric{
				Name:  metricapi.ServiceAccountCountMetricName,
				Value: stat.Count,
			})
		}
	}

	return metrics, nil
}
