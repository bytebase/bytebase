package collector

import (
	"context"

	"github.com/bytebase/bytebase/api"
	metricAPI "github.com/bytebase/bytebase/metric"
	"github.com/bytebase/bytebase/plugin/metric"
	"github.com/bytebase/bytebase/store"
	"go.uber.org/zap"
)

func init() {
	store.Register(metricAPI.InstanceCountMetricName, collector{})
}

type collector struct{}

func (collector) Collect(ctx context.Context, l *zap.Logger, s *store.Store) ([]metric.Metric, error) {
	instanceCountMetricList, err := s.CountInstanceGroupByEngineAndEnvironmentID(ctx, api.Normal)
	if err != nil {
		return nil, err
	}
	res := make([]metric.Metric, 0, len(instanceCountMetricList))
	for _, instanceCountMetric := range instanceCountMetricList {
		res = append(res, instanceCountMetric)
	}
	return res, nil
}
