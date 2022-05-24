package collector

import (
	"context"

	metricAPI "github.com/bytebase/bytebase/metric"
	"github.com/bytebase/bytebase/plugin/metric"
	"github.com/bytebase/bytebase/store"
	"go.uber.org/zap"
)

func init() {
	store.Register(metricAPI.ProjectCountMetricName, collector{})
}

type collector struct{}

func (collector) Collect(ctx context.Context, l *zap.Logger, s *store.Store) ([]metric.Metric, error) {
	projectCountMetricList, err := s.CountProjectGroupByTenantModeAndWorkflow(ctx)
	if err != nil {
		return nil, err
	}
	res := make([]metric.Metric, 0, len(projectCountMetricList))

	for _, projectCountMetric := range projectCountMetricList {
		res = append(res, projectCountMetric)
	}
	return res, nil
}
